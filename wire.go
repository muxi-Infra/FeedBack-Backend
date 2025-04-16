//go:build wireinject

package main

import (
	"feedback/config"
	"feedback/middleware"
	"feedback/pkg/feishu"
	"feedback/pkg/ijwt"
	log "feedback/pkg/logger"
	"feedback/web"
	"github.com/google/wire"
)

func InitApp() *App {
	wire.Build(
		wire.Struct(new(App), "*"),
		config.ProviderSet,
		log.ProviderSet,
		feishu.ProviderSet,
		ijwt.NewJWT,
		middleware.NewCorsMiddleware,
		middleware.NewAuthMiddleware,
		web.ProviderSet,
	)
	return &App{}
}
