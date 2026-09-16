package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/internal/testclock"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/configmetrics"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/constvar"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/errorx"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm"
	gormlog "gorm.io/gorm/logger"
)

type configFixture struct {
	db       *gorm.DB
	dao      dao.ConfigDAOV3
	cache    *ProjectConfigCacheV3
	admin    V3AdminService
	clock    *testclock.Clock
	cfg      *config.V3ConfigCacheConfig
	metrics  *configmetrics.Metrics
	log      logger.Logger
	logs     *observer.ObservedLogs
	instance *domain.ConfigInstanceV3
}

func newConfigFixture(t *testing.T) *configFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlog.Default.LogMode(gormlog.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, db.AutoMigrate(&model.FeedbackProjectV3{}, &model.FeedbackProjectTableV3{}, &model.FeedbackProjectScopeV3{}, &model.FeedbackProjectKeyV3{}, &model.ConfigAuditV3{}, &model.ConfigOutboxV3{}))
	core, logs := observer.New(zap.DebugLevel)
	f := &configFixture{db: db, dao: dao.NewConfigDAOV3(db), clock: testclock.New(), cfg: config.DefaultV3ConfigCacheConfig(), metrics: configmetrics.New(prometheus.NewRegistry()), log: logger.NewZapLogger(zap.New(core)), logs: logs, instance: domain.NewConfigInstanceV3()}
	f.cache = f.newCache(t, f.dao)
	f.admin = NewV3AdminService(dao.NewIntegrationDAOV3(db), f.dao, f.cache, f.clock, f.log, f.instance)
	return f
}
func (f *configFixture) newCache(t *testing.T, d dao.ConfigDAOV3) *ProjectConfigCacheV3 {
	c := NewProjectConfigCacheV3(d, f.cfg, f.clock, f.metrics, f.log)
	t.Cleanup(c.Close)
	return c
}
func configInput(scope string) domain.RegisterProjectInput {
	return domain.RegisterProjectInput{ProjectName: "Fictional project", School: "Fictional school", Tables: []domain.RegisterProjectTableInput{{TableIdentity: "fictional-feedback", TableName: "Fictional feedback", TableType: constvar.FeedbackTableType, TableToken: "fictional-sensitive-table-token", TableID: "fictional-table", ViewID: "fictional-view", Scopes: []string{scope}}}}
}
func configActor() domain.ConfigActorV3 {
	return domain.ConfigActorV3{AdminID: 1, RequestID: uuid.NewString()}
}
func (f *configFixture) register(t *testing.T) string {
	r, err := f.admin.RegisterProject(context.Background(), configInput(constvar.FeedbackScopeReadSelf), configActor())
	require.NoError(t, err)
	require.Equal(t, uint64(1), r.Receipt.Version)
	return r.ProjectID
}
func awaitConfig(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("configuration synchronization timed out")
	}
}

