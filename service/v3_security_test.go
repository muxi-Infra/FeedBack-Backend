package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang/mock/gomock"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	reqV3 "github.com/muxi-Infra/FeedBack-Backend/api/request/v3"
	respV1 "github.com/muxi-Infra/FeedBack-Backend/api/response/v1"
	respV2 "github.com/muxi-Infra/FeedBack-Backend/api/response/v2"
	respV3 "github.com/muxi-Infra/FeedBack-Backend/api/response/v3"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/ioc"
	"github.com/muxi-Infra/FeedBack-Backend/middleware"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/apikey"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/constvar"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/errorx"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
	larkmock "github.com/muxi-Infra/FeedBack-Backend/pkg/lark/mock"
	"github.com/muxi-Infra/FeedBack-Backend/repository/cache"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"github.com/muxi-Infra/FeedBack-Backend/service"
	"github.com/muxi-Infra/FeedBack-Backend/web"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const (
	projectA   = "fictional-project-a"
	projectB   = "fictional-project-b"
	studentA   = "fictional-student-a"
	studentB   = "fictional-student-b"
	testJWTKey = "fictional-jwt-secret-for-tests-only"
	testAPIKey = "fictional-api-key-for-tests-only"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// 本组读取和兑换测试不验证配置事件投递。
// 消费方法立即返回，避免生产构造函数启动的订阅协程持续运行。
type inertProjectEvents struct{}

func (inertProjectEvents) PublishProjectChanged(context.Context, string) error       { return nil }
func (inertProjectEvents) ConsumeProjectChanged(context.Context, func(string) error) {}

type securityFixture struct {
	db     *gorm.DB
	redis  *miniredis.Miniredis
	auth   service.V3AuthService
	admin  service.V3AdminService
	jwt    *ijwt.V3JWT
	router *gin.Engine
	lark   *larkmock.MockClient
}

func newSecurityFixture(t *testing.T) *securityFixture {
	t.Helper()
	// 使用单连接保证内存数据库仅供当前测试使用，同时实际执行 SQL，
	// 包括生产代码中的 WHERE 查询条件和密钥轮换事务。
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, db.AutoMigrate(&model.FeedbackProjectV3{}, &model.FeedbackProjectKeyV3{},
		&model.FeedbackProjectTableV3{}, &model.FeedbackProjectScopeV3{},
		&model.Sheet{}, &model.FAQRecord{}, &model.FAQResolution{}))

	r := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: r.Addr(), MaxRetries: -1})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	j := ijwt.NewV3JWT(config.JWTConfig{SecretKey: testJWTKey, Timeout: 3600}, nil)
	d := dao.NewIntegrationDAOV3(db)
	auth := service.NewV3AuthService(d, cache.NewIntegrationNonceStoreV3(client), inertProjectEvents{},
		service.NewProjectConfigCacheV3(), j, &config.IntegrationAuthConfig{TimestampSkew: 300})
	lark := larkmock.NewMockClient(gomock.NewController(t))
	sheets := service.NewSheetServiceForTest(db, lark)
	router := gin.New()
	v3 := router.Group("/api/v3")
	mw := middleware.NewV3AuthMiddleware(j).MiddlewareFunc()
	web.RegisterSheetHandlerV3(v3, controller.NewV3Sheet(sheets, nil, auth), mw)
	web.RegisterAuthRouterV3(v3, controller.NewV3Auth(auth, nil), mw)
	f := &securityFixture{db: db, redis: r, auth: auth, admin: service.NewV3AdminService(d, inertProjectEvents{}), jwt: j, router: router, lark: lark}
	for _, project := range []string{projectA, projectB} {
		require.NoError(t, db.Create(&model.FeedbackProjectV3{ProjectID: project, ProjectName: project, School: "Fictional School", Status: "active"}).Error)
		require.NoError(t, db.Create(&model.FeedbackProjectKeyV3{ProjectID: project, KeyID: project + "-key", APIKeyHash: apikey.Digest(testAPIKey), Status: "active"}).Error)
		for _, kind := range []string{constvar.FeedbackTableType, constvar.FAQTableType} {
			identity := project + "-" + kind
			require.NoError(t, db.Create(&model.FeedbackProjectTableV3{
				ProjectID: project, TableIdentity: identity, PhysicalName: "Fictional " + kind,
				TableToken: "fictional-table-token", TableID: identity, ViewID: "fictional-view", TableType: kind, Status: "active",
			}).Error)
			scope := constvar.FeedbackScopeReadSelf
			if kind == constvar.FAQTableType {
				scope = constvar.FeedbackScopeRead
			}
			require.NoError(t, db.Create(&model.FeedbackProjectScopeV3{ProjectID: project, TableIdentity: identity, Scope: scope}).Error)
		}
	}
	// 交错插入不同用户和项目的数据，确保缺少任一隔离条件都会导致首页或后续页泄露数据。
	// 在不同表中复用学生和记录 ID，验证表隔离条件确实生效。
	f.record(t, projectA, studentA, "record-a1", "photo-a1")
	f.record(t, projectA, studentB, "record-b1", "photo-b1")
	f.record(t, projectB, studentA, "record-project-b", "photo-project-b")
	f.record(t, projectA, studentA, "record-a2", "photo-a2")
	f.record(t, projectA, studentB, "record-b2", "photo-b2")
	f.record(t, projectB, studentA, "record-a1", "photo-other-table")
	for _, project := range []string{projectA, projectB} {
		require.NoError(t, db.Create(&model.FAQRecord{TableIdentify: ptr(project + "-faq"), RecordID: ptr("faq-shared-id"), Record: map[string]any{"content": project + " FAQ"}}).Error)
	}
	return f
}

