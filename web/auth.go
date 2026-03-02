package web

import (
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"

	"github.com/gin-gonic/gin"
)

func RegisterAuthRouter(r *gin.RouterGroup, ah controller.AuthHandler) {
	c := r.Group("/auth")
	{
		c.POST("/table-config/token", ginx.WrapReq(ah.GetTableToken))
		c.GET("/table-config/refresh", ginx.Wrap(ah.RefreshTableConfig))
		c.POST("/tenant/token", ginx.Wrap(ah.GetTenantToken))
	}
}
