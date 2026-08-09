package v1

type GenerateTableTokenReq struct {
	TableIdentify string `json:"table_identify" binding:"required"`
}

type ExchangeIntegrationTokenReq struct {
	ProjectID     string `json:"project_id" binding:"required"`
	KeyID         string `json:"key_id" binding:"required"`
	StudentID     string `json:"student_id" binding:"required"`
	TableIdentity string `json:"table_identity" binding:"required"`
	Timestamp     int64  `json:"timestamp" binding:"required"`
	Nonce         string `json:"nonce" binding:"required"`
	Signature     string `json:"signature" binding:"required"`
}
