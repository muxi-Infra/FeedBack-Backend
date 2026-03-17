package v1

type AIQueryReq struct {
	Query string `json:"query" binding:"required"` //用户的问题描述
}
