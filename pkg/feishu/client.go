package feishu

import (
	"feedback/config"
	"github.com/google/wire"
	lark "github.com/larksuite/oapi-sdk-go/v3"
)

var ProviderSet = wire.NewSet(NewClient)

func NewClient(conf config.ClientConfig) *lark.Client {
	return lark.NewClient(conf.AppID, conf.AppSecret)
}
