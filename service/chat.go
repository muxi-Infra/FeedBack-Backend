package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

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

func (s *ChatServiceImpl) Chat(ctx context.Context, query string, tableIdentity, userID string) (<-chan string, <-chan error) {
	out := make(chan string)
	errCh := make(chan error, 1)

	go func() {
		defer close(out)
		defer close(errCh)

		//如果存在list的情况,添加到上下文对话中去
		convID := tableIdentity + userID
		now := time.Now()

		// 存储到会话历史中去
		exists, err := s.cache.Exists(ctx, convID)
		if err != nil {
			errCh <- fmt.Errorf("检查对话存在失败: %w", err)
			return
		}

		// 如果不存在对话需要初始化
		if !exists {
			err := s.cache.SetConversationMeta(ctx, &model.Conversation{
				ID:        convID,
				UserID:    userID,
				CreatedAt: now,
				UpdatedAt: now,
			})
			if err != nil {
				s.log.Error("初始化对话失败",
					logger.String("conv_id", convID),
					logger.String("error", err.Error()),
				)
				errCh <- fmt.Errorf("初始化对话失败: %w", err)
				return
			}
		}

		// 写入用户消息
		err = s.cache.PushMessage(ctx, convID, schema.UserMessage(query))
		if err != nil {
			s.log.Error("写入用户消息失败",
				logger.String("conv_id", convID),
				logger.String("query", query),
				logger.String("error", err.Error()),
			)
			errCh <- fmt.Errorf("写入用户消息失败: %w", err)
			return
		}

		// 读取历史对话
		conversation, err := s.cache.GetFullConversation(ctx, convID)
		if err != nil {
			s.log.Error("读取对话历史失败",
				logger.String("conv_id", convID),
				logger.String("error", err.Error()),
			)
			errCh <- fmt.Errorf("读取对话历史失败: %w", err)
			return
		}

		// conversation.Messages
		// 将字符串 Query 转化为 Agent 接受的 Message 格式
		input := conversation.Messages

		// 直接调用预设好的 Agent
		stream, err := s.agent.Stream(ctx, input)
		if err != nil {
			s.log.Error("Agent generate failed",
				logger.String("query", query),
				logger.String("error", err.Error()),
			)
			errCh <- fmt.Errorf("AI 助理执行失败: %w", err)
			return

		}
		defer stream.Close()
		var fullReply strings.Builder
		var output *schema.Message
		for {
			frame, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					// 结束
					break
				}
				// 其他错误
				s.log.Error("Agent stream receive failed",
					logger.String("query", query),
					logger.String("error", err.Error()),
				)
				errCh <- fmt.Errorf("AI 助理执行失败: %w", err)
				return
			}
			if frame == nil || frame.Content == "" {
				continue
			}
			if output == nil {
				output = frame
			}
			out <- frame.Content
			fmt.Printf("%#v\n", frame)
			fullReply.WriteString(frame.Content)
		}

		// 写入 robot 消息
		output.Content = fullReply.String()
		err = s.cache.PushMessage(ctx, convID, output)
		if err != nil {
			s.log.Error("写入机器人消息失败",
				logger.String("conv_id", convID),
				logger.String("reply", output.Content),
				logger.String("error", err.Error()),
			)
			errCh <- fmt.Errorf("写入机器人消息失败: %w", err)
			return
		}

	}()

	return out, errCh
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
