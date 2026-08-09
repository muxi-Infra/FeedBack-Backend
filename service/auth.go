package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/apikey"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/retry"
	"github.com/muxi-Infra/FeedBack-Backend/repository/cache"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
)

const (
	TenantRefreshInterval         = time.Hour + 35*time.Minute
	NoticeRefreshInterval         = 10 * time.Minute
	SyncRefreshInterval           = 4 * time.Hour
	ConfigRefreshFallbackInterval = 5 * time.Minute
)

//go:generate mockgen -source=auth.go -destination=./mock/auth_mock.go -package=mocks
type AuthService interface {
	RefreshTableConfig() ([]domain.TableConfig, error)
	GetTableConfig(projectID, tableIdentity string) (domain.TableConfig, error)
	GetTenantToken() string
	ExchangeIntegrationToken(ctx context.Context, input domain.ExchangeIntegrationTokenInput) (string, int64, error)
}

type AuthServiceImpl struct {
	tenantToken    string // 上传资源（如：图片等）使用
	clientCfg      *config.ClientConfig
	mutex          sync.RWMutex
	log            logger.Logger
	jwtHandler     *ijwt.JWT
	integration    *config.IntegrationAuthConfig
	integrationDAO dao.IntegrationDAO
	configEvents   cache.ProjectConfigEventBus
	nonceStore     cache.IntegrationNonceStore
}

func NewAuthService(clientCfg *config.ClientConfig, log logger.Logger, jwtHandler *ijwt.JWT, integration *config.IntegrationAuthConfig, integrationDAO dao.IntegrationDAO, configEvents cache.ProjectConfigEventBus, nonceStore cache.IntegrationNonceStore) AuthService {
	s := &AuthServiceImpl{
		tenantToken:    "",
		clientCfg:      clientCfg,
		mutex:          sync.RWMutex{},
		log:            log,
		jwtHandler:     jwtHandler,
		integration:    integration,
		integrationDAO: integrationDAO,
		configEvents:   configEvents,
		nonceStore:     nonceStore,
	}
	// 启动时同步刷新一次表配置，失败只记录日志
	if _, err := s.RefreshTableConfig(); err != nil {
		s.log.Error("启动阶段 RefreshTableConfig 初始调用失败",
			logger.String("error", err.Error()),
		)
	}
	s.startTenantTokenRefresher()
	s.startNotifiableTableScanner()
	s.startSyncTableScanner()
	s.startProjectConfigRefreshers()

	return s
}

func (t *AuthServiceImpl) startProjectConfigRefreshers() {
	if t.configEvents != nil {
		go t.configEvents.ConsumeProjectChanged(context.Background(), func(projectID string) error {
			_, err := t.RefreshTableConfig()
			if err == nil {
				t.log.Info("项目配置缓存已刷新", logger.String("project_id", projectID))
			}
			return err
		})
	}

	go func() {
		ticker := time.NewTicker(ConfigRefreshFallbackInterval)
		defer ticker.Stop()
		for range ticker.C {
			if _, err := t.RefreshTableConfig(); err != nil {
				t.log.Error("项目配置兜底刷新失败", logger.String("error", err.Error()))
			}
		}
	}()
}

