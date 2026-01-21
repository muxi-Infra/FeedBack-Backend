package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/lark"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
)

var (
	wg       sync.WaitGroup
	errCount atomic.Int32
)

//go:generate mockgen -destination=./mock/message_mock.go -package=mocks github.com/muxi-Infra/FeedBack-Backend/service MessageService
type MessageService interface {
	SendFeedbackNotification(tableName, content, url string) error
}

type MessageServiceImpl struct {
	c   lark.Client
	log logger.Logger
	lc  *config.LarkMessage
}

func NewMessageService(c lark.Client, log logger.Logger, lc *config.LarkMessage) MessageService {
	return &MessageServiceImpl{
		c:   c,
		log: log,
		lc:  lc,
	}
}

func (m MessageServiceImpl) SendFeedbackNotification(tableName, content, url string) error {
	if len(content) > 30 {
		content = content[:30] + "......"
	}

	message := domain.LarkMessage{
		Type: "template",
		Data: domain.LarkMessageData{
			TemplateId:          m.lc.TemplateID,
			TemplateVersionName: "",
			TemplateVariable: map[string]interface{}{
				"table_name":       tableName,
				"feedback_content": content,
				"shared_url":       map[string]string{"url": url},
			},
		},
	}
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return errs.SerializationError(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sem := make(chan struct{}, 5)
	for _, r := range m.lc.ReceiveIDs {
		r := r // 避免闭包问题
		wg.Add(1)

		go func() {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				m.log.Warn("SendLarkMessage canceled by context",
					logger.String("receive_id", r.ID),
				)
				return
			}

			req := larkim.NewCreateMessageReqBuilder().
				ReceiveIdType(r.Type).
				Body(larkim.NewCreateMessageReqBodyBuilder().
					ReceiveId(r.ID).
					MsgType("interactive").
					Content(string(messageBytes)).
					Build()).
				Build()

			resp, err := m.c.SendNotice(ctx, req)
			// 处理错误
			if err != nil {
				errCount.Add(1)
				m.log.Error("SendLarkMessage failed",
					logger.String("receive_id", r.ID),
					logger.String("error", err.Error()),
				)
				return
			}

			// 服务端错误处理
			if !resp.Success() {
				errCount.Add(1)
				m.log.Error("Lark API error",
					logger.String("receive_id", r.ID),
					logger.String("request_id", resp.RequestId()),
					logger.String("error", larkcore.Prettify(resp.CodeError)),
				)
				return
			}
		}()
	}

	wg.Wait()
	if errCount.Load() > 0 {
		return errs.LarkMessagePartialFailureError(fmt.Errorf("send message failed: %v", errCount.Load()))
	}

	return nil
}

func (m MessageServiceImpl) SendCCNUBoxNotification(content, url string) error {
	//TODO implement me
	panic("implement me")
}
