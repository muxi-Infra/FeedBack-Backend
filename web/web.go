package web

import (
	"feedback/controller"
	oauth "feedback/controller/oauth/v3"
	"feedback/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
)

//	@title			木犀反馈系统 API
//	@version		1.0
//	@description	木犀反馈系统 API
//	@host			localhost:8080

var ProviderSet = wire.NewSet(
	NewGinEngine,
	oauth.NewOauth,
	wire.Bind(new(OauthHandler), new(*oauth.Oauth)),
	controller.NewSheet,
	wire.Bind(new(SheetHandler), new(*controller.Sheet)),
)

func NewGinEngine(corsMiddleware *middleware.CorsMiddleware, authMiddleware *middleware.AuthMiddleware,
	sh SheetHandler, oh OauthHandler) *gin.Engine {
	gin.ForceConsoleColor()
	r := gin.Default()
	// 跨域
	r.Use(corsMiddleware.MiddlewareFunc())

	RegisterSheetHandler(r, sh, authMiddleware.MiddlewareFunc())
	RegisterOauthRouter(r, oh)

	return r
}
