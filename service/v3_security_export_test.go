package service

import (
	"context"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/pkg/lark"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// NewSheetServiceForTest uses the production read paths without starting unrelated
// synchronization workers that consume process-wide queues indefinitely.
func NewSheetServiceForTest(db *gorm.DB, client lark.Client) SheetService {
	return &SheetServiceImpl{
		c: client, log: logger.NewZapLogger(zap.NewNop()),
		sheetDao: dao.NewSheetDAO(db), faqDAO: dao.NewFAQDAO(db),
		resolutionDAO: dao.NewFAQResolutionDAO(db),
	}
}

// ExchangeV3AtForTest advances timestamp validation with the Redis test clock.
func ExchangeV3AtForTest(s V3AuthService, ctx context.Context, input V3ExchangeInput, now time.Time) (string, int64, error) {
	return s.(*v3AuthService).exchangeAt(ctx, input, now)
}
