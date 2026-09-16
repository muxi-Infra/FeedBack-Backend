package middleware

import (
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConfigDiagnosticsRemainReachableAndLimiterAborts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr(), MaxRetries: -1})
	defer client.Close()
	limiter := NewLimitMiddleware(&config.LimiterConfig{Capacity: 1, FillInterval: 1, Quantum: 1}, client)
	r := gin.New()
	r.Use(limiter.Middleware())
	calls := 0
	r.GET("/business", func(c *gin.Context) { calls++; c.Status(http.StatusNoContent) })
	for _, path := range []string{"/api/v1/metrics", "/api/v1/health", "/api/v3/admin/integrations/projects/:project_id/config-status"} {
		r.GET(path, func(c *gin.Context) { c.AbortWithStatus(401) })
	}
	request := func(path string) int {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		return w.Code
	}
	require.Equal(t, 204, request("/business"))
	require.Equal(t, 429, request("/business"))
	require.Equal(t, 1, calls)
	m.Close()
	require.Equal(t, 500, request("/business"))
	require.Equal(t, 1, calls)
	for _, path := range []string{"/api/v1/metrics", "/api/v1/health", "/api/v3/admin/integrations/projects/fictional/config-status"} {
		require.Equal(t, 401, request(path))
	}
}
