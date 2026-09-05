package cache_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/muxi-Infra/FeedBack-Backend/repository/cache"
	"github.com/stretchr/testify/require"
)

func TestV3NonceStoreAtomicityAndExpiration(t *testing.T) {
	server := miniredis.RunT(t)
	// Independent clients/stores model multiple application instances sharing Redis.
	var stores []cache.IntegrationNonceStoreV3
	for i := 0; i < 2; i++ {
		client := redis.NewClient(&redis.Options{Addr: server.Addr(), MaxRetries: -1})
		t.Cleanup(func() { require.NoError(t, client.Close()) })
		stores = append(stores, cache.NewIntegrationNonceStoreV3(client))
	}
	ctx := context.Background()
	const key = "fictional-project:nonce"
	const ttl = 10 * time.Second
	type result struct {
		used bool
		err  error
	}
	start := make(chan struct{})
	results := make(chan result, 32)
	var wg sync.WaitGroup
	for i := 0; i < cap(results); i++ {
		wg.Add(1)
		go func(store cache.IntegrationNonceStoreV3) {
			defer wg.Done()
			<-start
			used, err := store.MarkUsed(ctx, key, ttl)
			results <- result{used, err}
		}(stores[i%len(stores)])
	}
	close(start)
	wg.Wait()
	close(results)
	success := 0
	for result := range results {
		require.NoError(t, result.err)
		if result.used {
			success++
		}
	}
	require.Equal(t, 1, success)
	require.Equal(t, ttl, server.TTL(key))
	server.FastForward(9 * time.Second)
	used, err := stores[1].MarkUsed(ctx, key, ttl)
	require.NoError(t, err)
	require.False(t, used)
	require.Equal(t, time.Second, server.TTL(key), "duplicate attempts must not extend the existing TTL")
	server.FastForward(time.Second)
	used, err = stores[0].MarkUsed(ctx, key, ttl)
	require.NoError(t, err)
	require.True(t, used, "the key can be reused after expiration")
}

func TestV3NonceStoreRejectsInvalidInput(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	for _, tc := range []struct {
		name, key string
		ttl       time.Duration
		client    *redis.Client
	}{
		{"empty_key", "", time.Second, client},
		{"zero_ttl", "fictional-nonce", 0, client},
		{"negative_ttl", "fictional-nonce", -time.Second, client},
		{"nil_client", "fictional-nonce", time.Second, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			used, err := cache.NewIntegrationNonceStoreV3(tc.client).MarkUsed(context.Background(), tc.key, tc.ttl)
			require.Error(t, err)
			require.False(t, used)
			require.Empty(t, server.Keys())
		})
	}
}
