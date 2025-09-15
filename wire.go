//go:build wireinject

package main

import (
	"feedback/config"
	"feedback/middleware"
	"feedback/pkg/feishu"
	"feedback/pkg/ijwt"
	log "feedback/pkg/logger"
	"feedback/repository/dao"
	"feedback/service"
	"feedback/web"
	"github.com/google/wire"
)

func InitApp() (*App, error) {
	wire.Build(
		wire.Struct(new(App), "*"),
		config.ProviderSet,
		log.ProviderSet,
		feishu.ProviderSet,
		ijwt.NewJWT,
		service.ProviderSet,
		middleware.NewCorsMiddleware,
		middleware.NewAuthMiddleware,
		web.ProviderSet,
		dao.ProviderSet,
	)
	return &App{}, nil
}
