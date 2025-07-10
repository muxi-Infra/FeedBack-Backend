package web

import (
	"feedback/controller"
	_ "feedback/docs" // 生成的swagger文档
	"feedback/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var ProviderSet = wire.NewSet(
	NewGinEngine,
	controller.NewOauth,
	wire.Bind(new(OauthHandler), new(*controller.Oauth)),
	controller.NewSheet,
	wire.Bind(new(SheetHandler), new(*controller.Sheet)),
)

func NewGinEngine(corsMiddleware *middleware.CorsMiddleware, authMiddleware *middleware.AuthMiddleware,
	sh SheetHandler, oh OauthHandler) *gin.Engine {
	gin.ForceConsoleColor()
	r := gin.Default()
	// 跨域
	r.Use(corsMiddleware.MiddlewareFunc())

	RegisterSwaggerHandler(r)

	RegisterSheetHandler(r, sh, authMiddleware.MiddlewareFunc())
	RegisterOauthRouter(r, oh)

	return r
}

func RegisterSwaggerHandler(r *gin.Engine) {
	
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
