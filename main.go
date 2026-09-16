package main

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/service"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

// @title		木犀反馈系统 API
// @version	1.0
// @host		localhost:8080
// @BasePath	/
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	err := config.InitNacos()
	if err != nil {
		panic(err)
	}
	app, cleanup, err := InitApp()
	if err != nil {
		panic(err)
	}
	defer cleanup()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err = app.run(ctx); err != nil {
		panic(err)
	}
}

type App struct {
	r             *gin.Engine
	configRuntime *service.ConfigRuntimeV3
}

func (app *App) run(ctx context.Context) error {
	if err := app.configRuntime.Start(ctx); err != nil {
		return err
	}
	server := &http.Server{Addr: ":8080", Handler: app.r, ReadHeaderTimeout: 10 * time.Second}
	result := make(chan error, 1)
	go func() { result <- server.ListenAndServe() }()
	var serveErr error
	select {
	case <-ctx.Done():
	case serveErr = <-result:
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err := server.Shutdown(shutdown)
	cancel()
	if err != nil {
		_ = server.Close()
	}
	shutdown, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	runtimeErr := app.configRuntime.Stop(shutdown)
	cancel()
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}
	return errors.Join(serveErr, err, runtimeErr)
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
