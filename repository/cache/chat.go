package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
)

const ConversationExpiration = 1 * time.Hour

type ChatCache interface {
	Exists(ctx context.Context, convID string) (bool, error) // 新增
	SetConversationMeta(ctx context.Context, conv *model.Conversation) error
	PushMessage(ctx context.Context, convID string, msg model.Message) error
	GetFullConversation(ctx context.Context, convID string) (*model.Conversation, error)
	DeleteConversation(ctx context.Context, convID string) error
}

type chatCache struct {
	client redis.Cmdable
}

func NewChatCache(client *redis.Client) ChatCache {
	return &chatCache{client: client}
}

func (c *chatCache) SetConversationMeta(ctx context.Context, conv *model.Conversation) error {
	key := c.GetFullKey("meta:" + conv.ID)
	// 存储前清空 Messages 指针，避免冗余序列化存入 meta
	temp := conv.Messages
	conv.Messages = nil
	defer func() { conv.Messages = temp }()

	data, err := json.Marshal(conv)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, ConversationExpiration).Err()
}

func (c *chatCache) PushMessage(ctx context.Context, convID string, msg model.Message) error {
	listKey := c.GetFullKey("list:" + convID)
	metaKey := c.GetFullKey("meta:" + convID)
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	pipe := c.client.Pipeline()
	pipe.RPush(ctx, listKey, data)
	pipe.Expire(ctx, listKey, ConversationExpiration)
	pipe.Expire(ctx, metaKey, ConversationExpiration) // 只要说话，元数据也续期

	_, err = pipe.Exec(ctx)
	return err
}

// GetFullHistory 组合 meta 和 list，返回完整结构体
func (c *chatCache) GetFullConversation(ctx context.Context, convID string) (*model.Conversation, error) {
	metaKey := c.GetFullKey("meta:" + convID)
	listKey := c.GetFullKey("list:" + convID)

	// 使用 Pipeline 同时获取元数据和消息列表，并一键续期
	pipe := c.client.Pipeline()
	metaGet := pipe.Get(ctx, metaKey)
	listRange := pipe.LRange(ctx, listKey, 0, -1)
	pipe.Expire(ctx, metaKey, ConversationExpiration)
	pipe.Expire(ctx, listKey, ConversationExpiration)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	// 1. 解析元数据 (Conversation 壳子)
	metaData, err := metaGet.Result()
	if err != nil {
		return nil, fmt.Errorf("fetch meta error: %w", err)
	}

	var conv model.Conversation
	if err := json.Unmarshal([]byte(metaData), &conv); err != nil {
		return nil, err
	}

	// 2. 解析消息列表 (Messages)
	listData, err := listRange.Result()
	if err != nil {
		return nil, fmt.Errorf("fetch messages error: %w", err)
	}

	messages := make([]model.Message, len(listData))
	for i, str := range listData {
		if err := json.Unmarshal([]byte(str), &messages[i]); err != nil {
			return nil, err
		}
	}

	// 3. 装配结果
	conv.Messages = messages
	return &conv, nil
}

func (c *chatCache) DeleteConversation(ctx context.Context, convID string) error {
	pipe := c.client.Pipeline()
	pipe.Del(ctx, c.GetFullKey("meta:"+convID))
	pipe.Del(ctx, c.GetFullKey("list:"+convID))
	_, err := pipe.Exec(ctx)
	return err
}

// Exists 快速判断会话是否存在，并顺便续期
func (c *chatCache) Exists(ctx context.Context, convID string) (bool, error) {
	metaKey := c.GetFullKey("meta:" + convID)
	listKey := c.GetFullKey("list:" + convID)

	// 使用 Exists 检查 metaKey
	n, err := c.client.Exists(ctx, metaKey).Result()
	if err != nil {
		return false, err
	}

	exists := n > 0

	// 如果存在，顺便异步续期（滑动窗口）
	if exists {
		go func() {
			// 这里使用新的上下文，防止主流程结束导致续期失败
			bgCtx := context.Background()
			pipe := c.client.Pipeline()
			pipe.Expire(bgCtx, metaKey, ConversationExpiration)
			pipe.Expire(bgCtx, listKey, ConversationExpiration)
			_, _ = pipe.Exec(bgCtx)
		}()
	}

	return exists, nil
}
func (c *chatCache) GetFullKey(s string) string {
	return fmt.Sprintf("feedback:chat:%s", s)
}
