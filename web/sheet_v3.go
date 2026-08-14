package web

import (
	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"
)

func RegisterSheetHandlerV3(r *gin.RouterGroup, h controller.V3SheetHandler, auth gin.HandlerFunc) {
	c := r.Group("/sheet")
	c.POST("/feedback/records", auth, ginx.WrapV3Claims(h.CreateFeedback))
	c.GET("/feedback/records", auth, ginx.WrapV3Claims(h.GetFeedback))
	c.GET("/feedback/record", auth, ginx.WrapV3Claims(h.GetRecord))
	c.GET("/faq/records", auth, ginx.WrapV3Claims(h.GetFAQ))
	c.POST("/faq/records", auth, ginx.WrapV3Claims(h.UpdateFAQ))
	c.GET("/feedback/photos/url", auth, ginx.WrapV3Claims(h.GetPhotoURL))
}
