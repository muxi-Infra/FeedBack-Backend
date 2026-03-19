package model

import (
	"time"

	"github.com/cloudwego/eino/schema"
)

// Conversation 会话主体，用于管理一组消息
type Conversation struct {
	ID        string            `json:"id" bson:"_id"`
	UserID    string            `json:"user_id" bson:"user_id"`
	CreatedAt time.Time         `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time         `json:"updated_at" bson:"updated_at"`
	Messages  []*schema.Message `json:"messages" bson:"messages"`
}
