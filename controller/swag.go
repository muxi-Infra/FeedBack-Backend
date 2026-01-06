package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/service"
)

type Swag struct {
	s service.SwagService
}

func NewSwag(s service.SwagService) *Swag {
	return &Swag{
		s: s,
	}
}

// GetOpenApi3 获取/重新生成 OpenAPI3 接口文档
//
//	@Summary		获取 OpenAPI3 接口文档 (YAML)
//	@Description	接口直接返回 docs/openapi3.yaml yaml格式的原始内容，使用BasicAuth进行验证
//	@Tags			Swag
//	@ID				get-openapi3
//	@Produce		application/x-yaml
//	@Security		BasicAuth
//	@Success		200	{string}	string				"成功返回 OpenAPI3 文档内容"
//	@Failure		401	{object}	response.Response	"未授权，BasicAuth 验证失败"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/api/v1/openapi [get]
func (s *Swag) GetOpenApi3(c *gin.Context) (response.Response, error) {
	content, err := s.s.GenerateOpenAPI()
	if err != nil {
		return response.Response{}, err
	}

	// 返回 YAML 字符串
	c.String(200, string(content))
	// 为保证返回的文件纯净性，不打印通用响应体
	c.Abort()
	return response.Response{}, nil
}
