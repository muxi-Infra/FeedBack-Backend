package model

import (
	"time"
)

// Role 定义消息角色类型
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message 每一条具体的对话消息
type Message struct {
	// 基础字段
	ID             string    `json:"id" bson:"_id"`                          // 消息唯一标识 (UUID)
	ConversationID string    `json:"conversation_id" bson:"conversation_id"` // 所属会话 ID
	Role           Role      `json:"role" bson:"role"`                       // 角色: system/user/assistant
	Content        string    `json:"content" bson:"content"`                 // 消息文本内容
	CreatedAt      time.Time `json:"created_at" bson:"created_at"`           // 创建时间

	// 扩展字段 (Metadata)
	// 使用 map[string]any 可以灵活存储：Token 消耗、模型名称、耗时、引用来源等
	Metadata map[string]any `json:"metadata,omitempty" bson:"metadata,omitempty"`
}

// Conversation 会话主体，用于管理一组消息
type Conversation struct {
	ID        string    `json:"id" bson:"_id"`
	UserID    string    `json:"user_id" bson:"user_id"`
	Messages  []Message `json:"messages" bson:"messages"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}
