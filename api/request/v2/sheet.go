package v2

// GetTableRecordByUserReq 获取用户历史反馈记录请求参数（通过用户 ID 获取）
type GetTableRecordByUserReq struct {
	TableIdentify *string `form:"table_identify" binding:"required"`
	StudentID     *string `form:"student_id" binding:"required"`  // 学号，用于标记用户身份
	PageToken     *string `form:"page_token" binding:"omitempty"` // 分页参数,第一次不需要
	LimitSize     *int    `form:"limit_size" binding:"omitempty"`
}

// SyncUnsyncedTableRecordsReq 同步指定表格下所有未同步的记录请求参数（不区分用户））
type SyncUnsyncedTableRecordsReq struct {
	TableIdentify *string `json:"table_identify" binding:"required"`
}

// ForceSyncUserTableRecordsReq 同步指定表格下所有记录请求参数（通过用户 ID 获取）
// 强制同步（不区分是否已经同步过）
type ForceSyncUserTableRecordsReq struct {
	TableIdentify *string `json:"table_identify" binding:"required"`
	StudentID     *string `json:"student_id" binding:"required"`
}

// ForceSyncTableRecordsReq 同步指定表格下所有记录请求参数
// 强制同步（不区分是否已经同步过）
// 慎用！！！
type ForceSyncTableRecordsReq struct {
	TableIdentify *string `json:"table_identify" binding:"required"`
}

// GetFAQProblemTableRecordReg 获取常见问题记录请求参数
type GetFAQProblemTableRecordReg struct {
	TableIdentify *string `form:"table_identify" binding:"required"`
	StudentID     *string `form:"student_id" binding:"required"` // 学号，用于标记用户身份
}

// FAQResolutionUpdateReq 更新常见问题解决状态请求参数
type FAQResolutionUpdateReq struct {
	TableIdentify *string `json:"table_identify" binding:"required"`
	RecordID      *string `json:"record_id" binding:"required"`
	UserID        *string `json:"user_id" binding:"required"`
	IsResolved    *bool   `json:"is_resolved" binding:"required"`
}

type SyncFaqRecordReq struct {
	TableIdentify *string `json:"table_identify" binding:"required"`
}
