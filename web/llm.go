package web

import (
	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"
)

func RegisterAIRouter(r *gin.RouterGroup, ah controller.ChatHandler, authMiddleware gin.HandlerFunc) {
	c := r.Group("/llm")
	{
		c.POST("/query", authMiddleware, ginx.WrapSSEReq(ah.Query))
		c.POST("/insert", ginx.WrapReq(ah.Insert))
		c.GET("/history", authMiddleware, ginx.WrapReq(ah.GetHistory))
		c.GET("/conversation", authMiddleware, ginx.WrapClaimsAndReq(ah.GetConversation))
	}
}
