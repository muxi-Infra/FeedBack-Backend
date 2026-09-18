package config

import (
	"errors"
	"time"
)

type V3ConfigCacheConfig struct {
	StreamKey           string        `mapstructure:"stream_key"`
	RefreshInterval     time.Duration `mapstructure:"refresh_interval"`
	RefreshTimeout      time.Duration `mapstructure:"refresh_timeout"`
	MaxAge              time.Duration `mapstructure:"max_age"`
	LoadTimeout         time.Duration `mapstructure:"load_timeout"`
	Concurrency         int           `mapstructure:"concurrency"`
	RetryMin            time.Duration `mapstructure:"retry_min"`
	RetryMax            time.Duration `mapstructure:"retry_max"`
	PublishInterval     time.Duration `mapstructure:"publish_interval"`
	OutboxLease         time.Duration `mapstructure:"outbox_lease"`
	OutboxRetention     time.Duration `mapstructure:"outbox_retention"`
	StreamMaxLen        int64         `mapstructure:"stream_max_len"`
	BacklogThreshold    int64         `mapstructure:"backlog_threshold"`
	HeartbeatInterval   time.Duration `mapstructure:"heartbeat_interval"`
	GroupLease          time.Duration `mapstructure:"group_lease"`
	MaintenanceInterval time.Duration `mapstructure:"maintenance_interval"`
}

func DefaultV3ConfigCacheConfig() *V3ConfigCacheConfig {
	return &V3ConfigCacheConfig{
		StreamKey:       "feedback:v3:project_config_changed",
		RefreshInterval: 30 * time.Second, RefreshTimeout: 10 * time.Second,
		MaxAge: 60 * time.Second, LoadTimeout: 2 * time.Second, Concurrency: 4,
		RetryMin: 200 * time.Millisecond, RetryMax: 30 * time.Second,
		PublishInterval: time.Second, OutboxLease: 15 * time.Second, OutboxRetention: 7 * 24 * time.Hour,
		StreamMaxLen: 10000, BacklogThreshold: 1000,
		HeartbeatInterval: 10 * time.Second, GroupLease: 120 * time.Second, MaintenanceInterval: time.Minute,
	}
}

func (c *V3ConfigCacheConfig) Validate() error {
	// Validate positive durations before comparing the refresh budget.
	if c.StreamKey == "" {
		return errors.New("v3_config_cache.stream_key must not be empty")
	}
	if c.RefreshInterval <= 0 {
		return errors.New("v3_config_cache.refresh_interval must be > 0")
	}
	if c.RefreshTimeout <= 0 {
		return errors.New("v3_config_cache.refresh_timeout must be > 0")
	}
	if c.LoadTimeout <= 0 {
		return errors.New("v3_config_cache.load_timeout must be > 0")
	}
	if c.MaxAge <= 0 {
		return errors.New("v3_config_cache.max_age must be > 0")
	}
	if c.RefreshInterval > c.MaxAge {
		return errors.New("v3_config_cache.refresh_interval must be <= max_age")
	}
	if c.RefreshTimeout > c.MaxAge-c.RefreshInterval {
		return errors.New("v3_config_cache.refresh_timeout must be <= max_age - refresh_interval")
	}
	if c.LoadTimeout > c.RefreshTimeout {
		return errors.New("v3_config_cache.load_timeout must be <= refresh_timeout")
	}

	// Bound refresh concurrency and retry delays.
	if c.Concurrency < 1 {
		return errors.New("v3_config_cache.concurrency must be >= 1")
	}
	if c.RetryMin <= 0 {
		return errors.New("v3_config_cache.retry_min must be > 0")
	}
	if c.RetryMax < c.RetryMin {
		return errors.New("v3_config_cache.retry_max must be >= retry_min")
	}

	// Keep the outbox lease longer than a publish operation.
	if c.PublishInterval <= 0 {
		return errors.New("v3_config_cache.publish_interval must be > 0")
	}
	if c.OutboxLease <= c.LoadTimeout {
		return errors.New("v3_config_cache.outbox_lease must be > load_timeout")
	}
	if c.OutboxRetention <= 0 {
		return errors.New("v3_config_cache.outbox_retention must be > 0")
	}

	// Keep backlog detection within the retained stream capacity.
	if c.StreamMaxLen < 1 {
		return errors.New("v3_config_cache.stream_max_len must be >= 1")
	}
	if c.BacklogThreshold < 1 {
		return errors.New("v3_config_cache.backlog_threshold must be >= 1")
	}
	if c.BacklogThreshold > c.StreamMaxLen {
		return errors.New("v3_config_cache.backlog_threshold must be <= stream_max_len")
	}

	// Allow consumer heartbeats to renew the group lease before expiry.
	if c.HeartbeatInterval <= 0 {
		return errors.New("v3_config_cache.heartbeat_interval must be > 0")
	}
	if c.GroupLease < time.Millisecond {
		return errors.New("v3_config_cache.group_lease must be >= 1ms")
	}
	if c.GroupLease/2 <= c.HeartbeatInterval {
		return errors.New("v3_config_cache.group_lease / 2 must be > heartbeat_interval")
	}
	if c.MaintenanceInterval <= 0 {
		return errors.New("v3_config_cache.maintenance_interval must be > 0")
	}
	return nil
}

func NewV3ConfigCacheConfig() (*V3ConfigCacheConfig, error) {
	c := DefaultV3ConfigCacheConfig()
	if err := vp.UnmarshalKey("v3_config_cache", c); err != nil {
		return nil, err
	}
	return c, c.Validate()
}
