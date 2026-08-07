// Package feedback 提供校园项目后端接入反馈中台的 Go SDK。
package feedback

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const defaultAudience = "feedback-center"

// Config 配置校园项目与反馈中台之间的服务端身份交换。
type Config struct {
	Endpoint     string
	ProjectID    string
	KeyID        string
	Issuer       string
	PrivateKey   []byte
	Audience     string
	HTTPClient   *http.Client
	AssertionTTL time.Duration
}

// Identity 表示已经由校园项目后端确认的用户身份。
type Identity struct {
	StudentID     string
	TableIdentity string
}

// Token 是反馈中台返回的短期访问令牌。
type Token struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type exchangeRequest struct {
	ProjectID string `json:"project_id"`
	KeyID     string `json:"key_id"`
	Assertion string `json:"assertion"`
}

type exchangeResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    Token  `json:"data"`
}

// Client 负责生成身份断言并向反馈中台兑换反馈 JWT。
type Client struct {
	endpoint     string
	projectID    string
	keyID        string
	issuer       string
	audience     string
	privateKey   *rsa.PrivateKey
	httpClient   *http.Client
	assertionTTL time.Duration
}

// NewClient 创建 SDK 客户端。私钥只保存在当前进程内，不会写入请求日志或发送给反馈中台。
func NewClient(config Config) (*Client, error) {
	endpoint := strings.TrimRight(strings.TrimSpace(config.Endpoint), "/")
	if endpoint == "" || strings.TrimSpace(config.ProjectID) == "" || strings.TrimSpace(config.KeyID) == "" || strings.TrimSpace(config.Issuer) == "" {
		return nil, errors.New("endpoint、project_id、key_id 和 issuer 不能为空")
	}
	privateKey, err := parsePrivateKey(config.PrivateKey)
	if err != nil {
		return nil, err
	}
	audience := strings.TrimSpace(config.Audience)
	if audience == "" {
		audience = defaultAudience
	}
	ttl := config.AssertionTTL
	if ttl <= 0 {
		ttl = time.Minute
	}
	if ttl > 5*time.Minute {
		return nil, errors.New("assertion 有效期不能超过 5 分钟")
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		endpoint: endpoint, projectID: strings.TrimSpace(config.ProjectID), keyID: strings.TrimSpace(config.KeyID),
		issuer: strings.TrimSpace(config.Issuer), audience: audience, privateKey: privateKey,
		httpClient: httpClient, assertionTTL: ttl,
	}, nil
}

// Exchange 为当前学生和指定反馈表兑换反馈平台 JWT。
func (c *Client) Exchange(ctx context.Context, identity Identity) (Token, error) {
	if c == nil || c.privateKey == nil {
		return Token{}, errors.New("feedback client 未初始化")
	}
	studentID := strings.TrimSpace(identity.StudentID)
	tableIdentity := strings.TrimSpace(identity.TableIdentity)
	if studentID == "" || tableIdentity == "" {
		return Token{}, errors.New("student_id 和 table_identity 不能为空")
	}
	now := time.Now()
	jti, err := randomID()
	if err != nil {
		return Token{}, fmt.Errorf("生成 assertion jti 失败: %w", err)
	}
	claims := jwt.MapClaims{
		"iss": c.issuer, "aud": jwt.ClaimStrings{c.audience},
		"iat": now.Unix(), "exp": now.Add(c.assertionTTL).Unix(), "jti": jti,
		"project_id": c.projectID, "student_id": studentID, "table_identity": tableIdentity,
	}
	assertion := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	assertion.Header["kid"] = c.keyID
	assertionString, err := assertion.SignedString(c.privateKey)
	if err != nil {
		return Token{}, fmt.Errorf("签发 assertion 失败: %w", err)
	}
	body, err := json.Marshal(exchangeRequest{ProjectID: c.projectID, KeyID: c.keyID, Assertion: assertionString})
	if err != nil {
		return Token{}, fmt.Errorf("编码 Token exchange 请求失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/api/v1/integrations/token/exchange", bytes.NewReader(body))
	if err != nil {
		return Token{}, fmt.Errorf("创建 Token exchange 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Token{}, fmt.Errorf("请求反馈中台失败: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Token{}, fmt.Errorf("读取 Token exchange 响应失败: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return Token{}, fmt.Errorf("token exchange 返回 HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	var result exchangeResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return Token{}, fmt.Errorf("解析 Token exchange 响应失败: %w", err)
	}
	if result.Code != 0 || result.Data.AccessToken == "" {
		return Token{}, fmt.Errorf("token exchange 失败: code=%d message=%s", result.Code, result.Message)
	}
	return result.Data, nil
}

func parsePrivateKey(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("私钥不是有效 PEM")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("解析 RSA 私钥失败: %w", err)
	}
	return key, nil
}

func randomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