func TestV3ConfigMutationAuditAndTombstone(t *testing.T) {
	f := newConfigFixture(t)
	ctx := context.Background()
	id := f.register(t)
	old, err := f.cache.Get(ctx, id, constvar.FeedbackTableType)
	require.NoError(t, err)
	require.True(t, old.HasScope(constvar.FeedbackScopeReadSelf))
	receipt, err := f.admin.UpdateProject(ctx, id, configInput(constvar.FeedbackScopeWrite), configActor())
	require.NoError(t, err)
	require.Equal(t, uint64(2), receipt.Version)
	current, err := f.cache.Get(ctx, id, constvar.FeedbackTableType)
	require.NoError(t, err)
	require.False(t, current.HasScope(constvar.FeedbackScopeReadSelf))
	require.True(t, current.HasScope(constvar.FeedbackScopeWrite))
	current.Scopes[0] = "mutated"
	again, err := f.cache.Get(ctx, id, constvar.FeedbackTableType)
	require.NoError(t, err)
	require.True(t, again.HasScope(constvar.FeedbackScopeWrite))
	receipt, err = f.admin.DeleteProject(ctx, id, configActor())
	require.NoError(t, err)
	require.Equal(t, uint64(3), receipt.Version)
	_, err = f.cache.Get(ctx, id, constvar.FeedbackTableType)
	require.Error(t, err)
	repeated, err := f.admin.DeleteProject(ctx, id, configActor())
	require.NoError(t, err)
	require.Equal(t, receipt.Version, repeated.Version)
	_, err = f.admin.UpdateProject(ctx, id, configInput(constvar.FeedbackScopeReadSelf), configActor())
	require.Error(t, err)
	_, err = f.admin.UpdateProject(ctx, "missing", configInput(constvar.FeedbackScopeReadSelf), configActor())
	require.Error(t, err)
	rows, err := f.dao.ListAudits(ctx, id, "", 0, 100)
	require.NoError(t, err)
	require.Len(t, rows, 3)
	var outbox []model.ConfigOutboxV3
	require.NoError(t, f.db.Find(&outbox).Error)
	require.Len(t, outbox, 3)
	for i, row := range rows {
		require.Equal(t, uint64(3-i), row.Version)
		require.Equal(t, uint64(1), row.AdminID)
		require.Equal(t, "committed", row.Result)
	}
	p, err := f.dao.Metadata(ctx, id)
	require.NoError(t, err)
	require.NotZero(t, p.DeletedAt)
	require.Equal(t, uint64(3), p.ConfigVersion)
	encoded, err := json.Marshal(rows)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "fictional-sensitive-table-token")
	for _, entry := range f.logs.All() {
		data, _ := json.Marshal(entry.ContextMap())
		require.NotContains(t, string(data), "fictional-sensitive-table-token")
	}
}

type snapshotHookDAO struct {
	dao.ConfigDAOV3
	hook func(context.Context, string) (domain.ProjectSnapshotV3, error)
}

func (d snapshotHookDAO) Snapshot(ctx context.Context, id string) (domain.ProjectSnapshotV3, error) {
	return d.hook(ctx, id)
}
func TestV3ConfigOldLoadCannotReturnAfterInvalidation(t *testing.T) {
	f := newConfigFixture(t)
	id := f.register(t)
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var first atomic.Bool
	f.cache.dao = snapshotHookDAO{ConfigDAOV3: f.dao, hook: func(ctx context.Context, id string) (domain.ProjectSnapshotV3, error) {
		s, err := f.dao.Snapshot(ctx, id)
		if first.CompareAndSwap(false, true) {
			close(entered)
			select {
			case <-release:
			case <-ctx.Done():
				return s, ctx.Err()
			}
		}
		return s, err
	}}
	var got V3TableConfig
	var readErr error
	go func() {
		defer close(done)
		got, readErr = f.cache.Get(context.Background(), id, constvar.FeedbackTableType)
	}()
	t.Cleanup(func() { f.cache.cancel(); awaitConfig(t, done) })
	awaitConfig(t, entered)
	_, err := f.admin.UpdateProject(context.Background(), id, configInput(constvar.FeedbackScopeWrite), configActor())
	require.NoError(t, err)
	close(release)
	awaitConfig(t, done)
	require.NoError(t, readErr)
	require.False(t, got.HasScope(constvar.FeedbackScopeReadSelf))
	require.True(t, got.HasScope(constvar.FeedbackScopeWrite))
	require.Equal(t, uint64(2), f.cache.Status(context.Background(), id, "test").AppliedVersion)
}

