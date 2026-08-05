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
	dao.NewSheetDAO,
	dao.NewFAQDAO,
	dao.NewIntegrationDAO,
	dao.NewAdminUserDAO,
)

var CacheSet = wire.NewSet(
	cache.NewFAQResolutionStateCache,
)

func InitTables(db *gorm.DB) error {
	models := []any{
		&model.FAQResolution{},
		&model.Sheet{},
		&model.FAQRecord{},
		&model.FeedbackProject{},
		&model.FeedbackProjectKey{},
		&model.FeedbackProjectTable{},
		&model.FeedbackProjectScope{},
		&model.AdminUser{},
	}

	return db.AutoMigrate(models...)
}
