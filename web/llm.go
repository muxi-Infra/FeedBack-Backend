package web

import (
	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"
)

func RegisterAIRouter(r *gin.RouterGroup, ah controller.ChatHandler, authMiddleware gin.HandlerFunc) {
	c := r.Group("/llm")
	{
		c.POST("/chat", authMiddleware, ginx.WrapSSE(ah.Chat))
		c.POST("/query", authMiddleware, ginx.WrapClaimsAndReq(ah.Query))
		c.POST("/insert", ginx.WrapReq(ah.Insert))
		c.GET("/history", authMiddleware, ginx.WrapClaimsAndReq(ah.GetHistory))
	}
}
