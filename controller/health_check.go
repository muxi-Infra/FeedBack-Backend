package controller

import (
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/service"

	"github.com/gin-gonic/gin"
)

// HealthCheck 健康检查接口
//
//	@Summary		健康检查，返回当前服务占用的资源等信息
//	@Description	健康检查，返回当前服务占用的资源等信息
//	@ID				health-check
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	response.Response{data=response.HealthCheckResponse}	"成功返回健康检查结果"
//	@Failure		500	{object}	response.Response{data=string}							"服务器内部错误"
//	@Router			/api/v1/health [get]
func HealthCheck(c *gin.Context) (response.Response, error) {
	resp := service.HealthCheck()

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp,
	}, nil
}
