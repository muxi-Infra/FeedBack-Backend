package config

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestConfigCacheDefaultsAndExplicitZero(t *testing.T) {
	previous := vp
	t.Cleanup(func() { vp = previous })
	vp = viper.New()
	c, err := NewV3ConfigCacheConfig()
	require.NoError(t, err)
	require.Equal(t, 60*time.Second, c.MaxAge)
	vp.Set("v3_config_cache.max_age", 0)
	_, err = NewV3ConfigCacheConfig()
	require.Error(t, err)
	for _, mutate := range []func(*V3ConfigCacheConfig){func(c *V3ConfigCacheConfig) { c.RefreshInterval = 0 }, func(c *V3ConfigCacheConfig) { c.Concurrency = 0 }, func(c *V3ConfigCacheConfig) { c.GroupLease = c.HeartbeatInterval }} {
		c := DefaultV3ConfigCacheConfig()
		mutate(c)
		require.Error(t, c.Validate())
	}
}

func TestRedisConfigurationRejectsInvalidAddress(t *testing.T) {
	previous := vp
	t.Cleanup(func() { vp = previous })
	vp = viper.New()
	for _, addr := range []string{"", "localhost", "localhost:0", "localhost:70000", ":6379"} {
		vp.Set("redis.addr", addr)
		_, err := NewRedisConfig()
		require.Error(t, err)
	}
	vp.Set("redis.addr", "localhost:6379")
	_, err := NewRedisConfig()
	require.NoError(t, err)
	vp.Set("redis.db", -1)
	_, err = NewRedisConfig()
	require.Error(t, err)
}
