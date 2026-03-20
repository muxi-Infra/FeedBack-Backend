package repository

import (
	"github.com/google/wire"
	"github.com/muxi-Infra/FeedBack-Backend/repository/cache"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/repository/es"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"gorm.io/gorm"
)

var ProviderSet = wire.NewSet(DaoSet, CacheSet)

var DaoSet = wire.NewSet(
	dao.NewFAQResolutionDAO,
	dao.NewSheetDAO,
	dao.NewChatDAO,
	es.NewFAQESRepo,
	dao.NewFAQDAO,
)

var CacheSet = wire.NewSet(
	cache.NewFAQResolutionStateCache,
	cache.NewChatCache,
)

func InitTables(db *gorm.DB) error {
	models := []any{
		&model.FAQResolution{},
		&model.Sheet{},
		&model.FAQRecord{},
		&model.Conversation{},
		&model.Message{},
	}

	return db.AutoMigrate(models...)
}
