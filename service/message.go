package service

import (
	"context"
	"encoding/json"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/lark"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
)

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

	// TODO
	for i := 0; i < len(m.lc.ReceiveIDs); i++ {
		req := larkim.NewCreateMessageReqBuilder().
			ReceiveIdType(m.lc.ReceiveIDs[i].Type).
			Body(larkim.NewCreateMessageReqBodyBuilder().
				ReceiveId(m.lc.ReceiveIDs[i].ID).
				MsgType(`interactive`).
				Content(string(messageBytes)).
				Build()).
			Build()

		resp, err := m.c.SendNotice(context.Background(), req)
		// 处理错误
		if err != nil {
			m.log.Error("SendLarkMessage 调用失败",
				logger.String("error", err.Error()),
			)
			return errs.LarkRequestError(err)
		}

		// 服务端错误处理
		if !resp.Success() {
			m.log.Error("SendLarkMessage Lark 接口错误",
				logger.String("request_id", resp.RequestId()),
				logger.String("error", larkcore.Prettify(resp.CodeError)),
			)
			return errs.LarkResponseError(err)
		}
	}
	return nil
}
