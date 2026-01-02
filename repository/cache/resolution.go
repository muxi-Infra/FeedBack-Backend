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
)

type FAQResolutionStateCache interface {
	IncAAndGetB(keyA, keyB string) (uint64, uint64, error)
	IncAAndDecB(keyA, keyB string) (uint64, uint64, error)
}

type faqResolutionStateCache struct {
	cache             redis.Cmdable
	incAAndDecBScript *redis.Script
	incAAndGetBScript *redis.Script
}

func NewFAQResolutionStateCache(cache *redis.Client) FAQResolutionStateCache {
	return &faqResolutionStateCache{
		cache:             cache,
		incAAndDecBScript: redis.NewScript(incAAndDecBScriptSrc),
		incAAndGetBScript: redis.NewScript(incAAndGetBScriptSrc),
	}
}

// IncAAndGetB keyA 自增 1 并同时返回 keyA 和 keyB 的值
func (c *faqResolutionStateCache) IncAAndGetB(keyA, keyB string) (uint64, uint64, error) {
	ctx := context.Background()
	res, err := c.incAAndGetBScript.Run(ctx, c.cache, []string{keyA, keyB}).Result()
	if err != nil {
		return 0, 0, err
	}

	vals := res.([]interface{})
	a := uint64(vals[0].(int64))
	b := uint64(vals[1].(int64))

	return a, b, nil
}

func (c *faqResolutionStateCache) IncAAndDecB(keyA, keyB string) (uint64, uint64, error) {
	ctx := context.Background()
	res, err := c.incAAndDecBScript.Run(ctx, c.cache, []string{keyA, keyB}).Result()
	if err != nil {
		return 0, 0, err
	}

	vals := res.([]interface{})
	a := uint64(vals[0].(int64))
	b := uint64(vals[1].(int64))

	return a, b, nil
}
