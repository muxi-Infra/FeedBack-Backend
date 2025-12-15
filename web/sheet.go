package web

import (
	"github.com/muxi-Infra/FeedBack-Backend/api/request"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"

	"github.com/gin-gonic/gin"
)

type SheetHandler interface {
	CreateApp(c *gin.Context, r request.CreateAppReq, uc ijwt.UserClaims) (response.Response, error)
	CopyApp(c *gin.Context, r request.CopyAppReq, uc ijwt.UserClaims) (response.Response, error)
	CreateAppTableRecord(c *gin.Context, r request.CreateAppTableRecordReq, uc ijwt.UserClaims) (response.Response, error)
	GetAppTableRecord(c *gin.Context, r request.GetAppTableRecordReq, uc ijwt.UserClaims) (response.Response, error)
	GetPhotoUrl(c *gin.Context, r request.GetPhotoUrlReq, uc ijwt.UserClaims) (res response.Response, err error)
	GetNormalRecord(c *gin.Context, r request.GetAppTableRecordReq, uc ijwt.UserClaims) (response.Response, error)
}

func RegisterSheetHandler(r *gin.Engine, sh SheetHandler, authMiddleware gin.HandlerFunc) {
	c := r.Group("/sheet")
	{
		c.POST("/createapp", authMiddleware, ginx.WrapClaimsAndReq(sh.CreateApp))
		c.POST("/copyapp", authMiddleware, ginx.WrapClaimsAndReq(sh.CopyApp))
		c.POST("/createrecord", authMiddleware, ginx.WrapClaimsAndReq(sh.CreateAppTableRecord))
		c.POST("/getrecord", authMiddleware, ginx.WrapClaimsAndReq(sh.GetAppTableRecord))
		c.POST("/getphotourl", authMiddleware, ginx.WrapClaimsAndReq(sh.GetPhotoUrl))
		c.POST("/getnormal", authMiddleware, ginx.WrapClaimsAndReq(sh.GetNormalRecord))
	}
}
