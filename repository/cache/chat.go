package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/go-redis/redis/v8"
)

const ConversationExpiration = 1*time.Hour + 30*time.Minute

type InsertPosition bool

const (
	PositionTail InsertPosition = false
	PositionHead InsertPosition = true
)

type ChatCache interface {
	PushMessages(ctx context.Context, convID uint, atHead InsertPosition, msgs ...*schema.Message) error
	TrimMessageLeft(ctx context.Context, convID uint, n int64) error
	GetMSGList(ctx context.Context, convID uint) ([]*schema.Message, error)
	SetSummary(ctx context.Context, convID uint, summary *schema.Message) error
	GetSummary(ctx context.Context, convID uint) (*schema.Message, error)
}

type chatCache struct {
	client redis.Cmdable
}

func NewChatCache(client *redis.Client) ChatCache {
	return &chatCache{client: client}
}

// refreshExpiry 统一为该会话的所有相关 Key 续期
func (c *chatCache) refreshExpiry(ctx context.Context, pipe redis.Pipeliner, convID uint) {
	listKey := c.GetFullKey(fmt.Sprintf("list:%d", convID))
	summaryKey := c.GetFullKey(fmt.Sprintf("summary:%d", convID))

	pipe.Expire(ctx, listKey, ConversationExpiration)
	pipe.Expire(ctx, summaryKey, ConversationExpiration)
}

func (c *chatCache) PushMessages(ctx context.Context, convID uint, atHead InsertPosition, msgs ...*schema.Message) error {
	if len(msgs) == 0 {
		return nil
	}

	listKey := c.GetFullKey(fmt.Sprintf("list:%d", convID))
	values := make([]interface{}, 0, len(msgs))

	for i := 0; i < len(msgs); i++ {
		idx := i
		if atHead {
			idx = len(msgs) - 1 - i // LPush 需要逆序保持视觉顺序
		}
		data, err := json.Marshal(msgs[idx])
		if err != nil {
			return err
		}
		values = append(values, data)
	}

	pipe := c.client.Pipeline()
	if atHead {
		pipe.LPush(ctx, listKey, values...)
	} else {
		pipe.RPush(ctx, listKey, values...)
	}

	// 修改：同步更新 Summary 的过期时间
	c.refreshExpiry(ctx, pipe, convID)

	_, err := pipe.Exec(ctx)
	return err
}

func (c *chatCache) TrimMessageLeft(ctx context.Context, convID uint, n int64) error {
	if n <= 0 {
		return nil
	}
	listKey := c.GetFullKey(fmt.Sprintf("list:%d", convID))

	pipe := c.client.Pipeline()
	pipe.LTrim(ctx, listKey, n, -1)

	// 修改：同步更新 Summary 的过期时间
	c.refreshExpiry(ctx, pipe, convID)

	_, err := pipe.Exec(ctx)
	return err
}

func (c *chatCache) GetMSGList(ctx context.Context, convID uint) ([]*schema.Message, error) {
	listKey := c.GetFullKey(fmt.Sprintf("list:%d", convID))

	pipe := c.client.Pipeline()
	listRange := pipe.LRange(ctx, listKey, 0, -1)

	// 修改：读的时候也同步续期
	c.refreshExpiry(ctx, pipe, convID)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	listData, err := listRange.Result()
	if err != nil {
		return nil, err
	}

	messages := make([]*schema.Message, len(listData))
	for i, str := range listData {
		if err := json.Unmarshal([]byte(str), &messages[i]); err != nil {
			return nil, err
		}
	}

	return messages, nil
}

func (c *chatCache) SetSummary(ctx context.Context, convID uint, summary *schema.Message) error {
	if summary == nil {
		return nil
	}

	summaryKey := c.GetFullKey(fmt.Sprintf("summary:%d", convID))
	data, err := json.Marshal(summary)
	if err != nil {
		return err
	}

	// 这里不需要专门调用 refreshExpiry，因为 Set 已经带了过期时间
	// 但为了严谨，我们可以用 Pipeline 让 List 也同步续期
	pipe := c.client.Pipeline()
	pipe.Set(ctx, summaryKey, data, ConversationExpiration)
	pipe.Expire(ctx, c.GetFullKey(fmt.Sprintf("list:%d", convID)), ConversationExpiration)

	_, err = pipe.Exec(ctx)
	return err
}

func (c *chatCache) GetSummary(ctx context.Context, convID uint) (*schema.Message, error) {
	summaryKey := c.GetFullKey(fmt.Sprintf("summary:%d", convID))

	data, err := c.client.Get(ctx, summaryKey).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var msg schema.Message
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

func (c *chatCache) GetFullKey(s string) string {
	return fmt.Sprintf("feedback:chat:%s", s)
}