func TestV3ConfigFallbackAndAuthorizationDeadline(t *testing.T) {
	f := newConfigFixture(t)
	ctx := context.Background()
	id := f.register(t)
	other := f.newCache(t, f.dao)
	_, err := other.Get(ctx, id, constvar.FeedbackTableType)
	require.NoError(t, err)
	_, err = f.admin.UpdateProject(ctx, id, configInput(constvar.FeedbackScopeWrite), configActor())
	require.NoError(t, err)
	stale, err := other.Get(ctx, id, constvar.FeedbackTableType)
	require.NoError(t, err)
	require.True(t, stale.HasScope(constvar.FeedbackScopeReadSelf))
	f.clock.Advance(30 * time.Second)
	require.NoError(t, other.Reconcile(ctx))
	updated, err := other.Get(ctx, id, constvar.FeedbackTableType)
	require.NoError(t, err)
	require.False(t, updated.HasScope(constvar.FeedbackScopeReadSelf))
	_, err = f.admin.DeleteProject(ctx, id, configActor())
	require.NoError(t, err)
	require.NoError(t, other.Reconcile(ctx))
	_, err = other.Get(ctx, id, constvar.FeedbackTableType)
	require.Error(t, err)
	second := f.register(t)
	_, err = other.Get(ctx, second, constvar.FeedbackTableType)
	require.NoError(t, err)
	other.dao = snapshotHookDAO{ConfigDAOV3: f.dao, hook: func(context.Context, string) (domain.ProjectSnapshotV3, error) {
		return domain.ProjectSnapshotV3{}, errors.New("fictional-sensitive-failure")
	}}
	f.clock.Advance(60 * time.Second)
	_, err = other.Get(ctx, second, constvar.FeedbackTableType)
	require.Equal(t, 503, errorx.ToCustomError(err).HttpCode)
	require.Equal(t, errs.V3ConfigUnavailableCode, errorx.ToCustomError(err).Code)
	for _, entry := range f.logs.All() {
		data, _ := json.Marshal(entry.ContextMap())
		require.NotContains(t, string(data), "fictional-sensitive-failure")
	}
}

func TestV3ConfigPartialRefreshPreservesOtherProjects(t *testing.T) {
	f := newConfigFixture(t)
	ctx := context.Background()
	a, b := f.register(t), f.register(t)
	other := f.newCache(t, f.dao)
	require.NoError(t, other.Reconcile(ctx))
	for _, id := range []string{a, b} {
		_, err := f.admin.UpdateProject(ctx, id, configInput(constvar.FeedbackScopeWrite), configActor())
		require.NoError(t, err)
	}
	other.dao = snapshotHookDAO{ConfigDAOV3: f.dao, hook: func(ctx context.Context, id string) (domain.ProjectSnapshotV3, error) {
		if id == a {
			return domain.ProjectSnapshotV3{}, errors.New("unavailable")
		}
		return f.dao.Snapshot(ctx, id)
	}}
	require.Error(t, other.Reconcile(ctx))
	good, err := other.Get(ctx, b, constvar.FeedbackTableType)
	require.NoError(t, err)
	require.True(t, good.HasScope(constvar.FeedbackScopeWrite))
	_, err = other.Get(ctx, a, constvar.FeedbackTableType)
	require.Equal(t, 503, errorx.ToCustomError(err).HttpCode)
	other.mu.Lock()
	require.NotNil(t, other.items[a].snapshot)
	require.Equal(t, uint64(1), other.items[a].snapshot.Project.ConfigVersion)
	other.mu.Unlock()
	other.dao = f.dao
	require.NoError(t, other.Reconcile(ctx))
}

func TestV3ConfigDuplicateAndOlderEventsDoNotRegress(t *testing.T) {
	f := newConfigFixture(t)
	ctx := context.Background()
	id := f.register(t)
	require.NoError(t, f.cache.Reconcile(ctx))
	_, err := f.admin.UpdateProject(ctx, id, configInput(constvar.FeedbackScopeWrite), configActor())
	require.NoError(t, err)
	require.NoError(t, f.cache.Refresh(ctx, id, "event"))
	before := testutil.ToFloat64(f.metrics.Refresh.WithLabelValues("event", "applied"))
	for _, version := range []uint64{2, 1, 2} {
		f.cache.Notify(domain.ConfigEventV3{ProjectID: id, Version: version})
		require.NoError(t, f.cache.Refresh(ctx, id, "event"))
	}
	require.Equal(t, before, testutil.ToFloat64(f.metrics.Refresh.WithLabelValues("event", "applied")))
	require.Equal(t, uint64(2), f.cache.Status(ctx, id, "test").AppliedVersion)
}

