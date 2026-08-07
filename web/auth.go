package web

import (
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"

	"github.com/gin-gonic/gin"
)

func RegisterAuthRouter(r *gin.RouterGroup, ah controller.AuthHandler, authMiddleware gin.HandlerFunc) {
	c := r.Group("/auth")
	{
		// 租户 Token 仅允许已通过反馈 JWT 认证的用户获取，用于上传图片等应用级操作。
		c.POST("/tenant/token", authMiddleware, ginx.WrapClaims(ah.GetTenantToken))
	}
	r.POST("/integrations/token/exchange", ginx.WrapReq(ah.ExchangeIntegrationToken))
}
