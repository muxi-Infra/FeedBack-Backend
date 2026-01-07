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
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/feishu"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
)

const RefreshInterval = time.Hour + 35*time.Minute

type AuthService interface {
	RefreshTableConfig() ([]Table, error)
	GetTableConfig(tableIdentity *string) (Table, error)
	GetTenantToken() string
}

type AuthServiceImpl struct {
	tableCfg     map[string]Table
	tenantToken  string // 上传资源（如：图片等）使用
	baseTableCfg *config.BaseTable
	clientCfg    *config.ClientConfig
	mutex        sync.RWMutex
	c            feishu.Client
	log          logger.Logger
}

type Table struct {
	Identity   string
	Name       string
	TableToken string
	TableID    string
	ViewID     string
}

func NewTableService(baseCfg *config.BaseTable, clientCfg *config.ClientConfig, c feishu.Client, log logger.Logger) AuthService {
	svc := &AuthServiceImpl{
		tableCfg:     make(map[string]Table),
		tenantToken:  "",
		baseTableCfg: baseCfg,
		clientCfg:    clientCfg,
		mutex:        sync.RWMutex{},
		c:            c,
		log:          log,
	}
	// 启动时同步刷新一次表配置，失败只记录日志
	if _, err := svc.RefreshTableConfig(); err != nil {
		svc.log.Error("启动阶段 RefreshTableConfig 初始调用失败",
			logger.String("error", err.Error()),
		)
	}
	svc.startTenantTokenRefresher()

	return svc
}

func (t *AuthServiceImpl) RefreshTableConfig() ([]Table, error) {
	// 创建请求对象
	req := larkbitable.NewSearchAppTableRecordReqBuilder().
		AppToken(t.baseTableCfg.TableToken).
		TableId(t.baseTableCfg.TableID).
		PageToken("").
		PageSize(50). // 分页大小，先给 50， 应该用不到这么多
		Body(larkbitable.NewSearchAppTableRecordReqBodyBuilder().
			ViewId(t.baseTableCfg.ViewID).
			FieldNames([]string{`table_identity`, `table_name`, `table_token`, `table_id`, `view_id`}).
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
		return nil, errs.FeishuRequestError(err)
	}

	// 服务端错误处理
	if !resp.Success() {
		t.log.Error("RefreshTableConfig Lark 接口错误",
			logger.String("request_id", resp.RequestId()),
			logger.String("error", larkcore.Prettify(resp.CodeError)),
		)
		return nil, errs.FeishuResponseError(err)
	}

	var tables []Table
	for _, item := range resp.Data.Items {
		var table Table
		if item.Fields != nil {
			extract := func(key string) string {
				val, ok := item.Fields[key]
				if !ok || val == nil {
					return ""
				}

				switch v := val.(type) {
				case string:
					return v
				case []interface{}:
					if len(v) == 0 {
						return ""
					}
					elem := v[0]
					if s, ok := elem.(map[string]interface{}); ok {
						if txt, ok := s["text"]; ok {
							if ss, ok := txt.(string); ok {
								return ss
							}
						}
						return ""
					}
					return ""
				case map[string]interface{}:
					// 尝试获取 text 字段
					if txt, ok := v["text"]; ok {
						if s, ok := txt.(string); ok {
							return s
						}
					}
					return ""
				default:
					return fmt.Sprintf("%v", v)
				}
			}

			table.Identity = extract("table_identity")
			table.Name = extract("table_name")
			table.TableToken = extract("table_token")
			table.TableID = extract("table_id")
			table.ViewID = extract("view_id")
		}

		if table.Identity != "" {
			tables = append(tables, table)
		}
	}

	// 同步更新配置（在临界区内替换 map，避免并发读写风险）
	newTables := make(map[string]Table)
	for _, table := range tables {
		if table.Identity != "" {
			newTables[table.Identity] = table
		}
	}

	t.mutex.Lock()
	t.tableCfg = newTables
	t.mutex.Unlock()

	return tables, nil
}

func (t *AuthServiceImpl) GetTableConfig(tableIdentity *string) (Table, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	table, exists := t.tableCfg[*tableIdentity]
	if !exists {
		return Table{},
			errs.TableIdentifyNotFoundError(fmt.Errorf("table identity %s not found", tableIdentity))
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
	// TODO 重试机制
	if _, err := t.refreshTenantToken(); err != nil {
		t.log.Error(
			"启动阶段 RefreshTenantToken 初始调用失败",
			logger.String("error", err.Error()),
		)
	}

	// 后台定时刷新
	ticker := time.NewTicker(RefreshInterval)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// TODO 重试机制
				if _, err := t.refreshTenantToken(); err != nil {
					t.log.Error(
						"定时刷新租户 Token 失败",
						logger.String("error", err.Error()),
					)
				}
			}
		}
	}()
}
