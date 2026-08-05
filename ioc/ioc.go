package ioc

import (
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	InitMysql,
	InitCasbin,
	InitRedis,
	InitLogger,
	InitPrometheus,
	InitClient,
)
