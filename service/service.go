package service

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewAuthServiceImpl,
	NewSheetService,
	wire.Bind(new(AuthService), new(*AuthServiceImpl)),
	wire.Bind(new(TableConfigProvider), new(*AuthServiceImpl)),
)
