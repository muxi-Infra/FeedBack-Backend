package web

import (
	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/api/request"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"
)

type MessageHandler interface {
	TriggerNotification(c *gin.Context, req request.TriggerNotificationReq) (response.Response, error)
}

func RegisterMessageRouter(r *gin.RouterGroup, mh MessageHandler) {
	c := r.Group("/message")
	{
		c.POST("trigger", ginx.WrapReq(mh.TriggerNotification))
	}
}
