package ioc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	mysqlclient "github.com/go-sql-driver/mysql"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/repository"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitMysql(cfg *config.MysqlConfig) (*gorm.DB, func(), error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&parseTime=true&loc=Local",
		cfg.UserName, cfg.Password, cfg.Addr, cfg.DBName)

	logFile, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, nil, err
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.New(
			log.New(logFile, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             200 * time.Millisecond, // 慢 SQL 阈值
				LogLevel:                  logger.Warn,            // 日志级别
				IgnoreRecordNotFoundError: true,                   // 是否忽略记录未找到错误
				Colorful:                  false,                  // 是否彩色打印
				ParameterizedQueries:      true,
			},
		),
	})
	if err != nil {
		_ = logFile.Close()
		return nil, nil, mysqlStartupError("initialization", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		_ = logFile.Close()
		return nil, nil, mysqlStartupError("connection pool", err)
	}
	cleanup := func() { _ = sqlDB.Close(); _ = logFile.Close() }
	err = repository.InitTables(db)
	if err != nil {
		cleanup()
		return nil, nil, mysqlStartupError("migration", err)
	}
	return db, cleanup, nil
}

// Startup errors are printed by main; never include a driver message or DSN.
func mysqlStartupError(stage string, err error) error {
	var serverErr *mysqlclient.MySQLError
	if errors.As(err, &serverErr) {
		return fmt.Errorf("mysql %s failed (mysql_code=%d)", stage, serverErr.Number)
	}
	class := "driver"
	var networkErr net.Error
	var dnsErr *net.DNSError
	switch {
	case errors.Is(err, context.Canceled):
		class = "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		class = "timeout"
	case errors.As(err, &networkErr) && networkErr.Timeout():
		class = "timeout"
	case errors.As(err, &dnsErr):
		class = "dns"
	case errors.As(err, &networkErr):
		class = "network"
	}
	return fmt.Errorf("mysql %s failed (error_class=%s)", stage, class)
}
