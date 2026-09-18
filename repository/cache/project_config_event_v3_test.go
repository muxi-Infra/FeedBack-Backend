package cache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/internal/testclock"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/configclock"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/configmetrics"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newConfigBusTest(t *testing.T, client *redis.Client, key string, clock configclock.Clock) *redisProjectConfigEventBusV3 {
	t.Helper()
	cfg := config.DefaultV3ConfigCacheConfig()
	cfg.StreamKey = key
	b := NewProjectConfigEventBusV3(client, logger.NewZapLogger(zap.NewNop()), cfg, clock, configmetrics.New(prometheus.NewRegistry()), domain.NewConfigInstanceV3()).(*redisProjectConfigEventBusV3)
	t.Cleanup(func() { _ = b.Close() })
	return b
}
func configSignal(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(8 * time.Second):
		t.Fatal("stream synchronization timed out")
	}
}
func consumeConfigTest(t *testing.T, b *redisProjectConfigEventBusV3, handler func(context.Context, domain.ConfigEventV3) error) (chan struct{}, chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	ready, done := make(chan struct{}, 10), make(chan struct{})
	go func() {
		defer close(done)
		b.Consume(ctx, handler, func(context.Context) error { ready <- struct{}{}; return nil })
	}()
	t.Cleanup(func() { cancel(); _ = b.Close(); configSignal(t, done) })
	return ready, done
}

type configCommandHook struct {
	before func(redis.Cmder) error
	after  func(redis.Cmder)
}

func (h configCommandHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	if h.before != nil {
		return ctx, h.before(cmd)
	}
	return ctx, nil
}
func (h configCommandHook) AfterProcess(_ context.Context, cmd redis.Cmder) error {
	if h.after != nil {
		h.after(cmd)
	}
	return nil
}
func (configCommandHook) BeforeProcessPipeline(ctx context.Context, _ []redis.Cmder) (context.Context, error) {
	return ctx, nil
}
func (configCommandHook) AfterProcessPipeline(context.Context, []redis.Cmder) error { return nil }

func TestV3ConfigBroadcastToIndependentGroups(t *testing.T) {
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	key := "test:config:" + uuid.NewString()
	a := newConfigBusTest(t, client, key, configclock.New())
	b := newConfigBusTest(t, client, key, configclock.New())
	require.NotEqual(t, a.group, b.group)
	aSeen, bSeen := make(chan struct{}, 1), make(chan struct{}, 1)
	aReady, _ := consumeConfigTest(t, a, func(_ context.Context, e domain.ConfigEventV3) error {
		if e.Version == 2 {
			aSeen <- struct{}{}
		}
		return nil
	})
	bReady, _ := consumeConfigTest(t, b, func(_ context.Context, e domain.ConfigEventV3) error {
		if e.Version == 2 {
			bSeen <- struct{}{}
		}
		return nil
	})
	configSignal(t, aReady)
	configSignal(t, bReady)
	_, err := a.Publish(context.Background(), domain.ConfigEventV3{ProjectID: "fictional", Version: 2, ChangeID: uuid.NewString(), ChangedAt: time.Now(), Kind: "update"})
	require.NoError(t, err)
	configSignal(t, aSeen)
	configSignal(t, bSeen)
}

func TestV3ConfigRecoversHandlerAndAckFailures(t *testing.T) {
	for _, failure := range []string{"handler", "ack", "read"} {
		t.Run(failure, func(t *testing.T) {
			m := miniredis.RunT(t)
			client := redis.NewClient(&redis.Options{Addr: m.Addr(), MaxRetries: -1})
			t.Cleanup(func() { _ = client.Close() })
			clock := testclock.New()
			b := newConfigBusTest(t, client, "test:config:"+uuid.NewString(), clock)
			var calls atomic.Int32
			var once atomic.Bool
			acked := make(chan struct{}, 10)
			client.AddHook(configCommandHook{before: func(cmd redis.Cmder) error {
				if failure == "ack" && cmd.Name() == "xack" && once.CompareAndSwap(false, true) {
					return errors.New("injected_ack_failure")
				}
				return nil
			}, after: func(cmd redis.Cmder) {
				if cmd.Name() == "xack" && cmd.Err() == nil {
					acked <- struct{}{}
				}
			}})
			b.reader.AddHook(configCommandHook{before: func(cmd redis.Cmder) error {
				if failure == "read" && cmd.Name() == "xreadgroup" && once.CompareAndSwap(false, true) {
					return errors.New("injected_read_failure")
				}
				return nil
			}})
			ready, _ := consumeConfigTest(t, b, func(context.Context, domain.ConfigEventV3) error {
				n := calls.Add(1)
				if failure == "handler" && n == 1 {
					return errors.New("injected_load_failure")
				}
				return nil
			})
			configSignal(t, ready)
			_, err := b.Publish(context.Background(), domain.ConfigEventV3{ProjectID: "fictional", Version: 1, ChangedAt: clock.Now()})
			require.NoError(t, err)
			configSignal(t, clock.Created)
			clock.Advance(30 * time.Second)
			configSignal(t, acked)
			pending, err := client.XPending(context.Background(), b.stream, b.group).Result()
			require.NoError(t, err)
			require.Zero(t, pending.Count)
			if failure != "read" {
				require.GreaterOrEqual(t, calls.Load(), int32(2))
			}
		})
	}
}

