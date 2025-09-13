package model

type LikeMessage struct {
	ID          string    `json:"id"`
	Timestamp   int64     `json:"timestamp"`
	Attempts    int       `json:"attempts"` // 尝试次数
	MaxAttempts int       `json:"max_attempts"`
	Data        *LikeData `json:"data"`
}

type LikeData struct {
	AppToken string `json:"app_token"`
	TableID  string `json:"table_id"`
	RecordID string `json:"record_id"`
	UserID   string `json:"user_id"`
	IsLike   int    `json:"is_like"` // 0 = 未解决 1 = 已解决
	Action   string `json:"action"`  // add 点赞 remove 取消
}
