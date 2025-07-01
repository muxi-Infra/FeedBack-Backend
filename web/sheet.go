package web

import (
	"feedback/api/request"
	"feedback/api/response"
	"feedback/pkg/ginx"
	"feedback/pkg/ijwt"
	"github.com/gin-gonic/gin"
)

type SheetHandler interface {
	CreateApp(c *gin.Context, r request.CreateAppReq, uc ijwt.UserClaims) (response.Response, error)
	CopyApp(c *gin.Context, r request.CopyAppReq, uc ijwt.UserClaims) (response.Response, error)
	CreateAppTableRecord(c *gin.Context, r request.CreateAppTableRecordReq, uc ijwt.UserClaims) (response.Response, error)
	GetAppTableRecord(c *gin.Context, r request.GetAppTableRecordReq, uc ijwt.UserClaims) (response.Response, error)
}

func RegisterSheetHandler(r *gin.Engine, sh SheetHandler, authMiddleware gin.HandlerFunc) {
	c := r.Group("/sheet")
	{
		c.POST("/createapp", authMiddleware, ginx.WrapClaimsAndReq(sh.CreateApp))
		c.POST("/copyapp", authMiddleware, ginx.WrapClaimsAndReq(sh.CopyApp))
		c.POST("/createrecord", authMiddleware, ginx.WrapClaimsAndReq(sh.CreateAppTableRecord))
		c.POST("/getrecored", authMiddleware, ginx.WrapClaimsAndReq(sh.GetAppTableRecord))
	}
}
