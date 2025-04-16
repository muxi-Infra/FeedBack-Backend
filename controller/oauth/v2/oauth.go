package v2

import (
	"bytes"
	"context"
	"encoding/json"
	"feedback/api/request"
	"feedback/api/response"
	"feedback/config"
	"feedback/pkg/ijwt"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
	"io"
	"log"
	"math/rand"
	"net/http"
)

type Oauth struct {
	oauthConfig *oauth2.Config
	jwtHandler  *ijwt.JWT
}

var oauthEndpoint = oauth2.Endpoint{
	AuthURL:  "https://accounts.feishu.cn/open-apis/authen/v1/authorize",
	TokenURL: "https://open.feishu.cn/open-apis/authen/v2/oauth/token",
}

func NewOauth(c config.ClientConfig, jwtHandler *ijwt.JWT) *Oauth {
	return &Oauth{
		oauthConfig: &oauth2.Config{
			ClientID:     c.AppID,
			ClientSecret: c.AppSecret,
			RedirectURL:  "http://localhost:8080/callback", // 请先添加该重定向 URL，配置路径：开发者后台 -> 开发配置 -> 安全设置 -> 重定向 URL -> 添加
			Endpoint:     oauthEndpoint,
			Scopes:       []string{"offline_access", "bitable:app", "base:app:create"},
		},
		jwtHandler: jwtHandler,
	}
}

// IndexController godoc
//
//	@Summary		首页
//	@Description	显示欢迎页面并提供飞书登录入口
//	@Tags			Auth
//	@Produce		html
//	@Success		200	{string}	string	"HTML 页面"
//	@Router			/ [get]
func (o Oauth) IndexController(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	var username string
	session := sessions.Default(c)
	if session.Get("user") != nil {
		username = session.Get("user").(string)
	}
	html := fmt.Sprintf(`<html><head><style>body{font-family:Arial,sans-serif;background:#f4f4f4;margin:0;display:flex;justify-content:center;align-items:center;height:100vh}.container{text-align:center;background:#fff;padding:30px;border-radius:10px;box-shadow:0 0 10px rgba(0,0,0,0.1)}a{padding:10px 20px;font-size:16px;color:#fff;background:#007bff;border-radius:5px;text-decoration:none;transition:0.3s}a:hover{background:#0056b3}}</style></head><body><div class="container"><h2>欢迎%s！</h2><a href="/login">使用飞书登录</a></div></body></html>`, username)
	c.String(http.StatusOK, html)
}

