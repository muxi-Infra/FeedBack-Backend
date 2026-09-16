//go:build integration

package service

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/internal/integrationenv"
	"github.com/muxi-Infra/FeedBack-Backend/internal/testclock"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/configclock"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/configmetrics"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/constvar"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	eventcache "github.com/muxi-Infra/FeedBack-Backend/repository/cache"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func newMySQLConfigFixture(t *testing.T) *configFixture {
	db := integrationenv.MySQL(t)
	require.NoError(t, db.AutoMigrate(&model.FeedbackProjectV3{}, &model.FeedbackProjectTableV3{}, &model.FeedbackProjectScopeV3{}, &model.FeedbackProjectKeyV3{}, &model.ConfigAuditV3{}, &model.ConfigOutboxV3{}))
	f := &configFixture{db: db, dao: dao.NewConfigDAOV3(db), clock: testclock.New(), cfg: config.DefaultV3ConfigCacheConfig(), metrics: configmetrics.New(prometheus.NewRegistry()), log: logger.NewZapLogger(zap.NewNop()), instance: domain.NewConfigInstanceV3()}
	f.cache = f.newCache(t, f.dao)
	f.admin = NewV3AdminService(dao.NewIntegrationDAOV3(db), f.dao, f.cache, f.clock, f.log, f.instance)
	return f
}

func TestIntegrationConfigMySQLConcurrentVersionsAndSnapshot(t *testing.T) {
	f := newMySQLConfigFixture(t)
	ctx := context.Background()
	id := f.register(t)
	const writers = 8
	start := make(chan struct{})
	results := make(chan error, writers)
	for i := 0; i < writers; i++ {
		go func() {
			<-start
			_, err := f.admin.UpdateProject(ctx, id, configInput(constvar.FeedbackScopeWrite), configActor())
			results <- err
		}()
	}
	close(start)
	for i := 0; i < writers; i++ {
		require.NoError(t, <-results)
	}
	audits, err := f.dao.ListAudits(ctx, id, "", 0, 100)
	require.NoError(t, err)
	require.Len(t, audits, writers+1)
	for i, row := range audits {
		require.Equal(t, uint64(writers+1-i), row.Version)
		require.Equal(t, row.Version-1, row.PreviousVersion)
	}

	type snapshotKey struct{}
	read, release := make(chan struct{}), make(chan struct{})
	require.NoError(t, f.db.Callback().Query().After("gorm:query").Register("test_snapshot_barrier", func(tx *gorm.DB) {
		if tx.Statement.Context.Value(snapshotKey{}) != nil && tx.Statement.Table == "feedback_projects" {
			close(read)
			<-release
		}
	}))
	defer f.db.Callback().Query().Remove("test_snapshot_barrier")
	var snapshot domain.ProjectSnapshotV3
	result := make(chan error, 1)
	go func() {
		var err error
		snapshot, err = f.dao.Snapshot(context.WithValue(ctx, snapshotKey{}, true), id)
		result <- err
	}()
	awaitConfig(t, read)
	_, err = f.admin.UpdateProject(ctx, id, configInput(constvar.FeedbackScopeReadSelf), configActor())
	close(release)
	require.NoError(t, err)
	require.NoError(t, <-result)
	require.Equal(t, uint64(writers+1), snapshot.Project.ConfigVersion)
	require.Equal(t, []string{constvar.FeedbackScopeWrite}, snapshot.Scopes["fictional-feedback"])
	current, err := f.dao.Snapshot(ctx, id)
	require.NoError(t, err)
	require.Equal(t, uint64(writers+2), current.Project.ConfigVersion)
	require.Equal(t, []string{constvar.FeedbackScopeReadSelf}, current.Scopes["fictional-feedback"])
}

