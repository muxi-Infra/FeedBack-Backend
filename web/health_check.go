package web

import (
	"feedback/controller"
	"feedback/pkg/ginx"

	"github.com/gin-gonic/gin"
)

func RegisterHealthCheckHandler(r *gin.Engine) {
	r.GET("/health", ginx.Wrap(controller.HealthCheck))
}
