package middleware

import (
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type LoggerMiddleware struct {
	log logger.Logger
}

func NewLoggerMiddleware(log logger.Logger) *LoggerMiddleware {
	return &LoggerMiddleware{
		log: log,
	}
}

// MiddlewareFunc 处理响应逻辑
func (lm *LoggerMiddleware) MiddlewareFunc() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		path := ctx.Request.URL.Path
		ctx.Next() // 处理请求

		cost := time.Since(start)
		if len(ctx.Errors) > 0 {
			// 有错误记录错误日志
			lm.log.Error("HTTP request error",
				zap.String("method", ctx.Request.Method),
				zap.String("path", path),
				zap.Int("status", ctx.Writer.Status()),
				zap.String("client_ip", ctx.ClientIP()),
				zap.Duration("latency", cost),
				zap.String("errors", ctx.Errors.String()),
			)
		} else {
			// 正常请求记录访问日志
			lm.log.Info("HTTP request success",
				zap.String("method", ctx.Request.Method),
				zap.String("path", path),
				zap.Int("status", ctx.Writer.Status()),
				zap.String("client_ip", ctx.ClientIP()),
				zap.Duration("latency", cost),
			)
		}
	}
}