func ptr[T any](v T) *T { return &v }

func (f *securityFixture) record(t *testing.T, project, student, id, photo string) {
	t.Helper()
	require.NoError(t, f.db.Create(&model.Sheet{TableIdentify: ptr(project + "-feedback"), UserID: ptr(student), RecordID: ptr(id),
		Record: map[string]any{"content": project + "/" + student + "/" + id, "截图": []string{photo}}}).Error)
}

func (f *securityFixture) token(t *testing.T, project, student string) string {
	t.Helper()
	token, _, err := f.jwt.Issue(project, student)
	require.NoError(t, err)
	return token
}

type securityResponse struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
}

func (f *securityFixture) request(t *testing.T, method, path, token string, body any, status, code int) json.RawMessage {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		require.NoError(t, err)
	}
	req := httptest.NewRequest(method, "/api/v3"+path, bytes.NewReader(payload))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	require.Equal(t, status, w.Code, "unexpected HTTP status for %s %s", method, path)
	var response securityResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, code, response.Code)
	if status != http.StatusOK {
		require.True(t, len(response.Data) == 0 || string(response.Data) == "null", "denial must not return data")
	}
	return response.Data
}

func TestV3FeedbackListIdentityIsolation(t *testing.T) {
	f := newSecurityFixture(t)
	for _, tc := range []struct {
		project, student string
		ids              []string
	}{
		{projectA, studentA, []string{"record-a2", "record-a1"}},
		{projectA, studentB, []string{"record-b2", "record-b1"}},
		{projectB, studentA, []string{"record-a1", "record-project-b"}},
	} {
		t.Run(tc.project+"/"+tc.student, func(t *testing.T) {
			token := f.token(t, tc.project, tc.student)
			// 客户端传入的身份和物理表字段不能覆盖令牌中的身份声明。
			query := url.Values{"student_id": {studentB}, "project_id": {projectB}, "table_identity": {projectB + "-feedback"}, "table_identify": {projectB + "-feedback"}, "limit_size": {"1"}}
			var ids []string
			for page := 0; page < 3; page++ {
				data := f.request(t, http.MethodGet, "/sheet/feedback/records?"+query.Encode(), token, nil, 200, 0)
				var records domain.TableRecords
				require.NoError(t, json.Unmarshal(data, &records))
				require.Len(t, records.Records, 1)
				r := records.Records[0]
				ids = append(ids, *r.RecordID)
				require.Equal(t, tc.project+"/"+tc.student+"/"+*r.RecordID, r.Record["content"])
				require.NotNil(t, records.HasMore)
				if !*records.HasMore {
					require.Nil(t, records.PageToken)
					break
				}
				require.NotNil(t, records.PageToken)
				query.Set("page_token", *records.PageToken)
			}
			require.Equal(t, tc.ids, ids)
		})
	}
}

