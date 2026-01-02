package web

import (
	"github.com/muxi-Infra/FeedBack-Backend/api/request"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"

	"github.com/gin-gonic/gin"
)

type SheetHandler interface {
	CreatTableRecord(c *gin.Context, r request.CreatTableRecordReg, uc ijwt.UserClaims) (response.Response, error)
	GetTableRecordReqByKey(c *gin.Context, r request.GetTableRecordReq, uc ijwt.UserClaims) (response.Response, error)
	GetFAQResolutionRecord(c *gin.Context, r request.GetFAQProblemTableRecordReg, uc ijwt.UserClaims) (response.Response, error)
	UpdateFAQResolutionRecord(c *gin.Context, r request.FAQResolutionUpdateReq, uc ijwt.UserClaims) (response.Response, error)
	GetPhotoUrl(c *gin.Context, r request.GetPhotoUrlReq, uc ijwt.UserClaims) (res response.Response, err error)
}

func RegisterSheetHandler(r *gin.RouterGroup, sh SheetHandler, authMiddleware gin.HandlerFunc) {
	c := r.Group("/sheet")
	{
		c.POST("/records", authMiddleware, ginx.WrapClaimsAndReq(sh.CreatTableRecord))
		c.GET("/records", authMiddleware, ginx.WrapClaimsAndReq(sh.GetTableRecordReqByKey))
		c.GET("/records/faq", authMiddleware, ginx.WrapClaimsAndReq(sh.GetFAQResolutionRecord))
		c.POST("records/faq", authMiddleware, ginx.WrapClaimsAndReq(sh.UpdateFAQResolutionRecord))
		c.GET("/photos/url", authMiddleware, ginx.WrapClaimsAndReq(sh.GetPhotoUrl))
	}
}
