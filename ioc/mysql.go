package ioc

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitMysql(cfg *config.MysqlConfig) *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&parseTime=true&loc=Local",
		cfg.UserName, cfg.Password, cfg.Addr, cfg.DBName)

	logFile, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.New(
			log.New(logFile, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             200 * time.Millisecond, // 慢 SQL 阈值
				LogLevel:                  logger.Warn,            // 日志级别
				IgnoreRecordNotFoundError: true,                   // 是否忽略记录未找到错误
				Colorful:                  true,                   // 是否彩色打印
			},
		),
	})
	if err != nil {
		panic(fmt.Sprintf("Mysql 连接失败: %v", err))
	}

	err = dao.InitTables(db)
	if err != nil {
		panic(err)
	}
	return db
}