func TestV3FeedbackRecordIdentityIsolation(t *testing.T) {
	f := newSecurityFixture(t)
	for _, tc := range []struct {
		name, project, student, record string
		status                         int
	}{
		{"own", projectA, studentA, "record-a1", 200},
		{"other_student", projectA, studentA, "record-b1", 404},
		{"other_student_can_read_own", projectA, studentB, "record-b1", 200},
		{"other_project_same_student", projectA, studentA, "record-project-b", 404},
		{"other_project_can_read_own", projectB, studentA, "record-project-b", 200},
		{"same_record_id_other_table", projectB, studentA, "record-a1", 200},
		{"unknown_record", projectA, studentA, "missing-record", 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code := 0
			if tc.status == 404 {
				code = errs.TableRecordNotFoundCode
			}
			data := f.request(t, http.MethodGet, "/sheet/feedback/record?record_id="+tc.record+"&student_id="+studentB+"&project_id="+projectB, f.token(t, tc.project, tc.student), nil, tc.status, code)
			if tc.status == 200 {
				var r respV1.GetTableRecordByRecordIdResp
				require.NoError(t, json.Unmarshal(data, &r))
				require.Equal(t, tc.project+"/"+tc.student+"/"+tc.record, r.Record["content"])
			}
		})
	}
}

func TestV3FeedbackPhotoOwnership(t *testing.T) {
	for _, tc := range []struct {
		name, project, student, record string
		photos                         []string
		status, code                   int
	}{
		{"own_photo", projectA, studentA, "record-a1", []string{"photo-a1"}, 200, 0},
		{"other_student_record", projectA, studentA, "record-b1", []string{"photo-b1"}, 404, errs.TableRecordNotFoundCode},
		{"other_student_can_read_own", projectA, studentB, "record-b1", []string{"photo-b1"}, 200, 0},
		{"other_project_record", projectA, studentA, "record-project-b", []string{"photo-project-b"}, 404, errs.TableRecordNotFoundCode},
		{"other_project_can_read_own", projectB, studentA, "record-project-b", []string{"photo-project-b"}, 200, 0},
		{"foreign_token_on_own_record", projectA, studentA, "record-a1", []string{"photo-b1"}, 403, errs.V3FeedbackPhotoForbiddenCode},
		{"same_owner_different_record", projectA, studentA, "record-a1", []string{"photo-a2"}, 403, errs.V3FeedbackPhotoForbiddenCode},
		{"same_record_id_other_project", projectA, studentA, "record-a1", []string{"photo-other-table"}, 403, errs.V3FeedbackPhotoForbiddenCode},
		{"mixed_owned_and_foreign", projectA, studentA, "record-a1", []string{"photo-a1", "photo-b1"}, 403, errs.V3FeedbackPhotoForbiddenCode},
		{"empty_token", projectA, studentA, "record-a1", []string{""}, 403, errs.V3FeedbackPhotoForbiddenCode},
		{"missing_tokens", projectA, studentA, "record-a1", nil, 400, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newSecurityFixture(t)
			// 被拒绝的请求不设置飞书调用期望；一旦发起外部调用，测试就会失败。
			if tc.status == 200 {
				f.lark.EXPECT().GetPhotoUrl(gomock.Any(), gomock.Eq(larkdrive.NewBatchGetTmpDownloadUrlMediaReqBuilder().FileTokens(tc.photos).Build())).Return(&larkdrive.BatchGetTmpDownloadUrlMediaResp{
					Data: &larkdrive.BatchGetTmpDownloadUrlMediaRespData{TmpDownloadUrls: []*larkdrive.TmpDownloadUrl{{FileToken: ptr(tc.photos[0]), TmpDownloadUrl: ptr("https://images.example.invalid/owned")}}},
				}, nil).Times(1)
			}
			query := url.Values{"record_id": {tc.record}, "file_tokens": tc.photos, "project_id": {projectB}, "student_id": {studentB}}
			data := f.request(t, http.MethodGet, "/sheet/feedback/photos/url?"+query.Encode(), f.token(t, tc.project, tc.student), nil, tc.status, tc.code)
			if tc.status == 200 {
				var r respV1.GetPhotoUrlResp
				require.NoError(t, json.Unmarshal(data, &r))
				require.Len(t, r.Files, 1)
				require.Equal(t, tc.photos[0], *r.Files[0].FileToken)
				require.Equal(t, "https://images.example.invalid/owned", *r.Files[0].TmpDownloadURL)
			}
		})
	}
}

