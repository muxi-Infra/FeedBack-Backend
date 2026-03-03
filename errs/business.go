package errs

import (
	"net/http"

	"github.com/muxi-Infra/FeedBack-Backend/pkg/errorx"
)

/*
20xxxx - 业务错误
*/

const (
	TokenGeneratedErrorCode                 = 200000 + iota // Token 生成失败
	TableIdentifyNotFoundCode                               // 表格标识未找到
	TableIdentifierInvalidCode                              // 表格标识无效
	CreateRecordEmptyContentCode                            // 新增表格记录反馈内容为空
	CreateRecordEmptyStudentIDCode                          // 新增表格记录学号为空
	CreateRecordInvalidStudentIDCode                        // 新增表格记录学号不合法
	FAQResolutionChangeErrorCode                            // FAQ 解决状态更新失败
	FAQResolutionFindErrorCode                              // FAQ 解决状态查询失败
	FAQResolutionExistCode                                  // FAQ 解决状态已存在
	FAQResolutionCountGetErrorCode                          // FAQ 解决状态计数获取失败
	FAQResolutionChangeLimitExceededCode                    // FAQ 解决状态修改次数达到上限
	FileTokenInvalidErrorCode                               // 文件 Token 无效
	LarkMessagePartialFailureCode                           // 飞书消息部分发送失败
	AppNotificationChannelFullErrorCode                     // 应用通知通道已满
	TableNotificationNotConfiguredErrorCode                 // 表格通知未配置错误
	TableRecordNotFoundCode                                 // 表格记录未找到
	PageTokenInvalidCode                                    // 分页 Token 无效
	APIDeprecatedCode                                       // API 已废弃
	CreateRecordDBErrorCode                                 // 创建表格记录数据库错误
	UpdateRecordDBErrorCode                                 // 更新表格记录数据库错误
	GetUnsyncedRecordsByTableErrorCode                      // 根据表格标识获取未同步记录错误
	CountSheetRecordByUserErrorCode                         // 根据用户统计表格记录错误
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
		return errorx.New(http.StatusBadRequest, CreateRecordEmptyStudentIDCode, "新增表格记录学号为空", err)
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
	FileTokenInvalidError = func(err error) error {
		return errorx.New(http.StatusBadRequest, FileTokenInvalidErrorCode, "文件 Token 无效", err)
	}
	LarkMessagePartialFailureError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, LarkMessagePartialFailureCode, "飞书消息部分发送失败", err)
	}
	AppNotificationChannelFullError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, AppNotificationChannelFullErrorCode, "应用通知通道已满", err)
	}
	TableNotificationNotConfiguredError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, TableNotificationNotConfiguredErrorCode, "表格通知未配置错误", err)
	}
	TableRecordNotFoundError = func(err error) error {
		return errorx.New(http.StatusNotFound, TableRecordNotFoundCode, "表格记录未找到", err)
	}
	PageTokenInvalidError = func(err error) error {
		return errorx.New(http.StatusBadRequest, PageTokenInvalidCode, "分页 Token 无效", err)
	}
	APIDeprecatedError = func(err error) error {
		return errorx.New(http.StatusGone, APIDeprecatedCode, "API 已废弃", err)
	}
	CreateRecordDBError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, CreateRecordDBErrorCode, "创建表格记录数据库错误", err)
	}
	UpdateRecordDBError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, UpdateRecordDBErrorCode, "更新表格记录数据库错误", err)
	}
	GetUnsyncedRecordsByTableError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, GetUnsyncedRecordsByTableErrorCode, "根据表格标识获取未同步记录错误", err)
	}
	CountSheetRecordByUserError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, CountSheetRecordByUserErrorCode, "根据用户统计表格记录错误", err)
	}
)
