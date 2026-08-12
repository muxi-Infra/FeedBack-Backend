package ginx

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/constvar"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/errorx"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
)

// WrapV3Req 绑定 V3 请求体并统一处理业务错误。
func WrapV3Req[Req any](fn func(*gin.Context, Req) (response.Response, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req Req
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, response.Response{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("请求参数错误: %v", err),
			})
			return
		}
		res, err := fn(c, req)
		if err != nil {
			writeV3Error(c, err)
			return
		}
		c.JSON(http.StatusOK, res)
	}
}

func WrapV3Claims[Req any](fn func(*gin.Context, Req, string, string) (response.Response, error)) gin.HandlerFunc {
	return WrapV3Req(func(c *gin.Context, req Req) (response.Response, error) {
		value, ok := c.Get(constvar.V3ClaimsContextKey)
		if !ok {
			return response.Response{}, fmt.Errorf("V3 claims 不存在")
		}
		claims, ok := value.(ijwt.V3UserClaims)
		if !ok {
			return response.Response{}, fmt.Errorf("V3 claims 类型错误")
		}
		return fn(c, req, claims.ProjectID, claims.StudentID)
	})
}

// WrapV3ClaimsNoReq 读取 V3 用户 Claims，并调用不需要请求参数的处理函数。
func WrapV3ClaimsNoReq(fn func(*gin.Context, string, string) (response.Response, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, ok := c.Get(constvar.V3ClaimsContextKey)
		if !ok {
			writeV3Error(c, fmt.Errorf("V3 claims 不存在"))
			return
		}
		claims, ok := value.(ijwt.V3UserClaims)
		if !ok {
			writeV3Error(c, fmt.Errorf("V3 claims 类型错误"))
			return
		}
		res, err := fn(c, claims.ProjectID, claims.StudentID)
		if err != nil {
			writeV3Error(c, err)
			return
		}
		c.JSON(http.StatusOK, res)
	}
}

func writeV3Error(c *gin.Context, err error) {
	custom := errorx.ToCustomError(err)
	c.Error(err)
	c.JSON(custom.HttpCode, response.Response{
		Code:    custom.Code,
		Message: custom.Msg,
	})
}
