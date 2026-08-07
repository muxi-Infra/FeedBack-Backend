package cache

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
)

const (
	projectConfigChangeStream = "feedback:project_config_changed"
)

// ProjectConfigEventBus 负责通过 Redis Stream 传递项目配置变更通知。
// 事件只携带 project_id，不携带公钥、table_token 等敏感配置。
// todo 后续可以加一个任务队列统一处理所有任务
type ProjectConfigEventBus interface {
	PublishProjectChanged(ctx context.Context, projectID string) error
	ConsumeProjectChanged(ctx context.Context, handler func(projectID string) error)
}

type redisProjectConfigEventBus struct {
	client        *redis.Client
	log           logger.Logger
	consumerGroup string
	consumerName  string
}

func NewProjectConfigEventBus(client *redis.Client, log logger.Logger) ProjectConfigEventBus {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "unknown"
	}
	instanceID := fmt.Sprintf("%s-%d", hostname, os.Getpid())
	return &redisProjectConfigEventBus{
		client:        client,
		log:           log,
		consumerGroup: "feedback-config-refresh-" + instanceID,
		consumerName:  instanceID,
	}
}

func (b *redisProjectConfigEventBus) PublishProjectChanged(ctx context.Context, projectID string) error {
	if b.client == nil {
		return errors.New("redis client is nil")
	}
	if projectID == "" {
		return errors.New("project_id is required")
	}
	return b.client.XAdd(ctx, &redis.XAddArgs{
		Stream: projectConfigChangeStream,
		Values: map[string]interface{}{
			"project_id": projectID,
			"changed_at": strconv.FormatInt(time.Now().Unix(), 10),
		},
	}).Err()
}

func (b *redisProjectConfigEventBus) ConsumeProjectChanged(ctx context.Context, handler func(projectID string) error) {
	if b.client == nil || handler == nil {
		return
	}
	if err := b.client.XGroupCreateMkStream(ctx, projectConfigChangeStream, b.consumerGroup, "$").Err(); err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		b.log.Error("创建项目配置 Redis Stream 消费组失败", logger.String("error", err.Error()))
		return
	}

	for {
		if ctx.Err() != nil {
			return
		}
		result, err := b.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    b.consumerGroup,
			Consumer: b.consumerName,
			Streams:  []string{projectConfigChangeStream, ">"},
			Count:    20,
			Block:    5 * time.Second,
		}).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) || ctx.Err() != nil {
				continue
			}
			b.log.Error("读取项目配置 Redis Stream 失败", logger.String("error", err.Error()))
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range result {
			for _, message := range stream.Messages {
				projectID, ok := message.Values["project_id"].(string)
				if !ok || projectID == "" {
					b.log.Warn("忽略无效的项目配置刷新事件", logger.String("message_id", message.ID))
					_ = b.client.XAck(ctx, projectConfigChangeStream, b.consumerGroup, message.ID).Err()
					continue
				}
				if err := handler(projectID); err != nil {
					b.log.Error("处理项目配置刷新事件失败", logger.String("project_id", projectID), logger.String("error", err.Error()))
					continue
				}
				if err := b.client.XAck(ctx, projectConfigChangeStream, b.consumerGroup, message.ID).Err(); err != nil {
					b.log.Error("确认项目配置刷新事件失败", logger.String("message_id", message.ID), logger.String("error", err.Error()))
				}
			}
		}
	}
}
