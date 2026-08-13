package web

import (
	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"
)

func RegisterMessageRouter(r *gin.RouterGroup, mh controller.MessageHandler) {
	c := r.Group("/message")
	{
		c.POST("trigger", ginx.WrapReq(mh.TriggerNotification))
	}
}
