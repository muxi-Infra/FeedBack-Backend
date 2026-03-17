package controller

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewSwag,
	NewAuth,
	NewSheet,
	NewSheetV2,
	NewMessage,
	NewAI,
)
