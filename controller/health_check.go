package controller

import (
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/service"

	"github.com/gin-gonic/gin"
	respV1 "github.com/muxi-Infra/FeedBack-Backend/api/response/v1"
)

// HealthCheck 系统健康检查
//
//	@Summary		系统健康检查
//	@Description	检查系统运行状态，返回服务健康信息、资源使用情况等监控数据。用于负载均衡器健康检查和系统监控。
//	@Tags			Health
//	@ID				health-check
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	response.Response{data=respV1.HealthCheckResponse}	"系统运行正常，返回健康检查详情"
//	@Failure		500	{object}	response.Response{data=string}						"系统异常或部分服务不可用"
//	@Router			/api/v1/health [get]
func HealthCheck(c *gin.Context) (response.Response, error) {
	serviceResult := service.HealthCheck(c)

	resp := respV1.HealthCheckResponse{
		Status:     serviceResult.Status,
		ResponseMs: serviceResult.ResponseMs,
		System:     serviceResult.System,
		Process:    serviceResult.Process,
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp,
	}, nil
}