func TestV3FAQScopeAndProjectIsolation(t *testing.T) {
	for _, scope := range []string{"", constvar.FeedbackScopeReadSelf, constvar.FeedbackScopeWrite, constvar.FeedbackScopeRead} {
		t.Run("scope="+scope, func(t *testing.T) {
			f := newSecurityFixture(t)
			require.NoError(t, f.db.Where("project_id = ? AND table_identity = ?", projectA, projectA+"-faq").Delete(&model.FeedbackProjectScopeV3{}).Error)
			if scope != "" {
				require.NoError(t, f.db.Create(&model.FeedbackProjectScopeV3{ProjectID: projectA, TableIdentity: projectA + "-faq", Scope: scope}).Error)
			}
			// 其他表或项目的读取权限不能授予当前 FAQ 的访问权限。
			require.NoError(t, f.db.Create(&model.FeedbackProjectScopeV3{ProjectID: projectA, TableIdentity: projectA + "-feedback", Scope: constvar.FeedbackScopeRead}).Error)
			for _, project := range []string{projectB, projectA, projectB, projectA} {
				status, code := 200, 0
				if project == projectA && scope != constvar.FeedbackScopeRead {
					status, code = 403, errs.V3ProjectTokenScopeForbiddenCode
				}
				data := f.request(t, http.MethodGet, "/sheet/faq/records?project_id="+projectB+"&table_identity="+projectB+"-faq", f.token(t, project, studentA), nil, status, code)
				if status == 200 {
					var r respV2.GetTableRecordByRecordIdResp
					require.NoError(t, json.Unmarshal(data, &r))
					require.Len(t, r.Records, 1)
					require.Equal(t, project+" FAQ", r.Records[0].Record["content"])
				}
			}
		})
	}
}

func signedExchange(project, student, keyID, key, nonce string, timestamp int64) reqV3.ExchangeFeedbackTokenReq {
	payload := strings.Join([]string{project, keyID, student, strconv.FormatInt(timestamp, 10), nonce}, "\n")
	return reqV3.ExchangeFeedbackTokenReq{ProjectID: project, StudentID: student, KeyID: keyID, Nonce: nonce, Timestamp: timestamp, Signature: apikey.Sign(key, payload)}
}

func exchangeInput(r reqV3.ExchangeFeedbackTokenReq) service.V3ExchangeInput {
	return service.V3ExchangeInput{ProjectID: r.ProjectID, KeyID: r.KeyID, StudentID: r.StudentID, Timestamp: r.Timestamp, Nonce: r.Nonce, Signature: r.Signature}
}

func (f *securityFixture) exchange(t *testing.T, req reqV3.ExchangeFeedbackTokenReq, status, code int) string {
	t.Helper()
	data := f.request(t, http.MethodPost, "/integrations/token/exchange", "", req, status, code)
	if status != 200 {
		return ""
	}
	var result respV3.ExchangeFeedbackTokenResp
	require.NoError(t, json.Unmarshal(data, &result))
	require.NotEmpty(t, result.AccessToken)
	require.Equal(t, "Bearer", result.TokenType)
	require.Positive(t, result.ExpiresIn)
	claims, err := f.jwt.Parse(result.AccessToken)
	require.NoError(t, err)
	require.Equal(t, req.ProjectID, claims.ProjectID)
	require.Equal(t, req.StudentID, claims.StudentID)
	return result.AccessToken
}

