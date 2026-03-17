package service

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/repository/es"
)

type AIService interface {
	Query(ctx context.Context, query string) (string, error)
	Insert(ctx context.Context) error
}

type AIServiceImpl struct {
	agent  *react.Agent
	log    logger.Logger
	faqDAO dao.FAQResolutionDAO
	esDAO  es.FAQESRepo
}

func (s *AIServiceImpl) Insert(ctx context.Context) error {
	user, err := s.faqDAO.ListResolutionsByUser()
	if err != nil {
		return err
	}
}

func NewAIService(
	agent *react.Agent,
	log logger.Logger,
	faqDAO dao.FAQResolutionDAO,
	esDAO es.FAQESRepo,
) AIService {
	return &AIServiceImpl{
		agent:  agent,
		log:    log,
		faqDAO: faqDAO,
		esDAO:  esDAO,
	}
}

// Query 处理用户的提问
func (s *AIServiceImpl) Query(ctx context.Context, query string) (string, error) {
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
