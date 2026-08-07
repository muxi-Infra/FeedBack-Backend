package web

import (
	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	"github.com/muxi-Infra/FeedBack-Backend/middleware"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"
)

func RegisterMessageRouter(
	r *gin.RouterGroup,
	mh controller.MessageHandler,
	auth *middleware.AdminAuthMiddleware,
	permission *middleware.AdminPermissionMiddleware,
) {
	c := r.Group("/message", auth.MiddlewareFunc())
	{
		// 通知触发属于后台运营操作，复用 audit:create 权限
		c.POST("/trigger", permission.Require("audit", "create"), ginx.WrapReq(mh.TriggerNotification))
	}
}
