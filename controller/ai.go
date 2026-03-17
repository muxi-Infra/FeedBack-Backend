package controller

import (
	"github.com/gin-gonic/gin"
	reqV1 "github.com/muxi-Infra/FeedBack-Backend/api/request/v1"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	respV1 "github.com/muxi-Infra/FeedBack-Backend/api/response/v1"
	"github.com/muxi-Infra/FeedBack-Backend/service"
)

type AIHandler interface {
	Query(c *gin.Context, req reqV1.AIQueryReq) (response.Response, error)
}

type AI struct {
	s service.AIService
}

func NewAI(s service.AIService) AIHandler {
	return &AI{
		s: s,
	}
}

// Query AI 客服咨询
//
//	@Summary		AI 客服咨询
//	@Description	提交用户问题，由 AI 助理结合历史 FAQ 数据库进行分析并返回解答。
//	@Tags			AI
//	@ID				ai-query
//	@Accept			json
//	@Produce		json
//	@Param			request	body		reqV1.AIQueryReq						true	"AI 查询请求参数"
//	@Success		200		{object}	response.Response{data=respV1.AIQueryResp}	"成功返回 AI 答复"
//	@Failure		400		{object}	response.Response						"请求参数错误"
//	@Failure		500		{object}	response.Response						"服务器内部错误"
//	@Router			/api/v1/ai/query [post]
func (a *AI) Query(c *gin.Context, req reqV1.AIQueryReq) (response.Response, error) {
	// 调用 AIService 执行 Agent 逻辑
	answer, err := a.s.Query(c.Request.Context(), req.Query)
	if err != nil {
		// 这里可以直接返回错误，由 Gin 的中间件或上层统一处理 errs
		return response.Response{}, err
	}

	resp := respV1.AIQueryResp{
		Answer: answer,
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp,
	}, nil
}
