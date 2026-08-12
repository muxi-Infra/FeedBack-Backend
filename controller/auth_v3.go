package controller

import (
	"errors"

	"github.com/gin-gonic/gin"
	reqV3 "github.com/muxi-Infra/FeedBack-Backend/api/request/v3"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	respV3 "github.com/muxi-Infra/FeedBack-Backend/api/response/v3"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/service"
)

type V3AuthHandler interface {
	Exchange(c *gin.Context, req reqV3.ExchangeFeedbackTokenReq) (response.Response, error)
	GetTenantToken(c *gin.Context, projectID, studentID string) (response.Response, error)
}

type V3Auth struct {
	service service.V3AuthService
	// todo 当前租户令牌由 V1 的 AuthService 提供，后续 V1 下线后迁移到 V3 服务。
	legacy service.AuthService
}

func NewV3Auth(s service.V3AuthService, legacy service.AuthService) V3AuthHandler {
	return &V3Auth{
		service: s,
		legacy:  legacy,
	}
}

// Exchange 通过接入项目后端签名的请求交换用户级反馈 JWT。
//
//	@Summary	V3 交换反馈访问令牌
//	@Tags		V3Auth
//	@Accept		json
//	@Produce	json
//	@Param		request	body		reqV3.ExchangeFeedbackTokenReq	true	"交换请求"
//	@Success	200		{object}	response.Response{data=respV3.ExchangeFeedbackTokenResp}
//	@Router		/api/v3/integrations/token/exchange [post]
func (h *V3Auth) Exchange(c *gin.Context, req reqV3.ExchangeFeedbackTokenReq) (response.Response, error) {
	token, expires, err := h.service.Exchange(c.Request.Context(), service.V3ExchangeInput{
		ProjectID: req.ProjectID,
		KeyID:     req.KeyID,
		StudentID: req.StudentID,
		Timestamp: req.Timestamp,
		Nonce:     req.Nonce,
		Signature: req.Signature,
	})
	if err != nil {
		return response.Response{}, err
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data: respV3.ExchangeFeedbackTokenResp{
			AccessToken: token,
			TokenType:   "Bearer",
			ExpiresIn:   expires,
		},
	}, nil
}

// GetTenantToken 为 V3 图片上传保留租户令牌接口，但必须先通过 V3 JWT。
//
//	@Summary	获取 V3 图片上传租户令牌
//	@Tags	V3Auth
//	@Produce	json
//	@Param	Authorization	header	string	true	"Bearer 用户反馈 Token"
//	@Success	200	{object}	response.Response{data=respV3.TenantTokenResp}
//	@Router	/api/v3/auth/tenant/token [post]
func (h *V3Auth) GetTenantToken(c *gin.Context, projectID, studentID string) (response.Response, error) {
	// todo 底层使用的是 V1 版本的，后续 V1 不需要的时候将其迁移到此处
	if projectID == "" || studentID == "" {
		return response.Response{}, errs.V3IdentityRequiredError(errors.New("project_id or student_id is empty"))
	}
	return response.Response{
		Code:    0,
		Message: "Success",
		Data: respV3.TenantTokenResp{
			AccessToken: h.legacy.GetTenantToken(),
		},
	}, nil
}
