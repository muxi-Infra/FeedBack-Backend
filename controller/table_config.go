package controller

import (
	"errors"
	"fmt"

	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
	"github.com/muxi-Infra/FeedBack-Backend/service"
)

// tableConfigFromClaims 根据令牌中的项目和表标识，从反馈配置缓存中获取真实飞书配置。
// 飞书凭证不再放入用户 JWT。
func tableConfigFromClaims(auth service.AuthService, claims ijwt.UserClaims) (domain.TableConfig, error) {
	if auth == nil {
		return domain.TableConfig{}, errors.New("auth service is not configured")
	}
	return auth.GetTableConfig(claims.ProjectID, claims.TableIdentity)
}

// tableConfigWithScope 从服务端配置缓存读取表格配置并校验当前接口权限。
func tableConfigWithScope(auth service.AuthService, claims ijwt.UserClaims, scope string) (domain.TableConfig, error) {
	tableConfig, err := tableConfigFromClaims(auth, claims)
	if err != nil {
		return domain.TableConfig{}, err
	}
	if !tableConfig.HasScope(scope) {
		return domain.TableConfig{}, errs.IntegrationScopeDeniedError(
			fmt.Errorf("%s scope is required", scope),
		)
	}
	return tableConfig, nil
}
