package main

import (
	"context"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/configclock"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/configmetrics"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/constvar"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"github.com/muxi-Infra/FeedBack-Backend/service"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type shutdownConfigDAO struct{ dao.ConfigDAOV3 }

func (shutdownConfigDAO) Snapshot(ctx context.Context, id string) (domain.ProjectSnapshotV3, error) {
	return domain.ProjectSnapshotV3{
		Project: model.FeedbackProjectV3{ID: 1, ProjectID: id, Status: "active", ConfigVersion: 1},
		Tables:  []model.FeedbackProjectTableV3{{TableType: constvar.FeedbackTableType}},
	}, ctx.Err()
}
func (d shutdownConfigDAO) ListMetadata(ctx context.Context, _ uint64, _ int) ([]model.FeedbackProjectV3, error) {
	s, err := d.Snapshot(ctx, "fictional")
	return []model.FeedbackProjectV3{s.Project}, err
}
func (shutdownConfigDAO) ClaimOutbox(context.Context, string, time.Time, time.Duration, int) ([]model.ConfigOutboxV3, error) {
	return nil, nil
}
func (shutdownConfigDAO) OutboxStats(context.Context) (int64, *time.Time, error) {
	return 0, nil, nil
}
func (shutdownConfigDAO) PruneOutbox(context.Context, time.Time) error { return nil }

type shutdownConfigBus struct{ started chan context.Context }

func (b *shutdownConfigBus) Consume(ctx context.Context, _ func(context.Context, domain.ConfigEventV3) error, _ func(context.Context) error) {
	b.started <- ctx
	<-ctx.Done()
}
func (*shutdownConfigBus) Publish(context.Context, domain.ConfigEventV3) (string, error) {
	return "", nil
}
func (*shutdownConfigBus) Heartbeat(context.Context) error        { return nil }
func (*shutdownConfigBus) Maintain(context.Context) (bool, error) { return false, nil }
func (*shutdownConfigBus) Close() error                           { return nil }

func newShutdownApp(t *testing.T) (*App, *service.ProjectConfigCacheV3, *shutdownConfigBus) {
	t.Helper()
	cfg := config.DefaultV3ConfigCacheConfig()
	clock := configclock.New()
	m := configmetrics.New(prometheus.NewRegistry())
	log := logger.NewZapLogger(zap.NewNop())
	d := shutdownConfigDAO{}
	local := service.NewProjectConfigCacheV3(d, cfg, clock, m, log)
	bus := &shutdownConfigBus{started: make(chan context.Context, 1)}
	runtime := service.NewConfigRuntimeV3(local, bus, d, cfg, clock, m, log, domain.NewConfigInstanceV3())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, runtime.Stop(ctx))
	})
	return &App{configRuntime: runtime}, local, bus
}

func waitShutdownSignal(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown synchronization timed out")
	}
}

func TestShutdownDrainsHTTPBeforeClosingConfiguration(t *testing.T) {
	app, local, bus := newShutdownApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	address := make(chan string, 1)
	entered, release, draining, done := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	server := &http.Server{
		Addr: "127.0.0.1:0",
		BaseContext: func(listener net.Listener) context.Context {
			address <- listener.Addr().String()
			return context.Background()
		},
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			close(entered)
			<-release
			if _, err := local.Get(req.Context(), "fictional", constvar.FeedbackTableType); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	server.RegisterOnShutdown(func() { close(draining) })
	var runErr error
	go func() { defer close(done); runErr = app.runServer(ctx, server) }()
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(release) })
		cancel()
		_ = server.Close()
		waitShutdownSignal(t, done)
	})
	var addr string
	select {
	case addr = <-address:
	case <-time.After(5 * time.Second):
		t.Fatal("HTTP listener did not start")
	}
	status, failure := make(chan int, 1), make(chan error, 1)
	client := &http.Client{Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()
	go func() {
		resp, err := client.Get("http://" + addr)
		if err != nil {
			failure <- err
			return
		}
		_ = resp.Body.Close()
		status <- resp.StatusCode
	}()
	waitShutdownSignal(t, entered)
	var runtimeCtx context.Context
	select {
	case runtimeCtx = <-bus.started:
	case <-time.After(5 * time.Second):
		t.Fatal("configuration runtime did not start")
	}
	cancel()
	waitShutdownSignal(t, draining)
	require.NoError(t, runtimeCtx.Err(), "configuration must remain available while HTTP requests drain")
	releaseOnce.Do(func() { close(release) })
	select {
	case code := <-status:
		require.Equal(t, http.StatusNoContent, code)
	case err := <-failure:
		t.Fatal(err)
	case <-time.After(5 * time.Second):
		t.Fatal("HTTP request did not finish")
	}
	waitShutdownSignal(t, done)
	require.NoError(t, runErr)
	require.ErrorIs(t, runtimeCtx.Err(), context.Canceled)
	require.ErrorIs(t, local.Refresh(context.Background(), "fictional", "request"), context.Canceled)
}

func TestHTTPListenFailureStopsConfiguration(t *testing.T) {
	app, local, _ := newShutdownApp(t)
	err := app.runServer(context.Background(), &http.Server{Addr: "127.0.0.1:invalid"})
	require.Error(t, err)
	require.ErrorIs(t, local.Refresh(context.Background(), "fictional", "request"), context.Canceled)
}
