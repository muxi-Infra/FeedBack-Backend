//go:build wireinject

package main

import (
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/ioc"
	"github.com/muxi-Infra/FeedBack-Backend/middleware"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/feishu"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/service"
	"github.com/muxi-Infra/FeedBack-Backend/web"

	"github.com/google/wire"
)

func InitApp() (*App, error) {
	wire.Build(
		wire.Struct(new(App), "*"),
		config.ProviderSet,
		ioc.ProviderSet,
		logger.NewZapLogger,
		feishu.ProviderSet,
		ijwt.NewJWT,
		service.ProviderSet,
		middleware.NewCorsMiddleware,
		middleware.NewAuthMiddleware,
		middleware.NewBasicAuthMiddleware,
		middleware.NewLoggerMiddleware,
		middleware.NewPrometheusMiddleware,
		middleware.NewLimitMiddleware,
		web.ProviderSet,
		dao.ProviderSet,
	)
	return &App{}, nil
}
