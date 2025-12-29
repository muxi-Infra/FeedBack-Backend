package errs

import (
	"net/http"

	"github.com/muxi-Infra/FeedBack-Backend/pkg/errorx"
)

/*
30xxxx - 第三方服务错误
*/

const (
	FeishuRequestErrorCode  = 300000 + iota // 飞书请求接口失败
	FeishuResponseErrorCode                 // 飞书服务返回错误 	// 读取响应流错误
)

var (
	FeishuRequestError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, FeishuRequestErrorCode, "飞书请求接口失败", err)
	}
	FeishuResponseError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, FeishuResponseErrorCode, "飞书服务返回错误", err)
	}
)
