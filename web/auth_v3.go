package web

import (
	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"
)

// RegisterAuthRouterV3 注册 V3 独立认证路由。
func RegisterAuthRouterV3(r *gin.RouterGroup, h controller.V3AuthHandler, auth gin.HandlerFunc) {
	r.POST("/integrations/token/exchange", ginx.WrapV3Req(h.Exchange))
	r.POST("/auth/tenant/token", auth, ginx.WrapV3ClaimsNoReq(h.GetTenantToken))
}
