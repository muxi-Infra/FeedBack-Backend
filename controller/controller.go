package controller

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewSwag,
	NewAuth,
	NewSheet,
	NewSheetV2,
	NewMessage,
	NewV3Auth,
	NewV3Sheet,
	NewV3Admin,
	NewV3Sync,
	NewAdminAuthV3,
)
