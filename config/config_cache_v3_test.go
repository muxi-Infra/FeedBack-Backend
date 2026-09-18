package config

import (
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
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
	require.EqualError(t, err, "v3_config_cache.max_age must be > 0")
}

func TestConfigCacheValidatePositiveDurations(t *testing.T) {
	fields := []struct {
		name string
		set  func(*V3ConfigCacheConfig, time.Duration)
	}{
		{"refresh_interval", func(c *V3ConfigCacheConfig, d time.Duration) { c.RefreshInterval = d }},
		{"refresh_timeout", func(c *V3ConfigCacheConfig, d time.Duration) { c.RefreshTimeout = d }},
		{"load_timeout", func(c *V3ConfigCacheConfig, d time.Duration) { c.LoadTimeout = d }},
		{"max_age", func(c *V3ConfigCacheConfig, d time.Duration) { c.MaxAge = d }},
		{"retry_min", func(c *V3ConfigCacheConfig, d time.Duration) { c.RetryMin = d }},
		{"publish_interval", func(c *V3ConfigCacheConfig, d time.Duration) { c.PublishInterval = d }},
		{"outbox_retention", func(c *V3ConfigCacheConfig, d time.Duration) { c.OutboxRetention = d }},
		{"heartbeat_interval", func(c *V3ConfigCacheConfig, d time.Duration) { c.HeartbeatInterval = d }},
		{"maintenance_interval", func(c *V3ConfigCacheConfig, d time.Duration) { c.MaintenanceInterval = d }},
	}
	for _, field := range fields {
		for _, d := range []time.Duration{-time.Nanosecond, 0} {
			t.Run(field.name+"/"+d.String(), func(t *testing.T) {
				c := DefaultV3ConfigCacheConfig()
				field.set(c, d)
				require.EqualError(t, c.Validate(), "v3_config_cache."+field.name+" must be > 0")
			})
		}
	}
}

func TestConfigCacheValidateConstraints(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*V3ConfigCacheConfig)
		want   string
	}{
		{"empty stream key", func(c *V3ConfigCacheConfig) { c.StreamKey = "" }, "stream_key must not be empty"},
		{"interval exceeds max age", func(c *V3ConfigCacheConfig) { c.RefreshInterval = c.MaxAge + time.Nanosecond }, "refresh_interval must be <= max_age"},
		{"interval exhausts max age", func(c *V3ConfigCacheConfig) { c.RefreshInterval = c.MaxAge }, "refresh_timeout must be <= max_age - refresh_interval"},
		{"refresh exceeds budget", func(c *V3ConfigCacheConfig) { c.RefreshTimeout = c.MaxAge - c.RefreshInterval + time.Nanosecond }, "refresh_timeout must be <= max_age - refresh_interval"},
		{"load exceeds refresh", func(c *V3ConfigCacheConfig) { c.LoadTimeout = c.RefreshTimeout + time.Nanosecond }, "load_timeout must be <= refresh_timeout"},
		{"zero concurrency", func(c *V3ConfigCacheConfig) { c.Concurrency = 0 }, "concurrency must be >= 1"},
		{"negative concurrency", func(c *V3ConfigCacheConfig) { c.Concurrency = -1 }, "concurrency must be >= 1"},
		{"retry max below min", func(c *V3ConfigCacheConfig) { c.RetryMax = c.RetryMin - time.Nanosecond }, "retry_max must be >= retry_min"},
		{"outbox lease equals load", func(c *V3ConfigCacheConfig) { c.OutboxLease = c.LoadTimeout }, "outbox_lease must be > load_timeout"},
		{"outbox lease below load", func(c *V3ConfigCacheConfig) { c.OutboxLease = c.LoadTimeout - time.Nanosecond }, "outbox_lease must be > load_timeout"},
		{"zero stream capacity", func(c *V3ConfigCacheConfig) { c.StreamMaxLen = 0 }, "stream_max_len must be >= 1"},
		{"negative stream capacity", func(c *V3ConfigCacheConfig) { c.StreamMaxLen = -1 }, "stream_max_len must be >= 1"},
		{"zero backlog threshold", func(c *V3ConfigCacheConfig) { c.BacklogThreshold = 0 }, "backlog_threshold must be >= 1"},
		{"negative backlog threshold", func(c *V3ConfigCacheConfig) { c.BacklogThreshold = -1 }, "backlog_threshold must be >= 1"},
		{"backlog exceeds capacity", func(c *V3ConfigCacheConfig) { c.BacklogThreshold = c.StreamMaxLen + 1 }, "backlog_threshold must be <= stream_max_len"},
		{"group lease below millisecond", func(c *V3ConfigCacheConfig) {
			c.GroupLease = time.Millisecond - time.Nanosecond
			c.HeartbeatInterval = time.Nanosecond
		}, "group_lease must be >= 1ms"},
		{"heartbeat equals half lease", func(c *V3ConfigCacheConfig) { c.GroupLease = 2 * c.HeartbeatInterval }, "group_lease / 2 must be > heartbeat_interval"},
		{"heartbeat exceeds half lease", func(c *V3ConfigCacheConfig) { c.GroupLease = 2*c.HeartbeatInterval - time.Nanosecond }, "group_lease / 2 must be > heartbeat_interval"},
		{"half lease rounds down", func(c *V3ConfigCacheConfig) { c.GroupLease = 2*c.HeartbeatInterval + time.Nanosecond }, "group_lease / 2 must be > heartbeat_interval"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := DefaultV3ConfigCacheConfig()
			tt.mutate(c)
			require.EqualError(t, c.Validate(), "v3_config_cache."+tt.want)
		})
	}
}

