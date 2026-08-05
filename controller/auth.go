package controller

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	reqV1 "github.com/muxi-Infra/FeedBack-Backend/api/request/v1"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	respV1 "github.com/muxi-Infra/FeedBack-Backend/api/response/v1"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
	"github.com/muxi-Infra/FeedBack-Backend/service"
	"golang.org/x/sync/singleflight"
)

type AuthHandler interface {
	GetTableToken(c *gin.Context, req reqV1.GenerateTableTokenReq) (response.Response, error)
	ExchangeIntegrationToken(c *gin.Context, req reqV1.ExchangeIntegrationTokenReq) (response.Response, error)
	RefreshTableConfig(c *gin.Context) (response.Response, error)
	GetTenantToken(c *gin.Context) (response.Response, error)
}

type Auth struct {
	jwtHandler *ijwt.JWT
	group      *singleflight.Group
	s          service.AuthService
}

func NewAuth(jwtHandler *ijwt.JWT, s service.AuthService) AuthHandler {
	return &Auth{
		jwtHandler: jwtHandler,
		group:      &singleflight.Group{},
		s:          s,
	}
}

// ExchangeIntegrationToken validates a trusted project's signed identity
// assertion and issues a feedback-service access token.
//
//	@Summary		项目身份换取反馈 Token
//	@Description	校验已登记项目的身份断言，签发绑定项目和学生身份的短期反馈访问 Token。
//	@Tags			Auth
//	@ID				integration-token-exchange
//	@Accept			json
//	@Produce		json
//	@Param			request	body		reqV1.ExchangeIntegrationTokenReq	true	"项目身份断言"
//	@Success		200		{object}	response.Response{data=respV1.ExchangeIntegrationTokenResp}
//	@Failure		400		{object}	response.Response
//	@Failure		401		{object}	response.Response
//	@Failure		500		{object}	response.Response
//	@Router			/api/v1/integrations/token/exchange [post]
func (o Auth) ExchangeIntegrationToken(c *gin.Context, req reqV1.ExchangeIntegrationTokenReq) (response.Response, error) {
	token, expiresIn, err := o.s.ExchangeIntegrationToken(req.ProjectID, req.KeyID, req.Assertion)
	if err != nil {
		return response.Response{}, err
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data: respV1.ExchangeIntegrationTokenResp{
			AccessToken: token,
			TokenType:   "Bearer",
			ExpiresIn:   expiresIn,
		},
	}, nil
}

// GetTableToken 获取表格访问令牌
//
//	@Summary		获取表格访问令牌
//	@Description	根据表格标识符生成JWT访问令牌，用于后续的表格数据操作。该令牌包含表格配置信息和访问权限。
//	@Tags			Auth
//	@ID				get-table-token
//	@Accept			json
//	@Produce		json
//	@Param			request	body		reqV1.GenerateTableTokenReq								true	"获取Token请求参数"
//	@Success		200		{object}	response.Response{data=respV1.GenerateTableTokenResp}	"成功返回 JWT 令牌"
//	@Failure		400		{object}	response.Response										"请求参数错误"
//	@Failure		500		{object}	response.Response										"服务器内部错误"
//	@Router			/api/v1/auth/table-config/token [post]
func (o Auth) GetTableToken(c *gin.Context, req reqV1.GenerateTableTokenReq) (response.Response, error) {
	if strings.HasSuffix(strings.TrimSpace(req.TableIdentify), "-faq") {
		return response.Response{}, errs.IntegrationScopeDeniedError(errors.New("FAQ table tokens must be issued through project integration"))
	}

	tableCfg, err := o.s.GetTableConfig(&req.TableIdentify)
	if err != nil {
		return response.Response{}, err
	}

	token, err := o.jwtHandler.SetJWTToken(*tableCfg.TableIdentity, *tableCfg.TableName, *tableCfg.TableToken, *tableCfg.TableID, *tableCfg.ViewID)
	if err != nil {
		return response.Response{}, errs.TokenGeneratedError(err)
	}

	c.Header("Authorization", token)

	resp := respV1.GenerateTableTokenResp{
		AccessToken: token,
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp,
	}, nil
}

// RefreshTableConfig 刷新表格配置信息
//
//	@Summary		刷新表格配置缓存
//	@Description	刷新并重新加载系统中所有支持的表格配置信息，返回当前可用的表格列表及其基本信息。通常用于配置更新后的缓存刷新。
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

// GetTenantToken 获取租户访问令牌
//
//	@Summary		获取租户访问令牌
//	@Description	获取飞书应用的租户访问令牌，主要用于文件上传、图片处理等需要应用级权限的操作。该令牌具有较高的访问权限。
//	@Tags			Auth
//	@ID				get-tenant-token
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	response.Response{data=respV1.GenerateTenantToken}	"成功返回 JWT 令牌"
//	@Failure		400	{object}	response.Response									"请求参数错误"
//	@Failure		500	{object}	response.Response									"服务器内部错误"
//	@Router			/api/v1/auth/tenant/token [post]
func (o Auth) GetTenantToken(c *gin.Context) (response.Response, error) {
	token := o.s.GetTenantToken()

	resp := respV1.GenerateTenantToken{
		AccessToken: token,
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp,
	}, nil
}
