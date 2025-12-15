package web

import (
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"

	"github.com/gin-gonic/gin"
)

func RegisterHealthCheckHandler(r *gin.Engine) {
	r.GET("/health", ginx.Wrap(controller.HealthCheck))
}
