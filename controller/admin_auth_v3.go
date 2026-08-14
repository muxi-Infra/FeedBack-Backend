package controller

import (
	"github.com/gin-gonic/gin"
	reqV3 "github.com/muxi-Infra/FeedBack-Backend/api/request/v3"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	respV3 "github.com/muxi-Infra/FeedBack-Backend/api/response/v3"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/service"
)

type AdminAuthHandlerV3 interface {
	Login(*gin.Context, reqV3.AdminLoginReq) (response.Response, error)
}

type AdminAuthV3 struct {
	service service.AdminAuthServiceV3
}

func NewAdminAuthV3(s service.AdminAuthServiceV3) AdminAuthHandlerV3 {
	return &AdminAuthV3{
		service: s,
	}
}

// Login 管理后台登录。
//
//	@Summary	V3 管理员登录
//	@Tags		V3Admin
//	@Accept		json
//	@Produce	json
//	@Param		request	body		reqV3.AdminLoginReq	true	"管理员账号"
//	@Success	200		{object}	response.Response{data=respV3.AdminLoginResp}
//	@Router		/api/v3/admin/login [post]
func (h *AdminAuthV3) Login(c *gin.Context, req reqV3.AdminLoginReq) (response.Response, error) {
	result, err := h.service.Login(c.Request.Context(), domain.AdminLoginInputV3{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		return response.Response{}, err
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data: respV3.AdminLoginResp{
			AccessToken: result.AccessToken,
			TokenType:   result.TokenType,
			ExpiresIn:   result.ExpiresIn,
		},
	}, nil
}
