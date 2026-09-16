package config

import (
	"fmt"
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
	if c.StreamKey == "" || c.RefreshInterval <= 0 || c.RefreshTimeout <= 0 || c.LoadTimeout <= 0 ||
		c.MaxAge <= 0 || c.RefreshInterval > c.MaxAge || c.RefreshTimeout > c.MaxAge-c.RefreshInterval || c.LoadTimeout > c.RefreshTimeout ||
		c.Concurrency < 1 || c.RetryMin <= 0 || c.RetryMax < c.RetryMin ||
		c.PublishInterval <= 0 || c.OutboxLease <= c.LoadTimeout || c.OutboxRetention <= 0 ||
		c.StreamMaxLen < 1 || c.BacklogThreshold < 1 || c.BacklogThreshold > c.StreamMaxLen ||
		c.HeartbeatInterval <= 0 || c.GroupLease < time.Millisecond || c.GroupLease/2 <= c.HeartbeatInterval || c.MaintenanceInterval <= 0 {
		return fmt.Errorf("invalid v3_config_cache timing or capacity")
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
