package dao

import (
	"errors"

	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FAQResolutionDAO interface {
	GetResolutionByUserAndRecord(userID, tableIdentify, recordID *string) (*model.FAQResolution, error)
	ListResolutionsByUser(userID, tableIdentify *string) ([]model.FAQResolution, error)
	CreateOrUpsertFAQResolution(input *model.FAQResolution) error
}

type faqResolutionDAO struct {
	db *gorm.DB
}

func NewFAQResolutionDAO(gorm *gorm.DB) FAQResolutionDAO {
	return &faqResolutionDAO{
		db: gorm,
	}
}

// GetResolutionByUserAndRecord 获取特定用户和记录的FAQ解决状态（单条记录）
func (f *faqResolutionDAO) GetResolutionByUserAndRecord(userID, tableIdentify, recordID *string) (*model.FAQResolution, error) {
	if userID == nil || tableIdentify == nil || recordID == nil {
		return nil, errors.New("user_id or table_identify or record_id is nil")
	}

	var res model.FAQResolution
	err := f.db.
		Where("user_id = ? AND table_identify = ? AND record_id = ?", userID, tableIdentify, recordID).
		Take(&res).Error

	// 如果没有找到记录，返回 nil 而不是错误
	// 这里表示的是用户未选择该 FAQ 解决状态
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &res, err
}

// ListResolutionsByUser 获取用户的所有FAQ解决状态（多条记录）
func (f *faqResolutionDAO) ListResolutionsByUser(userID, tableIdentify *string) ([]model.FAQResolution, error) {
	if userID == nil || tableIdentify == nil {
		return nil, errors.New("user_id or table_identify is nil")
	}

	var list []model.FAQResolution
	err := f.db.
		Where("user_id = ? AND table_identify = ?", userID, tableIdentify).
		Find(&list).Error // 没有数据时 err = nil, list = []

	return list, err
}

func (f *faqResolutionDAO) CreateOrUpsertFAQResolution(m *model.FAQResolution) error {
	// 逻辑层兜底校验
	if m.UserID == nil || m.RecordID == nil {
		return errors.New("user_id or record_id is nil")
	}

	// 存在时更新，不存在时插入
	err := f.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "table_identify"},
			{Name: "record_id"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"is_resolved": gorm.Expr("VALUES(is_resolved)"),
			"frequency":   gorm.Expr("VALUES(frequency)"),
			"updated_at":  gorm.Expr("NOW(3)"),
		}),
	}).Create(m).Error

	return err
}
