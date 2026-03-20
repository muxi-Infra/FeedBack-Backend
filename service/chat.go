package service

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/repository/cache"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/repository/es"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
)

type ChatService interface {
	Query(ctx context.Context, query string, convID uint) (string, error)
	Insert(ctx context.Context, tableIdentify string) error
	GetConversation(ctx context.Context, tableIdentify, userID string) (*domain.Conversation, error)
	GetHistory(ctx context.Context, convID uint, lastID uint, limit int) ([]*domain.Message, error)
}

type ChatServiceImpl struct {
	agent           *react.Agent
	log             logger.Logger
	faqDAO          dao.FAQDAO
	chatDAO         dao.ChatDAO
	esDAO           es.FAQESRepo
	embedder        embedding.Embedder
	cache           cache.ChatCache
	summaryRunnable compose.Runnable[[]*schema.Message, *schema.Message]
}

func NewChatService(
	agent *react.Agent,
	log logger.Logger,
	faqDAO dao.FAQDAO,
	esDAO es.FAQESRepo,
	embedder embedding.Embedder,
	cache cache.ChatCache,
	chatDAO dao.ChatDAO,
	summaryRunnable compose.Runnable[[]*schema.Message, *schema.Message],
) ChatService {
	return &ChatServiceImpl{
		agent:           agent,
		log:             log,
		faqDAO:          faqDAO,
		esDAO:           esDAO,
		embedder:        embedder,
		cache:           cache,
		chatDAO:         chatDAO,
		summaryRunnable: summaryRunnable,
	}
}

func (s *ChatServiceImpl) GetHistory(ctx context.Context, convID uint, lastID uint, limit int) ([]*domain.Message, error) {
	message, err := s.chatDAO.GetMessagesByCursor(ctx, convID, lastID, limit)
	if err != nil {
		return nil, err
	}
	var res = make([]*domain.Message, len(message))
	for i := range message {
		res[i] = &domain.Message{
			ID:             message[i].ID,
			CreatedAt:      message[i].CreatedAt,
			UpdatedAt:      message[i].UpdatedAt,
			ConversationID: message[i].ConversationID,
			Role:           message[i].Role,
			Content:        message[i].Content,
		}
	}
	return res, nil
}

func (s *ChatServiceImpl) GetConversation(ctx context.Context, tableIdentify, userID string) (*domain.Conversation, error) {
	conversation, err := s.chatDAO.FirstOrCreateConversation(ctx, tableIdentify, userID)
	if err != nil {
		return nil, err
	}
	// 3. 组装并返回 domain.Conversation
	return &domain.Conversation{
		ID:        conversation.ID,
		UserID:    conversation.UserID,
		CreatedAt: conversation.CreatedAt,
		UpdatedAt: conversation.UpdatedAt,
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
func (s *ChatServiceImpl) Query(ctx context.Context, query string, convID uint) (string, error) {

	// 获取会话信息以得到会话的id
	conversation, err := s.chatDAO.FirstConversation(ctx, convID)
	if err != nil {
		return "", err
	}

	// 读取历史对话
	msgs, err := s.cache.GetMSGList(ctx, conversation.ID)
	if err != nil {
		return "", err
	}

	// 获取上下文
	sum, err := s.cache.GetSummary(ctx, conversation.ID)
	if err != nil {
		return "", err
	}

	// 如果达到阈值进行上下文压缩
	if len(msgs) >= 10 {
		var tmps = msgs[:5]

		if sum != nil {
			//上次的总结和被压缩的上下文
			tmps = append([]*schema.Message{sum}, tmps...)
		}
		// 总结
		summary, err := s.summary(ctx, tmps)
		if err != nil {
			return "", err
		}
		// 设置总结
		err = s.cache.SetSummary(ctx, conversation.ID, summary)
		if err != nil {
			return "", err
		}
		// 清除被压缩的上下文
		err = s.cache.TrimMessageLeft(ctx, conversation.ID, 5)
		if err != nil {
			return "", err
		}
		//清理上下文保证获取到的上下文是正常的
		msgs = msgs[5:]

	}

	if sum != nil {
		msgs = append([]*schema.Message{sum}, msgs...)
	}

	//添加用户消息
	userMSG := schema.UserMessage(query)
	msgs = append(msgs, userMSG)

	// 直接调用预设好的 Agent
	output, err := s.agent.Generate(ctx, msgs)
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

	// TODO需要做事务保证一致性
	// Redis
	err = s.cache.PushMessages(ctx, conversation.ID, cache.PositionTail, userMSG, output)
	if err != nil {
		return "", err
	}
	// MYSQL
	//创建用户消息
	err = s.chatDAO.CreateMessages(ctx, &model.Message{
		ConversationID: conversation.ID,
		Role:           model.User,
		Content:        userMSG.Content,
		RawData:        model.EinoMessage{Message: userMSG},
	}, &model.Message{
		ConversationID: conversation.ID,
		Role:           model.Assistant,
		Content:        output.Content,
		RawData:        model.EinoMessage{Message: output},
	})
	if err != nil {
		return "", err
	}

	// 刷新数据库会话更新时间
	err = s.chatDAO.SaveConversation(ctx, conversation)
	if err != nil {
		return "", err
	}

	return output.Content, nil
}

func (s *ChatServiceImpl) summary(ctx context.Context, msgs []*schema.Message) (*schema.Message, error) {

	summary, err := s.summaryRunnable.Invoke(ctx, msgs)
	if err != nil {
		return nil, err
	}

	return summary, nil
}
