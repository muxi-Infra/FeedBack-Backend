package web

import (
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	_ "github.com/muxi-Infra/FeedBack-Backend/docs" // 生成的swagger文档
	"github.com/muxi-Infra/FeedBack-Backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var ProviderSet = wire.NewSet(
	NewGinEngine,
	controller.NewOauth,
	wire.Bind(new(OauthHandler), new(*controller.Oauth)),
	controller.NewSheet,
	wire.Bind(new(SheetHandler), new(*controller.Sheet)),
	controller.NewLike,
	wire.Bind(new(LikeHandler), new(*controller.Like)),
)

func NewGinEngine(corsMiddleware *middleware.CorsMiddleware,
	authMiddleware *middleware.AuthMiddleware,
	basicAuthMiddleware *middleware.BasicAuthMiddleware,
	logMiddleware *middleware.LoggerMiddleware,
	prometheusMiddleware *middleware.PrometheusMiddleware,
	limitMiddleware *middleware.LimitMiddleware,
	sh SheetHandler, oh OauthHandler, lh LikeHandler) *gin.Engine {
	gin.ForceConsoleColor()
	r := gin.Default()
	// 跨域
	r.Use(corsMiddleware.MiddlewareFunc())
	r.Use(logMiddleware.MiddlewareFunc())
	r.Use(prometheusMiddleware.MiddlewareFunc())
	r.Use(limitMiddleware.Middleware())

	// Prometheus metrics endpoint with basic auth
	reg := prometheusMiddleware.GetRegistry()
	r.GET("/metrics", basicAuthMiddleware.MiddlewareFunc(), gin.WrapH(promhttp.HandlerFor(
		reg,
		promhttp.HandlerOpts{},
	)))
	r.GET("/swagger/*any", basicAuthMiddleware.MiddlewareFunc(), ginSwagger.WrapHandler(swaggerFiles.Handler))

	RegisterHealthCheckHandler(r)

	RegisterSheetHandler(r, sh, authMiddleware.MiddlewareFunc())
	RegisterOauthRouter(r, oh)
	RegisterLikeHandler(r, lh, authMiddleware.MiddlewareFunc())

	return r
}
