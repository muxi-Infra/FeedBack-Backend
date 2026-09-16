package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/configclock"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/configmetrics"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
)

type projectCacheEntryV3 struct {
	snapshot        *domain.ProjectSnapshotV3
	generation      uint64
	required        uint64
	confirmed       time.Time
	loaded          time.Time
	invalidated     time.Time
	lastInvalidated time.Time
	eventID         string
	errorClass      string
	flight          chan struct{}
}
type ProjectConfigStatusV3 struct {
	InstanceID      string    `json:"instance_id"`
	ProjectID       string    `json:"project_id"`
	TargetVersion   *uint64   `json:"target_version"`
	RequiredVersion uint64    `json:"required_version"`
	AppliedVersion  uint64    `json:"applied_version"`
	State           string    `json:"state"`
	LastEventID     string    `json:"last_event_id,omitempty"`
	ConfirmedAt     time.Time `json:"confirmed_at"`
	LoadedAt        time.Time `json:"loaded_at"`
	InvalidatedAt   time.Time `json:"invalidated_at"`
	ErrorClass      string    `json:"error_class,omitempty"`
}
type ProjectConfigCacheV3 struct {
	mu        sync.Mutex
	items     map[string]*projectCacheEntryV3
	dao       dao.ConfigDAOV3
	cfg       *config.V3ConfigCacheConfig
	clock     configclock.Clock
	metrics   *configmetrics.Metrics
	log       logger.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	closed    bool
	wg        sync.WaitGroup
	slots     chan struct{}
	updates   chan string
	reconcile chan struct{}
}

func NewProjectConfigCacheV3(d dao.ConfigDAOV3, cfg *config.V3ConfigCacheConfig, clock configclock.Clock, m *configmetrics.Metrics, log logger.Logger) *ProjectConfigCacheV3 {
	ctx, cancel := context.WithCancel(context.Background())
	return &ProjectConfigCacheV3{items: make(map[string]*projectCacheEntryV3), dao: d, cfg: cfg, clock: clock, metrics: m, log: log, ctx: ctx, cancel: cancel, slots: make(chan struct{}, cfg.Concurrency), updates: make(chan string, 1000), reconcile: make(chan struct{}, 1)}
}
func (c *ProjectConfigCacheV3) Close() {
	c.mu.Lock()
	c.closed = true
	c.cancel()
	c.mu.Unlock()
	c.wg.Wait()
}
func (c *ProjectConfigCacheV3) entry(id string) *projectCacheEntryV3 {
	e := c.items[id]
	if e == nil {
		e = &projectCacheEntryV3{}
		c.items[id] = e
	}
	return e
}
func (c *ProjectConfigCacheV3) valid(e *projectCacheEntryV3, now time.Time) bool {
	return e.snapshot != nil && e.snapshot.Project.ConfigVersion >= e.required && !e.confirmed.IsZero() && now.Sub(e.confirmed) < c.cfg.MaxAge
}
func (c *ProjectConfigCacheV3) TriggerReconcile() {
	select {
	case c.reconcile <- struct{}{}:
	default:
	}
}
func (c *ProjectConfigCacheV3) Notify(event domain.ConfigEventV3) {
	c.mu.Lock()
	e := c.entry(event.ProjectID)
	if event.MessageID != "" {
		e.eventID = event.MessageID
	}
	changed := event.Version == 0 || event.Version > e.required
	if changed {
		e.lastInvalidated = c.clock.Now()
		e.generation++
		if event.Version > e.required {
			e.required = event.Version
		}
		if event.Version == 0 {
			e.confirmed = time.Time{}
		}
		if e.invalidated.IsZero() {
			e.invalidated = c.clock.Now()
		}
	}
	c.mu.Unlock()
	if changed {
		c.log.Info("cache_invalidated", logger.String("project_id", event.ProjectID), logger.Uint64("version", event.Version))
	}
	select {
	case c.updates <- event.ProjectID:
	default:
		c.TriggerReconcile()
	}
}

