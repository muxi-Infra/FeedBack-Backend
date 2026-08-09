package cache

import (
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

// IntegrationNonceStore 用于记录已经使用过的集成认证 nonce，防止请求重放。
type IntegrationNonceStore interface {
	MarkUsed(ctx context.Context, key string, expiration time.Duration) (bool, error)
}

type redisIntegrationNonceStore struct {
	client *redis.Client
}

// NewIntegrationNonceStore 创建基于 Redis 的集成认证 nonce 存储。
func NewIntegrationNonceStore(client *redis.Client) IntegrationNonceStore {
	return &redisIntegrationNonceStore{client: client}
}

func (s *redisIntegrationNonceStore) MarkUsed(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	if s.client == nil {
		return false, errors.New("redis client is nil")
	}
	if key == "" {
		return false, errors.New("nonce key is required")
	}
	if expiration <= 0 {
		return false, errors.New("nonce expiration must be positive")
	}
	return s.client.SetNX(ctx, key, "1", expiration).Result()
}
