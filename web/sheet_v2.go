package web

import (
	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"
)

func RegisterSheetHandlerV2(r *gin.RouterGroup, sh controller.SheetV2Handler, authMiddleware gin.HandlerFunc) {
	c := r.Group("/sheet")
	{
		c.GET("/records", authMiddleware, ginx.WrapClaimsAndReq(sh.GetTableRecordReqByUser))
		c.POST("/sync", authMiddleware, ginx.WrapClaimsAndReq(sh.SyncUnsyncedTableRecords))
		c.POST("sync/user", authMiddleware, ginx.WrapClaimsAndReq(sh.ForceSyncUserTableRecords))
		c.POST("/sync/force", authMiddleware, ginx.WrapClaimsAndReq(sh.ForceSyncTableRecords))
		c.GET("/records/faq", authMiddleware, ginx.WrapClaimsAndReq(sh.GetFAQRecord))
		c.POST("/records/faq", authMiddleware, ginx.WrapClaimsAndReq(sh.UpdateFAQResolutionRecord))
		c.POST("/sync/faq", authMiddleware, ginx.WrapClaimsAndReq(sh.SyncFAQRecord))
	}
}
