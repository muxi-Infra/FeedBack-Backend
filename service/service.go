package service

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewOauth,
	NewLikeService,
	NewSheetService,
	wire.Bind(new(AuthService), new(*AuthServiceImpl)),
	wire.Bind(new(AuthTokenProvider), new(*AuthServiceImpl)),
)
