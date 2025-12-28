package ioc

import (
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(InitRedis, InitLogger, InitPrometheus, InitClient)
