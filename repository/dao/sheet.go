package dao

import (
	"errors"

	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FAQResolutionDAO interface {
	GetResolutionByUserAndRecord(userID, recordID *string) (*model.FAQResolution, error)
	ListResolutionsByUser(userID *string) ([]model.FAQResolution, error)
	UpsertFAQResolution(input *domain.FAQResolution) error
}

type faqResolution struct {
	db *gorm.DB
}

func NewFAQResolutionDAO(gorm *gorm.DB) FAQResolutionDAO {
	return &faqResolution{
		db: gorm,
	}
}

// GetResolutionByUserAndRecord 获取特定用户和记录的FAQ解决状态（单条记录）
func (f *faqResolution) GetResolutionByUserAndRecord(userID, recordID *string) (*model.FAQResolution, error) {
	if userID == nil || recordID == nil {
		return nil, errors.New("user_id or record_id is nil")
	}

	var res model.FAQResolution
	err := f.db.
		Where("user_id = ? AND record_id = ?", userID, recordID).
		First(&res).Error

	// 如果没有找到记录，返回 nil 而不是错误
	// 这里表示的是用户未选择该 FAQ 解决状态
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &res, err
}

// ListResolutionsByUser 获取用户的所有FAQ解决状态（多条记录）
func (f *faqResolution) ListResolutionsByUser(userID *string) ([]model.FAQResolution, error) {
	if userID == nil {
		return nil, errors.New("user_id is nil")
	}

	var list []model.FAQResolution
	err := f.db.
		Where("user_id = ?", userID).
		Find(&list).Error // 没有数据时 err = nil, list = []

	return list, err
}

func (f *faqResolution) UpsertFAQResolution(input *domain.FAQResolution) error {
	// 逻辑层兜底校验
	if input.UserID == nil || input.RecordID == nil {
		return errors.New("user_id or record_id is nil")
	}

	m := model.FAQResolution{
		UserID:     input.UserID,
		RecordID:   input.RecordID,
		IsResolved: input.IsResolved,
	}

	// 存在时更新，不存在时插入
	err := f.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "record_id"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"is_resolved",
		}),
	}).Create(&m).Error

	return err
}
