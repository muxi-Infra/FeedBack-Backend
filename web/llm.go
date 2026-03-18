package web

import (
	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"
)

func RegisterAIRouter(r *gin.RouterGroup, ah controller.ChatHandler) {
	c := r.Group("/llm")
	{
		c.POST("/query", ginx.WrapReq(ah.Query))
		c.POST("/insert", ginx.WrapReq(ah.Insert))
	}
}
