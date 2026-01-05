package errs

import (
	"net/http"

	"github.com/muxi-Infra/FeedBack-Backend/pkg/errorx"
)

/*
20xxxx - 业务错误
*/

const (
	TokenGeneratedErrorCode              = 200000 + iota // Token 生成失败
	TableIdentifyNotFoundCode                            // 表格标识未找到
	TableIdentifierInvalidCode                           // 表格标识无效
	FAQResolutionChangeErrorCode                         // FAQ 解决状态更新失败
	FAQResolutionFindErrorCode                           // FAQ 解决状态查询失败
	FAQResolutionExistCode                               // FAQ 解决状态已存在
	FAQResolutionCountGetErrorCode                       // FAQ 解决状态计数获取失败
	FAQResolutionChangeLimitExceededCode                 // FAQ 解决状态修改次数达到上限
)

var (
	TokenGeneratedError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, TokenGeneratedErrorCode, "Token 生成失败", err)
	}
	TableIdentifyNotFoundError = func(err error) error {
		return errorx.New(http.StatusNotFound, TableIdentifyNotFoundCode, "表格标识未找到", err)
	}
	TableIdentifierInvalidError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, TableIdentifierInvalidCode, "表格标识无效", err)
	}
	FAQResolutionChangeError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, FAQResolutionChangeErrorCode, "FAQ 解决状态更新失败", err)
	}
	FAQResolutionFindError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, FAQResolutionFindErrorCode, "FAQ 解决状态查询失败", err)
	}
	FAQResolutionExistError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, FAQResolutionExistCode, "FAQ 解决状态已存在", err)
	}
	FAQResolutionCountGetError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, FAQResolutionCountGetErrorCode, "FAQ 解决状态计数获取失败", err)
	}
	FAQResolutionChangeLimitExceededError = func(err error) error {
		return errorx.New(http.StatusTooManyRequests, FAQResolutionChangeLimitExceededCode, "FAQ 解决状态修改次数达到上限", err)
	}
)
