package main

import (
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/muxi-Infra/FeedBack-Backend/config"
)

func init() {
	// 预加载.env文件,用于本地开发
	_ = godotenv.Load()
}

// @title		木犀反馈系统 API
// @version	1.0
// @host		localhost:8080
// @BasePath	/
func main() {
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
	r *gin.Engine
}
