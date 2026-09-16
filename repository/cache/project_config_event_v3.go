package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/configclock"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/configmetrics"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
)

const projectConfigChangeStreamV3 = "feedback:v3:project_config_changed"

type ProjectConfigEventBusV3 interface {
	Publish(context.Context, domain.ConfigEventV3) (string, error)
	Consume(context.Context, func(context.Context, domain.ConfigEventV3) error, func(context.Context) error)
	Heartbeat(context.Context) error
	Maintain(context.Context) (bool, error)
	Close() error
}
type redisProjectConfigEventBusV3 struct {
	client              *redis.Client
	reader              *redis.Client
	log                 logger.Logger
	cfg                 *config.V3ConfigCacheConfig
	clock               configclock.Clock
	metrics             *configmetrics.Metrics
	stream, group, name string
}

func NewProjectConfigEventBusV3(client *redis.Client, log logger.Logger, cfg *config.V3ConfigCacheConfig, clock configclock.Clock, m *configmetrics.Metrics, instance *domain.ConfigInstanceV3) ProjectConfigEventBusV3 {
	opts := *client.Options()
	opts.MaxRetries = -1
	opts.ReadTimeout = 2 * time.Second
	opts.WriteTimeout = 2 * time.Second
	opts.DialTimeout = 2 * time.Second
	return &redisProjectConfigEventBusV3{client: client, reader: redis.NewClient(&opts), log: log, cfg: cfg, clock: clock, metrics: m, stream: cfg.StreamKey, group: "feedback-v3-config-refresh-v2:" + instance.ID, name: instance.ID}
}
func (b *redisProjectConfigEventBusV3) registry() string { return b.stream + ":v2:groups" }
func (b *redisProjectConfigEventBusV3) lease(group string) string {
	return b.stream + ":v2:lease:" + group
}
func (b *redisProjectConfigEventBusV3) Close() error { return b.reader.Close() }
func (b *redisProjectConfigEventBusV3) Publish(ctx context.Context, e domain.ConfigEventV3) (string, error) {
	return b.client.XAdd(ctx, &redis.XAddArgs{Stream: b.stream, MaxLen: b.cfg.StreamMaxLen, Approx: true, Values: map[string]any{
		"project_id": e.ProjectID, "changed_at": strconv.FormatInt(e.ChangedAt.Unix(), 10), "config_version": strconv.FormatUint(e.Version, 10), "change_id": e.ChangeID, "kind": e.Kind, "schema_version": "1",
	}}).Result()
}

var createConfigGroup = redis.NewScript(`
local r=redis.pcall('XGROUP','CREATE',KEYS[1],ARGV[1],'$','MKSTREAM')
if type(r)=='table' and r.err and not string.find(r.err,'BUSYGROUP',1,true) then return redis.error_reply(r.err) end
redis.call('SET',KEYS[2],'1','PX',ARGV[2])
redis.call('SADD',KEYS[3],ARGV[1])
return 1
`)
var renewConfigGroup = redis.NewScript(`
if redis.call('SISMEMBER',KEYS[2],ARGV[1])==0 then return 0 end
redis.call('SET',KEYS[1],'1','PX',ARGV[2])
return 1
`)
var cleanConfigGroup = redis.NewScript(`
if redis.call('EXISTS',KEYS[2])==1 then return 0 end
redis.call('XGROUP','DESTROY',KEYS[1],ARGV[1])
redis.call('SREM',KEYS[3],ARGV[1])
return 1
`)

func (b *redisProjectConfigEventBusV3) Heartbeat(ctx context.Context) error {
	return renewConfigGroup.Run(ctx, b.client, []string{b.lease(b.group), b.registry()}, b.group, b.cfg.GroupLease.Milliseconds()).Err()
}
func parseConfigEvent(m redis.XMessage) (domain.ConfigEventV3, error) {
	e := domain.ConfigEventV3{MessageID: m.ID}
	var ok bool
	e.ProjectID, ok = m.Values["project_id"].(string)
	if !ok || strings.TrimSpace(e.ProjectID) == "" || len(e.ProjectID) > 64 {
		return e, errors.New("invalid_project")
	}
	if v, exists := m.Values["schema_version"]; exists && v != "1" {
		return e, errors.New("invalid_schema")
	}
	if value, exists := m.Values["config_version"]; exists {
		s, ok := value.(string)
		if !ok {
			return e, errors.New("invalid_version")
		}
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil || v == 0 {
			return e, errors.New("invalid_version")
		}
		e.Version = v
	}
	e.ChangeID, _ = m.Values["change_id"].(string)
	e.Kind, _ = m.Values["kind"].(string)
	if t, ok := m.Values["changed_at"].(string); ok {
		seconds, err := strconv.ParseInt(t, 10, 64)
		if err == nil {
			e.ChangedAt = time.Unix(seconds, 0)
		}
	}
	return e, nil
}

