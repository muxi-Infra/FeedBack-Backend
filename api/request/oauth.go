package request

type GenerateTokenReq struct {
	// Token string `json:"token" binding:"required"`
	// 更改为 TableID
	TableID       string `json:"table_id" binding:"required"`        // 反馈表格ID
	NormalTableID string `json:"normal_table_id" binding:"required"` // 常见问题表格ID
}
