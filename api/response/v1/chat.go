package v1

import "github.com/muxi-Infra/FeedBack-Backend/domain"

type ChatQueryResp struct {
	Answer string `json:"answer"` //回答的结果
}

type GetHistoryResp struct {
	Messages []*domain.Message `json:"messages"`
}

type GetConversationResp struct {
	Conversation domain.Conversation `json:"conversation"`
}
