package main

import (
	"fmt"
	"os"

	"github.com/casbin/casbin/v3"
	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

// @title		木犀反馈系统 API
// @version	1.0
// @host		localhost:8080
// @BasePath	/
func main() {
	// 加载项目根目录的 .env。已有的系统环境变量不会被覆盖。
	if err := gotenv.Load(); err != nil && !os.IsNotExist(err) {
		panic(fmt.Errorf("加载 .env 失败: %w", err))
	}

	err := config.InitNacos()
	if err != nil {
		panic(err)
	}
	app, err := InitApp()
	if err != nil {
		panic(err)
	}
	app.r.Run(":8080")
}

type App struct {
	r        *gin.Engine
	enforcer *casbin.Enforcer
}

func initViper() {
	cfile := pflag.String("config", "config/config.yaml", "配置文件路径")
	pflag.Parse()

	viper.SetConfigType("yaml")
	viper.SetConfigFile(*cfile)
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
}
