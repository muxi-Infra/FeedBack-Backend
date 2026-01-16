package controller

import (
	"github.com/muxi-Infra/FeedBack-Backend/api/request"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
	"github.com/muxi-Infra/FeedBack-Backend/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/singleflight"
)

type Auth struct {
	jwtHandler *ijwt.JWT
	group      *singleflight.Group
	s          service.AuthService
}

func NewAuth(jwtHandler *ijwt.JWT, s service.AuthService) *Auth {
	return &Auth{
		jwtHandler: jwtHandler,
		group:      &singleflight.Group{},
		s:          s,
	}
}

// GetTableToken godoc
//
//	@Summary		获取 table token 接口
//	@Description	获取 table token 接口
//	@Tags			Auth
//	@ID				get-table-token
//	@Accept			json
//	@Produce		json
//	@Param			request	body		request.GenerateTableTokenReq	true	"获取Token请求参数"
//	@Success		200		{object}	response.Response				"成功返回 JWT 令牌"
//	@Failure		400		{object}	response.Response				"请求参数错误"
//	@Failure		500		{object}	response.Response				"服务器内部错误"
//	@Router			/api/v1/auth/table-config/token [post]
func (o Auth) GetTableToken(c *gin.Context, req request.GenerateTableTokenReq) (response.Response, error) {
	tableCfg, err := o.s.GetTableConfig(&req.TableIdentify)
	if err != nil {
		return response.Response{}, err
	}

	token, err := o.jwtHandler.SetJWTToken(tableCfg.Identity, tableCfg.Name, tableCfg.TableToken, tableCfg.TableID, tableCfg.ViewID)
	if err != nil {
		return response.Response{}, errs.TokenGeneratedError(err)
	}

	c.Header("Authorization", token)

	return response.Response{
		Code:    0,
		Message: "Success",
		Data: map[string]string{
			"access_token": token,
		},
	}, nil
}

// RefreshTableConfig godoc
//
//	@Summary		刷新表格 token 配置接口
//	@Description	刷新并返回目前支持索引的表格的公开配置
//	@Tags			Auth
//	@ID				refresh-table-config
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	response.Response	"成功返回目前支持索引的表格的公开配置"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/api/v1/auth/table-config/refresh [get]
func (o Auth) RefreshTableConfig(c *gin.Context) (response.Response, error) {
	tableCfgs, err := o.s.RefreshTableConfig()
	if err != nil {
		return response.Response{}, err
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    tableCfgs,
	}, nil
}

// GetTenantToken godoc
//
//	@Summary		获取 tenant token 接口
//	@Description	获取 tenant token 接口，用于上传图片等
//	@Tags			Auth
//	@ID				get-tenant-token
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	response.Response	"成功返回 JWT 令牌"
//	@Failure		400	{object}	response.Response	"请求参数错误"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/api/v1/auth/tenant/token [post]
func (o Auth) GetTenantToken(c *gin.Context) (response.Response, error) {
	token := o.s.GetTenantToken()

	return response.Response{
		Code:    0,
		Message: "Success",
		Data: map[string]string{
			"access_token": token,
		},
	}, nil
}
