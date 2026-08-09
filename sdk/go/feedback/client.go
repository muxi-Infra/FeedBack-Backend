// Package feedback 提供校园项目后端接入反馈中台的 Go SDK。
package feedback

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Config 配置校园项目与反馈中台之间的服务端身份交换。
type Config struct {
	Endpoint   string
	ProjectID  string
	KeyID      string
	APIKey     string
	HTTPClient *http.Client
}

// Identity 表示已经由校园项目后端确认的用户身份。
type Identity struct {
	StudentID     string
	TableIdentity string
}

// Token 反馈中台返回的短期访问令牌。
type Token struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type exchangeRequest struct {
	ProjectID     string `json:"project_id"`
	KeyID         string `json:"key_id"`
	StudentID     string `json:"student_id"`
	TableIdentity string `json:"table_identity"`
	Timestamp     int64  `json:"timestamp"`
	Nonce         string `json:"nonce"`
	Signature     string `json:"signature"`
}

type exchangeResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    Token  `json:"data"`
}

// Client 负责使用 API Key 生成 HMAC 请求签名并兑换反馈 JWT。
type Client struct {
	endpoint   string
	projectID  string
	keyID      string
	apiKey     string
	httpClient *http.Client
}

// NewClient 创建 SDK 客户端。API Key 只应保存在校园项目后端。
func NewClient(config Config) (*Client, error) {
	endpoint := strings.TrimRight(strings.TrimSpace(config.Endpoint), "/")
	if endpoint == "" || strings.TrimSpace(config.ProjectID) == "" || strings.TrimSpace(config.KeyID) == "" || strings.TrimSpace(config.APIKey) == "" {
		return nil, errors.New("endpoint、project_id、key_id 和 api_key 不能为空")
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{endpoint: endpoint, projectID: strings.TrimSpace(config.ProjectID), keyID: strings.TrimSpace(config.KeyID), apiKey: strings.TrimSpace(config.APIKey), httpClient: httpClient}, nil
}

// Exchange 为当前学生和指定反馈表兑换反馈平台 JWT。
func (c *Client) Exchange(ctx context.Context, identity Identity) (Token, error) {
	if c == nil || c.apiKey == "" {
		return Token{}, errors.New("feedback client 未初始化")
	}
	studentID := strings.TrimSpace(identity.StudentID)
	tableIdentity := strings.TrimSpace(identity.TableIdentity)
	if studentID == "" || tableIdentity == "" {
		return Token{}, errors.New("student_id 和 table_identity 不能为空")
	}
	nonce, err := randomID()
	if err != nil {
		return Token{}, fmt.Errorf("生成 nonce 失败: %w", err)
	}
	timestamp := time.Now().Unix()
	payload := fmt.Sprintf("%s\n%s\n%s\n%s\n%d\n%s", c.projectID, c.keyID, studentID, tableIdentity, timestamp, nonce)
	bodyData := exchangeRequest{
		ProjectID:     c.projectID,
		KeyID:         c.keyID,
		StudentID:     studentID,
		TableIdentity: tableIdentity,
		Timestamp:     timestamp,
		Nonce:         nonce,
		Signature:     sign(c.apiKey, payload),
	}
	body, err := json.Marshal(bodyData)
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

func sign(apiKey, payload string) string {
	digest := sha256.Sum256([]byte(apiKey))
	mac := hmac.New(sha256.New, []byte(hex.EncodeToString(digest[:])))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func randomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
