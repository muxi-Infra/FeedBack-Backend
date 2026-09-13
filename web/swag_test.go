package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	"github.com/muxi-Infra/FeedBack-Backend/docs"
	"github.com/stretchr/testify/assert"
)

func TestOpenAPIResponse(t *testing.T) {
	r := gin.New()
	RegisterSwagHandler(
		r.Group("/api/v1"),
		controller.NewSwag(),
		gin.BasicAuth(gin.Accounts{"test": "secret"}),
	)

	tests := []struct {
		name       string
		username   string
		password   string
		wantStatus int
	}{
		{"valid auth", "test", "secret", http.StatusOK},
		{"missing auth", "", "", http.StatusUnauthorized},
		{"invalid auth", "test", "wrong", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodGet, "/api/v1/openapi", nil,
			)
			if tt.username != "" {
				req.SetBasicAuth(tt.username, tt.password)
			}

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				assert.Equal(t,
					"application/x-yaml; charset=utf-8",
					rec.Header().Get("Content-Type"),
				)
				assert.Equal(t, docs.OpenAPI, rec.Body.String())
			} else {
				assert.Empty(t, rec.Body.String())
				assert.NotEmpty(t, rec.Header().Get("WWW-Authenticate"))
			}
		})
	}
}
