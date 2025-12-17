package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/feishu"

	"golang.org/x/oauth2"
)

type AuthService interface {
	StartAutoRefresh(accessToken string, refreshToken string, t time.Duration)
	AutoRefreshToken() error

	StartRefresh(accessToken string, refreshToken string)
	GetAccessToken() string
	GetTenantAccessToken() string
}

type AuthServiceImpl struct {
	oauthConfig *oauth2.Config

	mutex        sync.RWMutex
	accessToken  string
	refreshToken string // 不对外展示
	first        bool   // 是否为第一次设定

	tenantAccessToken string // 自建应用发送消息需要使用的 token
}

func NewOauth(c config.ClientConfig) AuthService {
	return &AuthServiceImpl{
		oauthConfig: &oauth2.Config{
			ClientID:     c.AppID,
			ClientSecret: c.AppSecret,
			RedirectURL:  "http://localhost:8080/callback", // 请先添加该重定向 URL，配置路径：开发者后台 -> 开发配置 -> 安全设置 -> 重定向 URL -> 添加
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.feishu.cn/open-apis/authen/v1/authorize",
				TokenURL: "https://open.feishu.cn/open-apis/authen/v2/oauth/token",
			},
			Scopes: []string{"offline_access", "bitable:app", "base:app:create"},
		},
		accessToken:       "",
		refreshToken:      "",
		first:             true,
		tenantAccessToken: "",
	}
}

func (o *AuthServiceImpl) StartRefresh(accessToken string, refreshToken string) {
	o.mutex.Lock()
	o.accessToken = accessToken
	o.refreshToken = refreshToken
	o.mutex.Unlock()

	userRefresher := &feishu.UserTokenRefresher{
		Oauth:        o.oauthConfig,
		Mutex:        &o.mutex,
		AccessToken:  &o.accessToken,
		RefreshToken: &o.refreshToken,
	}

	tenantRefresher := &feishu.TenantTokenRefresher{
		Oauth:             o.oauthConfig,
		Mutex:             &o.mutex,
		TenantAccessToken: &o.tenantAccessToken,
	}

	var userAuto feishu.AutoRefresher
	userAuto.Start(userRefresher)

	var tenantAuto feishu.AutoRefresher
	tenantAuto.Start(tenantRefresher)
}

// StartAutoRefresh 启动定时刷新协程
func (o *AuthServiceImpl) StartAutoRefresh(accessToken string, refreshToken string, t time.Duration) {
	o.mutex.Lock()
	o.accessToken = accessToken
	o.refreshToken = refreshToken
	o.mutex.Unlock()

	if o.first == true {
		go func(t time.Duration) {
			o.mutex.Lock()
			o.first = false
			o.mutex.Unlock()
			ticker := time.NewTicker(t)
			defer ticker.Stop()
			for {
				<-ticker.C
				err := o.AutoRefreshToken()
				if err != nil {
					// TODO log
					fmt.Printf("refresh token failed:%v\n", err)
				}
			}
		}(t)
	}
}

func (o *AuthServiceImpl) AutoRefreshToken() error {
	token := o.getRefreshToken()

	requestBody := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": token,
		"client_id":     o.oauthConfig.ClientID,
		"client_secret": o.oauthConfig.ClientSecret,
	}
	jsonBody, _ := json.Marshal(requestBody)

	// 调用飞书 token 接口
	resp, err := http.Post(
		"https://open.feishu.cn/open-apis/authen/v2/oauth/token",
		"application/json; charset=utf-8",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return errs.FeishuRequestError(err)
	}
	defer resp.Body.Close()
	// 解析响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errs.ReadResponseError(err)
	}

	// 解析 JSON
	var tokenResp map[string]interface{}
	if err = json.Unmarshal(body, &tokenResp); err != nil {
		return errs.DeserializationError(err)
	}

	accessToken, ok1 := tokenResp["access_token"].(string)
	refreshToken, ok2 := tokenResp["refresh_token"].(string)
	// 更新缓存
	if ok1 && ok2 {
		o.mutex.Lock()
		o.accessToken = accessToken
		o.refreshToken = refreshToken
		o.mutex.Unlock()
	}

	// 返回 token 数据
	return nil
}

func (o *AuthServiceImpl) AutoRefreshTenantToken() error {
	requestBody := map[string]string{
		"app_id":     o.oauthConfig.ClientID,
		"app_secret": o.oauthConfig.ClientSecret,
	}

	jsonBody, _ := json.Marshal(requestBody)

	// 发起请求
	resp, err := http.Post("https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal",
		"application/json; charset=utf-8",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return errs.FeishuRequestError(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errs.ReadResponseError(err)
	}

	// 解析响应
	var res map[string]interface{}
	if err = json.Unmarshal(body, &res); err != nil {
		return err
	}

	tenantAccessToken, ok := res["tenant_access_token"].(string)
	if ok {
		o.mutex.Lock()
		o.tenantAccessToken = tenantAccessToken
		o.mutex.Unlock()
	}

	return nil
}

func (o *AuthServiceImpl) GetAccessToken() string {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return o.accessToken
}

func (o *AuthServiceImpl) GetTenantAccessToken() string {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return o.tenantAccessToken
}

// GetRefreshToken 获取刷新令牌
// 该方法不对外展示
func (o *AuthServiceImpl) getRefreshToken() string {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return o.refreshToken
}