func (t *AuthServiceImpl) RefreshTableConfig() ([]domain.TableConfig, error) {
	return t.refreshTableConfigFromDB()
	/*
		// 创建请求对象
		req := larkbitable.NewSearchAppTableRecordReqBuilder().
			AppToken(t.baseTableCfg.TableToken).
			TableId(t.baseTableCfg.TableID).
			PageToken("").
			PageSize(50). // 分页大小，先给 50， 应该用不到这么多
			Body(larkbitable.NewSearchAppTableRecordReqBodyBuilder().
				ViewId(t.baseTableCfg.ViewID).
				FieldNames([]string{`table_identity`, `table_name`, `table_token`, `table_id`, `view_id`, `notice`}).
				Build()).
			Build()

		// 发起请求
		ctx := context.Background()
		resp, err := t.c.GetAppTableRecord(ctx, req)

		// 处理错误
		if err != nil {
			t.log.Error("RefreshTableConfig 调用失败",
				logger.String("error", err.Error()),
			)
			return nil, errs.LarkRequestError(err)
		}

		// 服务端错误处理
		if !resp.Success() {
			t.log.Error("RefreshTableConfig Lark 接口错误",
				logger.String("request_id", resp.RequestId()),
				logger.String("error", larkcore.Prettify(resp.CodeError)),
			)
			return nil, errs.LarkResponseError(err)
		}

		var tables []domain.TableConfig
		for _, item := range resp.Data.Items {
			var table domain.TableConfig
			if item.Fields != nil {
				fields := simplifyFields(item.Fields)

				if v, ok := fields["table_identity"].(string); ok {
					table.TableIdentity = &v
				}
				if v, ok := fields["table_name"].(string); ok {
					table.TableName = &v
				}
				if v, ok := fields["table_token"].(string); ok {
					table.TableToken = &v
				}
				if v, ok := fields["table_id"].(string); ok {
					table.TableID = &v
				}
				if v, ok := fields["view_id"].(string); ok {
					table.ViewID = &v
				}
				if v, ok := fields["notice"].(string); ok {
					table.Notice = v == "yes"
				}
			}

			if *table.TableIdentity != "" {
				tables = append(tables, table)
			}
		}

		// 同步更新配置（在临界区内替换 map，避免并发读写风险）
		newTables := make(map[string]domain.TableConfig)
		for _, table := range tables {
			if *table.TableIdentity != "" {
				newTables[*table.TableIdentity] = table
			}
		}

		t.mutex.Lock()
		tableCfg = newTables
		t.mutex.Unlock()

	*/
}

func (t *AuthServiceImpl) refreshTableConfigFromDB() ([]domain.TableConfig, error) {
	if t.integrationDAO == nil {
		return nil, errs.IntegrationProjectDatabaseError(errors.New("integration dao is not configured"))
	}

	ctx := context.Background()
	projects, err := t.integrationDAO.ListProjects(ctx)
	if err != nil {
		return nil, errs.IntegrationProjectDatabaseError(err)
	}

	var tables []domain.TableConfig
	newTables := make(map[string]domain.TableConfig)
	for _, project := range projects {
		if project.Status != ProjectStatusActive {
			continue
		}
		projectTables, err := t.integrationDAO.ListProjectTables(ctx, project.ProjectID)
		if err != nil {
			return nil, errs.IntegrationProjectDatabaseError(err)
		}
		for _, projectTable := range projectTables {
			if projectTable.Status != ProjectStatusActive || strings.TrimSpace(projectTable.TableIdentity) == "" {
				continue
			}
			identity := projectTable.TableIdentity
			projectID := project.ProjectID
			name := projectTable.PhysicalName
			token := projectTable.TableToken
			tableID := projectTable.TableID
			viewID := projectTable.ViewID
			scopeModels, err := t.integrationDAO.ListProjectScopes(ctx, project.ProjectID, projectTable.TableIdentity)
			if err != nil {
				return nil, errs.IntegrationProjectDatabaseError(err)
			}
			scopes := make([]string, 0, len(scopeModels))
			for _, scope := range scopeModels {
				scopes = append(scopes, scope.Scope)
			}
			table := domain.TableConfig{
				ProjectID:     projectID,
				TableIdentity: &identity,
				TableName:     &name,
				TableToken:    &token,
				TableID:       &tableID,
				ViewID:        &viewID,
				Notice:        projectTable.Notice,
				Scopes:        scopes,
			}
			tables = append(tables, table)
			newTables[tableConfigCacheKey(projectID, identity)] = table
		}
	}

	runtimeTableConfigCache.Replace(newTables)
	return tables, nil
}

