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
	CreateRecordEmptyContentCode                         // 新增表格记录反馈内容为空
	CreateRecordEmptyStudentIDCodeCode                   // 新增表格记录学号为空
	CreateRecordInvalidStudentIDCode                     // 新增表格记录学号不合法
	FAQResolutionChangeErrorCode                         // FAQ 解决状态更新失败
	FAQResolutionFindErrorCode                           // FAQ 解决状态查询失败
	FAQResolutionExistCode                               // FAQ 解决状态已存在
	FAQResolutionCountGetErrorCode                       // FAQ 解决状态计数获取失败
	FAQResolutionChangeLimitExceededCode                 // FAQ 解决状态修改次数达到上限
	SwagMakeFailureErrorCode                             // 生成 Swagger 文档失败
	SwagOpenFailureErrorCode                             // 打开 Swagger 文档失败
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
	CreateRecordEmptyContentError = func(err error) error {
		return errorx.New(http.StatusBadRequest, CreateRecordEmptyContentCode, "新增表格记录反馈内容为空", err)
	}
	CreateRecordEmptyStudentIDError = func(err error) error {
		return errorx.New(http.StatusBadRequest, CreateRecordEmptyStudentIDCodeCode, "新增表格记录学号为空", err)
	}
	CreateRecordInvalidStudentIDError = func(err error) error {
		return errorx.New(http.StatusBadRequest, CreateRecordInvalidStudentIDCode, "新增表格记录学号不合法", err)
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
	SwagMakeFailureError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, SwagMakeFailureErrorCode, "生成 Swagger 文档失败", err)
	}
	SwagOpenFailureError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, SwagOpenFailureErrorCode, "打开 Swagger 文档失败", err)
	}
)
