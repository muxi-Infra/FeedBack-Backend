package dao

import (
	"context"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"gorm.io/gorm"
)

type ChatDAO interface {
	// 会话相关
	SaveConversation(ctx context.Context, conv *model.Conversation) error
	FirstOrCreateConversation(ctx context.Context, tableIdentity, userID string) (*model.Conversation, error)
	FirstConversation(ctx context.Context, convID uint) (*model.Conversation, error)
	// 消息相关
	CreateMessages(ctx context.Context, msgs ...*model.Message) error
	// GetMessagesByCursor 使用游标(Message ID)获取特定会话的消息列表
	// lastID: 上一次查询最后一条消息的 ID，首次查询传 0
	// limit: 获取的数量
	GetMessagesByCursor(ctx context.Context, convID uint, lastID uint, limit int) ([]*model.Message, error)
}

type chatDAO struct {
	db *gorm.DB
}

func NewChatDAO(db *gorm.DB) ChatDAO {
	return &chatDAO{db: db}
}

// SaveConversation 创建或更新会话元数据,一般只会用于创建
func (c *chatDAO) SaveConversation(ctx context.Context, conv *model.Conversation) error {
	return c.db.WithContext(ctx).Save(conv).Error
}

func (c *chatDAO) FirstOrCreateConversation(ctx context.Context, tableIdentity, userID string) (*model.Conversation, error) {
	var conv model.Conversation
	// 定义 1 小时前的时间点
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	// 1. 匹配 user_id 和 table_identity
	// 2. 存在（EXISTS）一条属于该会话的消息，且其 update_at > 一小时前

	err := c.db.WithContext(ctx).
		Where("user_id = ? AND table_identity = ?", userID, tableIdentity).
		Where("updated_at > ?", oneHourAgo).
		FirstOrCreate(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// CreateMessage 插入单条消息
func (c *chatDAO) CreateMessages(ctx context.Context, msgs ...*model.Message) error {
	return c.db.WithContext(ctx).Create(msgs).Error
}

// GetMessagesByCursor 游标分页查询
// 逻辑：查找 ID > lastID 的消息，按 ID 正序排列（最早的消息在前）
// 如果需要查看历史（往回滚），则逻辑相反：ID < lastID，按 ID 倒序
func (c *chatDAO) GetMessagesByCursor(ctx context.Context, convID uint, lastID uint, limit int) ([]*model.Message, error) {
	var msgs []*model.Message

	query := c.db.WithContext(ctx).
		Where("conversation_id = ?", convID)

	// 如果传入了游标 ID，则只获取比该 ID 更早的消息
	if lastID > 0 {
		query = query.Where("id < ?", lastID)
	}

	err := query.Order("id DESC").Limit(limit).Find(&msgs).Error
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

func (c *chatDAO) FirstConversation(ctx context.Context, convID uint) (*model.Conversation, error) {
	var conv model.Conversation
	err := c.db.WithContext(ctx).First(&conv, "id = ?", convID).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}
