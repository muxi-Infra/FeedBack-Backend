package web

import (
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	_ "github.com/muxi-Infra/FeedBack-Backend/docs" // 生成的 swagger 文档
	"github.com/muxi-Infra/FeedBack-Backend/middleware"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var ProviderSet = wire.NewSet(
	NewGinEngine,
	controller.NewSwag,
	wire.Bind(new(SwagHandler), new(*controller.Swag)),
	controller.NewAuth,
	wire.Bind(new(AuthHandler), new(*controller.Auth)),
	controller.NewSheet,
	wire.Bind(new(SheetHandler), new(*controller.Sheet)),
)

func NewGinEngine(corsMiddleware *middleware.CorsMiddleware,
	authMiddleware *middleware.AuthMiddleware,
	basicAuthMiddleware *middleware.BasicAuthMiddleware,
	logMiddleware *middleware.LoggerMiddleware,
	prometheusMiddleware *middleware.PrometheusMiddleware,
	limitMiddleware *middleware.LimitMiddleware,
	swag SwagHandler,
	sh SheetHandler, ah AuthHandler) *gin.Engine {
	gin.ForceConsoleColor()
	r := gin.Default()

	// 全局中间件
	r.Use(corsMiddleware.MiddlewareFunc())       // 跨域中间件
	r.Use(logMiddleware.MiddlewareFunc())        // 日志中间件
	r.Use(prometheusMiddleware.MiddlewareFunc()) // Prometheus 监控中间件
	r.Use(limitMiddleware.Middleware())          // 限流中间件

	api := r.Group("/api/v1")

	// Swagger 文档使用 basic auth 保护
	RegisterSwagHandler(api, swag, basicAuthMiddleware.MiddlewareFunc())
	// Prometheus metrics 使用 basic auth 保护
	RegisterPrometheusHandler(api, prometheusMiddleware, basicAuthMiddleware)

	// 健康检查
	RegisterHealthCheckHandler(api)

	// 业务路由
	RegisterAuthRouter(api, ah)
	RegisterSheetHandler(api, sh, authMiddleware.MiddlewareFunc())

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
