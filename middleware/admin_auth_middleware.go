package middleware

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/casbin/casbin/v3"
	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
)

const (
	AdminIDContextKey     = "admin_id"
	AdminClaimsContextKey = "admin_claims"
)

// AdminAuthMiddleware 负责解析和校验管理后台 JWT。
// 它只确认“是谁”，不负责判断“能做什么”；
// 具体权限由 AdminPermissionMiddleware 处理。
type AdminAuthMiddleware struct {
	jwt *ijwt.AdminJWT
}

func NewAdminAuthMiddleware(adminJWT *ijwt.AdminJWT) *AdminAuthMiddleware {
	return &AdminAuthMiddleware{jwt: adminJWT}
}

func (m *AdminAuthMiddleware) MiddlewareFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := strings.TrimSpace(c.GetHeader("Authorization"))
		parts := strings.Fields(authorization)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			abortAdminUnauthorized(c, errors.New("管理员认证头格式错误"))
			return
		}

		claims, err := m.jwt.Parse(parts[1])
		if err != nil || claims.Subject == "" {
			if err == nil {
				err = errors.New("管理员 Token 缺少 subject")
			}
			abortAdminUnauthorized(c, err)
			return
		}

		adminID, err := strconv.ParseUint(claims.Subject, 10, 64)
		if err != nil || adminID == 0 {
			abortAdminUnauthorized(c, errors.New("管理员 Token subject 无效"))
			return
		}

		c.Set(AdminIDContextKey, adminID)
		c.Set(AdminClaimsContextKey, claims)
		c.Next()
	}
}

// AdminPermissionMiddleware 使用 Casbin 校验管理员权限。
// object 和 action 由路由显式指定，不直接使用 URL，避免 URL 变化导致权限含义变化。
type AdminPermissionMiddleware struct {
	enforcer *casbin.Enforcer
}

func NewAdminPermissionMiddleware(enforcer *casbin.Enforcer) *AdminPermissionMiddleware {
	return &AdminPermissionMiddleware{enforcer: enforcer}
}

func (m *AdminPermissionMiddleware) Require(object, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, exists := c.Get(AdminIDContextKey)
		if !exists {
			abortAdminUnauthorized(c, errors.New("管理员身份未找到"))
			return
		}
		adminID, ok := value.(uint64)
		if !ok || adminID == 0 {
			abortAdminUnauthorized(c, errors.New("管理员身份无效"))
			return
		}

		allowed, err := m.enforcer.Enforce(strconv.FormatUint(adminID, 10), object, action)
		if err != nil {
			abortAdminForbidden(c, err)
			return
		}
		if !allowed {
			abortAdminForbidden(c, errors.New("管理员没有执行该操作的权限"))
			return
		}
		c.Next()
	}
}

// GetAdminID 从请求上下文中读取已通过认证的管理员 ID。
func GetAdminID(c *gin.Context) (uint64, bool) {
	value, exists := c.Get(AdminIDContextKey)
	if !exists {
		return 0, false
	}
	id, ok := value.(uint64)
	return id, ok && id != 0
}

func abortAdminUnauthorized(c *gin.Context, err error) {
	c.Error(err)
	c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{
		Code: http.StatusUnauthorized, Message: "管理员身份认证失败", Data: nil,
	})
}

func abortAdminForbidden(c *gin.Context, err error) {
	c.Error(err)
	c.AbortWithStatusJSON(http.StatusForbidden, response.Response{
		Code: http.StatusForbidden, Message: "管理员权限不足", Data: nil,
	})
}
