package web

import (
	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"
)

func RegisterAIRouter(r *gin.RouterGroup, ah controller.AIHandler) {
	c := r.Group("/ai")
	{
		c.POST("/query", ginx.WrapReq(ah.Query))
	}
}
