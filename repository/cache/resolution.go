package cache

import (
	"context"
	_ "embed"

	"github.com/go-redis/redis/v8"
)

var (
	//go:embed scripts/inc_a_and_dec_b.lua
	incAAndDecBScriptSrc string

	//go:embed scripts/inc_a_and_get_b.lua
	incAAndGetBScriptSrc string

	//go:embed scripts/get_a_and_get_b.lua
	getAAndGetBScriptSrc string
)

type FAQResolutionStateCache interface {
	IncAAndGetB(ctx context.Context, keyA, keyB string) (uint64, uint64, error)
	IncAAndDecB(ctx context.Context, keyA, keyB string) (uint64, uint64, error)
	GetAAndGetB(ctx context.Context, keyA, keyB string) (uint64, uint64, error)
	Delete(ctx context.Context, keys ...string) error
}

type faqResolutionStateCache struct {
	cache             redis.Cmdable
	incAAndDecBScript *redis.Script
	incAAndGetBScript *redis.Script
	getAAndGetBScript *redis.Script
}

func NewFAQResolutionStateCache(cache *redis.Client) FAQResolutionStateCache {
	return &faqResolutionStateCache{
		cache:             cache,
		incAAndDecBScript: redis.NewScript(incAAndDecBScriptSrc),
		incAAndGetBScript: redis.NewScript(incAAndGetBScriptSrc),
		getAAndGetBScript: redis.NewScript(getAAndGetBScriptSrc),
	}
}

// IncAAndGetB keyA 自增 1 并同时返回 keyA 和 keyB 的值
func (c *faqResolutionStateCache) IncAAndGetB(ctx context.Context, keyA, keyB string) (uint64, uint64, error) {
	res, err := c.incAAndGetBScript.Run(ctx, c.cache, []string{keyA, keyB}).Result()
	if err != nil {
		return 0, 0, err
	}

	vals := res.([]interface{})
	a := uint64(vals[0].(int64))
	b := uint64(vals[1].(int64))

	return a, b, nil
}

func (c *faqResolutionStateCache) IncAAndDecB(ctx context.Context, keyA, keyB string) (uint64, uint64, error) {
	res, err := c.incAAndDecBScript.Run(ctx, c.cache, []string{keyA, keyB}).Result()
	if err != nil {
		return 0, 0, err
	}

	vals := res.([]interface{})
	a := uint64(vals[0].(int64))
	b := uint64(vals[1].(int64))

	return a, b, nil
}

func (c *faqResolutionStateCache) GetAAndGetB(ctx context.Context, keyA, keyB string) (uint64, uint64, error) {
	res, err := c.getAAndGetBScript.Run(ctx, c.cache, []string{keyA, keyB}).Result()
	if err != nil {
		return 0, 0, err
	}

	vals := res.([]interface{})
	a := uint64(vals[0].(int64))
	b := uint64(vals[1].(int64))

	return a, b, nil
}

func (c *faqResolutionStateCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	err := c.cache.Del(ctx, keys...).Err()
	return err
}