func TestV3ConfigStartupRedisRecovery(t *testing.T) {
	m := miniredis.RunT(t)
	address := m.Addr()
	m.Close()
	client := redis.NewClient(&redis.Options{Addr: address, MaxRetries: -1, DialTimeout: 100 * time.Millisecond})
	t.Cleanup(func() { _ = client.Close() })
	clock := testclock.New()
	b := newConfigBusTest(t, client, "test:config:"+uuid.NewString(), clock)
	seen := make(chan struct{}, 1)
	ready, _ := consumeConfigTest(t, b, func(context.Context, domain.ConfigEventV3) error { seen <- struct{}{}; return nil })
	configSignal(t, clock.Created)
	recovered := miniredis.NewMiniRedis()
	require.NoError(t, recovered.StartAddr(address))
	t.Cleanup(recovered.Close)
	clock.Advance(30 * time.Second)
	configSignal(t, ready)
	_, err := b.Publish(context.Background(), domain.ConfigEventV3{ProjectID: "fictional", Version: 2, ChangedAt: clock.Now()})
	require.NoError(t, err)
	configSignal(t, seen)
}

func TestV3ConfigGroupLeaseCleanupAndLegacyEvent(t *testing.T) {
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	b := newConfigBusTest(t, client, "test:config:"+uuid.NewString(), configclock.New())
	ctx := context.Background()
	for _, group := range []string{b.group, "feedback-v3-config-refresh-v2:dead", "legacy-group"} {
		require.NoError(t, client.XGroupCreateMkStream(ctx, b.stream, group, "$").Err())
	}
	require.NoError(t, client.SAdd(ctx, b.registry(), b.group, "feedback-v3-config-refresh-v2:dead").Err())
	require.NoError(t, b.Heartbeat(ctx))
	_, err := b.Maintain(ctx)
	require.NoError(t, err)
	raw, err := client.Do(ctx, "XINFO", "GROUPS", b.stream).Result()
	require.NoError(t, err)
	groups, err := configGroups(raw)
	require.NoError(t, err)
	require.Len(t, groups, 2)
	m.FastForward(b.cfg.GroupLease)
	_, _ = b.Maintain(ctx)
	raw, err = client.Do(ctx, "XINFO", "GROUPS", b.stream).Result()
	require.NoError(t, err)
	groups, err = configGroups(raw)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, "legacy-group", groups[0].name)
	e, err := parseConfigEvent(redis.XMessage{ID: "1-0", Values: map[string]any{"project_id": "fictional", "changed_at": "1800000000"}})
	require.NoError(t, err)
	require.Zero(t, e.Version)
}

func TestV3ConfigGroupResponseCompatibility(t *testing.T) {
	for _, tt := range []struct {
		name  string
		tail  []any
		lag   int64
		exact bool
	}{
		{"missing lag", nil, 0, false},
		{"known lag", []any{"entries-read", int64(1), "lag", int64(9)}, 9, true},
		{"unknown lag", []any{"entries-read", nil, "lag", nil}, 0, false},
		{"zero lag and future field", []any{"lag", int64(0), "future", int64(1)}, 0, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fields := append([]any{"name", "g", "consumers", int64(1), "pending", int64(2), "last-delivered-id", "1-0"}, tt.tail...)
			g, err := configGroups([]any{fields})
			require.NoError(t, err)
			require.Len(t, g, 1)
			require.Equal(t, int64(2), g[0].pending)
			require.Equal(t, tt.lag, g[0].lag)
			require.Equal(t, tt.exact, g[0].exact)
		})
	}
}

