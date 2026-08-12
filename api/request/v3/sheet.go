package v3

type CreateFeedbackReq struct {
	Content     *string        `json:"content" binding:"required"`
	Images      []string       `json:"images"`
	ContactInfo *string        `json:"contact_info"`
	ExtraRecord map[string]any `json:"extra_record"`
}

type GetFeedbackReq struct {
	PageToken *string `form:"page_token"`
	LimitSize int     `form:"limit_size"`
}

type GetRecordReq struct {
	RecordID string `form:"record_id" binding:"required"`
}

// GetFAQReq 查询当前项目 FAQ。FAQ 内容从本地数据库读取
type GetFAQReq struct{}

type UpdateFAQReq struct {
	RecordID   *string `json:"record_id" binding:"required"`
	IsResolved *bool   `json:"is_resolved" binding:"required"`
}

type GetPhotoURLReq struct {
	RecordID   string   `form:"record_id" binding:"required"`
	FileTokens []string `form:"file_tokens" binding:"required,min=1"`
}
