package request

type FAQResolution struct {
	RecordID *string `json:"record_id" binding:"required"`
	UserID   *string `json:"user_id" binding:"required"`
	Like     *bool   `json:"is_like" binding:"required"`
}

type LikeReq struct {
	RecordID string `json:"record_id" binding:"required"`
	UserID   string `json:"user_id" binding:"required"`
	IsLike   int    `json:"is_like"`
	Action   string `json:"action" binding:"required"`
}
