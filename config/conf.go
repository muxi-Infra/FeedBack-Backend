package config

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/google/wire"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/spf13/viper"
)

var ProviderSet = wire.NewSet(
	NewClientConfig,
	NewJWTConfig,
	NewMiddlewareConfig,
	NewBaseTable,
	NewBatchNoticeConfig,
	NewMysqlConfig,
	NewRedisConfig,
	NewLimiterConfig,
	NewBasicAuthConfig,
	NewLogConfig,
)

var vp *viper.Viper

func InitNacos() error {
	// 从 nacos 获取
	content, err := getConfigFromNacos()
	if err != nil {
		log.Println(err)
		// 本地兜底获取
		localPath := "./config/config.yaml"
		fileContent, err := os.ReadFile(localPath)
		if err != nil {
			// 如果本地文件也读取失败，则彻底失败
			log.Fatalf("无法读取本地配置文件 %s，且 Nacos 配置获取失败: %v", localPath, err)
			return err
		}
		content = string(fileContent)
	}

	vp = viper.New()
	vp.SetConfigType("yaml")
	err = vp.ReadConfig(bytes.NewBuffer([]byte(content)))
	if err != nil {
		return err
	}

	return nil
}

func getConfigFromNacos() (string, error) {
	server, port, namespace, user, pass, group, dataId, err := parseNacosDSN()
	if err != nil {
		return "", err
	}

	serverConfigs := []constant.ServerConfig{
		{
			IpAddr: server,
			Port:   port,
			Scheme: "http",
		},
	}

	clientConfig := constant.ClientConfig{
		NamespaceId:         namespace,
		Username:            user,
		Password:            pass,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		CacheDir:            "./data/configCache",
	}

	configClient, err := clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": serverConfigs,
		"clientConfig":  clientConfig,
	})
	if err != nil {
		log.Fatal("初始化失败:", err)
	}

	content, err := configClient.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})
	if err != nil {
		log.Fatal("拉取配置失败:", err)
	}
	return content, nil
}

// DSN 示例： localhost:8848?namespace=default&username=nacos&password=1234&group=QA&dataId=my-service
func parseNacosDSN() (server string, port uint64, ns, user, pass, group, dataId string, err error) {
	var dsn string
	flag.StringVar(&dsn, "nacos-dsn", "", "Nacos DSN")
	flag.Parse()

	if dsn == "" {
		dsn = os.Getenv("NACOSDSN")
	}

	if dsn == "" {
		err = errors.New("nacos-dsn must be provided via --nacos-dsn or NACOSDSN environment variable")
		return
	}

	parts := strings.SplitN(dsn, "?", 2)
	host := parts[0]
	params := url.Values{}

	if len(parts) == 2 {
		params, _ = url.ParseQuery(parts[1])
	}

	hostParts := strings.Split(host, ":")
	server = hostParts[0]
	if len(hostParts) > 1 {
		p, _ := strconv.Atoi(hostParts[1])
		port = uint64(p)
	} else {
		port = 8848
	}

	ns = params.Get("namespace") // 当namespace是public时，此处填空字符串。

	user = params.Get("username")
	pass = params.Get("password")
	group = params.Get("group")
	dataId = params.Get("dataId")
	return
}

type ClientConfig struct {
	AppID     string `yaml:"appID"` // 应用凭证
	AppSecret string `yaml:"appSecret"`
}

func NewClientConfig() *ClientConfig {
	clientConfig := &ClientConfig{}
	err := vp.UnmarshalKey("client", &clientConfig)
	if err != nil {
		panic(err)
	}
	if clientConfig.AppID == "" || clientConfig.AppSecret == "" {
		panic("client 配置无效: appID 和 appSecret 不能为空")
	}

	//fmt.Printf("clientConfig :%v\n", clientConfig)
	return clientConfig
}

type JWTConfig struct {
	SecretKey string `yaml:"secretKey"` //秘钥
	EncKey    string `yaml:"encKey"`
	Timeout   int    `yaml:"timeout"` //过期时间
}

