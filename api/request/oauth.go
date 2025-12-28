package request

type GenerateTokenReq struct {
	TableCode string `json:"table_code" binding:"required"` // 反馈表格ID
}