func TestIntegrationConfigMySQLLeasesAndAuditRollback(t *testing.T) {
	f := newMySQLConfigFixture(t)
	ctx := context.Background()
	id := f.register(t)
	claims := make(chan []model.ConfigOutboxV3, 2)
	failures := make(chan error, 2)
	var wg sync.WaitGroup
	for _, owner := range []string{"a", "b"} {
		wg.Add(1)
		go func(owner string) {
			defer wg.Done()
			rows, err := f.dao.ClaimOutbox(ctx, owner, f.clock.Now(), f.cfg.OutboxLease, 4)
			claims <- rows
			failures <- err
		}(owner)
	}
	wg.Wait()
	close(claims)
	close(failures)
	var claimed []model.ConfigOutboxV3
	for err := range failures {
		require.NoError(t, err)
	}
	for rows := range claims {
		claimed = append(claimed, rows...)
	}
	require.Len(t, claimed, 1)
	f.clock.Advance(f.cfg.OutboxLease)
	again, err := f.dao.ClaimOutbox(ctx, "c", f.clock.Now(), f.cfg.OutboxLease, 4)
	require.NoError(t, err)
	require.Len(t, again, 1)
	require.Equal(t, claimed[0].ChangeID, again[0].ChangeID)
	require.Error(t, f.dao.FinishOutbox(ctx, claimed[0], "1-0", f.clock.Now()))
	require.NoError(t, f.dao.FinishOutbox(ctx, again[0], "2-0", f.clock.Now()))
	bad := NewV3AdminService(dao.NewIntegrationDAOV3(f.db), failingAuditDAO{f.dao}, f.cache, f.clock, f.log, f.instance)
	_, err = bad.UpdateProject(ctx, id, configInput(constvar.FeedbackScopeWrite), configActor())
	require.Error(t, err)
	snapshot, err := f.dao.Snapshot(ctx, id)
	require.NoError(t, err)
	require.Equal(t, uint64(1), snapshot.Project.ConfigVersion)
	require.Equal(t, []string{constvar.FeedbackScopeReadSelf}, snapshot.Scopes["fictional-feedback"])
	audits, err := f.dao.ListAudits(ctx, id, "", 0, 100)
	require.NoError(t, err)
	require.Len(t, audits, 1)
}

func TestIntegrationConfigMySQLUpgradeIsRepeatable(t *testing.T) {
	db := integrationenv.MySQL(t)
	require.NoError(t, db.AutoMigrate(&model.FeedbackProjectV3{}))
	require.NoError(t, db.Migrator().DropColumn(&model.FeedbackProjectV3{}, "config_version"))
	require.NoError(t, db.Exec("INSERT INTO feedback_projects(project_id,project_name,school,status,deleted_at) VALUES ('legacy','Legacy','Fictional','active',0)").Error)
	ddl, err := os.ReadFile("../docs/migrations/098-config-cache.sql")
	require.NoError(t, err)
	for _, statement := range strings.Split(string(ddl), ";") {
		if strings.TrimSpace(statement) != "" {
			require.NoError(t, db.Exec(statement).Error)
		}
	}
	for i := 0; i < 2; i++ {
		require.NoError(t, db.AutoMigrate(&model.FeedbackProjectV3{}, &model.ConfigAuditV3{}, &model.ConfigOutboxV3{}))
	}
	p, err := dao.NewConfigDAOV3(db).Metadata(context.Background(), "legacy")
	require.NoError(t, err)
	require.Equal(t, uint64(1), p.ConfigVersion)
	var n int64
	require.NoError(t, db.Model(&model.ConfigAuditV3{}).Count(&n).Error)
	require.Zero(t, n)
}

func TestIntegrationConfigTwoRuntimesApplyAndExit(t *testing.T) {
	f := newMySQLConfigFixture(t)
	client, key := integrationenv.Redis(t)
	ctx := context.Background()
	f.cfg.StreamKey = key
	f.admin = NewV3AdminService(dao.NewIntegrationDAOV3(f.db), f.dao, f.cache, configclock.New(), f.log, f.instance)
	start := func() (*ProjectConfigCacheV3, *ConfigRuntimeV3) {
		instance := domain.NewConfigInstanceV3()
		m := configmetrics.New(prometheus.NewRegistry())
		clock := configclock.New()
		local := NewProjectConfigCacheV3(f.dao, f.cfg, clock, m, f.log)
		events := eventcache.NewProjectConfigEventBusV3(client, f.log, f.cfg, clock, m, instance)
		runtime := NewConfigRuntimeV3(local, events, f.dao, f.cfg, clock, m, f.log, instance)
		require.NoError(t, runtime.Start(ctx))
		t.Cleanup(func() {
			stop, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			require.NoError(t, runtime.Stop(stop))
		})
		return local, runtime
	}
	a, _ := start()
	b, _ := start()
	id := f.register(t)
	require.Eventually(t, func() bool {
		return a.Status(ctx, id, "a").AppliedVersion == 1 && b.Status(ctx, id, "b").AppliedVersion == 1
	}, 8*time.Second, 20*time.Millisecond)
	_, err := f.admin.UpdateProject(ctx, id, configInput(constvar.FeedbackScopeWrite), configActor())
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		return a.Status(ctx, id, "a").AppliedVersion == 2 && b.Status(ctx, id, "b").AppliedVersion == 2
	}, 8*time.Second, 20*time.Millisecond)
	for _, local := range []*ProjectConfigCacheV3{a, b} {
		cfg, err := local.Get(ctx, id, constvar.FeedbackTableType)
		require.NoError(t, err)
		require.True(t, cfg.HasScope(constvar.FeedbackScopeWrite))
		require.False(t, cfg.HasScope(constvar.FeedbackScopeReadSelf))
	}
}
