package middleware

import (
	"errors"

	"strings"

	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"

	"github.com/gin-gonic/gin"
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
			return
		}
		// Bearer Token 处理
		segs := strings.Split(authCode, " ")
		if len(segs) != 2 || segs[0] != "Bearer" {
			ctx.Error(errors.New("请求头格式错误"))
			return
		}

		uc, err := am.jwtHandler.ParseToken(segs[1])
		if err != nil {
			ctx.Error(err)
			return
		}

		ginx.SetClaims(ctx, uc)

		// 继续处理请求
		ctx.Next()
	}
}
