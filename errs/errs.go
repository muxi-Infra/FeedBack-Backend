package errs

import (
	"net/http"

	"github.com/muxi-Infra/FeedBack-Backend/pkg/errorx"
)

const (
	userIdOrPasswordErrorCode = iota + 40001
	unauthorizedErrorCode
)

const (
	internalServerErrorCode = iota + 50001
	crawlerServerErrorCode
	grabSeatErrorCode
	getHistoryErrorCode
	createClientErrorCode
)

var (
	UserIdOrPasswordError = func(err error) error {
		return errorx.New(http.StatusUnauthorized, userIdOrPasswordErrorCode, "账号或者密码错误!", err)
	}
	UnauthorizedError = func(err error) error {
		return errorx.New(http.StatusUnauthorized, unauthorizedErrorCode, "Authorization错误，请重新登录", err)
	}
)

var (
	InternalServerError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, internalServerErrorCode, "服务器内部错误", err)
	}
	CrawlerServerError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, crawlerServerErrorCode, "爬虫服务器错误", err)
	}
	GrabSeatError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, grabSeatErrorCode, "抢座失败，请稍后重试", err)
	}
	GetHistoryError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, getHistoryErrorCode, "获取历史记录失败", err)
	}
	CreateClientError = func(err error) error {
		return errorx.New(http.StatusInternalServerError, createClientErrorCode, "创建HTTP客户端失败", err)
	}
)
