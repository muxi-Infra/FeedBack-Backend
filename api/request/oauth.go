package request

type GenerateTokenReq struct {
	TableIdentity string `json:"table_identity" binding:"required"` // 反馈表格 Identity，反馈表的唯一标识
}
