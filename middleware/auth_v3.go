package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/constvar"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
)

const V3ClaimsContextKey = constvar.V3ClaimsContextKey

type V3AuthMiddleware struct {
	jwt *ijwt.V3JWT
}

func NewV3AuthMiddleware(jwt *ijwt.V3JWT) *V3AuthMiddleware {
	return &V3AuthMiddleware{
		jwt: jwt,
	}
}

func (m *V3AuthMiddleware) MiddlewareFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Error(errors.New("V3 Authorization 头格式错误"))
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{
				Code:    http.StatusUnauthorized,
				Message: "认证头格式错误",
			})
			return
		}

		claims, err := m.jwt.Parse(parts[1])
		if err != nil {
			c.Error(err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{
				Code:    http.StatusUnauthorized,
				Message: "无效或过期的 V3 身份令牌",
			})
			return
		}
		c.Set(V3ClaimsContextKey, claims)
		c.Next()
	}
}
