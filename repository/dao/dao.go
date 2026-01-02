package dao

import (
	"github.com/google/wire"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"gorm.io/gorm"
)

var ProviderSet = wire.NewSet(NewFAQResolutionDAO)

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(&model.FAQResolution{})
}
