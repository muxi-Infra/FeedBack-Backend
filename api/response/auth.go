package response

import "github.com/muxi-Infra/FeedBack-Backend/domain"

// GenerateTableTokenResp 生成表格访问令牌返回参数
type GenerateTableTokenResp struct {
	AccessToken string `json:"access_token"`
}

// GenerateTenantToken 生成租户访问令牌返回参数
type GenerateTenantToken struct {
	AccessToken string `json:"access_token"`
}

type RefreshTableConfigResp struct {
	TableConfig []domain.TableConfig `json:"table_config"`
}
