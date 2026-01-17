package cache

import (
	"context"
	_ "embed"
	"errors"

	"github.com/go-redis/redis/v8"
)

var (
	//go:embed scripts/get_and_reset.lua
	getAndReset string
)

type MessageCountCache interface {
	GetAndReset(key string) (uint64, error)
	Increment(key string) error
}

type messageCountCache struct {
	cache             redis.Cmdable
	getAndResetScript *redis.Script
}

func NewMessageCache(cache *redis.Client) MessageCountCache {
	return &messageCountCache{
		cache:             cache,
		getAndResetScript: redis.NewScript(getAndReset),
	}
}

func (m *messageCountCache) GetAndReset(key string) (uint64, error) {
	ctx := context.Background()
	res, err := m.getAndResetScript.Run(ctx, m.cache, []string{key}).Result()
	if err != nil {
		return 0, err
	}

	val, ok := res.(int64)
	if !ok {
		return 0, errors.New("unexpected return type")
	}

	return uint64(val), nil
}

func (m *messageCountCache) Increment(key string) error {
	ctx := context.Background()
	return m.cache.Incr(ctx, key).Err()
}
