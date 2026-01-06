package web

import (
	"github.com/muxi-Infra/FeedBack-Backend/api/request"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"

	"github.com/gin-gonic/gin"
)

type AuthHandler interface {
	GetTableToken(c *gin.Context, req request.GenerateTableTokenReq) (response.Response, error)
	RefreshTableConfig(c *gin.Context) (response.Response, error)
	GetTenantToken(c *gin.Context) (response.Response, error)
}

func RegisterAuthRouter(r *gin.RouterGroup, ah AuthHandler) {
	c := r.Group("/auth")
	{
		c.POST("/table-config/token", ginx.WrapReq(ah.GetTableToken))
		c.GET("/table-config/refresh", ginx.Wrap(ah.RefreshTableConfig))
		c.POST("/tenant/token", ginx.Wrap(ah.GetTenantToken))
	}
}
