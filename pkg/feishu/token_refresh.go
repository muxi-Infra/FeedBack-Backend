package feishu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// 这里使用策略模式优化一下 token 刷新的代码

type TokenRefresher interface {
	Refresh() error
	Interval() time.Duration
}

// AutoRefresher 使用策略的一方，即在策略模式中的 “上下文”
type AutoRefresher struct {
	once sync.Once
}

func (a *AutoRefresher) Start(refresher TokenRefresher) {
	a.once.Do(func() {
		go func() {
			ticker := time.NewTicker(refresher.Interval())
			//ticker := time.NewTicker(time.Second * 25)
			defer ticker.Stop()

			for {
				<-ticker.C
				err := refresher.Refresh()
				if err != nil {
					// TODO log
					fmt.Printf("refresh token failed:%v\n", err)
				}
			}
		}()
	})
}

// UserTokenRefresher user token
type UserTokenRefresher struct {
	Oauth *oauth2.Config
	Mutex *sync.RWMutex

	AccessToken  *string
	RefreshToken *string
}

func (u *UserTokenRefresher) Interval() time.Duration {
	return time.Duration(25) * time.Minute
}

func (u *UserTokenRefresher) Refresh() error {
	//fmt.Printf("UserTokenRefresher 启动刷新 token\n")

	requestBody := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": *u.RefreshToken,
		"client_id":     u.Oauth.ClientID,
		"client_secret": u.Oauth.ClientSecret,
	}

	jsonBody, _ := json.Marshal(requestBody)

	// 调用飞书 token 接口
	resp, err := http.Post(
		"https://open.feishu.cn/open-apis/authen/v2/oauth/token",
		"application/json; charset=utf-8",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 解析响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// 解析 JSON
	var tokenResp map[string]interface{}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return err
	}

	accessToken, ok1 := tokenResp["access_token"].(string)
	refreshToken, ok2 := tokenResp["refresh_token"].(string)

	if !ok1 || !ok2 {
		return fmt.Errorf("invalid user token response: %s", string(body))
	}

	//fmt.Printf("刷新 user token 成功\n")

	u.Mutex.Lock()
	*u.AccessToken = accessToken
	*u.RefreshToken = refreshToken
	u.Mutex.Unlock()

	return nil
}

// TenantTokenRefresher tenant token
type TenantTokenRefresher struct {
	Oauth *oauth2.Config
	Mutex *sync.RWMutex

	TenantAccessToken *string
}

func (t *TenantTokenRefresher) Interval() time.Duration {
	return time.Duration(1) * time.Hour // 最大有效期是 2 小时
}

func (t *TenantTokenRefresher) Refresh() error {
	//fmt.Printf("TenantTokenRefresher 启动刷新 token\n")

	requestBody := map[string]string{
		"app_id":     t.Oauth.ClientID,
		"app_secret": t.Oauth.ClientSecret,
	}

	jsonBody, _ := json.Marshal(requestBody)

	// 发起请求
	resp, err := http.Post("https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal",
		"application/json; charset=utf-8",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// 解析响应
	var res map[string]interface{}
	if err = json.Unmarshal(body, &res); err != nil {
		return err
	}

	tenantAccessToken, ok := res["tenant_access_token"].(string)
	if !ok {
		return fmt.Errorf("invalid tenant token response: %s", string(body))
	}

	// fmt.Printf("刷新 tenant token 成功\n")

	t.Mutex.Lock()
	*t.TenantAccessToken = tenantAccessToken
	t.Mutex.Unlock()

	return nil
}
