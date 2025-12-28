package controller

import (
	"fmt"
	"github.com/muxi-Infra/FeedBack-Backend/api/request"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/singleflight"
)

type Oauth struct {
	jwtHandler *ijwt.JWT
	group      *singleflight.Group
	tableCfg   *config.AppTable
}

func NewOauth(jwtHandler *ijwt.JWT, tableCfg *config.AppTable) *Oauth {
	return &Oauth{
		jwtHandler: jwtHandler,
		group:      &singleflight.Group{},
		tableCfg:   tableCfg,
	}
}

// GetToken godoc
//
//	@Summary		获取 token 接口
//	@Description	获取 token 接口
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	response.Response	"成功返回 JWT 令牌"
//	@Failure		400	{object}	response.Response	"请求参数错误"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/get_token [post]
func (o Oauth) GetToken(c *gin.Context, req request.GenerateTokenReq) (response.Response, error) {
	if !o.tableCfg.IsValidTableIdentity(req.TableIdentity) {
		return response.Response{},
			errs.TableIDInvalidError(fmt.Errorf("无效的表Identity"))
	}

	token, err := o.jwtHandler.SetJWTToken(req.TableIdentity)
	if err != nil {
		return response.Response{}, errs.TokenGeneratedError(err)
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data: map[string]string{
			"access_token": token,
		},
	}, nil
}