func TestConfigCacheValidateValidBoundaries(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*V3ConfigCacheConfig)
	}{
		{"defaults", func(c *V3ConfigCacheConfig) {}},
		{"refresh equals budget", func(c *V3ConfigCacheConfig) { c.RefreshTimeout = c.MaxAge - c.RefreshInterval }},
		{"load equals refresh", func(c *V3ConfigCacheConfig) { c.LoadTimeout = c.RefreshTimeout }},
		{"minimum concurrency", func(c *V3ConfigCacheConfig) { c.Concurrency = 1 }},
		{"equal retry bounds", func(c *V3ConfigCacheConfig) { c.RetryMax = c.RetryMin }},
		{"outbox lease above load", func(c *V3ConfigCacheConfig) { c.OutboxLease = c.LoadTimeout + time.Nanosecond }},
		{"minimum stream capacity", func(c *V3ConfigCacheConfig) { c.StreamMaxLen = 1; c.BacklogThreshold = 1 }},
		{"backlog equals capacity", func(c *V3ConfigCacheConfig) { c.BacklogThreshold = c.StreamMaxLen }},
		{"minimum group lease", func(c *V3ConfigCacheConfig) {
			c.GroupLease = time.Millisecond
			c.HeartbeatInterval = c.GroupLease/2 - time.Nanosecond
		}},
		{"half lease above heartbeat", func(c *V3ConfigCacheConfig) { c.GroupLease = 2*c.HeartbeatInterval + 2*time.Nanosecond }},
		{"minimum positive durations", func(c *V3ConfigCacheConfig) {
			c.RefreshInterval, c.RefreshTimeout, c.LoadTimeout = time.Nanosecond, time.Nanosecond, time.Nanosecond
			c.MaxAge = 2 * time.Nanosecond
			c.RetryMin, c.RetryMax = time.Nanosecond, time.Nanosecond
			c.PublishInterval, c.OutboxRetention = time.Nanosecond, time.Nanosecond
			c.HeartbeatInterval, c.MaintenanceInterval = time.Nanosecond, time.Nanosecond
		}},
		{"maximum refresh budget", func(c *V3ConfigCacheConfig) {
			c.MaxAge = time.Duration(1<<63 - 1)
			c.RefreshInterval = c.MaxAge - c.RefreshTimeout
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := DefaultV3ConfigCacheConfig()
			tt.mutate(c)
			require.NoError(t, c.Validate())
		})
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