type failingAuditDAO struct{ dao.ConfigDAOV3 }

func (d failingAuditDAO) RecordChange(context.Context, *gorm.DB, domain.ConfigActorV3, domain.ConfigEventV3, uint64, string) error {
	return errors.New("audit unavailable")
}
func TestV3ConfigAuditFailureRollsBackConfiguration(t *testing.T) {
	f := newConfigFixture(t)
	ctx := context.Background()
	id := f.register(t)
	admin := NewV3AdminService(dao.NewIntegrationDAOV3(f.db), failingAuditDAO{f.dao}, f.cache, f.clock, f.log, f.instance)
	_, err := admin.UpdateProject(ctx, id, configInput(constvar.FeedbackScopeWrite), configActor())
	require.Error(t, err)
	s, err := f.dao.Snapshot(ctx, id)
	require.NoError(t, err)
	require.Equal(t, uint64(1), s.Project.ConfigVersion)
	require.Equal(t, []string{constvar.FeedbackScopeReadSelf}, s.Scopes["fictional-feedback"])
	rows, err := f.dao.ListAudits(ctx, id, "", 0, 100)
	require.NoError(t, err)
	require.Len(t, rows, 1)
}

type testConfigBus struct {
	mu        sync.Mutex
	fail      bool
	published []domain.ConfigEventV3
	started   chan struct{}
}

func (b *testConfigBus) Publish(_ context.Context, e domain.ConfigEventV3) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.fail {
		return "", errors.New("redis unavailable")
	}
	b.published = append(b.published, e)
	return "1-0", nil
}
func (b *testConfigBus) Consume(ctx context.Context, _ func(context.Context, domain.ConfigEventV3) error, _ func(context.Context) error) {
	if b.started != nil {
		close(b.started)
	}
	<-ctx.Done()
}
func (b *testConfigBus) Heartbeat(context.Context) error        { return nil }
func (b *testConfigBus) Maintain(context.Context) (bool, error) { return false, nil }
func (b *testConfigBus) Close() error                           { return nil }

type failedFinishDAO struct {
	dao.ConfigDAOV3
	fail atomic.Bool
}

func (d *failedFinishDAO) FinishOutbox(ctx context.Context, row model.ConfigOutboxV3, id string, at time.Time) error {
	if d.fail.CompareAndSwap(true, false) {
		return errors.New("lost finish")
	}
	return d.ConfigDAOV3.FinishOutbox(ctx, row, id, at)
}
func TestV3ConfigOutboxFailureAndDuplicatePublication(t *testing.T) {
	f := newConfigFixture(t)
	ctx := context.Background()
	id := f.register(t)
	bus := &testConfigBus{fail: true}
	r := NewConfigRuntimeV3(f.cache, bus, f.dao, f.cfg, f.clock, f.metrics, f.log, f.instance)
	r.publish(ctx)
	n, _, err := f.dao.OutboxStats(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
	other := f.newCache(t, f.dao)
	require.NoError(t, other.Reconcile(ctx))
	require.Equal(t, uint64(1), other.Status(ctx, id, "other").AppliedVersion)
	bus.fail = false
	f.clock.Advance(time.Second)
	d := &failedFinishDAO{ConfigDAOV3: f.dao}
	d.fail.Store(true)
	r.dao = d
	r.publish(ctx)
	require.Len(t, bus.published, 1)
	f.clock.Advance(f.cfg.OutboxLease)
	r.publish(ctx)
	require.Len(t, bus.published, 2)
	require.Equal(t, bus.published[0].ChangeID, bus.published[1].ChangeID)
	n, _, err = f.dao.OutboxStats(ctx)
	require.NoError(t, err)
	require.Zero(t, n)
}
func TestV3ConfigRuntimeCancellationStopsTimersAndLoads(t *testing.T) {
	f := newConfigFixture(t)
	f.register(t)
	bus := &testConfigBus{started: make(chan struct{})}
	r := NewConfigRuntimeV3(f.cache, bus, f.dao, f.cfg, f.clock, f.metrics, f.log, f.instance)
	require.NoError(t, r.Start(context.Background()))
	awaitConfig(t, bus.started)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, r.Stop(ctx))
	require.Zero(t, f.clock.Active())
	require.ErrorIs(t, f.cache.Refresh(context.Background(), "missing", "test"), context.Canceled)
}

