package cache

import (
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

// IntegrationNonceStoreV3 防止同一个交换请求被重复使用。
type IntegrationNonceStoreV3 interface {
	MarkUsed(ctx context.Context, key string, expiration time.Duration) (bool, error)
}

type redisIntegrationNonceStoreV3 struct {
	client *redis.Client
}

func NewIntegrationNonceStoreV3(client *redis.Client) IntegrationNonceStoreV3 {
	return &redisIntegrationNonceStoreV3{
		client: client,
	}
}

func (s *redisIntegrationNonceStoreV3) MarkUsed(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	if s.client == nil {
		return false, errors.New("redis client is nil")
	}
	if key == "" || expiration <= 0 {
		return false, errors.New("invalid nonce arguments")
	}
	return s.client.SetNX(ctx, key, "1", expiration).Result()
}