func NewJWTConfig() JWTConfig {
	jwtConf := JWTConfig{}
	err := vp.UnmarshalKey("jwt", &jwtConf)
	if err != nil {
		panic(err)
	}
	if jwtConf.SecretKey == "" || jwtConf.EncKey == "" {
		panic("jwt 配置无效: secretKey, encKey 不能为空")
	}
	if jwtConf.Timeout <= 0 {
		panic("jwt 配置无效: timeout 必须大于 0")
	}

	//fmt.Printf("jwtConf :%v\n", jwtConf)
	return jwtConf
}

type MiddlewareConfig struct {
	AllowedOrigins []string `yaml:"allowedOrigins"`
}

func NewMiddlewareConfig() *MiddlewareConfig {
	middlewareConfig := vp.Get("middleware.allowedorigins")
	var allowedOrigins []string
	if arr, ok := middlewareConfig.([]interface{}); ok {
		for _, v := range arr {
			if str, ok := v.(string); ok {
				allowedOrigins = append(allowedOrigins, str)
			}
		}
	}
	mc := &MiddlewareConfig{
		AllowedOrigins: allowedOrigins,
	}

	//fmt.Printf("middlewareConfig :%v\n", middlewareConfig)
	return mc
}

type BaseTable struct {
	TableToken string `yaml:"tableToken"`
	TableID    string `yaml:"tableId"`
	ViewID     string `yaml:"viewId"`
}

func NewBaseTable() *BaseTable {
	baseTableCfg := &BaseTable{}
	err := vp.UnmarshalKey("baseTable", &baseTableCfg)
	if err != nil {
		panic(err)
	}
	if baseTableCfg.TableToken == "" || baseTableCfg.TableID == "" || baseTableCfg.ViewID == "" {
		fmt.Println(baseTableCfg)
		panic("base_table 配置无效: tableToken, tableID, 和 viewID 不能为空")
	}

	//fmt.Printf("baseTable :%v\n", baseTable)
	return baseTableCfg
}

type BatchNoticeConfig struct {
	OpenIDs []OpenID `mapstructure:"open_ids" yaml:"open_ids" json:"open_ids"`
	ChatIDs []ChatID `mapstructure:"chat_ids" yaml:"chat_ids" json:"chat_ids"`
	Content Content  `mapstructure:"content" yaml:"content" json:"content"`
}

func NewBatchNoticeConfig() *BatchNoticeConfig {
	openIDs := make([]OpenID, 0)
	chatIDs := make([]ChatID, 0)
	openIDsData := vp.Get("open_ids")
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
	chatIDsData := vp.Get("chat_ids")
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

	batchNoticeConfig := &BatchNoticeConfig{
		OpenIDs: openIDs,
		ChatIDs: chatIDs,
		Content: Content{
			Type: vp.GetString("content.type"),
			Data: Data{
				TemplateID: vp.GetString("content.data.template_id"),
				TemplateVariable: TemplateVariable{
					FeedbackSource: vp.GetString("content.data.template_variable.feedback_source"),
					DailyNewCount:  vp.GetInt("content.data.template_variable.daily_new_count"),
					TableUrl: TableUrl{
						PCUrl:      vp.GetString("content.data.template_variable.table_url.pc_url"),
						AndroidUrl: vp.GetString("content.data.template_variable.table_url.android_url"),
						IOSUrl:     vp.GetString("content.data.template_variable.table_url.ios_url"),
						Url:        vp.GetString("content.data.template_variable.table_url.url"),
					},
				},
			},
		},
	}

	if batchNoticeConfig.Content.Type == "" || batchNoticeConfig.Content.Data.TemplateID == "" {
		panic("batch_notice 配置无效: content.type 和 content.data.template_id 不能为空")
	}

	//fmt.Printf("batchNoticeConfig :%v\n", batchNoticeConfig)
	return batchNoticeConfig
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
	FeedbackSource string   `mapstructure:"feedback_source" yaml:"feedback_source" json:"feedback_source"`
	DailyNewCount  int      `mapstructure:"daily_new_count" yaml:"daily_new_count" json:"daily_new_count"`
	TableUrl       TableUrl `mapstructure:"table_url" yaml:"table_url" json:"table_url"`
}

