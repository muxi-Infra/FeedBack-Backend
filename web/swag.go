package web

import (
	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	_ "github.com/muxi-Infra/FeedBack-Backend/docs" // 生成的 swagger 文档
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ginx"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type SwagHandler interface {
	GetOpenApi3(c *gin.Context) (response.Response, error)
}

// RegisterSwagHandler 注册 Swagger 文档路由，使用 Basic Auth 保护
func RegisterSwagHandler(r *gin.RouterGroup, sh SwagHandler, basicAuthMiddleware gin.HandlerFunc) {
	r.GET("/openapi", basicAuthMiddleware, ginx.Wrap(sh.GetOpenApi3))
	r.GET("/swagger/*any", basicAuthMiddleware, ginSwagger.WrapHandler(swaggerFiles.Handler))
}
