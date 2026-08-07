package web

import (
	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	"github.com/muxi-Infra/FeedBack-Backend/middleware"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"
)

// RegisterIntegrationAdminRouter 注册项目集成配置管理路由。
// 所有接口都要求 Admin JWT，并通过 Casbin 校验项目级操作权限。
func RegisterIntegrationAdminRouter(
	r *gin.RouterGroup,
	h controller.IntegrationAdminHandler,
	auth *middleware.AdminAuthMiddleware,
	permission *middleware.AdminPermissionMiddleware,
) {
	projects := r.Group("/integrations/projects", auth.MiddlewareFunc())
	projects.POST("", permission.Require("project", "create"), ginx.WrapReq(h.RegisterProject))
	projects.GET("", permission.Require("project", "read"), ginx.Wrap(h.ListProjects))
	projects.GET("/:project_id", permission.Require("project", "read"), ginx.Wrap(h.GetProject))
	projects.PUT("/:project_id", permission.Require("project", "update"), ginx.WrapReq(h.UpdateProject))
	projects.PUT("/:project_id/config", permission.Require("project", "update"), ginx.WrapReq(h.UpdateProjectConfig))
	projects.DELETE("/:project_id", permission.Require("project", "delete"), ginx.Wrap(h.DeleteProject))
}
