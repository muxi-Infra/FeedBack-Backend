package request

type LikeReq struct {
	RecordID string `json:"record_id" binding:"required"`
	UserID   string `json:"user_id" binding:"required"`
	IsLike   int    `json:"is_like" binding:"required"`
	Action   string `json:"action" binding:"required"`
}
