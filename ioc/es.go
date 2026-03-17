package ioc

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/muxi-Infra/FeedBack-Backend/config"
)

func InitESClient(cfg *config.ESConfig) (*elasticsearch.Client, error) {
	esCfg := elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
		CloudID:   cfg.CloudID,
		APIKey:    cfg.APIKey,
		// 配置传输层，处理长连接和 TLS
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: 10 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // 跳过证书验证
			},
		},
	}

	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, err
	}

	// 尝试 Ping 一下确认连接正常
	_, err = client.Info()
	if err != nil {
		return nil, err
	}

	return client, nil
}
