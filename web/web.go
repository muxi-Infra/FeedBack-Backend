package web

import (
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	_ "github.com/muxi-Infra/FeedBack-Backend/docs" // 生成的 swagger 文档
	"github.com/muxi-Infra/FeedBack-Backend/middleware"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewGinEngine(corsMiddleware *middleware.CorsMiddleware,
	authMiddleware *middleware.AuthMiddleware,
	basicAuthMiddleware *middleware.BasicAuthMiddleware,
	logMiddleware *middleware.LoggerMiddleware,
	prometheusMiddleware *middleware.PrometheusMiddleware,
	limitMiddleware *middleware.LimitMiddleware,
	swag controller.SwagHandler,
	sh controller.SheetV1Handler,
	ah controller.AuthHandler,
	ch controller.ChatHandler,
	mh controller.MessageHandler,
	shV2 controller.SheetV2Handler,
) *gin.Engine {
	gin.ForceConsoleColor()
	r := gin.Default()

	// 全局中间件
	r.Use(corsMiddleware.MiddlewareFunc())       // 跨域中间件
	r.Use(prometheusMiddleware.MiddlewareFunc()) // Prometheus 监控中间件
	r.Use(logMiddleware.MiddlewareFunc())        // 日志中间件
	r.Use(limitMiddleware.Middleware())          // 限流中间件

	apiV1 := r.Group("/api/v1")

	// Swagger 文档使用 basic auth 保护
	RegisterSwagHandler(apiV1, swag, basicAuthMiddleware.MiddlewareFunc())
	// Prometheus metrics 使用 basic auth 保护
	RegisterPrometheusHandler(apiV1, prometheusMiddleware, basicAuthMiddleware)

	// 健康检查
	RegisterHealthCheckHandler(apiV1)

	// 业务路由
	RegisterAIRouter(apiV1, ch, authMiddleware.MiddlewareFunc())
	RegisterAuthRouter(apiV1, ah)
	RegisterSheetHandler(apiV1, sh, authMiddleware.MiddlewareFunc())
	RegisterMessageRouter(apiV1, mh)

	// V2 版本的路由
	apiV2 := r.Group("/api/v2")

	RegisterSheetHandlerV2(apiV2, shV2, authMiddleware.MiddlewareFunc())

	return r
}

// RegisterPrometheusHandler 注册 Prometheus 监控路由，使用 Basic Auth 保护
func RegisterPrometheusHandler(r *gin.RouterGroup, prometheusMiddleware *middleware.PrometheusMiddleware, basicAuthMiddleware *middleware.BasicAuthMiddleware) {
	reg := prometheusMiddleware.GetRegistry()
	r.GET("/metrics", basicAuthMiddleware.MiddlewareFunc(), gin.WrapH(promhttp.HandlerFor(
		reg,
		promhttp.HandlerOpts{},
	)))
}

// RegisterHealthCheckHandler 注册健康检查路由
func RegisterHealthCheckHandler(r *gin.RouterGroup) {
	r.GET("/health", ginx.Wrap(controller.HealthCheck))
}
