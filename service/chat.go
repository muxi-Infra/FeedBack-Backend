package service

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/repository/cache"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/repository/es"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
)

type ChatService interface {
	Query(ctx context.Context, query string, tableIdentity, userID string) (string, error)
	Insert(ctx context.Context, tableIdentify string) error
	GetHistory(ctx context.Context, chatID string) (*domain.Conversation, error)
}

type ChatServiceImpl struct {
	agent    *react.Agent
	log      logger.Logger
	faqDAO   dao.FAQDAO
	esDAO    es.FAQESRepo
	embedder embedding.Embedder
	cache    cache.ChatCache
}

func NewChatService(
	agent *react.Agent,
	log logger.Logger,
	faqDAO dao.FAQDAO,
	esDAO es.FAQESRepo,
	embedder embedding.Embedder,
	cache cache.ChatCache,
) ChatService {
	return &ChatServiceImpl{
		agent:    agent,
		log:      log,
		faqDAO:   faqDAO,
		esDAO:    esDAO,
		embedder: embedder,
		cache:    cache,
	}
}

func (s *ChatServiceImpl) GetHistory(ctx context.Context, chatID string) (*domain.Conversation, error) {
	conv, err := s.cache.GetFullConversation(ctx, chatID)
	if err != nil {
		return nil, err
	}
	// 2. 将 []model.Message 转换为 []domain.Message
	domainMessages := make([]domain.Message, len(conv.Messages))
	for i, m := range conv.Messages {
		domainMessages[i] = domain.Message{
			ID:             m.ID,
			ConversationID: m.ConversationID,
			Role:           domain.Role(m.Role), // 强制类型转换
			Content:        m.Content,
			CreatedAt:      m.CreatedAt,
			Metadata:       m.Metadata,
		}
	}

	// 3. 组装并返回 domain.Conversation
	return &domain.Conversation{
		ID:           conv.ID,
		UserID:       conv.UserID,
		CreatedAt:    conv.CreatedAt,
		UpdatedAt:    conv.UpdatedAt,
		LastMessage:  domainMessages[len(domainMessages)-1].Content,
		MessageCount: len(domainMessages),
		Messages:     domainMessages,
	}, nil

}

func (s *ChatServiceImpl) Insert(ctx context.Context, tableIdentify string) error {
	records, err := s.faqDAO.GetFAQRecords(&tableIdentify)
	if err != nil {
		return err
	}
	var texts = make([]string, len(records))
	for i := range records {
		texts[i] = fmt.Sprintf("问题名称: %v\n解决方案:%v", records[i].Record["问题名称"], records[i].Record["解决方案"])
	}

	embedStrs, err := s.embedder.EmbedStrings(ctx, texts)
	if err != nil {
		return err
	}

	for i := range records {
		err := s.esDAO.SaveWithVector(ctx, &records[i], embedStrs[i])
		if err != nil {
			return err
		}
	}

	return nil
}

// Query 处理用户的提问
func (s *ChatServiceImpl) Query(ctx context.Context, query string, tableIdentity, userID string) (string, error) {
	// 1. 将字符串 Query 转化为 Agent 接受的 Message 格式
	input := []*schema.Message{
		schema.UserMessage(query),
	}

	// 2. 直接调用预设好的 Agent
	// 因为你的 Agent 已经预设了一切（Prompt, Tools, LLM），这里直接 Generate 即可
	output, err := s.agent.Generate(ctx, input)
	if err != nil {
		s.log.Error("Agent generate failed",
			logger.String("query", query),
			logger.String("error", err.Error()),
		)
		return "", fmt.Errorf("AI 助理执行失败: %w", err)

	}

	if output.Content == "" {
		return "抱歉，我未能生成有效的回答，请稍后再试。", nil
	}
	//TODO 记得抽象方法
	convID := tableIdentity + userID
	now := time.Now()

	// 存储到会话历史中去
	exists, err := s.cache.Exists(ctx, convID)
	if err != nil {
		return "", err
	}

	if !exists {
		err := s.cache.SetConversationMeta(ctx, &model.Conversation{
			ID:        convID,
			UserID:    userID,
			CreatedAt: now,
			UpdatedAt: now,
		})
		if err != nil {
			return "", err
		}
	}

	//用户消息
	err = s.cache.PushMessage(ctx, convID, model.Message{
		ID:             uuid.New().String(),
		ConversationID: convID,
		Role:           model.RoleUser,
		Content:        query,
		CreatedAt:      now,
	})
	if err != nil {
		return "", err
	}

	// robot消息
	err = s.cache.PushMessage(ctx, convID, model.Message{
		ID:             uuid.New().String(),
		ConversationID: convID,
		Role:           model.RoleSystem,
		Content:        output.Content,
		CreatedAt:      now,
	})
	if err != nil {
		return "", err
	}

	return output.Content, nil
}
