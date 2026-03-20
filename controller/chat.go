package controller

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	reqV1 "github.com/muxi-Infra/FeedBack-Backend/api/request/v1"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	respV1 "github.com/muxi-Infra/FeedBack-Backend/api/response/v1"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
	"github.com/muxi-Infra/FeedBack-Backend/service"
)

type ChatHandler interface {
	Chat(c *gin.Context, req reqV1.ChatQueryReq, uc ijwt.UserClaims) error
	Query(c *gin.Context, req reqV1.ChatQueryReq, uc ijwt.UserClaims) (response.Response, error)
	Insert(c *gin.Context, req reqV1.InsertReq) (response.Response, error)
	GetHistory(c *gin.Context, req reqV1.GetHistoryReq, uc ijwt.UserClaims) (response.Response, error)
}

type Chat struct {
	s service.ChatService
}

func NewChat(s service.ChatService) ChatHandler {
	return &Chat{
		s: s,
	}
}

// Chat AI 客服咨询
//
//	@Summary		AI 客服咨询
//	@Description	提交用户问题，由 AI 助理结合历史 FAQ 数据库进行分析并返回解答。
//	@Tags			Chat
//	@ID				llm-chat
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string				true	"Bearer Token"
//	@Param			request			body		reqV1.ChatQueryReq	true	"Chat 查询请求参数"
//	@Success		200				{object}	response.Response	"成功返回 Chat 答复"
//	@Failure		400				{object}	response.Response	"请求参数错误"
//	@Failure		500				{object}	response.Response	"服务器内部错误"
//	@Router			/api/v1/llm/chat [post]
func (a *Chat) Chat(c *gin.Context, req reqV1.ChatQueryReq, uc ijwt.UserClaims) error {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return fmt.Errorf("stream not supported")
	}

	msgCh, errCh := a.s.Chat(c.Request.Context(), req.Query, uc.TableIdentity, req.UserID)

	// 流式消费
	for {
		select {
		// 正常 token
		case msg, ok := <-msgCh:
			if !ok {
				// 流结束
				c.SSEvent("done", "complete")
				flusher.Flush()
				return nil
			}

			c.SSEvent("message", msg)
			flusher.Flush()

		// 错误处理
		case err := <-errCh:
			if err != nil {
				c.SSEvent("error", err.Error())
				flusher.Flush()
				return err
			}

		// 客户端断开
		case <-c.Request.Context().Done():
			fmt.Println("client disconnected")
			return nil
		}
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
//	@Param			Authorization	header		string											true	"Bearer Token"
//	@Param			request			body		reqV1.ChatQueryReq								true	"Chat 查询请求参数"
//	@Success		200				{object}	response.Response{data=respV1.ChatQueryResp}	"成功返回 Chat 答复"
//	@Failure		400				{object}	response.Response								"请求参数错误"
//	@Failure		500				{object}	response.Response								"服务器内部错误"
//	@Router			/api/v1/llm/query [post]
func (a *Chat) Query(c *gin.Context, req reqV1.ChatQueryReq, uc ijwt.UserClaims) (response.Response, error) {
	// 调用 ChatService 执行 Agent 逻辑
	answer, err := a.s.Query(c.Request.Context(), req.Query, uc.TableIdentity, req.UserID)
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

// Insert FAQ 数据入库（用于构建向量索引）
//
//	@Summary		插入 FAQ 数据
//	@Description	将指定 table_identify 的数据写入 FAQ 库，并构建向量索引（embedding + ES）
//	@Tags			Chat
//	@ID				llm-insert
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string				true	"Bearer Token"
//	@Param			request			body		reqV1.InsertReq		true	"插入请求参数"
//	@Success		200				{object}	response.Response	"插入成功"
//	@Failure		400				{object}	response.Response	"请求参数错误"
//	@Failure		500				{object}	response.Response	"服务器内部错误"
//	@Router			/api/v1/llm/insert [post]
func (a *Chat) Insert(c *gin.Context, req reqV1.InsertReq) (response.Response, error) {
	// 调用 ChatService 执行 Agent 逻辑
	err := a.s.Insert(c.Request.Context(), req.TableIdentify)
	if err != nil {
		// 这里可以直接返回错误，由 Gin 的中间件或上层统一处理 errs
		return response.Response{}, err
	}

	return response.Response{
		Code:    0,
		Message: "Success",
	}, nil
}

// GetHistory 获取会话历史记录
// @Summary      查询聊天历史
// @Description  根据当前用户的身份标识和请求的 UserID，从 Redis 缓存（或 DB）中拉取完整的对话上下文
// @Tags         Chat
// @ID           llm-get-history
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        request   query      reqV1.GetHistoryReq  true  "查询参数（UserID）"
// @Success      200       {object}  response.Response{data=respV1.GetHistoryResp} "返回完整的 Conversation 对象（含 Messages）"
// @Failure      404       {object}  response.Response    "会话已过期或不存在"
// @Failure      500       {object}  response.Response    "缓存查询异常"
// @Router       /api/v1/llm/history [get]
func (a *Chat) GetHistory(c *gin.Context, req reqV1.GetHistoryReq, uc ijwt.UserClaims) (response.Response, error) {
	history, err := a.s.GetHistory(c, uc.TableIdentity+req.UserID)
	if err != nil {
		return response.Response{}, err
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    respV1.GetHistoryResp{Conversation: *history},
	}, nil
}
