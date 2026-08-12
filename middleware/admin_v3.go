package middleware

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/constvar"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
)

const AdminIDContextKeyV3 = constvar.AdminIDContextKeyV3

type AdminAuthMiddlewareV3 struct {
	jwt *ijwt.AdminJWTV3
}

func NewAdminAuthMiddlewareV3(jwt *ijwt.AdminJWTV3) *AdminAuthMiddlewareV3 {
	return &AdminAuthMiddlewareV3{
		jwt: jwt,
	}
}

func (m *AdminAuthMiddlewareV3) MiddlewareFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{
				Code:    http.StatusUnauthorized,
				Message: "管理员身份认证失败",
			})
			return
		}

		claims, err := m.jwt.Parse(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{
				Code:    http.StatusUnauthorized,
				Message: "管理员身份认证失败",
			})
			return
		}

		id, err := strconv.ParseUint(claims.Subject, 10, 64)
		if err != nil || id == 0 {
			c.Error(errors.New("admin jwt subject is invalid"))
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{
				Code:    http.StatusUnauthorized,
				Message: "管理员身份认证失败",
			})
			return
		}
		c.Set(AdminIDContextKeyV3, id)
		c.Next()
	}
}
