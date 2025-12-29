package middleware

import (
	"errors"
	"net/http"

	"strings"

	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/gin-gonic/gin"
)

var (
	tableCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "per_table_requests",
			Help: "Total number of requests per table",
		},
		[]string{"user"},
	)
)

type AuthMiddleware struct {
	jwtHandler *ijwt.JWT
}

func NewAuthMiddleware(jwtHandler *ijwt.JWT) *AuthMiddleware {
	return &AuthMiddleware{jwtHandler: jwtHandler}
}

func (am *AuthMiddleware) MiddlewareFunc() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 从请求中提取并解析 Token
		authCode := ctx.GetHeader("Authorization")
		if authCode == "" {
			ctx.Error(errors.New("认证头部缺失"))
			ctx.JSON(http.StatusUnauthorized, response.Response{
				Code:    http.StatusUnauthorized,
				Message: "认证头部缺失",
				Data:    nil,
			})
			return
		}
		// Bearer Token 处理
		segs := strings.Split(authCode, " ")
		if len(segs) != 2 || segs[0] != "Bearer" {
			ctx.Error(errors.New("请求头格式错误"))
			ctx.JSON(http.StatusUnauthorized, response.Response{
				Code:    http.StatusUnauthorized,
				Message: "认证头格式错误",
				Data:    nil,
			})
			return
		}

		uc, err := am.jwtHandler.ParseToken(segs[1])
		if err != nil {
			ctx.Error(err)
			ctx.JSON(http.StatusUnauthorized, response.Response{
				Code:    http.StatusUnauthorized,
				Message: "无效或过期的身份令牌",
				Data:    nil,
			})
			return
		}

		ginx.SetClaims(ctx, uc)

		// tableCount.WithLabelValues(uc.TableID).Inc()

		// 继续处理请求
		ctx.Next()
	}
}