// LoginController godoc
//
//	@Summary		跳转飞书授权
//	@Description	重定向到飞书 OAuth2 授权页面
//	@Tags			Auth
//	@Produce		json
//	@Success		302	{string}	string	"重定向到飞书登录"
//	@Router			/login [get]
func (o Oauth) LoginController(c *gin.Context) {
	session := sessions.Default(c)

	// 生成随机 state 字符串，你也可以用其他有意义的信息来构建 state
	state := fmt.Sprintf("%d", rand.Int())
	// 将 state 存入 session 中
	session.Set("state", state)
	// 生成 PKCE 需要的 code verifier
	verifier := oauth2.GenerateVerifier()
	// 将 code verifier 存入 session 中
	session.Set("code_verifier", verifier)
	session.Save()

	url := o.oauthConfig.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("scope", "offline_access bitable:app base:app:create"),
		oauth2.S256ChallengeOption(verifier))
	// 用户点击登录时，重定向到授权页面
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// OauthCallbackController godoc
//
//	@Summary		授权回调
//	@Description	处理飞书 OAuth2 回调并获取用户信息
//	@Tags			Auth
//	@Produce		html
//	@Param			state	query		string	true	"OAuth 状态"
//	@Param			code	query		string	true	"授权码"
//	@Success		200		{string}	string	"HTML 页面"
//	@Failure		302		{string}	string	"重定向回首页"
//	@Router			/callback [get]
func (o Oauth) OauthCallbackController(c *gin.Context) (response.Response, error) {
	session := sessions.Default(c)
	ctx := context.Background()

	// 从 session 中获取 state
	expectedState := session.Get("state")
	state := c.Query("state")

	// 如果 state 不匹配，说明是 CSRF 攻击，拒绝处理
	if state != expectedState {
		log.Printf("invalid oauth state, expected '%s', got '%s'\n", expectedState, state)
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return response.Response{
			Code:    400, //TODO
			Message: "Invalid OAuth state",
			Data:    nil,
		}, fmt.Errorf("invalid oauth state, expected '%s', got '%s'", expectedState, state)
	}

	code := c.Query("code")

	// 特殊测试，获取 code 调试 oldtoken
	//test := true
	//if test {
	//	return response.Response{
	//		Code:    0,
	//		Message: "get code success",
	//		Data:    code,
	//	}, nil
	//}

	// 如果 code 为空，说明用户拒绝了授权
	if code == "" {
		log.Printf("error: %s", c.Query("error"))
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return response.Response{
			Code:    400, //TODO
			Message: "Authorization denied",
			Data:    nil,
		}, fmt.Errorf("authorization denied, error: %s", c.Query("error"))
	}

	codeVerifier, _ := session.Get("code_verifier").(string)
	// 使用获取到的 code 获取 token
	token, err := o.oauthConfig.Exchange(ctx, code, oauth2.VerifierOption(codeVerifier))
	if err != nil {
		log.Printf("oauthConfig.Exchange() failed with '%s'\n", err)
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return response.Response{
			Code:    400,
			Message: "Failed to exchange code for token",
			Data:    nil,
		}, fmt.Errorf("oauthConfig.Exchange() failed with '%s'\n", err)
	}

	// 直接返回 token 信息（JSON 格式）
	//return response.Response{
	//	Code:    0,
	//	Message: "Success",
	//	Data:    token,
	//}, nil

	client := o.oauthConfig.Client(ctx, token)
	req, err := http.NewRequest("GET", "https://open.feishu.cn/open-apis/authen/v1/user_info", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	// 使用 token 发起请求，获取用户信息
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("client.Get() failed with '%s'\n", err)
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return response.Response{}, err
	}
	defer resp.Body.Close()
	var user struct {
		Data struct {
			Name   string `json:"name"`
			OpenId string `json:"open_id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		log.Printf("json.NewDecoder() failed with '%s'\n", err)
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return response.Response{}, err
	}
	// 后续可以用获取到的用户信息构建登录态，此处仅为示例，请勿直接使用
	session.Set("user", user.Data.Name)
	session.Save()
	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    token,
	}, nil
}

// RefreshToken godoc
//
//	@Summary		刷新Token
//	@Description	使用 refresh_token 刷新 access_token
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		request.RefreshTokenReq	true	"刷新 token 请求参数"
//	@Success		200		{object}	response.Response		"成功返回新的 token 信息"
//	@Failure		400		{object}	response.Response		"请求参数错误或飞书接口调用失败"
//	@Failure		500		{object}	response.Response		"服务器内部错误"
//	@Router			/refresh_token [post]
func (o Oauth) RefreshToken(c *gin.Context, r request.RefreshTokenReq) (response.Response, error) {
	// 准备请求体
	requestBody := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": r.RefreshToken,
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
		return response.Response{
			Code:    500,
			Message: "Failed to call token endpoint",
			Data:    nil,
		}, err
	}
	defer resp.Body.Close()

	// 解析响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response.Response{
			Code:    500,
			Message: "Failed to read response body",
			Data:    nil,
		}, err
	}

	// 解析 JSON
	var tokenResp map[string]interface{}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return response.Response{
			Code:    500,
			Message: "Invalid JSON response",
			Data:    nil,
		}, err
	}

	// 返回 token 数据
	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    tokenResp,
	}, nil
}

// GenerateToken  封装 token 接口
//
//	@Summary		封装 token 接口
//	@Description	封装 token 接口，将飞书 token 简单封装成 JWT 令牌
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		request.GenerateTokenReq	true	"封装 token 请求参数"
//	@Success		200		{object}	response.Response			"成功返回 JWT 令牌"
//	@Failure		400		{object}	response.Response			"请求参数错误"
//	@Failure		500		{object}	response.Response			"服务器内部错误"
//	@Router			/generate_token [post]
func (o Oauth) GenerateToken(c *gin.Context, r request.GenerateTokenReq) (response.Response, error) {
	token, err := o.jwtHandler.SetJWTToken(r.Token)
	if err != nil {
		return response.Response{
			Code:    500,
			Message: "生成 token 失败",
			Data:    nil,
		}, err
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data: map[string]string{
			"token": "Bearer " + token,
		},
	}, nil
}
