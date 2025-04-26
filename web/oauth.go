package web

import (
	"feedback/api/request"
	"feedback/api/response"
	"feedback/pkg/ginx"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

type OauthHandler interface {
	IndexController(c *gin.Context)
	LoginController(c *gin.Context)
	OauthCallbackController(c *gin.Context) (response.Response, error)
	InitToken(c *gin.Context, r request.InitTokenReq) (response.Response, error)
	GetToken(c *gin.Context) (response.Response, error)
	//RefreshToken(*gin.Context, request.RefreshTokenReq) (response.Response, error)
	//GenerateToken(c *gin.Context, r request.GenerateTokenReq) (response.Response, error)
}

func RegisterOauthRouter(r *gin.Engine, oh OauthHandler) {
	// 使用 Cookie 存储 session
	store := cookie.NewStore([]byte("secret")) // 此处仅为示例，务必不要硬编码密钥
	r.Use(sessions.Sessions("mysession", store))

	r.GET("/", oh.IndexController)
	r.GET("/login", oh.LoginController)
	r.GET("/callback", ginx.Wrap(oh.OauthCallbackController))
	//r.POST("/refresh_token", ginx.WrapReq(oh.RefreshToken))
	//r.POST("/generate_token", ginx.WrapReq(oh.GenerateToken))
	r.POST("/init_token", ginx.WrapReq(oh.InitToken))
	r.POST("/get_token", ginx.Wrap(oh.GetToken))
}