func (t *AuthServiceImpl) GetTableConfig(projectID, tableIdentity string) (domain.TableConfig, error) {
	projectID = strings.TrimSpace(projectID)
	tableIdentity = strings.TrimSpace(tableIdentity)
	if projectID == "" || tableIdentity == "" {
		return domain.TableConfig{}, errs.TableIdentifyNotFoundError(fmt.Errorf("project id and table identity are required"))
	}

	table, exists := runtimeTableConfigCache.Get(tableConfigCacheKey(projectID, tableIdentity))
	if !exists {
		return domain.TableConfig{}, errs.TableIdentifyNotFoundError(fmt.Errorf("table identity %s not found for project %s", tableIdentity, projectID))
	}
	return table, nil
}

func (t *AuthServiceImpl) GetTenantToken() string {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.tenantToken
}

func (t *AuthServiceImpl) ExchangeIntegrationToken(ctx context.Context, input domain.ExchangeIntegrationTokenInput) (string, int64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return t.exchangeIntegrationTokenFromDB(ctx, input)
}

func (t *AuthServiceImpl) exchangeIntegrationTokenFromDB(ctx context.Context, input domain.ExchangeIntegrationTokenInput) (string, int64, error) {
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	input.KeyID = strings.TrimSpace(input.KeyID)
	input.StudentID = strings.TrimSpace(input.StudentID)
	input.TableIdentity = strings.TrimSpace(input.TableIdentity)
	input.Nonce = strings.TrimSpace(input.Nonce)
	input.Signature = strings.TrimSpace(input.Signature)
	if input.ProjectID == "" || input.KeyID == "" || input.StudentID == "" || input.TableIdentity == "" || input.Timestamp <= 0 || input.Nonce == "" || input.Signature == "" {
		return "", 0, errs.IntegrationTokenInvalidError(errors.New("project_id, key_id, student_id, table_identity, timestamp, nonce and signature are required"))
	}
	if t.integrationDAO == nil {
		return "", 0, errs.IntegrationProjectDatabaseError(errors.New("integration dao is not configured"))
	}
	project, err := t.integrationDAO.GetProject(ctx, input.ProjectID)
	if err != nil {
		return "", 0, errs.IntegrationProjectDatabaseError(err)
	}
	if project == nil || project.Status != ProjectStatusActive {
		return "", 0, errs.IntegrationTokenInvalidError(errors.New("integration project is not registered or disabled"))
	}
	key, err := t.integrationDAO.GetProjectKey(ctx, input.ProjectID, input.KeyID)
	if err != nil {
		return "", 0, errs.IntegrationProjectDatabaseError(err)
	}
	if key == nil || key.Status != ProjectStatusActive {
		return "", 0, errs.IntegrationTokenInvalidError(errors.New("integration project key is not registered or disabled"))
	}
	if key.ExpiresAt != nil && !key.ExpiresAt.After(time.Now()) {
		return "", 0, errs.IntegrationTokenInvalidError(errors.New("integration project key has expired"))
	}
	if time.Since(time.Unix(input.Timestamp, 0)) > 5*time.Minute || time.Until(time.Unix(input.Timestamp, 0)) > 5*time.Minute {
		return "", 0, errs.IntegrationTokenInvalidError(errors.New("integration request timestamp is expired"))
	}
	payload := integrationSignaturePayload(input)
	if err := apikey.Verify(key.APIKeyHash, payload, input.Signature); err != nil {
		return "", 0, errs.IntegrationTokenInvalidError(err)
	}
	if t.nonceStore == nil {
		return "", 0, errors.New("nonce store is required for replay protection")
	}
	accepted, err := t.nonceStore.MarkUsed(ctx, fmt.Sprintf("feedback:integration:nonce:%s:%s:%s", input.ProjectID, input.KeyID, input.Nonce), 5*time.Minute)
	if err != nil {
		return "", 0, errs.IntegrationProjectDatabaseError(err)
	}
	if !accepted {
		return "", 0, errs.IntegrationTokenInvalidError(errors.New("integration request nonce has already been used"))
	}
	table, err := t.integrationDAO.GetProjectTable(ctx, input.ProjectID, input.TableIdentity)
	if err != nil {
		return "", 0, errs.IntegrationProjectDatabaseError(err)
	}
	if table == nil || table.Status != ProjectStatusActive {
		return "", 0, errs.IntegrationTokenInvalidError(errors.New("table is not allowed for integration project"))
	}
	if t.jwtHandler == nil || t.integration == nil {
		return "", 0, errors.New("jwt handler or integration config is not configured")
	}
	ttlSeconds := t.integration.AccessTokenTTL
	if ttlSeconds <= 0 {
		return "", 0, errors.New("integration access token ttl must be positive")
	}
	accessToken, err := t.jwtHandler.SetIntegrationJWTToken(
		table.TableIdentity,
		project.ProjectID,
		input.StudentID,
		time.Duration(ttlSeconds)*time.Second,
	)
	if err != nil {
		return "", 0, errs.TokenGeneratedError(err)
	}
	return accessToken, int64(ttlSeconds), nil
}

