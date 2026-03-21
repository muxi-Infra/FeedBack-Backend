package v1

type ChatQueryReq struct {
	Query  string `json:"query" binding:"required"` //用户的问题描述
	ConvID uint   `json:"conv_id" binding:"required"`
}

type InsertReq struct {
	TableIdentify string `json:"tableIdentify" binding:"required"`
}

type GetHistoryReq struct {
	ConvID uint `form:"conv_id" binding:"required"`
	LastID uint `form:"last_id" ` // 向上滑动用的游标
	Limit  int  `form:"limit" `   // 每次会向上拉取更早的 limit 条记录
}

type GetConversationReq struct {
	UserID string `form:"user_id" binding:"required"`
}
