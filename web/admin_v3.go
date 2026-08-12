package web

import (
	"github.com/gin-gonic/gin"
	reqV3 "github.com/muxi-Infra/FeedBack-Backend/api/request/v3"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	"github.com/muxi-Infra/FeedBack-Backend/middleware"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"
)

func RegisterAdminAuthRouterV3(r *gin.RouterGroup, h controller.AdminAuthHandlerV3) {
	r.POST("/admin/login", ginx.WrapReq(func(c *gin.Context, req reqV3.AdminLoginReq) (response.Response, error) { return h.Login(c, req) }))
}

// RegisterAdminRouterV3 注册 V3 项目管理接口。
func RegisterAdminRouterV3(r *gin.RouterGroup, h controller.V3AdminHandler, syncHandler controller.V3SyncHandler, auth gin.HandlerFunc, permissions *middleware.AdminPermissionMiddlewareV3) {
	r.POST("/admin/integrations/projects", auth, permissions.Require("integration", "create"), ginx.WrapReq(h.RegisterProject))
	r.GET("/admin/integrations/projects", auth, permissions.Require("integration", "read"), ginx.WrapReq(h.ListProjects))
	r.GET("/admin/integrations/projects/:project_id", auth, permissions.Require("integration", "read"), ginx.Wrap(h.GetProject))
	r.PUT("/admin/integrations/projects/:project_id", auth, permissions.Require("integration", "update"), ginx.WrapReq(h.UpdateProject))
	r.DELETE("/admin/integrations/projects/:project_id", auth, permissions.Require("integration", "delete"), ginx.Wrap(h.DeleteProject))
	r.POST("/admin/integrations/projects/:project_id/keys/rotate", auth, permissions.Require("key", "update"), ginx.Wrap(h.RotateAPIKey))
	r.POST("/admin/sheet/feedback/sync", auth, permissions.Require("audit", "create"), ginx.WrapReq(syncHandler.SyncUnsynced))
	r.POST("/admin/sheet/feedback/sync/user", auth, permissions.Require("audit", "create"), ginx.WrapReq(syncHandler.ForceSyncUser))
	r.POST("/admin/sheet/feedback/sync/force", auth, permissions.Require("audit", "create"), ginx.WrapReq(syncHandler.ForceSyncAll))
	r.POST("/admin/sheet/faq/sync", auth, permissions.Require("audit", "create"), ginx.WrapReq(syncHandler.SyncFAQ))
}
