package errs

import (
	"net/http"

	"github.com/muxi-Infra/FeedBack-Backend/pkg/errorx"
)

/*
10xxxx - 系统 / 基础设施错误
*/

const (
	InternalServerErrorCode  = 100000 + iota // 服务内部错误
	SerializationErrorCode                   // 序列化错误
	DeserializationErrorCode                 // 反序列化错误
)

var (
	InternalServerError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, InternalServerErrorCode, "服务内部错误", err)
	}
	SerializationError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, SerializationErrorCode, "序列化错误", err)
	}
	DeserializationError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, DeserializationErrorCode, "反序列化错误", err)
	}
)
