package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/apolloconfig/agollo/v4"
	"github.com/apolloconfig/agollo/v4/agcache"
	"github.com/apolloconfig/agollo/v4/env/config"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	NewClientConfig,
	NewJWTConfig,
	NewMiddlewareConfig,
	NewAppTable,
	NewBatchNoticeConfig,
)

var (
	cache  agcache.CacheInterface
	client agollo.Client
	c      *config.AppConfig
)

func InitApollo() error {
	// 解析命令行参数
	var secret string
	flag.StringVar(&secret, "apollo-secret", "", "Apollo config server secret")
	flag.Parse()

	if secret == "" {
		// 尝试从环境变量获取
		secret = os.Getenv("APOLLO_SECRET")
	}

	if secret == "" {
		return fmt.Errorf("apollo secret must be provided via --apollo-secret or APOLLO_SECRET environment variable")
	}
	c = &config.AppConfig{
		AppID:          "feedback",
		Cluster:        "default",
		IP:             "http://apollo.muxixyz.com:8080",
		NamespaceName:  "config.yaml",
		IsBackupConfig: true,
		Secret:         secret,
	}
	var err error
	client, err = agollo.StartWithConfig(func() (*config.AppConfig, error) {
		return c, nil
	})
	cache = client.GetConfigCache(c.NamespaceName)
	return err
}

type ClientConfig struct {
	AppID     string `yaml:"appID"` // 应用凭证
	AppSecret string `yaml:"appSecret"`
}

func NewClientConfig() ClientConfig {
	return ClientConfig{
		AppID:     getStringFromCache("client.appid"),
		AppSecret: getStringFromCache("client.appsecret"),
	}
}

type JWTConfig struct {
	SecretKey string `yaml:"secretKey"` //秘钥
	Timeout   int    `yaml:"timeout"`   //过期时间
}

func NewJWTConfig() JWTConfig {
	secretKey, _ := cache.Get("jwt.secretkey")
	timeout, _ := cache.Get("jwt.timeout")
	return JWTConfig{
		SecretKey: secretKey.(string),
		Timeout:   timeout.(int),
	}
}

type MiddlewareConfig struct {
	AllowedOrigins []string `yaml:"allowedOrigins"`
}

func NewMiddlewareConfig() *MiddlewareConfig {
	middlewareConfig, _ := cache.Get("middleware.allowedorigins")
	var allowedOrigins []string
	if arr, ok := middlewareConfig.([]interface{}); ok {
		for _, v := range arr {
			if str, ok := v.(string); ok {
				allowedOrigins = append(allowedOrigins, str)
			}
		}
	}
	return &MiddlewareConfig{
		AllowedOrigins: allowedOrigins,
	}
}

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
	appToken := getStringFromCache("app_table.app_token")
	tables := make(map[string]Table)
	appTablesId, _ := cache.Get("app_tables_id")
	if arr, ok := appTablesId.([]interface{}); ok {
		for _, v := range arr {
			tableName := fmt.Sprintf("app_table.tables.%s.name", v.(string))
			tableID := fmt.Sprintf("app_table.tables.%s.table_id", v.(string))
			viewID := fmt.Sprintf("app_table.tables.%s.view_id", v.(string))
			tables[v.(string)] = Table{
				Name:    getStringFromCache(tableName),
				TableID: getStringFromCache(tableID),
				ViewID:  getStringFromCache(viewID),
			}
		}
	}
	return &AppTable{
		AppToken: appToken,
		Tables:   tables,
	}
}

func getStringFromCache(key string) string {
	val, _ := cache.Get(key)
	if str, ok := val.(string); ok {
		return str
	}
	return ""
}

type BatchNoticeConfig struct {
	OpenIDs []OpenID `mapstructure:"open_ids" yaml:"open_ids" json:"open_ids"`
	ChatIDs []ChatID `mapstructure:"chat_ids" yaml:"chat_ids" json:"chat_ids"`
	Content Content  `mapstructure:"content" yaml:"content" json:"content"`
}

func NewBatchNoticeConfig() *BatchNoticeConfig {
	openIDs := make([]OpenID, 0)
	chatIDs := make([]ChatID, 0)
	openIDsData, _ := cache.Get("open_ids")
	if arr, ok := openIDsData.([]interface{}); ok {
		for _, v := range arr {
			if m, ok := v.(map[string]interface{}); ok {
				openIDs = append(openIDs, OpenID{
					Name:   m["name"].(string),
					OpenID: m["open_id"].(string),
				})
			}
		}
	}
	chatIDsData, _ := cache.Get("chat_ids")
	if arr, ok := chatIDsData.([]interface{}); ok {
		for _, v := range arr {
			if m, ok := v.(map[string]interface{}); ok {
				chatIDs = append(chatIDs, ChatID{
					Name:   m["name"].(string),
					ChatID: m["chat_id"].(string),
				})
			}
		}
	}

	return &BatchNoticeConfig{
		OpenIDs: openIDs,
		ChatIDs: chatIDs,
		Content: Content{
			Type: getStringFromCache("content.type"),
			Data: Data{
				TemplateID: getStringFromCache("content.data.template_id"),
				TemplateVariable: TemplateVariable{
					FeedbackContent: getStringFromCache("content.data.template_variable.feedback_content"),
					FeedbackSource:  getStringFromCache("content.data.template_variable.feedback_source"),
					FeedbackType:    getStringFromCache("content.data.template_variable.feedback_type"),
				},
			},
		},
	}
}

// Content 发送消息的内容
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

// OpenID 发送消息的人员
type OpenID struct {
	Name   string `mapstructure:"name" yaml:"name" json:"name"`
	OpenID string `mapstructure:"open_id" yaml:"open_id" json:"open_id"`
}

// ChatID 发送给群组
type ChatID struct {
	Name   string `mapstructure:"name" yaml:"name" json:"name"`
	ChatID string `mapstructure:"chat_id" yaml:"chat_id" json:"chat_id"`
}

func (t *AppTable) IsValidTableID(tableID string) bool {
	_, ok := t.Tables[tableID]
	return ok
}
