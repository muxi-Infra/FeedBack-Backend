package ioc

import (
	"context"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/config"

	"github.com/go-redis/redis/v8"
)

func InitRedis(conf *config.RedisConfig, log logger.Logger) (*redis.Client, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	rdb := redis.NewClient(&redis.Options{
		Addr:         conf.Addr,
		Password:     conf.Password,
		DB:           conf.DB,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Warn("redis_startup_degraded", logger.String("error_class", "connection"))
	}
	return rdb, func() { _ = rdb.Close() }
}
