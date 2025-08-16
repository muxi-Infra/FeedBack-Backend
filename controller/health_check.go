package controller

import (
	"feedback/api/response"
	"feedback/service"
	"github.com/gin-gonic/gin"
)

// HealthCheck 健康检查接口
//
//	@Summary		健康检查，返回当前服务占用的资源等信息
//	@Description	健康检查，返回当前服务占用的资源等信息
//	@ID				health-check
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	response.Response	"成功返回复制结果"
//	@Failure		400	{object}	response.Response	"请求参数错误或飞书接口调用失败"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/sheet/copyapp [post]
func HealthCheck(c *gin.Context) (response.Response, error) {
	resp := service.HealthCheck()

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp,
	}, nil
}
