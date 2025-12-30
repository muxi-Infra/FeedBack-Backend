package ioc

import (
	"fmt"

	"github.com/muxi-Infra/FeedBack-Backend/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitMysql(cfg *config.MysqlConfig) *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&parseTime=true&loc=Local",
		cfg.UserName, cfg.Password, cfg.Addr, cfg.DBName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("Mysql 连接失败: %v", err))
	}
	return db
}
