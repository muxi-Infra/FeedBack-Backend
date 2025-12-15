package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	reqCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gin_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	reqDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gin_http_request_duration_seconds",
			Help:    "Histogram of request durations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

type PrometheusMiddleware struct {
	reg *prometheus.Registry
}

func NewPrometheusMiddleware(reg *prometheus.Registry) *PrometheusMiddleware {
	reg.MustRegister(reqCounter)
	reg.MustRegister(reqDuration)

	return &PrometheusMiddleware{
		reg: reg,
	}
}

func (pm *PrometheusMiddleware) GetRegistry() *prometheus.Registry {
	return pm.reg
}

func (pm *PrometheusMiddleware) MiddlewareFunc() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next()

		path := ctx.FullPath()
		if path == "" {
			// 对于未注册的路由，FullPath 可能为空
			// prometheus 使用 "unknown" 作为标签值防止爆炸
			path = "unknown"
		}
		if path == "/metrics" {
			// 跳过 /metrics 路由的监控
			return
		}

		method := ctx.Request.Method
		status := strconv.Itoa(ctx.Writer.Status())
		duration := time.Since(start).Seconds()
		reqCounter.WithLabelValues(method, path, status).Inc()
		reqDuration.WithLabelValues(method, path).Observe(duration)
	}
}
