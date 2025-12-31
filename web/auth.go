package web

import (
	"github.com/muxi-Infra/FeedBack-Backend/api/request"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"

	"github.com/gin-gonic/gin"
)

type AuthHandler interface {
	GetToken(c *gin.Context, req request.GenerateTokenReq) (response.Response, error)
	RefreshTableConfig(c *gin.Context) (response.Response, error)
}

func RegisterAuthRouter(r *gin.RouterGroup, ah AuthHandler) {
	c := r.Group("/auth")
	{
		c.POST("/token", ginx.WrapReq(ah.GetToken))
		c.GET("/table-config/refresh", ginx.Wrap(ah.RefreshTableConfig))
	}
}