func (b *redisProjectConfigEventBusV3) process(ctx context.Context, items []redis.XStream, handler func(context.Context, domain.ConfigEventV3) error, reconcile func(context.Context) error) (bool, string) {
	var failed atomic.Bool
	var wg sync.WaitGroup
	slots := make(chan struct{}, b.cfg.Concurrency)
	var reconcileOnce sync.Once
	var reconcileErr error
	last := ""
messageLoop:
	for _, stream := range items {
		for _, msg := range stream.Messages {
			if ctx.Err() != nil {
				failed.Store(true)
				break
			}
			last = msg.ID
			select {
			case slots <- struct{}{}:
			case <-ctx.Done():
				failed.Store(true)
				break messageLoop
			}
			wg.Add(1)
			go func(msg redis.XMessage) {
				defer wg.Done()
				defer func() { <-slots }()
				b.metrics.Events.WithLabelValues("received").Inc()
				b.log.Info("event_received", logger.String("message_id", msg.ID))
				e, err := parseConfigEvent(msg)
				if err != nil {
					b.metrics.Events.WithLabelValues("invalid").Inc()
					b.log.Warn("config_event_invalid", logger.String("message_id", msg.ID), logger.String("error_class", err.Error()))
					// Empty bodies include entries trimmed while still referenced by this group's PEL.
					reconcileOnce.Do(func() { reconcileErr = reconcile(ctx) })
					if reconcileErr != nil {
						failed.Store(true)
						return
					}
				} else if err = handler(ctx, e); err != nil {
					failed.Store(true)
					b.metrics.Events.WithLabelValues("retry").Inc()
					return
				}
				ackCtx, cancel := context.WithTimeout(ctx, b.cfg.LoadTimeout)
				err = b.client.XAck(ackCtx, b.stream, b.group, msg.ID).Err()
				cancel()
				if err != nil {
					failed.Store(true)
					b.metrics.Events.WithLabelValues("ack_failed").Inc()
				} else {
					b.metrics.Events.WithLabelValues("acked").Inc()
					b.log.Info("event_acked", logger.String("message_id", msg.ID))
				}
			}(msg)
		}
	}
	wg.Wait()
	return failed.Load(), last
}
func (b *redisProjectConfigEventBusV3) Consume(ctx context.Context, handler func(context.Context, domain.ConfigEventV3) error, reconcile func(context.Context) error) {
	defer b.metrics.Consumer.Set(0)
	stop := context.AfterFunc(ctx, func() { _ = b.reader.Close() })
	defer stop()
	ready := false
	pendingCursor := "0"
	attempt := 0
	for ctx.Err() == nil {
		if !ready {
			op, cancel := context.WithTimeout(ctx, b.cfg.LoadTimeout)
			err := createConfigGroup.Run(op, b.client, []string{b.stream, b.lease(b.group), b.registry()}, b.group, b.cfg.GroupLease.Milliseconds()).Err()
			cancel()
			if err != nil {
				b.metrics.Consumer.Set(0)
				b.log.Warn("config_consumer_start_failed", logger.String("error_class", domain.ConfigErrorClass(err)))
				if !configclock.Wait(ctx, b.clock, configclock.Backoff(b.cfg.RetryMin, b.cfg.RetryMax, attempt)) {
					return
				}
				attempt++
				continue
			}
			if err = reconcile(ctx); err != nil {
				if !configclock.Wait(ctx, b.clock, configclock.Backoff(b.cfg.RetryMin, b.cfg.RetryMax, attempt)) {
					return
				}
				attempt++
				continue
			}
			ready = true
			b.metrics.Consumer.Set(1)
			pendingCursor = "0"
			attempt = 0
		}
		failed := false
		processed := false
		// One PEL page and one new page per pass preserve fairness when a project keeps failing.
		for _, position := range []string{pendingCursor, ">"} {
			block := time.Duration(-1)
			if position == ">" {
				block = time.Second
			}
			op, cancel := context.WithTimeout(ctx, 2*time.Second)
			items, err := b.reader.XReadGroup(op, &redis.XReadGroupArgs{Group: b.group, Consumer: b.name, Streams: []string{b.stream, position}, Count: 20, Block: block}).Result()
			cancel()
			if errors.Is(err, redis.Nil) {
				if position != ">" {
					pendingCursor = "0"
				}
				continue
			}
			if err != nil {
				failed = true
				b.metrics.Consumer.Set(0)
				b.metrics.Events.WithLabelValues("read_failed").Inc()
				if strings.Contains(err.Error(), "NOGROUP") {
					ready = false
					break
				}
				continue
			}
			b.metrics.Consumer.Set(1)
			bad, last := b.process(ctx, items, handler, reconcile)
			processed = processed || last != ""
			failed = failed || bad
			if position != ">" {
				if last == "" {
					pendingCursor = "0"
				} else {
					pendingCursor = last
				}
			}
		}
		if failed {
			if !configclock.Wait(ctx, b.clock, configclock.Backoff(b.cfg.RetryMin, b.cfg.RetryMax, attempt)) {
				return
			}
			attempt++
		} else if processed {
			attempt = 0
		}
	}
}

