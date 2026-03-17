package service

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/repository/es"
)

type ChatService interface {
	Query(ctx context.Context, query string) (string, error)
	Insert(ctx context.Context, tableIdentify string) error
}

type ChatServiceImpl struct {
	agent    *react.Agent
	log      logger.Logger
	faqDAO   dao.FAQDAO
	esDAO    es.FAQESRepo
	embedder embedding.Embedder
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

func NewChatService(
	agent *react.Agent,
	log logger.Logger,
	faqDAO dao.FAQDAO,
	esDAO es.FAQESRepo,
	embedder embedding.Embedder,
) ChatService {
	return &ChatServiceImpl{
		agent:    agent,
		log:      log,
		faqDAO:   faqDAO,
		esDAO:    esDAO,
		embedder: embedder,
	}
}

// Query 处理用户的提问
func (s *ChatServiceImpl) Query(ctx context.Context, query string) (string, error) {
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

	// 3. 提取 Agent 的最终回答
	// React Agent 的输出通常是经过推理后的最后一条助理消息
	if len(output.Content) > 0 {
		return output.Content, nil
	}

	return "抱歉，我未能生成有效的回答，请稍后再试。", nil
}
