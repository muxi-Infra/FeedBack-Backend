package ioc

import (
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(InitChatModel, InitLocalEmbedder, InitNLI, InitMysql, InitRedis, InitLogger, InitPrometheus, InitClient, InitESClient)
