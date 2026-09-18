//go:build integration

package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/internal/integrationenv"
	"github.com/muxi-Infra/FeedBack-Backend/internal/testclock"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/configclock"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestIntegrationConfigRedisBroadcastRecoveryAndCancel(t *testing.T) {
	client, key := integrationenv.Redis(t)
	ctx := context.Background()
	a := newConfigBusTest(t, client, key, configclock.New())
	b := newConfigBusTest(t, client, key, configclock.New())
	var fail atomic.Bool
	fail.Store(true)
	client.AddHook(configCommandHook{before: func(cmd redis.Cmder) error {
		if cmd.Name() == "xack" && fail.Swap(false) {
			return errors.New("injected ACK failure")
		}
		return nil
	}})
	aSeen, bSeen := make(chan struct{}, 10), make(chan struct{}, 10)
	var calls atomic.Int32
	aReady, _ := consumeConfigTest(t, a, func(context.Context, domain.ConfigEventV3) error {
		if calls.Add(1) == 1 {
			return errors.New("injected load failure")
		}
		aSeen <- struct{}{}
		return nil
	})
	bReady, _ := consumeConfigTest(t, b, func(context.Context, domain.ConfigEventV3) error { bSeen <- struct{}{}; return nil })
	configSignal(t, aReady)
	configSignal(t, bReady)
	_, err := a.Publish(ctx, domain.ConfigEventV3{ProjectID: "fictional", Version: 2, ChangedAt: time.Now()})
	require.NoError(t, err)
	configSignal(t, aSeen)
	configSignal(t, bSeen)
	require.Eventually(t, func() bool {
		ap, ae := client.XPending(ctx, key, a.group).Result()
		bp, be := client.XPending(ctx, key, b.group).Result()
		return ae == nil && be == nil && ap.Count == 0 && bp.Count == 0
	}, 8*time.Second, 10*time.Millisecond)
	require.GreaterOrEqual(t, calls.Load(), int32(2))
	_, err = a.Maintain(ctx)
	require.NoError(t, err)
	require.Equal(t, float64(1), testutil.ToFloat64(a.metrics.StreamStatsUp))
	require.Zero(t, testutil.ToFloat64(a.metrics.Unread))
}

func TestIntegrationConfigRedisReconcileFailureDoesNotBlockConsumption(t *testing.T) {
	client, key := integrationenv.Redis(t)
	assertConfigConsumptionAfterReconcileFailure(t, client, key)
}

func TestIntegrationConfigRedisPendingSweepFinishesDespiteNewFailures(t *testing.T) {
	client, key := integrationenv.Redis(t)
	assertConfigPendingSweepFinishes(t, client, key)
}

func TestIntegrationConfigRedisAtomicLeaseCleanup(t *testing.T) {
	client, key := integrationenv.Redis(t)
	ctx := context.Background()
	b := newConfigBusTest(t, client, key, configclock.New())
	for i := 0; i < 20; i++ {
		require.NoError(t, createConfigGroup.Run(ctx, client, []string{key, b.lease(b.group), b.registry()}, b.group, 120000).Err())
		require.NoError(t, client.Del(ctx, b.lease(b.group)).Err())
		var wg sync.WaitGroup
		failures := make(chan error, 2)
		wg.Add(2)
		go func() { defer wg.Done(); failures <- b.Heartbeat(ctx) }()
		go func() {
			defer wg.Done()
			failures <- cleanConfigGroup.Run(ctx, client, []string{key, b.lease(b.group), b.registry()}, b.group).Err()
		}()
		wg.Wait()
		close(failures)
		for err := range failures {
			require.NoError(t, err)
		}
		lease, err := client.Exists(ctx, b.lease(b.group)).Result()
		require.NoError(t, err)
		raw, err := client.Do(ctx, "XINFO", "GROUPS", key).Result()
		require.NoError(t, err)
		groups, err := configGroups(raw)
		require.NoError(t, err)
		if lease == 1 {
			require.Len(t, groups, 1)
			require.Equal(t, b.group, groups[0].name)
		} else {
			require.Empty(t, groups)
		}
	}
}

