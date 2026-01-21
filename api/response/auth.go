package response

// GenerateTableTokenResp 生成表格访问令牌返回参数
type GenerateTableTokenResp struct {
	AccessToken string `json:"access_token"`
}

// GenerateTenantToken 生成租户访问令牌返回参数
type GenerateTenantToken struct {
	AccessToken string `json:"access_token"`
}
