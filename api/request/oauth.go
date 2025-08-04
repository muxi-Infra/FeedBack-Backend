package request

type RefreshTokenReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type GenerateTokenReq struct {
	// Token string `json:"token" binding:"required"`
	// 更改为 TableID
	TableID       string `json:"table_id" binding:"required"` // 反馈表格ID
	NormalTableID string `json:"normal_table_id"`             // 常见问题表格ID
}

type InitTokenReq struct {
	AccessToken  string `json:"access_token" binding:"required"`
	RefreshToken string `json:"refresh_token" binding:"required"`
}
