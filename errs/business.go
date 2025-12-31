package errs

import (
	"net/http"

	"github.com/muxi-Infra/FeedBack-Backend/pkg/errorx"
)

/*
20xxxx - 业务错误
*/

const (
	TokenGeneratedErrorCode    = 200000 + iota // Token 生成失败
	TableIdentifyNotFoundCode                  // 表格标识未找到
	TableIdentifierInvalidCode                 // 表格标识无效
	RecordNotFoundCode                         // 记录不存在
	AddPendingLikeTaskCode                     // 添加待处理点赞任务失败
)

var (
	TokenGeneratedError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, TokenGeneratedErrorCode, "Token 生成失败", err)
	}
	TableIdentifyNotFoundError = func(err error) error {
		return errorx.New(http.StatusNotFound, TableIdentifyNotFoundCode, "表格标识未找到", err)
	}
	TableIdentifierInvalidError = func(err error) error {
		return errorx.New(http.StatusBadRequest, TableIdentifierInvalidCode, "表格标识无效", err)
	}
	RecordNotFoundError = func(err error) error {
		return errorx.New(http.StatusNotFound, RecordNotFoundCode, "记录不存在", err)
	}
	AddPendingLikeTaskError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, AddPendingLikeTaskCode, "添加待处理点赞任务失败", err)
	}
)