func TestV3ConfigTrimmedPendingRequiresReconcileBeforeAck(t *testing.T) {
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	b := newConfigBusTest(t, client, "test:config:"+uuid.NewString(), configclock.New())
	ctx := context.Background()
	require.NoError(t, client.XGroupCreateMkStream(ctx, b.stream, b.group, "0").Err())
	id, err := b.Publish(ctx, domain.ConfigEventV3{ProjectID: "fictional", Version: 1})
	require.NoError(t, err)
	_, err = client.XReadGroup(ctx, &redis.XReadGroupArgs{Group: b.group, Consumer: b.name, Streams: []string{b.stream, ">"}}).Result()
	require.NoError(t, err)
	items := []redis.XStream{{Stream: b.stream, Messages: []redis.XMessage{{ID: id, Values: nil}}}}
	var reconciled atomic.Int32
	failed, _ := b.process(ctx, items, func(context.Context, domain.ConfigEventV3) error {
		t.Fatal("missing body must not call project handler")
		return nil
	}, func(context.Context) error { reconciled.Add(1); return errors.New("db unavailable") })
	require.True(t, failed)
	failed, _ = b.process(ctx, items, nil, func(context.Context) error { reconciled.Add(1); return nil })
	require.False(t, failed)
	pending, err := client.XPending(ctx, b.stream, b.group).Result()
	require.NoError(t, err)
	require.Zero(t, pending.Count)
	require.Equal(t, int32(2), reconciled.Load())
	require.Equal(t, float64(1), testutil.ToFloat64(b.metrics.Events.WithLabelValues("acked")))
}

func TestV3ConfigFailingProjectDoesNotBlockAnotherProject(t *testing.T) {
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	b := newConfigBusTest(t, client, "test:config:"+uuid.NewString(), configclock.New())
	blocked, release, healthy := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var blockedOnce, releaseOnce, healthyOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	require.NoError(t, createConfigGroup.Run(context.Background(), client, []string{b.stream, b.lease(b.group), b.registry()}, b.group, 120000).Err())
	// Queue both entries before starting the reader so they form one complete batch.
	p := client.Pipeline()
	for _, id := range []string{"failing", "healthy"} {
		p.XAdd(context.Background(), &redis.XAddArgs{Stream: b.stream, Values: map[string]any{"project_id": id, "config_version": "1"}})
	}
	_, err := p.Exec(context.Background())
	require.NoError(t, err)
	ready, _ := consumeConfigTest(t, b, func(_ context.Context, e domain.ConfigEventV3) error {
		if e.ProjectID == "failing" {
			blockedOnce.Do(func() { close(blocked) })
			<-release
			return errors.New("load failure")
		}
		healthyOnce.Do(func() { close(healthy) })
		return nil
	})
	configSignal(t, ready)
	configSignal(t, blocked)
	configSignal(t, healthy)
	releaseOnce.Do(func() { close(release) })
}

type observingConfigClock struct {
	*testclock.Clock
	delays chan time.Duration
}

func (c observingConfigClock) NewTimer(d time.Duration) configclock.Timer {
	t := c.Clock.NewTimer(d)
	c.delays <- d
	return t
}
func TestV3ConfigPendingRetriesKeepBackoffAcrossEmptyPages(t *testing.T) {
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	clock := observingConfigClock{Clock: testclock.New(), delays: make(chan time.Duration, 10)}
	b := newConfigBusTest(t, client, "test:config:"+uuid.NewString(), clock)
	ctx := context.Background()
	require.NoError(t, createConfigGroup.Run(ctx, client, []string{b.stream, b.lease(b.group), b.registry()}, b.group, 120000).Err())
	_, err := b.Publish(ctx, domain.ConfigEventV3{ProjectID: "failing", Version: 1, ChangedAt: clock.Now()})
	require.NoError(t, err)
	_, err = client.XReadGroup(ctx, &redis.XReadGroupArgs{Group: b.group, Consumer: b.name, Streams: []string{b.stream, ">"}, Count: 1, Block: -1}).Result()
	require.NoError(t, err)
	b.reader.AddHook(configCommandHook{before: func(cmd redis.Cmder) error {
		args := cmd.Args()
		if cmd.Name() == "xreadgroup" && args[len(args)-1] == ">" {
			return redis.Nil
		}
		return nil
	}})
	ready, _ := consumeConfigTest(t, b, func(context.Context, domain.ConfigEventV3) error { return errors.New("retry") })
	configSignal(t, ready)
	for i := 0; i < 5; i++ {
		select {
		case d := <-clock.delays:
			base := b.cfg.RetryMin * time.Duration(1<<i)
			require.GreaterOrEqual(t, d, base/2)
			require.LessOrEqual(t, d, base)
			clock.Advance(d)
		case <-time.After(5 * time.Second):
			t.Fatal("missing retry timer")
		}
	}
}

