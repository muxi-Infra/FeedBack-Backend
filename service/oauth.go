package service

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/feishu"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"

	"golang.org/x/oauth2"
)

// AuthService 暴露给 controller 的接口
type AuthService interface {
	StartRefresh(accessToken string, refreshToken string)
	GetLoginURL(state string, verifier string) string
	Code2Token(ctx context.Context, code string, codeVerifier string) (*oauth2.Token, error)
	GetUserInfoByToken(ctx context.Context, token *oauth2.Token) (*UserInfo, error)
}

// AuthTokenProvider 给其他 service 层（需要token) 暴露接口
type AuthTokenProvider interface {
	GetAccessToken() string
	GetTenantAccessToken() string
}

type AuthServiceImpl struct {
	oauthConfig *oauth2.Config

	log logger.Logger

	mutex        sync.RWMutex
	accessToken  string
	refreshToken string // 不对外展示

	tenantAccessToken string // 自建应用发送消息需要使用的 token
}

var oauthEndpoint = oauth2.Endpoint{
	AuthURL:  "https://accounts.feishu.cn/open-apis/authen/v1/authorize",
	TokenURL: "https://open.feishu.cn/open-apis/authen/v2/oauth/token",
}

func NewOauth(c config.ClientConfig, log logger.Logger) *AuthServiceImpl {
	return &AuthServiceImpl{
		oauthConfig: &oauth2.Config{
			ClientID:     c.AppID,
			ClientSecret: c.AppSecret,
			RedirectURL:  "http://localhost:8080/callback", // 请先添加该重定向 URL，配置路径：开发者后台 -> 开发配置 -> 安全设置 -> 重定向 URL -> 添加
			Endpoint:     oauthEndpoint,
			Scopes:       []string{"offline_access", "bitable:app", "base:app:create"},
		},
		log:               log,
		accessToken:       "",
		refreshToken:      "",
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

func (o *AuthServiceImpl) GetLoginURL(state string, verifier string) string {
	return o.oauthConfig.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("scope", strings.Join(o.oauthConfig.Scopes, " ")),
		oauth2.S256ChallengeOption(verifier))
}

func (o *AuthServiceImpl) Code2Token(ctx context.Context, code string, codeVerifier string) (*oauth2.Token, error) {
	return o.oauthConfig.Exchange(ctx, code, oauth2.VerifierOption(codeVerifier))
}

type UserInfo struct {
	Data struct {
		Name   string `json:"name"`
		OpenId string `json:"open_id"`
	} `json:"data"`
}

func (o *AuthServiceImpl) GetUserInfoByToken(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := o.oauthConfig.Client(ctx, token)
	req, err := http.NewRequest("GET", "https://open.feishu.cn/open-apis/authen/v1/user_info", nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	// 使用 token 发起请求，获取用户信息
	resp, err := client.Do(req)
	if err != nil {
		o.log.Error("client.Get() failed",
			logger.String("error", err.Error()))
		return nil, errs.FeishuRequestError(err)
	}
	defer resp.Body.Close()

	var user UserInfo
	if err = json.NewDecoder(resp.Body).Decode(&user); err != nil {
		o.log.Error("json.NewDecoder() failed",
			logger.String("error", err.Error()))
		return nil, errs.JsonDecodeError(err)
	}
	return &user, nil
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