func integrationSignaturePayload(input domain.ExchangeIntegrationTokenInput) string {
	return fmt.Sprintf("%s\n%s\n%s\n%s\n%d\n%s", input.ProjectID, input.KeyID, input.StudentID, input.TableIdentity, input.Timestamp, input.Nonce)
}

func (t *AuthServiceImpl) refreshTenantToken() (*string, error) {
	// 局部定义请求/响应结构体
	type TokenRequest struct {
		AppID     string `json:"app_id"`
		AppSecret string `json:"app_secret"`
	}
	type TokenResponse struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}

	// 构造请求
	url := "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"
	requestBody := TokenRequest{
		AppID:     t.clientCfg.AppID,
		AppSecret: t.clientCfg.AppSecret,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %v", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}
	var tokenResp TokenResponse
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		return nil, errs.DeserializationError(err)
	}

	// 检查响应码
	if tokenResp.Code != 0 {
		return nil, fmt.Errorf("获取token失败: code=%d, msg=%s", tokenResp.Code, tokenResp.Msg)
	}

	// 同步更新配置
	t.mutex.Lock()
	t.tenantToken = tokenResp.TenantAccessToken
	t.mutex.Unlock()

	return &tokenResp.TenantAccessToken, nil
}

func (t *AuthServiceImpl) startTenantTokenRefresher() {
	// 启动立即刷新一次
	if _, err := retry.Retry(t.refreshTenantToken); err != nil {
		t.log.Error(
			"启动阶段 RefreshTenantToken 初始调用失败",
			logger.String("error", err.Error()),
		)
	}

	// 后台定时刷新
	ticker := time.NewTicker(TenantRefreshInterval)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if _, err := retry.Retry(t.refreshTenantToken); err != nil {
					t.log.Error(
						"定时刷新租户 Token 失败",
						logger.String("error", err.Error()),
					)
				}
			}
		}
	}()
}

func (t *AuthServiceImpl) startNotifiableTableScanner() {
	ticker := time.NewTicker(NoticeRefreshInterval)
	// 生产者，定时扫描需要发送通知的表，并将其放入 noticeCh 中
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				for tableID, table := range runtimeTableConfigCache.Snapshot() {
					if !table.Notice {
						continue
					}

					select {
					case noticeCh <- table:
						t.log.Info("notifiable table queued",
							logger.String("table_id", tableID),
						)
					default:
						// ⚠️ channel 满了，直接丢，避免阻塞
						t.log.Warn("notice channel full, skip table",
							logger.String("table_id", tableID),
						)
					}
				}
			}
		}
	}()
}

func (t *AuthServiceImpl) startSyncTableScanner() {
	ticker := time.NewTicker(SyncRefreshInterval)
	// 生产者，定时扫描需要同步的表，并将其放入 syncTableCh 中
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				for tableID, table := range runtimeTableConfigCache.Snapshot() {
					select {
					case syncTableCh <- table:
						t.log.Info("sync table queued",
							logger.String("table_id", tableID))
					default:
						// ⚠️ channel 满了，直接丢，避免阻塞
						t.log.Warn("sync table channel full, skip table",
							logger.String("table_id", tableID))
					}
				}
			}
		}
	}()
}
