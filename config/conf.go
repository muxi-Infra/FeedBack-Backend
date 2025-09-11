package config

import (
	"fmt"
	"github.com/google/wire"
	"github.com/spf13/viper"

	"log"
)

var ProviderSet = wire.NewSet(
	NewClientConfig,
	NewJWTConfig,
	NewMiddlewareConfig,
	NewAppTable,
	NewBatchNoticeConfig,
	NewRedisConfig,
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
	return &appTable
}

func (t *AppTable) IsValidTableID(tableID string) bool {
	_, ok := t.Tables[tableID]
	return ok
}

// 批量发送消息的配置
type BatchNoticeConfig struct {
	OpenIDs []OpenID `mapstructure:"open_ids" yaml:"open_ids" json:"open_ids"`
	ChatIDs []ChatID `mapstructure:"chat_ids" yaml:"chat_ids" json:"chat_ids"`
	Content Content  `mapstructure:"content" yaml:"content" json:"content"`
}

func NewBatchNoticeConfig() *BatchNoticeConfig {
	var cfg BatchNoticeConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("unmarshal batch_notice_config failed: %v", err)
	}
	return &cfg
}

// 发送消息的内容
type Content struct {
	Type string `mapstructure:"type" yaml:"type" json:"type"`
	Data Data   `mapstructure:"data" yaml:"data" json:"data"`
}

type Data struct {
	TemplateID          string           `mapstructure:"template_id" yaml:"template_id" json:"template_id"`
	TemplateVersionName string           `mapstructure:"template_version_name,omitempty" yaml:"template_version_name" json:"template_version_name,omitempty"`
	TemplateVariable    TemplateVariable `mapstructure:"template_variable" yaml:"template_variable" json:"template_variable"`
}

type TemplateVariable struct {
	FeedbackContent string `mapstructure:"feedback_content" yaml:"feedback_content" json:"feedback_content"`
	FeedbackSource  string `mapstructure:"feedback_source" yaml:"feedback_source" json:"feedback_source"`
	FeedbackType    string `mapstructure:"feedback_type" yaml:"feedback_type" json:"feedback_type"`
}

// 发送消息的人员
type OpenID struct {
	Name   string `mapstructure:"name" yaml:"name" json:"name"`
	OpenID string `mapstructure:"open_id" yaml:"open_id" json:"open_id"`
}

// 发送给群组
type ChatID struct {
	Name   string `mapstructure:"name" yaml:"name" json:"name"`
	ChatID string `mapstructure:"chat_id" yaml:"chat_id" json:"chat_id"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr" mapstructure:"addr"`
	Password string `yaml:"password" mapstructure:"password"`
	DB       int    `yaml:"db" mapstructure:"db"`
}

func NewRedisConfig() *RedisConfig {
	var cfg RedisConfig
	if err := viper.Sub("redis").Unmarshal(&cfg); err != nil {
		log.Fatalf("unmarshal redis_config failed: %v", err)
	}
	fmt.Println(cfg)
	return &cfg
}
