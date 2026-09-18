package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/internal/testclock"
	"github.com/muxi-Infra/FeedBack-Backend/middleware"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/configmetrics"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/constvar"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"github.com/muxi-Infra/FeedBack-Backend/service"
	"github.com/muxi-Infra/FeedBack-Backend/web"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestV3ConfigAdminResponseHeadersStatusAndAudit(t *testing.T) {
	f := newSecurityFixture(t)
	token := registerSecurityAdminRoutes(t, f, []string{"1", "integration", "read"}, []string{"1", "integration", "update"})
	payload := map[string]any{"project_name": "Fictional updated", "school": "Fictional School", "tables": []any{map[string]any{"table_identity": projectA + "-feedback", "table_name": "Fictional table", "table_type": "feedback", "table_token": "fictional-table-token", "table_id": "fictional-table", "view_id": "fictional-view", "scopes": []string{constvar.FeedbackScopeWrite}}}}
	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPut, "/api/v3/admin/integrations/projects/"+projectA, bytes.NewReader(encoded))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "11111111-2222-4333-8444-555555555555")
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	require.Equal(t, "2", w.Header().Get("X-Config-Version"))
	require.NotEmpty(t, w.Header().Get("X-Config-Change-ID"))
	require.Equal(t, "asynchronous", w.Header().Get("X-Config-Propagation"))
	require.Contains(t, w.Body.String(), `"data":null`)
	status := f.request(t, http.MethodGet, "/admin/integrations/projects/"+projectA+"/config-status", token, nil, 200, 0)
	var result service.ProjectConfigStatusV3
	require.NoError(t, json.Unmarshal(status, &result))
	require.Equal(t, uint64(2), *result.TargetVersion)
	require.Zero(t, result.AppliedVersion)
	require.NotEmpty(t, result.InstanceID)
	audits := f.request(t, http.MethodGet, "/admin/integrations/config-audits?request_id=11111111-2222-4333-8444-555555555555", token, nil, 200, 0)
	var rows []model.ConfigAuditV3
	require.NoError(t, json.Unmarshal(audits, &rows))
	require.Len(t, rows, 1)
	require.Equal(t, uint64(1), rows[0].AdminID)
	require.NotContains(t, string(audits), "fictional-table-token")
	_, err = f.auth.GetTableConfig(context.Background(), projectA, constvar.FeedbackTableType)
	require.NoError(t, err)
	status = f.request(t, http.MethodGet, "/admin/integrations/projects/"+projectA+"/config-status", token, nil, 200, 0)
	require.NoError(t, json.Unmarshal(status, &result))
	require.Equal(t, uint64(2), result.AppliedVersion)
}

func TestV3ConfigExistingJWTRevocationWindow(t *testing.T) {
	for _, kind := range []string{"scope", "delete"} {
		t.Run(kind, func(t *testing.T) {
			f := newSecurityFixture(t)
			clock := testclock.New()
			d := dao.NewConfigDAOV3(f.db)
			local := service.NewProjectConfigCacheV3(d, config.DefaultV3ConfigCacheConfig(), clock, configmetrics.New(prometheus.NewRegistry()), logger.NewZapLogger(zap.NewNop()))
			t.Cleanup(local.Close)
			auth := service.NewV3AuthService(dao.NewIntegrationDAOV3(f.db), nil, local, f.jwt, &config.IntegrationAuthConfig{TimestampSkew: 300})
			router := gin.New()
			web.RegisterSheetHandlerV3(router.Group("/api/v3"), controller.NewV3Sheet(service.NewSheetServiceForTest(f.db, f.lark), nil, auth), middleware.NewV3AuthMiddleware(f.jwt).MiddlewareFunc())
			f.router = router
			token := f.token(t, projectA, studentA)
			path := "/sheet/feedback/record?record_id=record-a1"
			f.request(t, http.MethodGet, path, token, nil, 200, 0)
			actor := domain.ConfigActorV3{AdminID: 1, RequestID: "11111111-2222-4333-8444-555555555555"}
			if kind == "delete" {
				_, err := f.admin.DeleteProject(context.Background(), projectA, actor)
				require.NoError(t, err)
			} else {
				input := domain.RegisterProjectInput{ProjectName: "Fictional", School: "Fictional", Tables: []domain.RegisterProjectTableInput{{TableIdentity: projectA + "-feedback", TableName: "Fictional", TableToken: "fictional-table-token", TableID: "fictional-table", ViewID: "fictional-view", TableType: "feedback", Scopes: []string{constvar.FeedbackScopeWrite}}}}
				_, err := f.admin.UpdateProject(context.Background(), projectA, input, actor)
				require.NoError(t, err)
			}
			// This instance deliberately receives no event and performs no scheduled reconciliation.
			clock.Advance(59 * time.Second)
			f.request(t, http.MethodGet, path, token, nil, 200, 0)
			clock.Advance(time.Second)
			if kind == "scope" {
				f.request(t, http.MethodGet, path, token, nil, 403, errs.V3ProjectTokenScopeForbiddenCode)
			} else {
				f.request(t, http.MethodGet, path, token, nil, 500, errs.V3TableConfigErrorCode)
			}
		})
	}
}
