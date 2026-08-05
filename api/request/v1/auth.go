package v1

type GenerateTableTokenReq struct {
	TableIdentify string `json:"table_identify" binding:"required"` // 反馈表格 Identify，反馈表的唯一标识
}

type ExchangeIntegrationTokenReq struct {
	ProjectID string `json:"project_id" binding:"required"`
	KeyID     string `json:"key_id" binding:"required"`
	Assertion string `json:"assertion" binding:"required"`
}
