package middleware

import (
	"net/http"
	"strconv"

	"github.com/casbin/casbin/v3"
	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
)

type AdminPermissionMiddlewareV3 struct {
	enforcer *casbin.Enforcer
}

func NewAdminPermissionMiddlewareV3(enforcer *casbin.Enforcer) *AdminPermissionMiddlewareV3 {
	return &AdminPermissionMiddlewareV3{
		enforcer: enforcer,
	}
}

func (m *AdminPermissionMiddlewareV3) Require(object, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, ok := c.Get(AdminIDContextKeyV3)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{
				Code:    http.StatusUnauthorized,
				Message: "管理员身份认证失败",
			})
			return
		}

		id, ok := value.(uint64)
		if !ok || id == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{
				Code:    http.StatusUnauthorized,
				Message: "管理员身份认证失败",
			})
			return
		}

		allowed, err := m.enforcer.Enforce(strconv.FormatUint(id, 10), object, action)
		if err != nil || !allowed {
			if err != nil {
				c.Error(err)
			}
			c.AbortWithStatusJSON(http.StatusForbidden, response.Response{
				Code:    http.StatusForbidden,
				Message: "管理员权限不足",
			})
			return
		}
		c.Next()
	}
}
