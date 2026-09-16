package ioc

import (
	"fmt"
	"log"
	"os"
	"time"

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
		return nil, nil, fmt.Errorf("mysql initialization failed")
	}

	sqlDB, err := db.DB()
	if err != nil {
		_ = logFile.Close()
		return nil, nil, err
	}
	cleanup := func() { _ = sqlDB.Close(); _ = logFile.Close() }
	err = repository.InitTables(db)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("mysql migration failed")
	}
	return db, cleanup, nil
}
