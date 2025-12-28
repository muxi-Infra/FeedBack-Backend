package errs

import (
	"net/http"

	"github.com/muxi-Infra/FeedBack-Backend/pkg/errorx"
)

/*
20xxxx - 业务错误
*/

const (
	RecordNotFoundCode      = 200000 + iota // 记录不存在
	AddPendingLikeTaskCode                  // 添加待处理点赞任务失败
	TableIDInvalidCode                      // 表格 ID 无效
	TokenGeneratedErrorCode                 // Token 生成失败
)

var (
	RecordNotFoundError = func(err error) error {
		return errorx.New(http.StatusNotFound, RecordNotFoundCode, "记录不存在", err)
	}
	AddPendingLikeTaskError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, AddPendingLikeTaskCode, "添加待处理点赞任务失败", err)
	}
	TableIDInvalidError = func(err error) error {
		return errorx.New(http.StatusBadRequest, TableIDInvalidCode, "表格 ID 无效", err)
	}
	TableIDNotFoundError = func(err error) error {
		return errorx.New(http.StatusNotFound, TableIDInvalidCode, "表格 ID 未找到", err)
	}
	TokenGeneratedError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, TokenGeneratedErrorCode, "Token 生成失败", err)
	}
)
