package config

import (
	"github.com/google/wire"
	"github.com/spf13/viper"
	"log"
)

var ProviderSet = wire.NewSet(
	NewClientConfig,
	NewJWTConfig,
	NewMiddlewareConfig,
	NewAppTable,
)

type ClientConfig struct {
	AppID     string `yaml:"appID"` // 应用凭证
	AppSecret string `yaml:"appSecret"`
}

func NewClientConfig() ClientConfig {
	return ClientConfig{
		AppID:     viper.GetString("client.appID"),
		AppSecret: viper.GetString("client.appSecret"),
	}
}

type JWTConfig struct {
	SecretKey string `yaml:"secretKey"` //秘钥
	Timeout   int    `yaml:"timeout"`   //过期时间
}

func NewJWTConfig() JWTConfig {
	return JWTConfig{
		SecretKey: viper.GetString("jwt.secretKey"),
		Timeout:   viper.GetInt("jwt.timeout"),
	}
}

type MiddlewareConfig struct {
	AllowedOrigins []string `yaml:"allowedOrigins"`
}

func NewMiddlewareConfig() *MiddlewareConfig {
	return &MiddlewareConfig{
		AllowedOrigins: viper.GetStringSlice("middleware.allowedOrigins"),
	}
}

// viper 默认默认使用的是 mapstructure 标签 来映射字段
// app_table
type AppTable struct {
	AppToken string           `yaml:"app_token" mapstructure:"app_token"`
	Tables   map[string]Table `yaml:"tables" mapstructure:"tables"` // 使用map方便获取
}

type Table struct {
	Name    string `yaml:"name" mapstructure:"name"`
	TableID string `yaml:"table_id" mapstructure:"table_id"`
	ViewID  string `yaml:"view_id" mapstructure:"view_id"`
}

func NewAppTable() *AppTable {
	var appTable AppTable
	// 反序列化
	if err := viper.Sub("app_table").Unmarshal(&appTable); err != nil {
		log.Fatalf("unmarshal app_table failed: %v", err)
	}
	//fmt.Println(appTable)
	return &appTable
}

func (t *AppTable) IsValidTableID(tableID string) bool {
	_, ok := t.Tables[tableID]
	return ok
}
