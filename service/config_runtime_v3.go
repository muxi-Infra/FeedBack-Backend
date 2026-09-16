package service

import (
	"context"
	"errors"
	"sync"

	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/configclock"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/configmetrics"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/repository/cache"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
)

type ConfigRuntimeV3 struct {
	local         *ProjectConfigCacheV3
	events        cache.ProjectConfigEventBusV3
	dao           dao.ConfigDAOV3
	cfg           *config.V3ConfigCacheConfig
	clock         configclock.Clock
	metrics       *configmetrics.Metrics
	log           logger.Logger
	instance      *domain.ConfigInstanceV3
	mu            sync.Mutex
	cancel        context.CancelFunc
	done          chan struct{}
	wg            sync.WaitGroup
	reconcileGate chan struct{}
	stopped       bool
}

func NewConfigRuntimeV3(local *ProjectConfigCacheV3, events cache.ProjectConfigEventBusV3, d dao.ConfigDAOV3, cfg *config.V3ConfigCacheConfig, clock configclock.Clock, m *configmetrics.Metrics, log logger.Logger, instance *domain.ConfigInstanceV3) *ConfigRuntimeV3 {
	return &ConfigRuntimeV3{local: local, events: events, dao: d, cfg: cfg, clock: clock, metrics: m, log: log, instance: instance, reconcileGate: make(chan struct{}, 1)}
}
func (r *ConfigRuntimeV3) Start(parent context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.done != nil || r.stopped {
		return errors.New("config runtime already started")
	}
	ctx, cancel := context.WithCancel(parent)
	r.cancel = cancel
	r.done = make(chan struct{})
	launch := func(fn func(context.Context)) { r.wg.Add(1); go func() { defer r.wg.Done(); fn(ctx) }() }
	launch(func(ctx context.Context) {
		r.events.Consume(ctx, func(ctx context.Context, e domain.ConfigEventV3) error {
			r.local.Notify(e)
			return r.local.Refresh(ctx, e.ProjectID, "event")
		}, r.reconcile)
	})
	launch(r.refreshLoop)
	launch(r.publishLoop)
	launch(r.maintenanceLoop)
	for i := 0; i < r.cfg.Concurrency; i++ {
		launch(func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					return
				case id := <-r.local.updates:
					_ = r.local.Refresh(ctx, id, "local")
				}
			}
		})
	}
	go func() { r.wg.Wait(); r.local.Close(); _ = r.events.Close(); close(r.done) }()
	r.log.Info("config_runtime_started", logger.String("instance_id", r.instance.ID))
	return nil
}
func (r *ConfigRuntimeV3) Stop(ctx context.Context) error {
	r.mu.Lock()
	cancel, done := r.cancel, r.done
	r.stopped = true
	r.mu.Unlock()
	if cancel == nil {
		r.local.Close()
		return r.events.Close()
	}
	cancel()
	_ = r.events.Close()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (r *ConfigRuntimeV3) reconcile(ctx context.Context) error {
	// Wait with cancellation so a scheduled scan is never discarded by an event-triggered scan.
	select {
	case r.reconcileGate <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-r.reconcileGate }()
	err := r.local.Reconcile(ctx)
	if err != nil {
		r.metrics.Reconciliations.WithLabelValues("failed").Inc()
		r.log.Warn("config_full_refresh_failed", logger.String("error_class", domain.ConfigErrorClass(err)))
	} else {
		r.metrics.Reconciliations.WithLabelValues("success").Inc()
	}
	return err
}
func (r *ConfigRuntimeV3) refreshLoop(ctx context.Context) {
	ticker := r.clock.NewTicker(r.cfg.RefreshInterval)
	defer ticker.Stop()
	_ = r.reconcile(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C():
			_ = r.reconcile(ctx)
		case <-r.local.reconcile:
			_ = r.reconcile(ctx)
		}
	}
}
func (r *ConfigRuntimeV3) publishLoop(ctx context.Context) {
	ticker := r.clock.NewTicker(r.cfg.PublishInterval)
	defer ticker.Stop()
	for {
		r.publish(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C():
		}
	}
}
func (r *ConfigRuntimeV3) publish(ctx context.Context) {
	op, cancel := context.WithTimeout(ctx, r.cfg.LoadTimeout)
	rows, err := r.dao.ClaimOutbox(op, r.instance.ID, r.clock.Now(), r.cfg.OutboxLease, r.cfg.Concurrency)
	cancel()
	if err != nil {
		r.metrics.Publish.WithLabelValues("claim_failed").Inc()
		return
	}
	var wg sync.WaitGroup
	for _, row := range rows {
		wg.Add(1)
		go func(row model.ConfigOutboxV3) { defer wg.Done(); r.publishRow(ctx, row) }(row)
	}
	wg.Wait()
}
func (r *ConfigRuntimeV3) publishRow(ctx context.Context, row model.ConfigOutboxV3) {
	op, cancel := context.WithTimeout(ctx, r.cfg.LoadTimeout)
	defer cancel()
	message, err := r.events.Publish(op, domain.ConfigEventV3{ProjectID: row.ProjectID, Version: row.Version, Kind: row.Kind, ChangeID: row.ChangeID, ChangedAt: row.CreatedAt})
	if err != nil {
		r.metrics.Publish.WithLabelValues("failed").Inc()
		r.log.Warn("config_publish_failed", logger.String("change_id", row.ChangeID), logger.String("error_class", domain.ConfigErrorClass(err)))
		retryCtx, retryCancel := context.WithTimeout(ctx, r.cfg.LoadTimeout)
		defer retryCancel()
		_ = r.dao.RetryOutbox(retryCtx, row, r.clock.Now().Add(configclock.Backoff(r.cfg.RetryMin, r.cfg.RetryMax, row.Attempts-1)), domain.ConfigErrorClass(err))
		return
	}
	if err = r.dao.FinishOutbox(op, row, message, r.clock.Now()); err != nil {
		r.metrics.Publish.WithLabelValues("finish_failed").Inc()
		return
	}
	r.metrics.Publish.WithLabelValues("published").Inc()
	r.log.Info("config_event_published", logger.String("change_id", row.ChangeID), logger.String("message_id", message), logger.String("project_id", row.ProjectID), logger.Uint64("version", row.Version))
}
func (r *ConfigRuntimeV3) maintenanceLoop(ctx context.Context) {
	heartbeat := r.clock.NewTicker(r.cfg.HeartbeatInterval)
	defer heartbeat.Stop()
	maintenance := r.clock.NewTicker(r.cfg.MaintenanceInterval)
	defer maintenance.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C():
			op, cancel := context.WithTimeout(ctx, r.cfg.LoadTimeout)
			if err := r.events.Heartbeat(op); err != nil {
				r.metrics.Consumer.Set(0)
			}
			cancel()
		case <-maintenance.C():
			op, cancel := context.WithTimeout(ctx, r.cfg.LoadTimeout)
			behind, err := r.events.Maintain(op)
			cancel()
			if err == nil && behind {
				r.local.TriggerReconcile()
			}
			op, cancel = context.WithTimeout(ctx, r.cfg.LoadTimeout)
			n, oldest, err := r.dao.OutboxStats(op)
			cancel()
			if err == nil {
				r.metrics.OutboxStatsUp.Set(1)
				r.metrics.OutboxPending.Set(float64(n))
				r.metrics.OutboxAge.Set(0)
				if oldest != nil {
					r.metrics.OutboxAge.Set(max(0, r.clock.Now().Sub(*oldest).Seconds()))
				}
			} else {
				r.metrics.OutboxStatsUp.Set(0)
			}
			op, cancel = context.WithTimeout(ctx, r.cfg.LoadTimeout)
			_ = r.dao.PruneOutbox(op, r.clock.Now().Add(-r.cfg.OutboxRetention))
			cancel()
		}
	}
}
