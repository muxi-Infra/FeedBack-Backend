package middleware

import (
	"slices"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type CorsMiddleware struct {
	allowedOrigins []string
}

func NewCorsMiddleware(conf *config.MiddlewareConfig) *CorsMiddleware {
	return &CorsMiddleware{allowedOrigins: conf.AllowedOrigins}
}

func (cm *CorsMiddleware) MiddlewareFunc() gin.HandlerFunc {
	return cors.New(cors.Config{
		// 允许的请求头
		AllowHeaders: []string{"Content-Type", "Authorization", "Origin", "Accept"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		// 是否允许携带凭证（如 Cookies）
		AllowCredentials: true,
		// 解决跨域问题,这个地方允许所有请求跨域了,之后要改成允许前端的请求,比如localhost
		AllowOriginFunc: func(origin string) bool {
			// 未配置允许来源时保留开发环境的宽松行为；生产环境应明确配置来源。
			if len(cm.allowedOrigins) == 0 {
				return true
			}
			return slices.Contains(cm.allowedOrigins, origin)
		},

		// 预检请求的缓存时间
		MaxAge: 12 * time.Hour,
	})
}
