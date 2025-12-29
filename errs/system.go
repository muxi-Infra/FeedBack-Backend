package errs

import (
	"net/http"

	"github.com/muxi-Infra/FeedBack-Backend/pkg/errorx"
)

/*
10xxxx - 系统 / 基础设施错误
*/

const (
	InternalServerErrorCode  = 100000 + iota // 内部服务器错误
	SerializationErrorCode                   // 序列化错误
	DeserializationErrorCode                 // 反序列化错误 	// 请求失败错误
	QueueOperationErrorCode                  // 队列任务操作错误
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
	QueueOperationError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, QueueOperationErrorCode, "队列任务操作错误", err)
	}
)