func TestV3TokenExchangeValidation(t *testing.T) {
	for _, tc := range []struct {
		name         string
		change       func(*reqV3.ExchangeFeedbackTokenReq)
		status, code int
	}{
		{"wrong_signing_key", func(r *reqV3.ExchangeFeedbackTokenReq) {
			*r = signedExchange(r.ProjectID, r.StudentID, r.KeyID, "fictional-wrong-key", r.Nonce, r.Timestamp)
		}, 401, errs.V3SignatureInvalidCode},
		{"malformed_signature", func(r *reqV3.ExchangeFeedbackTokenReq) { r.Signature = "not-hex" }, 401, errs.V3SignatureInvalidCode},
		{"tampered_student", func(r *reqV3.ExchangeFeedbackTokenReq) { r.StudentID = studentB }, 401, errs.V3SignatureInvalidCode},
		{"project_key_mismatch", func(r *reqV3.ExchangeFeedbackTokenReq) {
			*r = signedExchange(projectB, r.StudentID, r.KeyID, testAPIKey, r.Nonce, r.Timestamp)
		}, 401, errs.V3APIKeyInvalidCode},
		{"expired_timestamp", func(r *reqV3.ExchangeFeedbackTokenReq) {
			*r = signedExchange(r.ProjectID, r.StudentID, r.KeyID, testAPIKey, r.Nonce, r.Timestamp-3600)
		}, 401, errs.V3ExchangeExpiredCode},
		{"future_outside_window", func(r *reqV3.ExchangeFeedbackTokenReq) {
			*r = signedExchange(r.ProjectID, r.StudentID, r.KeyID, testAPIKey, r.Nonce, r.Timestamp+3600)
		}, 401, errs.V3ExchangeExpiredCode},
		{"missing_project", func(r *reqV3.ExchangeFeedbackTokenReq) { r.ProjectID = "" }, 400, 400},
		{"missing_student", func(r *reqV3.ExchangeFeedbackTokenReq) { r.StudentID = "" }, 400, 400},
		{"missing_nonce", func(r *reqV3.ExchangeFeedbackTokenReq) { r.Nonce = "" }, 400, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newSecurityFixture(t)
			valid := signedExchange(projectA, studentA, projectA+"-key", testAPIKey, "fictional-nonce", time.Now().Unix())
			invalid := valid
			tc.change(&invalid)
			f.exchange(t, invalid, tc.status, tc.code)
			require.Empty(t, f.redis.Keys(), "invalid requests must not consume nonces")
			token := f.exchange(t, valid, 200, 0)
			f.request(t, http.MethodGet, "/sheet/feedback/record?record_id=record-a1", token, nil, 200, 0)
			f.exchange(t, valid, 401, errs.V3ReplayRequestCode)
		})
	}
}

