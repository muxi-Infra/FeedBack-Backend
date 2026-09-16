package middleware

import (
	"github.com/google/uuid"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/constvar"
	"strings"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type LoggerMiddleware struct {
	log logger.Logger
}

func (lm *LoggerMiddleware) ConfigAuditV3() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/api/v3/admin/integrations/") {
			c.Next()
			return
		}
		requestID := c.GetHeader("X-Request-ID")
		if _, err := uuid.Parse(requestID); err != nil {
			requestID = uuid.NewString()
		}
		c.Set("config_request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
		if c.Request.Method != "GET" && c.Writer.Status() >= 400 {
			actor, _ := c.Get(constvar.AdminIDContextKeyV3)
			id, _ := actor.(uint64)
			lm.log.Warn("config_change_rejected", logger.String("request_id", requestID), logger.Uint64("admin_id", id), logger.String("project_id", c.Param("project_id")), logger.String("method", c.Request.Method), logger.String("route", c.FullPath()), logger.Int("status", c.Writer.Status()))
		}
	}
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

		// 跳过对 /api/v1/metrics 的日志记录
		if path == "/api/v1/metrics" {
			return
		}

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
