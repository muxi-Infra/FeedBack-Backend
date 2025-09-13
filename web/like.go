package web

import (
	"feedback/api/request"
	"feedback/api/response"
	"feedback/pkg/ginx"
	"feedback/pkg/ijwt"
	"github.com/gin-gonic/gin"
)

type LikeHandler interface {
	AddLikeTask(c *gin.Context, r request.LikeReq, uc ijwt.UserClaims) (response.Response, error)
}

func RegisterLikeHandler(r *gin.Engine, lh LikeHandler, authMiddleware gin.HandlerFunc) {
	c := r.Group("/like")
	{
		c.POST("/addtask", authMiddleware, ginx.WrapClaimsAndReq(lh.AddLikeTask))
	}
}
