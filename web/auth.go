package web

import (
	"github.com/muxi-Infra/FeedBack-Backend/api/request"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

type OauthHandler interface {
	GetToken(c *gin.Context, req request.GenerateTokenReq) (response.Response, error)
	RefreshTableConfig(c *gin.Context) (response.Response, error)
}

func RegisterOauthRouter(r *gin.Engine, oh OauthHandler) {
	// 使用 Cookie 存储 session
	store := cookie.NewStore([]byte("secret")) // 此处仅为示例，务必不要硬编码密钥
	r.Use(sessions.Sessions("mysession", store))

	r.POST("/get_token", ginx.WrapReq(oh.GetToken))
	r.GET("/refresh_table_config", ginx.Wrap(oh.RefreshTableConfig))
}
