package ioc

import (
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	InitMysql,
	InitCasbinV3,
	InitRedis,
	InitLogger,
	InitPrometheus,
	InitClient,
)