func TestV3TokenExchangeConcurrentReplay(t *testing.T) {
	f := newSecurityFixture(t)
	input := exchangeInput(signedExchange(projectA, studentA, projectA+"-key", testAPIKey, "concurrent-nonce", time.Now().Unix()))
	const count = 16
	start := make(chan struct{})
	type result struct {
		token string
		err   error
	}
	results := make(chan result, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			token, _, err := f.auth.Exchange(context.Background(), input)
			results <- result{token, err}
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	success := 0
	for result := range results {
		if result.err == nil {
			success++
			require.NotEmpty(t, result.token)
		} else {
			require.Empty(t, result.token)
			require.Equal(t, errs.V3ReplayRequestCode, errorx.ToCustomError(result.err).Code)
		}
	}
	require.Equal(t, 1, success, "SET NX must admit exactly one concurrent exchange")
	// nonce 按项目隔离；同一项目换用其他学生身份重新签名，仍应判定为重放。
	f.exchange(t, signedExchange(projectA, studentB, projectA+"-key", testAPIKey, input.Nonce, input.Timestamp), 401, errs.V3ReplayRequestCode)
	f.exchange(t, signedExchange(projectB, studentA, projectB+"-key", testAPIKey, input.Nonce, input.Timestamp), 200, 0)
	f.exchange(t, signedExchange(projectA, studentA, projectA+"-key", testAPIKey, "fresh-nonce", input.Timestamp), 200, 0)
}

func TestV3TokenExchangeFailsClosedOnNonceStoreError(t *testing.T) {
	f := newSecurityFixture(t)
	input := signedExchange(projectA, studentA, projectA+"-key", testAPIKey, "redis-error-nonce", time.Now().Unix())
	f.redis.SetError("ERR fictional nonce store failure")
	f.exchange(t, input, 500, errs.V3NonceErrorCode)
	f.redis.SetError("")
	f.exchange(t, input, 200, 0)
	f.exchange(t, input, 401, errs.V3ReplayRequestCode)
}

func TestV3NonceCoversEntireTimestampWindow(t *testing.T) {
	for _, offset := range []time.Duration{-300 * time.Second, 0, 240 * time.Second, 300 * time.Second} {
		t.Run(offset.String(), func(t *testing.T) {
			f := newSecurityFixture(t)
			now := time.Unix(1800000000, 0)
			input := exchangeInput(signedExchange(projectA, studentA, projectA+"-key", testAPIKey, "window-nonce", now.Add(offset).Unix()))
			_, _, err := service.ExchangeV3AtForTest(f.auth, context.Background(), input, now)
			require.NoError(t, err)
			// 时间戳有效窗口包含端点，因此 nonce 必须保留到最后一个有效秒结束，
			// 首次请求携带未来时间戳时也应满足这一要求。
			remaining := 300*time.Second + offset
			f.redis.FastForward(remaining)
			_, _, err = service.ExchangeV3AtForTest(f.auth, context.Background(), input, now.Add(remaining))
			require.Error(t, err, "replay at the last valid timestamp must be rejected")
			require.Equal(t, errs.V3ReplayRequestCode, errorx.ToCustomError(err).Code)
			f.redis.FastForward(time.Second)
			_, _, err = service.ExchangeV3AtForTest(f.auth, context.Background(), input, now.Add(remaining+time.Second))
			require.Error(t, err)
			require.Equal(t, errs.V3ExchangeExpiredCode, errorx.ToCustomError(err).Code)
			require.Empty(t, f.redis.Keys(), "expired nonce storage should be reclaimed")
		})
	}
}

func TestV3APIKeyRotation(t *testing.T) {
	f := newSecurityFixture(t)
	adminToken := registerSecurityAdminRoutes(t, f)
	now := time.Now().Unix()
	issued := f.exchange(t, signedExchange(projectA, studentA, projectA+"-key", testAPIKey, "before-rotation", now), 200, 0)
	data := f.request(t, http.MethodPost, "/admin/integrations/projects/"+projectA+"/keys/rotate", adminToken, nil, 200, 0)
	var rotated respV3.RotateAPIKeyResp
	require.NoError(t, json.Unmarshal(data, &rotated))
	keyID, newKey := rotated.KeyID, rotated.APIKey
	require.NotEmpty(t, keyID)
	require.NotEmpty(t, newKey)
	// 使用全新 nonce，确保兑换失败源于旧密钥已撤销，而非重复请求检查。
	f.exchange(t, signedExchange(projectA, studentA, projectA+"-key", testAPIKey, "old-key-fresh-nonce", now), 401, errs.V3APIKeyInvalidCode)
	f.exchange(t, signedExchange(projectA, studentA, keyID, testAPIKey, "wrong-key-fresh-nonce", now), 401, errs.V3SignatureInvalidCode)
	newToken := f.exchange(t, signedExchange(projectA, studentA, keyID, newKey, "new-key-fresh-nonce", now), 200, 0)
	f.exchange(t, signedExchange(projectB, studentA, projectB+"-key", testAPIKey, "other-project-fresh-nonce", now), 200, 0)
	// 密钥轮换撤销的是兑换凭据，已签发的反馈令牌仍应保持有效。
	for _, token := range []string{issued, newToken} {
		f.request(t, http.MethodGet, "/sheet/feedback/record?record_id=record-a1", token, nil, 200, 0)
	}
}

func registerSecurityAdminRoutes(t *testing.T, f *securityFixture) string {
	t.Helper()
	enforcer, err := ioc.InitCasbinV3(f.db)
	require.NoError(t, err)
	_, err = enforcer.AddPolicy("1", "key", "update")
	require.NoError(t, err)
	adminJWT := ijwt.NewAdminJWTV3(config.AdminJWTConfig{
		SecretKey: "fictional-admin-secret-for-tests-only", Issuer: "fictional-admin", Audience: "fictional-backend", Timeout: 3600,
	})
	web.RegisterAdminRouterV3(f.router.Group("/api/v3"), controller.NewV3Admin(f.admin),
		controller.NewV3Sync(service.NewSheetServiceForTest(f.db, f.lark), f.auth),
		middleware.NewAdminAuthMiddlewareV3(adminJWT).MiddlewareFunc(), middleware.NewAdminPermissionMiddlewareV3(enforcer))
	token, err := adminJWT.Issue(1)
	require.NoError(t, err)
	return token
}

func TestV3AdminProtectedRoutesRequireAuthentication(t *testing.T) {
	f := newSecurityFixture(t)
	registerSecurityAdminRoutes(t, f)
	for _, route := range f.router.Routes() {
		if !strings.HasPrefix(route.Path, "/api/v3/admin/") {
			continue
		}
		for name, token := range map[string]string{"missing_token": "", "invalid_token": "not-a-jwt", "feedback_token": f.token(t, projectA, studentA)} {
			t.Run(route.Method+route.Path+"/"+name, func(t *testing.T) {
				path := strings.ReplaceAll(strings.TrimPrefix(route.Path, "/api/v3"), ":project_id", projectA)
				f.request(t, route.Method, path, token, nil, 401, 401)
			})
		}
	}
}

func TestV3ProtectedRoutesRejectInvalidIdentity(t *testing.T) {
	f := newSecurityFixture(t)
	claims := jwt.MapClaims{"project_id": projectA, "student_id": studentA, "exp": time.Now().Add(time.Hour).Unix()}
	sign := func(claims jwt.MapClaims, key string, method jwt.SigningMethod) string {
		token, err := jwt.NewWithClaims(method, claims).SignedString([]byte(key))
		require.NoError(t, err)
		return token
	}
	tokens := map[string]string{"missing_header": "", "malformed_token": "not-a-jwt", "wrong_signature": sign(claims, "fictional-wrong-jwt-key", jwt.SigningMethodHS256), "wrong_algorithm": sign(claims, testJWTKey, jwt.SigningMethodHS384)}
	for _, missing := range []string{"project_id", "student_id"} {
		incomplete := jwt.MapClaims{}
		for k, v := range claims {
			if k != missing {
				incomplete[k] = v
			}
		}
		tokens["missing_"+missing] = sign(incomplete, testJWTKey, jwt.SigningMethodHS256)
		incomplete[missing] = ""
		tokens["empty_"+missing] = sign(incomplete, testJWTKey, jwt.SigningMethodHS256)
	}
	expired := jwt.MapClaims{"project_id": projectA, "student_id": studentA, "exp": time.Now().Add(-time.Hour).Unix()}
	tokens["expired"] = sign(expired, testJWTKey, jwt.SigningMethodHS256)
	// 枚举生产代码注册的路由，使后续新增的受保护路由也纳入鉴权失败测试。
	// 兑换接口的签名失败场景由独立用例覆盖。
	for _, route := range f.router.Routes() {
		if route.Path == "/api/v3/integrations/token/exchange" {
			continue
		}
		for name, token := range tokens {
			t.Run(route.Method+route.Path+"/"+name, func(t *testing.T) {
				f.request(t, route.Method, strings.TrimPrefix(route.Path, "/api/v3"), token, nil, 401, 401)
			})
		}
	}
	// 身份完整的令牌仍应能通过实际读取逻辑访问数据。
	f.request(t, http.MethodGet, "/sheet/feedback/record?record_id=record-a1", sign(claims, testJWTKey, jwt.SigningMethodHS256), nil, 200, 0)
}
