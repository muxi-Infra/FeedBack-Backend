package ioc

import (
	lark "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/muxi-Infra/FeedBack-Backend/config"
)

func InitClient(conf config.ClientConfig) *lark.Client {
	return lark.NewClient(conf.AppID, conf.AppSecret)
}
