package ginx

import (
	"errors"
	"feedback/pkg/ijwt"
	"github.com/gin-gonic/gin"
	"net/http"
)

const CTX = "claims"

func WrapClaimsAndReq[Req any, Resp any](fn func(*gin.Context, Req, ijwt.UserClaims) (Resp, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if len(ctx.Errors) > 0 {
			return
		}
		var req Req
		if err := ctx.Bind(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, err.Error())
			// TODO Logger
			return
		}

		claims, err := GetClaims(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, err.Error())
			// TODO Logger
			return
		}

		res, err := fn(ctx, req, claims)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, res)
			// TODO Logger
			return
		}
		ctx.JSON(http.StatusOK, res)
	}
}

// WrapReq .
func WrapReq[Req any, Resp any](fn func(*gin.Context, Req) (Resp, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if len(ctx.Errors) > 0 {
			return
		}
		var req Req
		if err := ctx.Bind(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, err.Error())
			return
		}

		res, err := fn(ctx, req)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, res)
			return
		}
		ctx.JSON(http.StatusOK, res)
	}
}

// Wrap .
func Wrap[Resp any](fn func(*gin.Context) (Resp, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if len(ctx.Errors) > 0 {
			return
		}
		res, err := fn(ctx)
		if err != nil {
			// TODO Logger
			return
		}
		ctx.JSON(http.StatusOK, res)
	}
}

// WrapClaims .
func WrapClaims[Resp any](fn func(*gin.Context, ijwt.UserClaims) (Resp, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if len(ctx.Errors) > 0 {
			return
		}
		claims, err := GetClaims(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, err.Error())
			// TODO Logger
			return
		}

		res, err := fn(ctx, claims)
		if err != nil {
			// TODO Logger
			return
		}
		ctx.JSON(http.StatusOK, res)
	}
}

func SetClaims(ctx *gin.Context, claims ijwt.UserClaims) {
	ctx.Set(CTX, claims)
}

func GetClaims(ctx *gin.Context) (ijwt.UserClaims, error) {
	val, ok := ctx.Get(CTX)
	if !ok {
		ctx.Error(errors.New("claims 不存在"))
		return ijwt.UserClaims{}, errors.New("claims 不存在")
	}
	claims, ok := val.(ijwt.UserClaims)
	if !ok {
		ctx.Error(errors.New("claims 断言失败"))
		return ijwt.UserClaims{}, errors.New("claims 断言失败")
	}
	return claims, nil
}
