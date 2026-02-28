package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/api/request"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/service"
)

type Message struct {
	s service.MessageService
}

func NewMessage(s service.MessageService) *Message {
	return &Message{
		s: s,
	}
}

// TriggerNotification 手动触发通知
//
//	@Summary		手动触发通知
//	@Description	将 `table_identify` 写入通知通道以触发下游消费
//	@Tags			Message
//	@ID				trigger-notification
//	@Accept			json
//	@Produce		json
//	@Param			request	body		request.TriggerNotificationReq	true	"触发通知请求参数"
//	@Success		200		{object}	response.Response				"触发成功"
//	@Failure		400		{object}	response.Response				"请求参数错误"
//	@Failure		500		{object}	response.Response				"服务器内部错误"
//	@Router			/api/v1/message/trigger [post]
func (m Message) TriggerNotification(c *gin.Context, req request.TriggerNotificationReq) (response.Response, error) {
	err := m.s.TriggerNotification(req.TableIdentify)
	if err != nil {
		return response.Response{}, err
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    nil,
	}, nil
}
