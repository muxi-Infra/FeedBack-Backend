package v1

type ChatQueryReq struct {
	Query  string `json:"query" binding:"required"` //用户的问题描述
	UserID string `json:"user_id" binding:"required"`
}

type InsertReq struct {
	TableIdentify string `json:"tableIdentify" binding:"required"` //用户的问题描述
}
type GetHistoryReq struct {
	UserID string `form:"user_id" binding:"required"`
}
