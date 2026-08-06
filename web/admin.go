package web

import (
	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	"github.com/muxi-Infra/FeedBack-Backend/middleware"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"
)

// RegisterAdminRouter 注册管理员登录和账号管理路由。
// 登录接口公开，其余接口统一经过 Admin JWT 和 Casbin 权限校验。
func RegisterAdminRouter(r *gin.RouterGroup, h controller.AdminHandler, auth *middleware.AdminAuthMiddleware, permission *middleware.AdminPermissionMiddleware) {
	admin := r.Group("/admin")
	admin.POST("/login", ginx.WrapReq(h.Login))

	protected := admin.Group("", auth.MiddlewareFunc())
	protected.POST("/users", permission.Require("admin", "create"), ginx.WrapReq(h.CreateAdmin))
	protected.GET("/users/:admin_id", permission.Require("admin", "read"), ginx.Wrap(h.GetAdmin))
	protected.PUT("/users/password", permission.Require("admin", "update"), ginx.WrapReq(h.ChangePassword))
	protected.PATCH("/users/:admin_id/status", permission.Require("admin", "update"), ginx.WrapReq(h.UpdateStatus))
}
