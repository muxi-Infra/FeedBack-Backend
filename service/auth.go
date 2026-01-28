package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkbitable "github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/lark"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/retry"
)

const (
	TenantRefreshInterval = time.Hour + 35*time.Minute
	NoticeRefreshInterval = 4 * time.Hour
)

//go:generate mockgen -destination=./mock/auth_mock.go -package=mocks github.com/muxi-Infra/FeedBack-Backend/service AuthService
type AuthService interface {
	RefreshTableConfig() ([]domain.TableConfig, error)
	GetTableConfig(tableIdentity *string) (domain.TableConfig, error)
	GetTenantToken() string
}

type AuthServiceImpl struct {
	tenantToken  string // 上传资源（如：图片等）使用
	baseTableCfg *config.BaseTable
	clientCfg    *config.ClientConfig
	mutex        sync.RWMutex
	c            lark.Client
	log          logger.Logger
}

func NewAuthService(baseCfg *config.BaseTable, clientCfg *config.ClientConfig, c lark.Client, log logger.Logger) AuthService {
	s := &AuthServiceImpl{
		tenantToken:  "",
		baseTableCfg: baseCfg,
		clientCfg:    clientCfg,
		mutex:        sync.RWMutex{},
		c:            c,
		log:          log,
	}
	// 启动时同步刷新一次表配置，失败只记录日志
	if _, err := s.RefreshTableConfig(); err != nil {
		s.log.Error("启动阶段 RefreshTableConfig 初始调用失败",
			logger.String("error", err.Error()),
		)
	}
	s.startTenantTokenRefresher()
	s.startNotifiableTableScanner()

	return s
}

func (t *AuthServiceImpl) RefreshTableConfig() ([]domain.TableConfig, error) {
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

	return tables, nil
}

func (t *AuthServiceImpl) GetTableConfig(tableIdentity *string) (domain.TableConfig, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	// 防止传入 nil 指针引起 panic
	if tableIdentity == nil {
		return domain.TableConfig{}, errs.TableIdentifyNotFoundError(fmt.Errorf("table identity is nil"))
	}

	table, exists := tableCfg[*tableIdentity]
	if !exists {
		return domain.TableConfig{}, errs.TableIdentifyNotFoundError(fmt.Errorf("table identity %s not found", *tableIdentity))
	}
	return table, nil
}

func (t *AuthServiceImpl) GetTenantToken() string {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.tenantToken
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

	body, err := ioutil.ReadAll(resp.Body)
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

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				t.mutex.RLock()
				defer t.mutex.RUnlock()

				for tableID, table := range tableCfg {
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
