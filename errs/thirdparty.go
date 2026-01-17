package errs

import (
	"net/http"

	"github.com/muxi-Infra/FeedBack-Backend/pkg/errorx"
)

/*
30xxxx - 第三方服务错误
*/

const (
	LarkRequestErrorCode  = 300000 + iota // 飞书请求接口失败
	LarkResponseErrorCode                 // 飞书服务返回错误
)

var (
	LarkRequestError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, LarkRequestErrorCode, "飞书请求接口失败", err)
	}
	LarkResponseError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, LarkResponseErrorCode, "飞书服务返回错误", err)
	}
)
