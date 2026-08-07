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
	NewAdminJWTConfig,
	NewMiddlewareConfig,
	NewBaseTable,
	NewLarkMessageConfig,
	NewCCNUBoxMessageConfig,
	NewIntegrationAuthConfig,
	NewMysqlConfig,
	NewRedisConfig,
	NewLimiterConfig,
	NewBasicAuthConfig,
	NewLogConfig,
)

var vp *viper.Viper

func InitNacos() error {
	localPath := "./config/config.yaml"
	source := strings.ToLower(strings.TrimSpace(os.Getenv("FEEDBACK_CONFIG_SOURCE")))
	if source == "" {
		source = "auto"
	}

	var (
		content string
		err     error
	)
	switch source {
	case "local":
		content, err = readLocalConfig(localPath)
	case "nacos":
		content, err = getConfigFromNacos()
	case "auto":
		// 本地配置存在时优先使用本地配置；本地文件不存在时再读取 Nacos。
		content, err = readLocalConfig(localPath)
		if errors.Is(err, os.ErrNotExist) {
			log.Println("本地配置不存在，尝试从 Nacos 获取")
			content, err = getConfigFromNacos()
		}
	default:
		return fmt.Errorf("不支持的 FEEDBACK_CONFIG_SOURCE: %q，可选值为 local、nacos、auto", source)
	}
	if err != nil {
		return fmt.Errorf("加载配置失败，source=%s: %w", source, err)
	}

	vp = viper.New()
	vp.SetConfigType("yaml")
	err = vp.ReadConfig(bytes.NewBuffer([]byte(content)))
	if err != nil {
		return err
	}

	return nil
}

func readLocalConfig(path string) (string, error) {
	fileContent, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(fileContent), nil
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
		return "", fmt.Errorf("初始化 Nacos 客户端失败: %w", err)
	}

	content, err := configClient.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})
	if err != nil {
		return "", fmt.Errorf("拉取 Nacos 配置失败: %w", err)
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
	Timeout   int    `yaml:"timeout"`   //过期时间
}

// AdminJWTConfig 管理后台 JWT 配置，与反馈接口 JWT 分离，避免两类令牌互相复用。
type AdminJWTConfig struct {
	SecretKey string `mapstructure:"secret_key" yaml:"secret_key"`
	Issuer    string `mapstructure:"issuer" yaml:"issuer"`
	Audience  string `mapstructure:"audience" yaml:"audience"`
	Timeout   int    `mapstructure:"timeout" yaml:"timeout"`
}

func NewAdminJWTConfig() AdminJWTConfig {
	cfg := AdminJWTConfig{}
	if err := vp.UnmarshalKey("admin_jwt", &cfg); err != nil {
		panic(fmt.Sprintf("无法解析 admin_jwt 配置: %v", err))
	}
	if cfg.SecretKey == "" || cfg.Issuer == "" || cfg.Audience == "" || cfg.Timeout <= 0 {
		panic("admin_jwt 配置无效: secret_key、issuer、audience 不能为空，timeout 必须大于 0")
	}
	return cfg
}

type IntegrationAuthConfig struct {
	AccessTokenTTL int `mapstructure:"access_token_ttl" yaml:"access_token_ttl"`
}

func NewIntegrationAuthConfig() *IntegrationAuthConfig {
	cfg := &IntegrationAuthConfig{}
	if err := vp.UnmarshalKey("integration", cfg); err != nil {
		panic(fmt.Sprintf("无法解析 integration 配置: %v", err))
	}
	if cfg.AccessTokenTTL <= 0 {
		cfg.AccessTokenTTL = 3600
	}
	return cfg
}

func NewJWTConfig() JWTConfig {
	jwtConf := JWTConfig{}
	err := vp.UnmarshalKey("jwt", &jwtConf)
	if err != nil {
		panic(err)
	}
	if jwtConf.SecretKey == "" {
		panic("jwt 配置无效: secretKey 不能为空")
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

// LarkMessage 发送的内容
type LarkMessage struct {
	TemplateID string      `mapstructure:"templateID" yaml:"templateID" json:"templateID"`
	ReceiveIDs []ReceiveID `mapstructure:"receiveIDs" yaml:"receiveIDs" json:"receiveIDs"`
}

type ReceiveID struct {
	Type string `mapstructure:"type" yaml:"type" json:"type"`
	ID   string `mapstructure:"id" yaml:"id" json:"id"`
}

func NewLarkMessageConfig() *LarkMessage {
	larkMessage := &LarkMessage{}
	err := vp.UnmarshalKey("larkMessage", &larkMessage)
	if err != nil {
		panic(fmt.Sprintf("无法解析 receive_ids 配置: %v", err))
	}
	if larkMessage.TemplateID == "" {
		panic("receive_ids 配置无效: templateID 不能为空")
	}
	if len(larkMessage.ReceiveIDs) == 0 {
		panic("receive_ids 配置无效: 至少需要一个接收者")
	}

	//fmt.Println(larkMessage)
	return larkMessage
}

type CCNUBoxMessage struct {
	TableIdentify string `yaml:"tableIdentify" json:"tableIdentify"`
	BasicUser     string `yaml:"basicUser" json:"basicUser"`
	BasicPassword string `yaml:"basicPassword" json:"basicPassword"`
	BaseURL       string `yaml:"baseURL" json:"baseURL"`
}

func NewCCNUBoxMessageConfig() *CCNUBoxMessage {
	ccnuBoxMessage := &CCNUBoxMessage{}
	err := vp.UnmarshalKey("ccnuBoxMessage", &ccnuBoxMessage)
	if err != nil {
		panic(fmt.Sprintf("无法解析 ccnuBoxMessage 配置: %v", err))
	}
	if ccnuBoxMessage.TableIdentify == "" || ccnuBoxMessage.BasicUser == "" || ccnuBoxMessage.BasicPassword == "" || ccnuBoxMessage.BaseURL == "" {
		panic("ccnuBoxMessage 配置无效: tableIdentify, basicUser, basicPassword, 和 baseURL 不能为空")
	}
	return ccnuBoxMessage
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
