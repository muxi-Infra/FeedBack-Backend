package controller

import (
	"github.com/gin-gonic/gin"
	reqV1 "github.com/muxi-Infra/FeedBack-Backend/api/request/v1"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	respV1 "github.com/muxi-Infra/FeedBack-Backend/api/response/v1"
	"github.com/muxi-Infra/FeedBack-Backend/service"
)

type ChatHandler interface {
	Query(c *gin.Context, req reqV1.ChatQueryReq) (response.Response, error)
}

type Chat struct {
	s service.ChatService
}

func NewChat(s service.ChatService) ChatHandler {
	return &Chat{
		s: s,
	}
}

// Query AI 客服咨询
//
//	@Summary		AI 客服咨询
//	@Description	提交用户问题，由 AI 助理结合历史 FAQ 数据库进行分析并返回解答。
//	@Tags			Chat
//	@ID				llm-query
//	@Accept			json
//	@Produce		json
//	@Param			request	body		reqV1.ChatQueryReq						true	"Chat 查询请求参数"
//	@Success		200		{object}	response.Response{data=respV1.ChatQueryResp}	"成功返回 Chat 答复"
//	@Failure		400		{object}	response.Response						"请求参数错误"
//	@Failure		500		{object}	response.Response						"服务器内部错误"
//	@Router			/api/v1/llm/query [post]
func (a *Chat) Query(c *gin.Context, req reqV1.ChatQueryReq) (response.Response, error) {
	// 调用 ChatService 执行 Agent 逻辑
	answer, err := a.s.Query(c.Request.Context(), req.Query)
	if err != nil {
		// 这里可以直接返回错误，由 Gin 的中间件或上层统一处理 errs
		return response.Response{}, err
	}

	resp := respV1.ChatQueryResp{
		Answer: answer,
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp,
	}, nil
}