// Refresh publishes complete snapshots, including deletion tombstones.
func (c *ProjectConfigCacheV3) Refresh(request context.Context, id, trigger string) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return context.Canceled
	}
	c.wg.Add(1)
	c.mu.Unlock()
	defer c.wg.Done()
	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.LoadTimeout)
	stop := context.AfterFunc(request, cancel)
	defer stop()
	defer cancel()
	for {
		if err := request.Err(); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		c.mu.Lock()
		e := c.entry(id)
		if c.valid(e, c.clock.Now()) {
			c.mu.Unlock()
			return nil
		}
		if e.flight != nil {
			done := e.flight
			c.mu.Unlock()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-done:
				continue
			}
		}
		e.flight = make(chan struct{})
		generation := e.generation
		c.mu.Unlock()
		started := c.clock.Now()
		var snapshot domain.ProjectSnapshotV3
		var err error
		select {
		case c.slots <- struct{}{}:
			started = c.clock.Now()
			snapshot, err = c.dao.Snapshot(ctx, id)
			<-c.slots
		case <-ctx.Done():
			err = ctx.Err()
		}
		if err == nil {
			err = ctx.Err()
		}
		if err == nil {
			err = request.Err()
		}
		c.mu.Lock()
		e = c.entry(id)
		obsolete := generation != e.generation || snapshot.Project.ConfigVersion < e.required || (e.snapshot != nil && snapshot.Project.ConfigVersion < e.snapshot.Project.ConfigVersion)
		if err == nil && c.clock.Now().Sub(started) >= c.cfg.MaxAge {
			err = context.DeadlineExceeded
		}
		result := "failed"
		if err == nil && !obsolete {
			result = "unchanged"
			if e.snapshot == nil || e.snapshot.Project.ConfigVersion != snapshot.Project.ConfigVersion {
				result = "applied"
			}
			e.snapshot = &snapshot
			e.required = max(e.required, snapshot.Project.ConfigVersion)
			e.confirmed = started
			e.loaded = c.clock.Now()
			e.errorClass = ""
			if result == "applied" {
				if age := c.clock.Now().Sub(snapshot.Project.UpdatedAt).Seconds(); !snapshot.Project.UpdatedAt.IsZero() && age >= 0 {
					c.metrics.ApplyAge.Observe(age)
				}
				if !e.invalidated.IsZero() {
					c.metrics.InvalidationDelay.Observe(c.clock.Now().Sub(e.invalidated).Seconds())
				}
			}
			e.invalidated = time.Time{}
		} else if err != nil {
			e.errorClass = domain.ConfigErrorClass(err)
		} else {
			result = "superseded"
		}
		close(e.flight)
		e.flight = nil
		c.mu.Unlock()
		c.metrics.Duration.WithLabelValues(trigger).Observe(c.clock.Now().Sub(started).Seconds())
		c.metrics.Refresh.WithLabelValues(trigger, result).Inc()
		if err != nil {
			c.log.Warn("refresh_failed", logger.String("project_id", id), logger.String("error_class", domain.ConfigErrorClass(err)))
			return err
		}
		if obsolete {
			continue
		}
		if result == "applied" {
			c.log.Info("config_applied", logger.String("project_id", id), logger.Uint64("version", snapshot.Project.ConfigVersion))
		}
		return nil
	}
}
func (c *ProjectConfigCacheV3) Get(ctx context.Context, id, kind string) (V3TableConfig, error) {
	if id == "" || kind == "" {
		return V3TableConfig{}, errs.V3InvalidInputError(errors.New("project_id and table_type required"))
	}
	counted := false
	for {
		if ctx.Err() != nil {
			return V3TableConfig{}, errs.V3ConfigUnavailableError(errors.New("canceled"))
		}
		c.mu.Lock()
		e := c.entry(id)
		if !c.closed && c.valid(e, c.clock.Now()) {
			s := e.snapshot
			if s.Active() {
				for _, table := range s.Tables {
					if table.TableType == kind {
						result := V3TableConfig{ProjectID: id, TableType: kind, Scopes: append([]string(nil), s.Scopes[table.TableIdentity]...), Table: table}
						c.mu.Unlock()
						if !counted {
							c.metrics.Cache.WithLabelValues("hit").Inc()
						}
						return result, nil
					}
				}
			}
			c.mu.Unlock()
			if !counted {
				c.metrics.Cache.WithLabelValues("hit").Inc()
			}
			return V3TableConfig{}, errs.V3TableConfigError(errors.New("project or table unavailable"))
		}
		reason := "miss"
		if e.snapshot != nil {
			reason = "expired"
			if e.snapshot.Project.ConfigVersion < e.required {
				reason = "outdated"
			}
		}
		c.mu.Unlock()
		if !counted {
			c.metrics.Cache.WithLabelValues(reason).Inc()
			counted = true
		}
		if err := c.Refresh(ctx, id, "request"); err != nil {
			return V3TableConfig{}, errs.V3ConfigUnavailableError(errors.New(domain.ConfigErrorClass(err)))
		}
		// Notify and this check share a mutex; the load result is never returned directly.
	}
}
func (c *ProjectConfigCacheV3) confirm(p model.FeedbackProjectV3, at time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entry(p.ProjectID)
	if p.ConfigVersion > e.required {
		e.lastInvalidated = c.clock.Now()
		e.required = p.ConfigVersion
		e.generation++
		if e.invalidated.IsZero() {
			e.invalidated = c.clock.Now()
		}
	}
	if e.snapshot != nil && e.snapshot.Project.ConfigVersion == p.ConfigVersion && p.ConfigVersion >= e.required && e.snapshot.Active() == (p.DeletedAt == 0 && p.Status == "active") && !e.confirmed.IsZero() {
		if at.After(e.confirmed) {
			e.confirmed = at
		}
		e.errorClass = ""
		return true
	}
	return false
}
func (c *ProjectConfigCacheV3) Reconcile(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.RefreshTimeout)
	defer cancel()
	var last uint64
	seen := make(map[string]bool)
	var combined error
	for {
		started := c.clock.Now()
		rows, err := c.dao.ListMetadata(ctx, last, 100)
		if err != nil {
			return err
		}
		jobs := make(chan string, len(rows))
		for _, p := range rows {
			seen[p.ProjectID] = true
			last = p.ID
			if c.confirm(p, started) {
				c.metrics.Refresh.WithLabelValues("reconcile", "unchanged").Inc()
			} else {
				jobs <- p.ProjectID
			}
		}
		close(jobs)
		var wg sync.WaitGroup
		var mu sync.Mutex
		for i := 0; i < c.cfg.Concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for id := range jobs {
					if err := c.Refresh(ctx, id, "reconcile"); err != nil {
						mu.Lock()
						combined = errors.Join(combined, err)
						mu.Unlock()
					}
				}
			}()
		}
		wg.Wait()
		if len(rows) < 100 {
			break
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	c.mu.Lock()
	var missing []string
	for id, e := range c.items {
		if !seen[id] && e.snapshot != nil {
			missing = append(missing, id)
			e.confirmed = time.Time{}
			e.generation++
		}
	}
	c.mu.Unlock()
	for _, id := range missing {
		combined = errors.Join(combined, c.Refresh(ctx, id, "reconcile"))
	}
	if combined == nil {
		combined = ctx.Err()
	}
	if combined == nil {
		c.metrics.LastFull.Set(float64(c.clock.Now().Unix()))
	}
	return combined
}
func (c *ProjectConfigCacheV3) Status(ctx context.Context, id, instance string) ProjectConfigStatusV3 {
	s := ProjectConfigStatusV3{ProjectID: id, InstanceID: instance, State: "not_loaded"}
	ctx, cancel := context.WithTimeout(ctx, c.cfg.LoadTimeout)
	defer cancel()
	p, err := c.dao.Metadata(ctx, id)
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.items[id]; e != nil {
		s.RequiredVersion = e.required
		s.LastEventID = e.eventID
		s.ConfirmedAt = e.confirmed
		s.LoadedAt = e.loaded
		s.InvalidatedAt = e.lastInvalidated
		s.ErrorClass = e.errorClass
		if e.snapshot != nil {
			s.AppliedVersion = e.snapshot.Project.ConfigVersion
			s.State = "applied"
			if !e.snapshot.Active() {
				s.State = "deleted"
			}
			if !c.valid(e, c.clock.Now()) {
				s.State = "stale"
			}
		}
	}
	if err != nil {
		s.State = "target_unknown"
		s.ErrorClass = domain.ConfigErrorClass(err)
		return s
	}
	s.TargetVersion = &p.ConfigVersion
	if s.AppliedVersion < p.ConfigVersion {
		s.State = "behind"
	}
	return s
}
