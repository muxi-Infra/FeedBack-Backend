package domain

import (
	"time"
)

// Conversation 会话主体，用于管理一组消息,目前附加消息比较少,后续可以根据需要逐步拓展,例如title等
type Conversation struct {
	ID        uint      `json:"id" bson:"_id"`
	UserID    string    `json:"user_id" bson:"user_id"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

type Message struct {
	ID             uint      `json:"id" bson:"_id"`
	CreatedAt      time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" bson:"updated_at"`
	ConversationID uint      `json:"conversation_id"`
	Role           string    `json:"role"` // user, assistant, system
	Content        string    `json:"content"`
}