func TestV3ConfigCloseCancelsActiveDatabaseLoad(t *testing.T) {
	f := newConfigFixture(t)
	entered := make(chan struct{})
	local := f.newCache(t, snapshotHookDAO{ConfigDAOV3: f.dao, hook: func(ctx context.Context, _ string) (domain.ProjectSnapshotV3, error) {
		close(entered)
		<-ctx.Done()
		return domain.ProjectSnapshotV3{}, ctx.Err()
	}})
	done := make(chan struct{})
	go func() { defer close(done); _ = local.Refresh(context.Background(), "fictional", "test") }()
	awaitConfig(t, entered)
	local.Close()
	awaitConfig(t, done)
}

type metadataHookDAO struct {
	dao.ConfigDAOV3
	list func(context.Context, uint64, int) ([]model.FeedbackProjectV3, error)
}

func (d metadataHookDAO) ListMetadata(ctx context.Context, after uint64, limit int) ([]model.FeedbackProjectV3, error) {
	return d.list(ctx, after, limit)
}

func TestV3ConfigIncompleteScanDoesNotInvalidateUnseenProjects(t *testing.T) {
	f := newConfigFixture(t)
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		require.NoError(t, f.db.Create(&model.FeedbackProjectV3{ProjectID: uuid.NewString(), ProjectName: "Fictional", School: "Fictional", Status: "active"}).Error)
	}
	id := f.register(t)
	other := f.newCache(t, metadataHookDAO{ConfigDAOV3: f.dao, list: func(ctx context.Context, after uint64, limit int) ([]model.FeedbackProjectV3, error) {
		if after > 0 {
			return nil, errors.New("second page unavailable")
		}
		return f.dao.ListMetadata(ctx, after, limit)
	}})
	_, err := other.Get(ctx, id, constvar.FeedbackTableType)
	require.NoError(t, err)
	confirmed := other.Status(ctx, id, "other").ConfirmedAt
	f.clock.Advance(time.Second)
	require.Error(t, other.Reconcile(ctx))
	status := other.Status(ctx, id, "other")
	require.Equal(t, confirmed, status.ConfirmedAt)
	require.Equal(t, "applied", status.State)
	other.mu.Lock()
	_, coldLoaded := other.items[id]
	count := len(other.items)
	other.mu.Unlock()
	require.True(t, coldLoaded)
	require.Equal(t, 101, count)
}

func TestV3ConfigSlowLoadDoesNotExtendAuthorizationLease(t *testing.T) {
	f := newConfigFixture(t)
	ctx := context.Background()
	id := f.register(t)
	var fail bool
	other := f.newCache(t, snapshotHookDAO{ConfigDAOV3: f.dao, hook: func(ctx context.Context, id string) (domain.ProjectSnapshotV3, error) {
		if fail {
			return domain.ProjectSnapshotV3{}, errors.New("unavailable")
		}
		s, err := f.dao.Snapshot(ctx, id)
		f.clock.Advance(59 * time.Second)
		return s, err
	}})
	_, err := other.Get(ctx, id, constvar.FeedbackTableType)
	require.NoError(t, err)
	fail = true
	f.clock.Advance(time.Second)
	_, err = other.Get(ctx, id, constvar.FeedbackTableType)
	require.Equal(t, 503, errorx.ToCustomError(err).HttpCode)
}
