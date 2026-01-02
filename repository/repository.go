package repository

import (
	"github.com/google/wire"
	"github.com/muxi-Infra/FeedBack-Backend/repository/cache"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"gorm.io/gorm"
)

var ProviderSet = wire.NewSet(DaoSet, CacheSet)

var DaoSet = wire.NewSet(
	dao.NewFAQResolutionDAO,
)

var CacheSet = wire.NewSet(
	cache.NewFAQResolutionStateCache,
)

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(&model.FAQResolution{})
}
