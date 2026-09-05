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

// NewSheetServiceForTest 使用生产代码的读取逻辑，
// 不启动与本组测试无关、长期消费进程级队列的同步工作协程。
func NewSheetServiceForTest(db *gorm.DB, client lark.Client) SheetService {
	return &SheetServiceImpl{
		c: client, log: logger.NewZapLogger(zap.NewNop()),
		sheetDao: dao.NewSheetDAO(db), faqDAO: dao.NewFAQDAO(db),
		resolutionDAO: dao.NewFAQResolutionDAO(db),
	}
}

// ExchangeV3AtForTest 让时间戳校验时间与 Redis 测试时钟同步推进。
func ExchangeV3AtForTest(s V3AuthService, ctx context.Context, input V3ExchangeInput, now time.Time) (string, int64, error) {
	return s.(*v3AuthService).exchangeAt(ctx, input, now)
}
