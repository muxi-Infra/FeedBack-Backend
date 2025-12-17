package errs

import (
	"net/http"

	"github.com/muxi-Infra/FeedBack-Backend/pkg/errorx"
)

const (
	recordNotFoundErrorCode = iota + 40001
)

const (
	internalServerErrorCode = iota + 50001
	serializationErrorCode
	deserializationErrorCode
	queueOperationErrorCode
	feishuRequestErrorCode
	feishuResponseErrorCode
	readResponseErrorCode
)

var (
	RecordNotFoundError = func(err error) error {
		return errorx.New(http.StatusNotFound, recordNotFoundErrorCode, "记录不存在", err)
	}
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
	FeishuRequestError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, feishuRequestErrorCode, "飞书请求接口失败", err)
	}
	FeishuResponseError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, feishuResponseErrorCode, "飞书服务返回错误", err)
	}
	ReadResponseError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, readResponseErrorCode, "读取响应流错误", err)
	}
)