type TableUrl struct {
	PCUrl      string `mapstructure:"pc_url" yaml:"pc_url" json:"pc_url"`
	AndroidUrl string `mapstructure:"android_url" yaml:"android_url" json:"android_url"`
	IOSUrl     string `mapstructure:"ios_url" yaml:"ios_url" json:"ios_url"`
	Url        string `mapstructure:"url" yaml:"url" json:"url"`
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

type RedisConfig struct {
	Addr     string `yaml:"addr" mapstructure:"addr"`
	Password string `yaml:"password" mapstructure:"password"`
	DB       int    `yaml:"db" mapstructure:"db"`
}

func NewRedisConfig() *RedisConfig {
	redisConfig := &RedisConfig{
		Addr:     vp.GetString("redis.addr"),
		Password: vp.GetString("redis.password"),
		DB:       vp.GetInt("redis.db"),
	}
	if redisConfig.Addr == "" {
		panic("redis 配置无效: addr 不能为空")
	}

	//fmt.Printf("redisConfig :%v\n", redisConfig)
	return redisConfig
}

type MysqlConfig struct {
	Addr     string `yaml:"addr" mapstructure:"addr"`
	DBName   string `yaml:"dbname" mapstructure:"dbname"`
	UserName string `yaml:"username" mapstructure:"username"`
	Password string `yaml:"password" mapstructure:"password"`
	LogFile  string `yaml:"logfile" mapstructure:"logfile"`
}

func NewMysqlConfig() *MysqlConfig {
	mysqlConfig := &MysqlConfig{}
	err := vp.UnmarshalKey("mysql", &mysqlConfig)
	if err != nil {
		panic(fmt.Sprintf("无法解析 MySQL 配置: %v", err))
	}

	if mysqlConfig.Addr == "" {
		panic("MySQL 配置无效: addr 不能为空")
	}
	if mysqlConfig.DBName == "" {
		panic("MySQL 配置无效: dbname 不能为空")
	}
	if mysqlConfig.UserName == "" || mysqlConfig.Password == "" {
		panic("MySQL 配置无效: username 和 password 不能为空")
	}
	if mysqlConfig.LogFile == "" {
		panic("MySQL 配置无效: logfile 不能为空")
	}
	return mysqlConfig
}

type LogConfig struct {
	File       string `yaml:"file"`
	MaxSize    int    `yaml:"maxSize"`
	MaxBackups int    `yaml:"maxBackups"`
	MaxAge     int    `yaml:"maxAge"`
	Compress   bool   `yaml:"compress"`
}

func NewLogConfig() *LogConfig {
	cfg := &LogConfig{}
	err := vp.UnmarshalKey("log", &cfg)
	if err != nil {
		panic(fmt.Sprintf("无法解析日志配置: %v", err))
	}
	if cfg.File == "" {
		panic("日志配置无效: file 不能为空")
	}

	return cfg
}

type LimiterConfig struct {
	Capacity     int `yaml:"capacity"`     // 令牌桶容量
	FillInterval int `yaml:"fillInterval"` // 每秒补充令牌的次数
	Quantum      int `yaml:"quantum"`      // 每次放置的令牌数
}

func NewLimiterConfig() *LimiterConfig {
	cfg := &LimiterConfig{}
	err := vp.UnmarshalKey("limiter", &cfg)
	if err != nil {
		panic(fmt.Sprintf("无法解析限流器配置: %v", err))
	}
	if cfg.Capacity <= 0 || cfg.FillInterval <= 0 || cfg.Quantum <= 0 {
		panic("限流器配置无效: capacity, fillInterval, 和 quantum 必须大于 0")
	}

	return cfg
}

type BasicAuthConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func NewBasicAuthConfig() []BasicAuthConfig {
	var users []BasicAuthConfig
	err := vp.UnmarshalKey("basicAuth", &users)
	if err != nil {
		panic(fmt.Sprintf("无法解析 BasicAuth 配置: %v", err))
	}
	if len(users) == 0 {
		panic("BasicAuth 配置无效: 至少需要一个用户")
	}
	for _, u := range users {
		if u.Username == "" || u.Password == "" {
			panic("BasicAuth 配置无效: username 和 password 不能为空")
		}
	}
	return users
}
