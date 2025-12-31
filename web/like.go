package web

import (
	"github.com/muxi-Infra/FeedBack-Backend/api/request"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"

	"github.com/gin-gonic/gin"
)

type LikeHandler interface {
	AddLikeTask(c *gin.Context, r request.LikeReq, uc ijwt.UserClaims) (response.Response, error)
}

func RegisterLikeHandler(r *gin.RouterGroup, lh LikeHandler, authMiddleware gin.HandlerFunc) {
	c := r.Group("/like")
	{
		c.POST("/addtask", authMiddleware, ginx.WrapClaimsAndReq(lh.AddLikeTask))
	}
}
