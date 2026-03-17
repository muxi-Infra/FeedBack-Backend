//go:build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/muxi-Infra/FeedBack-Backend/ai"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/controller"
	"github.com/muxi-Infra/FeedBack-Backend/ioc"
	"github.com/muxi-Infra/FeedBack-Backend/middleware"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/lark"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/repository"
	"github.com/muxi-Infra/FeedBack-Backend/service"
	"github.com/muxi-Infra/FeedBack-Backend/web"
)

func InitApp() (*App, error) {
	wire.Build(
		wire.Struct(new(App), "*"),
		config.ProviderSet,
		ioc.ProviderSet,
		logger.NewZapLogger,
		lark.ProviderSet,
		ijwt.NewJWT,
		repository.ProviderSet,
		service.ProviderSet,
		middleware.NewCorsMiddleware,
		middleware.NewAuthMiddleware,
		middleware.NewBasicAuthMiddleware,
		middleware.NewLoggerMiddleware,
		middleware.NewPrometheusMiddleware,
		middleware.NewLimitMiddleware,
		controller.ProviderSet,
		ai.ProviderSet,
		web.NewGinEngine,
	)
	return &App{}, nil
}
