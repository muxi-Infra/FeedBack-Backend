package web

import (
	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"
)

func RegisterSheetHandler(r *gin.RouterGroup, sh controller.SheetV1Handler, authMiddleware gin.HandlerFunc) {
	c := r.Group("/sheet")
	{
		c.POST("/records", authMiddleware, ginx.WrapClaimsAndReq(sh.CreateTableRecord))
		c.GET("/records", authMiddleware, ginx.WrapClaimsAndReq(sh.GetTableRecordReqByKey))
		c.GET("/record", authMiddleware, ginx.WrapClaimsAndReq(sh.GetTableRecordReqByRecordID))
		c.GET("/records/faq", authMiddleware, ginx.WrapClaimsAndReq(sh.GetFAQResolutionRecord))
		c.POST("records/faq", authMiddleware, ginx.WrapClaimsAndReq(sh.UpdateFAQResolutionRecord))
		c.GET("/photos/url", authMiddleware, ginx.WrapClaimsAndReq(sh.GetPhotoUrl))
	}
}