func TestIntegrationConfigRedisTrimmedPELAndLag(t *testing.T) {
	client, key := integrationenv.Redis(t)
	ctx := context.Background()
	b := newConfigBusTest(t, client, key, configclock.New())
	require.NoError(t, createConfigGroup.Run(ctx, client, []string{key, b.lease(b.group), b.registry()}, b.group, b.cfg.GroupLease.Milliseconds()).Err())
	id, err := b.Publish(ctx, domain.ConfigEventV3{ProjectID: "fictional", Version: 1, ChangedAt: time.Now()})
	require.NoError(t, err)
	_, err = client.XReadGroup(ctx, &redis.XReadGroupArgs{Group: b.group, Consumer: b.name, Streams: []string{key, ">"}, Count: 1, Block: -1}).Result()
	require.NoError(t, err)
	require.NoError(t, client.XDel(ctx, key, id).Err())
	items, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{Group: b.group, Consumer: b.name, Streams: []string{key, "0"}, Count: 1, Block: -1}).Result()
	require.NoError(t, err)
	reconciled := 0
	failed, _ := b.process(ctx, items, func(context.Context, domain.ConfigEventV3) error {
		t.Error("trimmed body must not be handled")
		return nil
	}, func(context.Context) error { reconciled++; return nil })
	require.False(t, failed)
	require.Equal(t, 1, reconciled)
	p, err := client.XPending(ctx, key, b.group).Result()
	require.NoError(t, err)
	require.Zero(t, p.Count)
	b.cfg.BacklogThreshold = 2
	b.stream = key + ":lag"
	require.NoError(t, createConfigGroup.Run(ctx, client, []string{b.stream, b.lease(b.group), b.registry()}, b.group, b.cfg.GroupLease.Milliseconds()).Err())
	for _, id := range []string{"100-0", "200-0", "300-0", "400-0"} {
		_, err = client.XAdd(ctx, &redis.XAddArgs{Stream: b.stream, ID: id, Values: map[string]any{"project_id": "fictional"}}).Result()
		require.NoError(t, err)
	}
	// An arbitrary position with a deletion ahead makes Redis 7 lag unknown; Redis 6 omits it.
	require.NoError(t, client.Do(ctx, "XGROUP", "SETID", b.stream, b.group, "150-0").Err())
	require.NoError(t, client.XDel(ctx, b.stream, "200-0").Err())
	behind, err := b.Maintain(ctx)
	require.NoError(t, err)
	require.True(t, behind)
	require.Zero(t, testutil.ToFloat64(b.metrics.UnreadExact))
	require.GreaterOrEqual(t, testutil.ToFloat64(b.metrics.Unread), float64(2))
}

func TestIntegrationConfigRedisLeaseCleanupAndStartupRetry(t *testing.T) {
	client, key := integrationenv.Redis(t)
	ctx := context.Background()
	clock := testclock.New()
	b := newConfigBusTest(t, client, key, clock)
	var unavailable atomic.Bool
	unavailable.Store(true)
	client.AddHook(configCommandHook{before: func(cmd redis.Cmder) error {
		if unavailable.Load() {
			return errors.New("injected startup outage")
		}
		return nil
	}})
	ready, _ := consumeConfigTest(t, b, func(context.Context, domain.ConfigEventV3) error { return nil })
	configSignal(t, clock.Created)
	unavailable.Store(false)
	clock.Advance(time.Second)
	configSignal(t, ready)
	require.NoError(t, b.Heartbeat(ctx))
	_, err := b.Maintain(ctx)
	require.NoError(t, err)
	groups, err := client.Do(ctx, "XINFO", "GROUPS", key).Result()
	require.NoError(t, err)
	parsed, err := configGroups(groups)
	require.NoError(t, err)
	require.Len(t, parsed, 1)
	stale := newConfigBusTest(t, client, key, configclock.New())
	require.NoError(t, createConfigGroup.Run(ctx, client, []string{key, stale.lease(stale.group), stale.registry()}, stale.group, 120000).Err())
	require.NoError(t, client.Del(ctx, stale.lease(stale.group)).Err())
	_, err = b.Maintain(ctx)
	require.NoError(t, err)
	groups, err = client.Do(ctx, "XINFO", "GROUPS", key).Result()
	require.NoError(t, err)
	parsed, err = configGroups(groups)
	require.NoError(t, err)
	require.Len(t, parsed, 1)
	require.Equal(t, b.group, parsed[0].name)
}
