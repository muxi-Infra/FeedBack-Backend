package request

// CreatTableRecordReg 创建表格记录请求参数
type CreatTableRecordReg struct {
	TableIdentify *string        `json:"table_identify" binding:"required"`
	StudentID     *string        `json:"student_id" binding:"required"`    // 学号，用于标记用户身份
	Content       *string        `json:"content" binding:"required"`       // 反馈内容
	Images        []string       `json:"images" binding:"omitempty"`       // 图片附件 URL 列表，可选
	ContactInfo   *string        `json:"contact_info" binding:"omitempty"` // 联系方式，可选
	ExtraRecord   map[string]any `json:"extra_record" binding:"omitempty"` // 额外记录列表，可选
}

// GetTableRecordReq 获取表格记录请求参数（个人历史记录）
type GetTableRecordReq struct {
	TableIdentify *string  `form:"table_identify" binding:"required"`
	KeyFieldName  *string  `form:"key_field" binding:"required"`    // 用于查询记录的关键值，一般使用学号的字段名
	KeyFieldValue *string  `form:"key_value" binding:"required"`    // 用于查询记录的关键值，一般使用学号的字段值
	RecordNames   []string `form:"record_names" binding:"required"` // 需要查询的字段名
	PageToken     *string  `form:"page_token" binding:"omitempty"`  // 分页参数,第一次不需要
}

// GetTableRecordByRecordIDReq 获取表格记录请求参数（通过记录 ID 获取）
type GetTableRecordByRecordIDReq struct {
	TableIdentify *string `form:"table_identify" binding:"required"`
	RecordID      *string `form:"record_id" binding:"required"` // 记录 ID
}

// GetFAQProblemTableRecordReg 获取常见问题记录请求参数
// 这个接口直接获取全部常见问题
// 筛选由前端完成，对应字段在前端过滤
type GetFAQProblemTableRecordReg struct {
	TableIdentify *string  `form:"table_identify" binding:"required"`
	StudentID     *string  `form:"student_id" binding:"required"`   // 学号，用于标记用户身份
	RecordNames   []string `form:"record_names" binding:"required"` // 需要查询的字段名
}

type FAQResolutionUpdateReq struct {
	TableIdentify       *string `json:"table_identify" binding:"required"`
	ResolvedFieldName   *string `json:"resolved_field_name" binding:"required"`
	UnresolvedFieldName *string `json:"unresolved_field_name" binding:"required"`
	RecordID            *string `json:"record_id" binding:"required"`
	UserID              *string `json:"user_id" binding:"required"`
	IsResolved          *bool   `json:"is_resolved" binding:"required"`
}

// GetPhotoUrlReq 获取附件 URL 请求参数
type GetPhotoUrlReq struct {
	FileTokens []string `form:"file_tokens" binding:"required"` // 附件 token
}
