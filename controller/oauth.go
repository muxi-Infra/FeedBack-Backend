package controller

import (
	"context"
	"encoding/json"
	"feedback/api/request"
	"feedback/api/response"
	"feedback/config"
	"feedback/pkg/ijwt"
	"feedback/service"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/sync/singleflight"
	"log"
	"math/rand"
	"net/http"
	"time"
)

type Oauth struct {
	oauthConfig *oauth2.Config
	jwtHandler  *ijwt.JWT
	group       *singleflight.Group

	ts *service.AuthService
}

var oauthEndpoint = oauth2.Endpoint{
	AuthURL:  "https://accounts.feishu.cn/open-apis/authen/v1/authorize",
	TokenURL: "https://open.feishu.cn/open-apis/authen/v2/oauth/token",
}

func NewOauth(c config.ClientConfig, jwtHandler *ijwt.JWT, tokenService *service.AuthService) *Oauth {
	return &Oauth{
		oauthConfig: &oauth2.Config{
			ClientID:     c.AppID,
			ClientSecret: c.AppSecret,
			RedirectURL:  "http://localhost:8080/callback", // 请先添加该重定向 URL，配置路径：开发者后台 -> 开发配置 -> 安全设置 -> 重定向 URL -> 添加
			Endpoint:     oauthEndpoint,
			Scopes:       []string{"offline_access", "bitable:app", "base:app:create"},
		},
		jwtHandler: jwtHandler,
		group:      &singleflight.Group{},
		ts:         tokenService,
	}
}

// IndexController godoc
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
	//		Message: "get old code success",
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

// GetToken godoc
//
//	@Summary		获取 token 接口
//	@Description	获取 token 接口
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	response.Response	"成功返回 JWT 令牌"
//	@Failure		400	{object}	response.Response	"请求参数错误"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/get_token [post]
func (o Oauth) GetToken(c *gin.Context, req request.GenerateTokenReq) (response.Response, error) {
	// todo
	// 判断携带参数是否为空
	if req.TableID == "" || req.NormalTableID == "" {
		return response.Response{
			Code:    400,
			Message: "请求参数为空",
			Data:    nil,
		}, fmt.Errorf("请求参数为空")
	}

	if !config.IsValidTableID(req.TableID) || !config.IsValidTableID(req.NormalTableID) {
		return response.Response{
			Code:    404,
			Message: "未找到相应的表格",
			Data:    nil,
		}, fmt.Errorf("未找到相应的表格")
	}

	token, err := o.jwtHandler.SetJWTToken(req.TableID, req.NormalTableID)
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
			"access_token": token,
		},
	}, nil
}

// InitToken 初始化 token
// InitToken godoc
//
//	@Summary		初始化 token 接口
//	@Description	初始化 token 接口
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		request.InitTokenReq	true	"初始化请求参数"
//	@Success		200		{object}	response.Response		"成功返回初始化结果"
//	@Failure		400		{object}	response.Response		"请求参数错误"
//	@Failure		500		{object}	response.Response		"服务器内部错误"
//	@Router			/init_token [post]
func (o Oauth) InitToken(c *gin.Context, r request.InitTokenReq) (response.Response, error) {
	// 启动定时刷新 token 协程
	go o.ts.StartAutoRefresh(r.AccessToken, r.RefreshToken, time.Duration(25)*time.Minute)

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    nil,
	}, nil
}
