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

const projectConfigChangeStreamV3 = "feedback:v3:project_config_changed"

// ProjectConfigEventBusV3 负责广播项目配置变更事件，事件只携带项目 ID。
type ProjectConfigEventBusV3 interface {
	PublishProjectChanged(context.Context, string) error
	ConsumeProjectChanged(context.Context, func(string) error)
}

type redisProjectConfigEventBusV3 struct {
	client *redis.Client
	log    logger.Logger
	group  string
	name   string
}

// NewProjectConfigEventBusV3 创建 V3 配置变更事件总线。
func NewProjectConfigEventBusV3(client *redis.Client, log logger.Logger) ProjectConfigEventBusV3 {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "unknown"
	}
	name := fmt.Sprintf("%s-%d", host, os.Getpid())
	return &redisProjectConfigEventBusV3{
		client: client,
		log:    log,
		group:  "feedback-v3-config-refresh-" + name,
		name:   name,
	}
}

func (b *redisProjectConfigEventBusV3) PublishProjectChanged(ctx context.Context, projectID string) error {
	if b.client == nil {
		return errors.New("redis client is nil")
	}
	if strings.TrimSpace(projectID) == "" {
		return errors.New("project_id is required")
	}
	return b.client.XAdd(ctx, &redis.XAddArgs{
		Stream: projectConfigChangeStreamV3,
		Values: map[string]interface{}{
			"project_id": projectID,
			"changed_at": strconv.FormatInt(time.Now().Unix(), 10),
		}}).Err()
}

func (b *redisProjectConfigEventBusV3) ConsumeProjectChanged(ctx context.Context, handler func(string) error) {
	if b.client == nil || handler == nil {
		return
	}
	if err := b.client.XGroupCreateMkStream(ctx, projectConfigChangeStreamV3, b.group, "$").Err(); err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		b.log.Error("创建 V3 项目配置 Redis 消费组失败", logger.String("error", err.Error()))
		return
	}
	for ctx.Err() == nil {
		items, err := b.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    b.group,
			Consumer: b.name,
			Streams: []string{
				projectConfigChangeStreamV3,
				">",
			},
			Count: 20,
			Block: 5 * time.Second,
		}).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) || ctx.Err() != nil {
				continue
			}
			b.log.Error("读取 V3 项目配置 Redis 事件失败", logger.String("error", err.Error()))
			continue
		}

		for _, stream := range items {
			for _, msg := range stream.Messages {
				projectID, ok := msg.Values["project_id"].(string)
				if ok && projectID != "" {
					if err := handler(projectID); err != nil {
						b.log.Error("处理 V3 配置刷新事件失败", logger.String("project_id", projectID), logger.String("error", err.Error()))
						continue
					}
				}
				if err := b.client.XAck(ctx, projectConfigChangeStreamV3, b.group, msg.ID).Err(); err != nil {
					b.log.Error("确认 V3 配置刷新事件失败", logger.String("message_id", msg.ID), logger.String("error", err.Error()))
				}
			}
		}
	}
}