type configGroupInfo struct {
	name, last   string
	pending, lag int64
	exact        bool
}

func configGroups(raw any) ([]configGroupInfo, error) {
	rows, ok := raw.([]interface{})
	if !ok {
		return nil, errors.New("invalid_group_response")
	}
	var groups []configGroupInfo
	for _, row := range rows {
		fields, ok := row.([]interface{})
		if !ok || len(fields)%2 != 0 {
			return nil, errors.New("invalid_group_fields")
		}
		g := configGroupInfo{}
		for i := 0; i < len(fields); i += 2 {
			key, _ := fields[i].(string)
			v := fields[i+1]
			switch key {
			case "name":
				g.name, _ = v.(string)
			case "last-delivered-id":
				g.last, _ = v.(string)
			case "pending":
				g.pending, _ = v.(int64)
			case "lag":
				g.lag, g.exact = v.(int64)
			}
		}
		if g.name == "" || g.last == "" {
			return nil, errors.New("missing_group_fields")
		}
		groups = append(groups, g)
	}
	return groups, nil
}
func (b *redisProjectConfigEventBusV3) Maintain(ctx context.Context) (bool, error) {
	b.metrics.StreamStatsUp.Set(0)
	// Only groups recorded by this version are eligible for lease-based cleanup.
	var cursor uint64
	for {
		groups, next, err := b.client.SScan(ctx, b.registry(), cursor, "*", 100).Result()
		if err != nil {
			return false, err
		}
		for _, group := range groups {
			if !strings.HasPrefix(group, "feedback-v3-config-refresh-v2:") {
				continue
			}
			if err = cleanConfigGroup.Run(ctx, b.client, []string{b.stream, b.lease(group), b.registry()}, group).Err(); err != nil {
				return false, err
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	raw, err := b.client.Do(ctx, "XINFO", "GROUPS", b.stream).Result()
	if err != nil {
		b.metrics.StreamStatsUp.Set(0)
		return false, err
	}
	groups, err := configGroups(raw)
	if err != nil {
		b.metrics.StreamStatsUp.Set(0)
		return false, err
	}
	for _, g := range groups {
		if g.name == b.group {
			first, err := b.client.XRangeN(ctx, b.stream, "-", "+", 1).Result()
			if err != nil {
				return false, err
			}
			gap := len(first) > 0 && g.last != "0-0" && streamIDBefore(g.last, first[0].ID)
			if !g.exact {
				rows, err := b.client.XRangeN(ctx, b.stream, g.last, "+", b.cfg.BacklogThreshold+2).Result()
				if err != nil {
					b.metrics.StreamStatsUp.Set(0)
					return false, err
				}
				g.lag = int64(len(rows))
				if len(rows) > 0 && rows[0].ID == g.last {
					g.lag--
				}
				g.exact = false
			}
			b.metrics.Pending.Set(float64(g.pending))
			b.metrics.Unread.Set(float64(g.lag))
			b.metrics.UnreadExact.Set(0)
			if g.exact {
				b.metrics.UnreadExact.Set(1)
			}
			b.metrics.OldestPending.Set(0)
			pending, err := b.client.XPending(ctx, b.stream, b.group).Result()
			if err != nil {
				b.metrics.StreamStatsUp.Set(0)
				return false, err
			}
			if pending.Count > 0 {
				parts := strings.SplitN(pending.Lower, "-", 2)
				ms, err := strconv.ParseInt(parts[0], 10, 64)
				if err == nil {
					b.metrics.OldestPending.Set(max(0, b.clock.Now().Sub(time.UnixMilli(ms)).Seconds()))
				}
			}
			b.metrics.StreamStatsUp.Set(1)
			return gap || g.pending+g.lag >= b.cfg.BacklogThreshold, nil
		}
	}
	b.metrics.StreamStatsUp.Set(0)
	return false, fmt.Errorf("config group unavailable")
}

func streamIDBefore(a, b string) bool {
	x, y := strings.SplitN(a, "-", 2), strings.SplitN(b, "-", 2)
	if len(x) != 2 || len(y) != 2 {
		return false
	}
	xm, xe := strconv.ParseUint(x[0], 10, 64)
	ym, ye := strconv.ParseUint(y[0], 10, 64)
	if xe != nil || ye != nil {
		return false
	}
	if xm != ym {
		return xm < ym
	}
	xs, xe := strconv.ParseUint(x[1], 10, 64)
	ys, ye := strconv.ParseUint(y[1], 10, 64)
	return xe == nil && ye == nil && xs < ys
}
