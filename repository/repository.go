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
	dao.NewIntegrationDAOV3,
	dao.NewAdminUserDAOV3,
)

var CacheSet = wire.NewSet(
	cache.NewFAQResolutionStateCache,
	cache.NewIntegrationNonceStoreV3,
	cache.NewProjectConfigEventBusV3,
)

func InitTables(db *gorm.DB) error {
	models := []any{
		&model.FAQResolution{},
		&model.Sheet{},
		&model.FAQRecord{},
		&model.FeedbackProjectV3{},
		&model.FeedbackProjectKeyV3{},
		&model.FeedbackProjectTableV3{},
		&model.FeedbackProjectScopeV3{},
		&model.AdminUserV3{},
	}

	return db.AutoMigrate(models...)
}
