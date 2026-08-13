package ioc

import (
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(InitMysql, InitRedis, InitLogger, InitPrometheus, InitClient)
