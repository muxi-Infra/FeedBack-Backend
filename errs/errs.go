package errs

import (
	"net/http"

	"github.com/muxi-Infra/FeedBack-Backend/pkg/errorx"
)

// 系统错误
const (
	// 系统内部错误（通用）
	internalServerErrorCode = iota + 100001

	// 序列化错误
	serializationErrorCode

	// 反序列化错误
	deserializationErrorCode

	// 队列操作错误
	queueOperationErrorCode

	// 读取响应错误
	readResponseErrorCode
)

// 业务错误
const (
	// 记录不存在
	recordNotFoundErrorCode = iota + 200001
)

// 第三方错误
const (
	// 飞书请求错误
	feishuRequestErrorCode = iota + 300001

	// 飞书服务返回错误
	feishuResponseErrorCode
)

var (
	InternalServerError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, internalServerErrorCode, "服务器内部错误", err)
	}
	SerializationError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, serializationErrorCode, "序列化错误", err)
	}
	DeserializationError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, deserializationErrorCode, "反序列化错误", err)
	}
	QueueOperationError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, queueOperationErrorCode, "队列任务操作错误", err)
	}
	ReadResponseError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, readResponseErrorCode, "读取响应流错误", err)
	}
)

var (
	RecordNotFoundError = func(err error) error {
		return errorx.New(http.StatusNotFound, recordNotFoundErrorCode, "记录不存在", err)
	}
)

var (
	FeishuRequestError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, feishuRequestErrorCode, "飞书请求接口失败", err)
	}
	FeishuResponseError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, feishuResponseErrorCode, "飞书服务返回错误", err)
	}
)