func TestV3ConfigReconcileFailureDoesNotBlockConsumption(t *testing.T) {
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	assertConfigConsumptionAfterReconcileFailure(t, client, "test:config:"+uuid.NewString())
}

func assertConfigConsumptionAfterReconcileFailure(t *testing.T, client *redis.Client, key string) {
	t.Helper()
	b := newConfigBusTest(t, client, key, configclock.New())
	ctx, cancel := context.WithCancel(context.Background())
	done, reconciled, acked := make(chan struct{}), make(chan struct{}, 2), make(chan struct{}, 2)
	client.AddHook(configCommandHook{after: func(cmd redis.Cmder) {
		if cmd.Name() == "xack" && cmd.Err() == nil {
			acked <- struct{}{}
		}
	}})
	go func() {
		defer close(done)
		b.Consume(ctx, func(context.Context, domain.ConfigEventV3) error { return nil }, func(context.Context) error {
			reconciled <- struct{}{}
			return errors.New("one project failed to load")
		})
	}()
	t.Cleanup(func() { cancel(); _ = b.Close(); configSignal(t, done) })
	for i := 0; i < 2; i++ {
		configSignal(t, reconciled)
		_, err := b.Publish(ctx, domain.ConfigEventV3{ProjectID: "healthy", Version: uint64(i + 1)})
		require.NoError(t, err)
		configSignal(t, acked)
		if i == 0 {
			require.NoError(t, client.XGroupDestroy(ctx, b.stream, b.group).Err())
		}
	}
}

func TestV3ConfigPendingSweepFinishesDespiteNewFailures(t *testing.T) {
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	assertConfigPendingSweepFinishes(t, client, "test:config:"+uuid.NewString())
}

func assertConfigPendingSweepFinishes(t *testing.T, client *redis.Client, key string) {
	t.Helper()
	clock := observingConfigClock{Clock: testclock.New(), delays: make(chan time.Duration, 1)}
	b := newConfigBusTest(t, client, key, clock)
	var batch int
	var oldCalls atomic.Int32
	var recovered atomic.Bool
	b.reader.AddHook(configCommandHook{before: func(cmd redis.Cmder) error {
		args := cmd.Args()
		if cmd.Name() != "xreadgroup" || args[len(args)-1] != ">" {
			return nil
		}
		batch++
		pipe := client.Pipeline()
		for i := 0; i < 20; i++ {
			pipe.XAdd(context.Background(), &redis.XAddArgs{Stream: b.stream, Values: map[string]any{"project_id": fmt.Sprintf("batch-%d", batch), "config_version": "1"}})
		}
		_, err := pipe.Exec(context.Background())
		return err
	}})
	ready, _ := consumeConfigTest(t, b, func(_ context.Context, e domain.ConfigEventV3) error {
		if e.ProjectID == "batch-1" {
			oldCalls.Add(1)
			if recovered.Load() {
				return nil
			}
		}
		return errors.New("injected load failure")
	})
	configSignal(t, ready)
	for round := 0; round < 6; round++ {
		select {
		case delay := <-clock.delays:
			if round == 1 {
				require.Equal(t, int32(40), oldCalls.Load())
				recovered.Store(true)
			}
			if round == 5 {
				require.Equal(t, int32(60), oldCalls.Load(), "the oldest page must be retried and acknowledged after recovery")
				pending, err := client.XPending(context.Background(), b.stream, b.group).Result()
				require.NoError(t, err)
				require.Equal(t, int64(100), pending.Count)
				return
			}
			clock.Advance(delay)
		case <-time.After(5 * time.Second):
			t.Fatal("missing retry timer")
		}
	}
}
