package v2

// GetTableRecordByUserReq 获取用户历史反馈记录请求参数（通过用户 ID 获取）
type GetTableRecordByUserReq struct {
	TableIdentify *string `form:"table_identify" binding:"required"`
	StudentID     *string `form:"student_id" binding:"required"`  // 学号，用于标记用户身份
	PageToken     *string `form:"page_token" binding:"omitempty"` // 分页参数,第一次不需要
	LimitSize     *int    `form:"limit_size" binding:"omitempty"`
}
