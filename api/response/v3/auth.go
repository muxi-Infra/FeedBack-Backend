package v3

type ExchangeFeedbackTokenResp struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// TenantTokenResp V3 图片上传使用的飞书租户令牌响应。
type TenantTokenResp struct {
	AccessToken string `json:"access_token"`
}
