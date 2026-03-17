package ioc

import (
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(InitChatModel, InitLocalEmbedder, InitMysql, InitRedis, InitLogger, InitPrometheus, InitClient, InitESClient)
