package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	reqV1 "github.com/muxi-Infra/FeedBack-Backend/api/request/v1"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	respV1 "github.com/muxi-Infra/FeedBack-Backend/api/response/v1"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/middleware"
	"github.com/muxi-Infra/FeedBack-Backend/service"
)

// AdminHandler 管理后台账号接口。
type AdminHandler interface {
	Login(c *gin.Context, req reqV1.AdminLoginReq) (response.Response, error)
	CreateAdmin(c *gin.Context, req reqV1.CreateAdminReq) (response.Response, error)
	GetAdmin(c *gin.Context) (response.Response, error)
	ChangePassword(c *gin.Context, req reqV1.ChangeAdminPasswordReq) (response.Response, error)
	UpdateStatus(c *gin.Context, req reqV1.UpdateAdminStatusReq) (response.Response, error)
}

type Admin struct {
	service service.AdminService
}

func NewAdmin(s service.AdminService) AdminHandler {
	return &Admin{service: s}
}

// Login 管理员登录。
//
//	@Summary		管理员登录
//	@Description	校验管理员账号和密码，返回 Admin JWT。
//	@Tags			Admin Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		reqV1.AdminLoginReq	true	"管理员登录信息"
//	@Success		200		{object}	response.Response{data=respV1.AdminLoginResponse}
//	@Failure		400		{object}	response.Response
//	@Failure		401		{object}	response.Response
//	@Router			/api/v1/admin/login [post]
func (h *Admin) Login(c *gin.Context, req reqV1.AdminLoginReq) (response.Response, error) {
	result, err := h.service.Login(c.Request.Context(), domain.AdminLoginInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		return response.Response{}, err
	}
	return response.Response{Code: 0, Message: "Success", Data: respV1.AdminLoginResponse{
		AccessToken: result.Token,
		TokenType:   "Bearer",
		ExpiresAt:   result.ExpiresAt,
		User:        toAdminUserResponse(result.User),
	}}, nil
}

// CreateAdmin 创建管理员账号。
//
//	@Summary		创建管理员
//	@Description	创建新的管理后台账号，需要具备 admin:create 权限。
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string					true	"Bearer Admin JWT"
//	@Param			request			body		reqV1.CreateAdminReq	true	"管理员信息"
//	@Success		200				{object}	response.Response{data=respV1.AdminUserResponse}
//	@Failure		400				{object}	response.Response
//	@Failure		401				{object}	response.Response
//	@Failure		403				{object}	response.Response
//	@Failure		409				{object}	response.Response
//	@Router			/api/v1/admin/users [post]
func (h *Admin) CreateAdmin(c *gin.Context, req reqV1.CreateAdminReq) (response.Response, error) {
	user, err := h.service.CreateAdmin(c.Request.Context(), domain.CreateAdminInput{
		Username: req.Username, DisplayName: req.DisplayName, Password: req.Password,
	})
	if err != nil {
		return response.Response{}, err
	}
	return response.Response{Code: 0, Message: "Success", Data: toAdminUserResponse(user)}, nil
}

// GetAdmin 查询管理员账号。
//
//	@Summary		查询管理员
//	@Description	根据管理员 ID 查询账号信息，需要具备 admin:read 权限。
//	@Tags			Admin
//	@Produce		json
//	@Param			Authorization	header		string	true	"Bearer Admin JWT"
//	@Param			admin_id		path		uint64	true	"管理员 ID"
//	@Success		200				{object}	response.Response{data=respV1.AdminUserResponse}
//	@Failure		400				{object}	response.Response
//	@Failure		401				{object}	response.Response
//	@Failure		403				{object}	response.Response
//	@Failure		404				{object}	response.Response
//	@Router			/api/v1/admin/users/{admin_id} [get]
func (h *Admin) GetAdmin(c *gin.Context) (response.Response, error) {
	id, err := strconv.ParseUint(c.Param("admin_id"), 10, 64)
	if err != nil || id == 0 {
		return response.Response{}, errs.AdminInvalidInputError(err)
	}
	user, err := h.service.GetAdmin(c.Request.Context(), id)
	if err != nil {
		return response.Response{}, err
	}
	return response.Response{Code: 0, Message: "Success", Data: toAdminUserResponse(user)}, nil
}

// ChangePassword 修改当前登录管理员的密码。
//
//	@Summary		修改管理员密码
//	@Description	根据当前 Admin JWT 修改自己的密码，需要具备 admin:update 权限。
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string							true	"Bearer Admin JWT"
//	@Param			request			body		reqV1.ChangeAdminPasswordReq	true	"密码信息"
//	@Success		200				{object}	response.Response
//	@Failure		400				{object}	response.Response
//	@Failure		401				{object}	response.Response
//	@Failure		403				{object}	response.Response
//	Router			/api/v1/admin/users/password [put]
func (h *Admin) ChangePassword(c *gin.Context, req reqV1.ChangeAdminPasswordReq) (response.Response, error) {
	adminID, ok := middleware.GetAdminID(c)
	if !ok {
		return response.Response{}, errs.AdminTokenError(nil)
	}
	err := h.service.ChangePassword(c.Request.Context(), domain.ChangeAdminPasswordInput{
		AdminID: adminID, CurrentPassword: req.CurrentPassword, NewPassword: req.NewPassword,
	})
	if err != nil {
		return response.Response{}, err
	}
	return response.Response{Code: 0, Message: "Success", Data: nil}, nil
}

// UpdateStatus 修改管理员状态。
//
//	@Summary		修改管理员状态
//	@Description	启用、锁定或禁用管理员账号，需要具备 admin:update 权限。
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string						true	"Bearer Admin JWT"
//	@Param			admin_id		path		uint64						true	"管理员 ID"
//	@Param			request			body		reqV1.UpdateAdminStatusReq	true	"管理员状态"
//	@Success		200				{object}	response.Response
//	@Failure		400				{object}	response.Response
//	@Failure		401				{object}	response.Response
//	@Failure		403				{object}	response.Response
//	@Failure		404				{object}	response.Response
//	@Router			/api/v1/admin/users/{admin_id}/status [patch]
func (h *Admin) UpdateStatus(c *gin.Context, req reqV1.UpdateAdminStatusReq) (response.Response, error) {
	id, err := strconv.ParseUint(c.Param("admin_id"), 10, 64)
	if err != nil || id == 0 {
		return response.Response{}, errs.AdminInvalidInputError(err)
	}
	if err := h.service.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		return response.Response{}, err
	}
	return response.Response{Code: 0, Message: "Success", Data: nil}, nil
}

func toAdminUserResponse(user domain.AdminUser) respV1.AdminUserResponse {
	return respV1.AdminUserResponse{
		ID: user.ID, Username: user.Username, DisplayName: user.DisplayName,
		Status: user.Status, LastLoginAt: user.LastLoginAt,
		CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt,
	}
}
