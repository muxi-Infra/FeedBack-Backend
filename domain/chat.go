package domain

import (
	"time"

	"github.com/cloudwego/eino/schema"
)

// Conversation 会话主体，用于管理一组消息
type Conversation struct {
	ID        string    `json:"id" bson:"_id"`
	UserID    string    `json:"user_id" bson:"user_id"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`

	// 冗余一些统计信息，方便在列表页展示
	LastMessage  string `json:"last_message" bson:"last_message"`
	MessageCount int    `json:"message_count" bson:"message_count"`

	Messages []*schema.Message `json:"messages" bson:"messages"`
}
