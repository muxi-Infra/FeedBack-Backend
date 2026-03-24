package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/repository/cache"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/repository/es"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
)

type ChatService interface {
	Query(ctx context.Context, query string, convID uint) (<-chan string, <-chan error)
	Insert(ctx context.Context, tableIdentify string) error
	GetConversation(ctx context.Context, tableIdentify, userID string) (*domain.Conversation, error)
	GetHistory(ctx context.Context, convID uint, lastID uint, limit int) ([]*domain.Message, error)
}

type ChatServiceImpl struct {
	runner          *adk.Runner
	log             logger.Logger
	faqDAO          dao.FAQDAO
	chatDAO         dao.ChatDAO
	esDAO           es.FAQESRepo
	embedder        embedding.Embedder
	cache           cache.ChatCache
	summaryRunnable compose.Runnable[[]*schema.Message, *schema.Message]
}

func NewChatService(
	agent *adk.Runner,
	log logger.Logger,
	faqDAO dao.FAQDAO,
	esDAO es.FAQESRepo,
	embedder embedding.Embedder,
	cache cache.ChatCache,
	chatDAO dao.ChatDAO,
	summaryRunnable compose.Runnable[[]*schema.Message, *schema.Message],
) ChatService {
	return &ChatServiceImpl{
		runner:          agent,
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
	//刷新一下当前的会话时间防止出现对话时过期的问题
	err = s.chatDAO.SaveConversation(ctx, conversation)
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
	records, err := s.faqDAO.GetFAQRecords(ctx, &tableIdentify)
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
func (s *ChatServiceImpl) Query(ctx context.Context, query string, convID uint) (<-chan string, <-chan error) {
	out := make(chan string)
	errCh := make(chan error, 1)
	const (
		SummaryThreshold = 20
		SummarySize      = 10
	)

	go func() {
		defer close(out)
		defer close(errCh)

		// 1. 获取会话信息
		conversation, err := s.chatDAO.FirstConversation(ctx, convID)
		if err != nil {
			errCh <- fmt.Errorf("获取会话 %d 失败: %w", convID, err)
			return
		}

		// 2. 读取历史对话
		msgs, err := s.cache.GetMSGList(ctx, conversation.ID)
		if err != nil {
			errCh <- fmt.Errorf("获取历史列表失败: %w", err)
			return
		}

		// 3. 获取摘要
		sum, err := s.cache.GetSummary(ctx, conversation.ID)
		if err != nil {
			errCh <- fmt.Errorf("获取摘要失败: %w", err)
			return
		}

		// 4. 上下文压缩逻辑 (已包含基本错误处理)
		if len(msgs) >= SummaryThreshold {
			var tmps = msgs[:SummarySize]
			if sum != nil {
				tmps = append([]*schema.Message{sum}, tmps...)
			}
			summary, err := s.summary(ctx, tmps)
			if err != nil {
				errCh <- fmt.Errorf("执行摘要总结失败: %w", err)
				return
			}
			if err = s.cache.SetSummary(ctx, conversation.ID, summary); err != nil {
				errCh <- fmt.Errorf("更新摘要缓存失败: %w", err)
				return
			}
			if err = s.cache.TrimMessageLeft(ctx, conversation.ID, SummarySize); err != nil {
				errCh <- fmt.Errorf("清理过期上下文失败: %w", err)
				return
			}
			msgs = msgs[SummarySize:]
			sum = summary // 更新当前使用的摘要
		}

		if sum != nil {
			msgs = append([]*schema.Message{sum}, msgs...)
		}

		userMSG := schema.UserMessage(query)
		msgs = append(msgs, userMSG)

		// 5. 调用 Agent 及其迭代器错误处理
		events := s.runner.Run(ctx, msgs)

		var sb strings.Builder
		for {
			event, ok := events.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				errCh <- fmt.Errorf("agent 运行异常: %w", event.Err)
				return
			}

			if event.Output == nil || event.Output.MessageOutput == nil {
				continue
			}
			mv := event.Output.MessageOutput
			if mv.Role != schema.Assistant {
				continue
			}

			if mv.IsStreaming {
				mv.MessageStream.SetAutomaticClose()
				for {
					frame, err := mv.MessageStream.Recv()
					if errors.Is(err, io.EOF) {
						break
					}
					// --- 补全点：流式读取层级错误 ---
					if err != nil {
						errCh <- fmt.Errorf("流式响应读取失败: %w", err)
						return
					}
					if frame != nil && frame.Content != "" {
						sb.WriteString(frame.Content)
						// 配合 select 防止向关闭的 chan 发送
						select {
						case out <- frame.Content:
						case <-ctx.Done():
							errCh <- ctx.Err()
							return
						}
					}
				}
				continue
			}

			if mv.Message != nil {
				sb.WriteString(mv.Message.Content)
				out <- mv.Message.Content
			}
		}

		unfiltered := sb.String()
		// 正则过滤掉 <think> 标签及其内容
		re := regexp.MustCompile(`(?s)<think>.*?</think>`)
		filtered := re.ReplaceAllString(unfiltered, "")

		// 写入 Redis (放过滤掉 think 标签后的内容来减少上下文长度)
		if err = s.cache.PushMessages(ctx, conversation.ID, cache.PositionTail, userMSG, schema.AssistantMessage(filtered, nil)); err != nil {
			errCh <- fmt.Errorf("同步消息到 Redis 失败: %w", err)
			return
		}

		// 写入 MySQL (放不滤 think 标签后的内容 保留原始信息)
		err = s.chatDAO.CreateMessages(ctx, &model.Message{
			ConversationID: conversation.ID,
			Role:           model.User,
			Content:        userMSG.Content,
			RawData:        model.EinoMessage{Message: userMSG},
		}, &model.Message{
			ConversationID: conversation.ID,
			Role:           model.Assistant,
			Content:        unfiltered,
			RawData:        model.EinoMessage{Message: schema.AssistantMessage(unfiltered, nil)},
		})
		if err != nil {
			errCh <- fmt.Errorf("持久化消息到数据库失败: %w", err)
			return
		}

		// 刷新会话时间
		if err = s.chatDAO.SaveConversation(ctx, conversation); err != nil {
			errCh <- fmt.Errorf("更新会话活跃时间失败: %w", err)
			return
		}
	}()

	return out, errCh
}
func (s *ChatServiceImpl) summary(ctx context.Context, msgs []*schema.Message) (*schema.Message, error) {

	summary, err := s.summaryRunnable.Invoke(ctx, msgs)
	if err != nil {
		return nil, err
	}

	return summary, nil
}
