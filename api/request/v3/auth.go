package v3

// ExchangeFeedbackTokenReq V3 项目后端交换用户反馈令牌的请求。
type ExchangeFeedbackTokenReq struct {
	ProjectID string `json:"project_id" binding:"required"`
	KeyID     string `json:"key_id" binding:"required"`
	StudentID string `json:"student_id" binding:"required"`
	Timestamp int64  `json:"timestamp" binding:"required"`
	Nonce     string `json:"nonce" binding:"required"`
	Signature string `json:"signature" binding:"required"`
}
