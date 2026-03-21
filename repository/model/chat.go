package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"

	"github.com/cloudwego/eino/schema"
	"gorm.io/gorm"
)

// 1. 定义 EinoMessage 类型，用于适配 GORM 的 JSON 存储
type EinoMessage struct {
	*schema.Message
}

const (
	User      = "user"
	Assistant = "assistant"
	System    = "system"
)

// 实现 sql.Scanner 接口：从数据库读取时自动反序列化
func (e *EinoMessage) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal JSONB value")
	}
	// 初始化内部指针防止 panic
	e.Message = &schema.Message{}
	return json.Unmarshal(bytes, e.Message)
}

// 实现 driver.Valuer 接口：写入数据库时自动序列化为 JSON
func (e EinoMessage) Value() (driver.Value, error) {
	if e.Message == nil {
		return nil, nil
	}
	return json.Marshal(e.Message)
}

// 2. Conversation 会话主体
type Conversation struct {
	gorm.Model
	UserID        string `gorm:"index;size:64" json:"user_id"`
	TableIdentity string `gorm:"size:64" json:"table_identity"`
}

// 3. Message 消息详情 (全量流水表)
type Message struct {
	gorm.Model
	ConversationID uint   `gorm:"index;size:64" json:"conversation_id"`
	Role           string `gorm:"size:20" json:"role"` // user, assistant, system
	Content        string `gorm:"type:longtext" json:"content"`
	// 使用自定义类型，GORM 会自动将其存为 JSON
	RawData EinoMessage `gorm:"type:json" json:"raw_data"`
}
