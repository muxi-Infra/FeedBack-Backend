package v1

import (
	"bytes"
	"encoding/json"
	"feedback/api/response"
	"feedback/config"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
)

type OldToken struct {
	AppID     string
	AppSecret string
}

func NewOldToken(c config.ClientConfig) *OldToken {
	return &OldToken{
		AppID:     c.AppID,
		AppSecret: c.AppSecret,
	}
}

// GetAppAccessToken godoc
//
//	@Summary		获取 App Access Token
//	@Description	通过 app_id 和 app_secret 获取 app_access_token
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Success		200		{object}	response.Response	"成功返回 app_access_token"
//	@Failure		400		{object}	response.Response	"请求参数错误或飞书接口调用失败"
//	@Failure		500		{object}	response.Response	"服务器内部错误"
//	@Router			/app_access_token [post]
func (o OldToken) GetAppAccessToken(c *gin.Context) (response.Response, error) {
	// 创建请求体
	requestBody := map[string]string{
		"app_id":     o.AppID,
		"app_secret": o.AppSecret,
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return response.Response{
			Code:    500,
			Message: "Failed to marshal request body",
			Data:    nil,
		}, err
	}

	// 发起请求
	req, err := http.NewRequest("POST", "https://open.feishu.cn/open-apis/auth/v3/app_access_token/internal", bytes.NewBuffer(body))
	if err != nil {
		return response.Response{
			Code:    500,
			Message: "Failed to create request",
			Data:    nil,
		}, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return response.Response{
			Code:    500,
			Message: "Failed to send request",
			Data:    nil,
		}, err
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return response.Response{
			Code:    500,
			Message: "Failed to read response body",
			Data:    nil,
		}, err
	}

	var tokenResp map[string]interface{}
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return response.Response{
			Code:    500,
			Message: "Failed to unmarshal response",
			Data:    nil,
		}, err
	}

	// 判断飞书返回的 code 是否为 0
	if code, ok := tokenResp["code"].(float64); !ok || code != 0 {
		return response.Response{
			Code:    400,
			Message: fmt.Sprintf("Feishu error: %v", tokenResp["msg"]),
			Data:    tokenResp,
		}, nil
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    tokenResp,
	}, nil
}
