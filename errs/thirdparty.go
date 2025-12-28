package errs

import (
	"net/http"

	"github.com/muxi-Infra/FeedBack-Backend/pkg/errorx"
)

/*
30xxxx - 第三方服务错误
*/

const (
	FeishuRequestErrorCode           = 300000 + iota // 飞书请求接口失败
	FeishuResponseErrorCode                          // 飞书服务返回错误
	FeishuOauthInvalidCode                           // 飞书 OAuth 无效
	FeishuAuthorizationDeniedCode                    // 飞书授权被拒绝
	FeishuOauthConfigChangeErrorCode                 // 飞书 OAuth 配置变更错误
	ReadResponseErrorCode                            // 读取响应流错误
)

var (
	FeishuRequestError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, FeishuRequestErrorCode, "飞书请求接口失败", err)
	}
	FeishuResponseError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, FeishuResponseErrorCode, "飞书服务返回错误", err)
	}
	FeishuOauthInvalidError = func(err error) error {
		return errorx.New(http.StatusUnauthorized, FeishuOauthInvalidCode, "飞书 OAuth 无效", err)
	}
	FeishuAuthorizationDeniedError = func(err error) error {
		return errorx.New(http.StatusUnauthorized, FeishuAuthorizationDeniedCode, "飞书授权被拒绝", err)
	}
	FeishuOauthConfigChangeError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, FeishuOauthConfigChangeErrorCode, "飞书 OAuth 配置变更错误", err)
	}
	ReadResponseError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, ReadResponseErrorCode, "读取响应流错误", err)
	}
)
