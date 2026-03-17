package dao

import (
	"errors"

	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FAQDAO interface {
	CreateOrUpdateSheetRecord(m *model.FAQRecord) error
	GetFAQRecords(tableIdentify *string) ([]model.FAQRecord, error)
	GetFAQRecordIDs(tableIdentify *string) ([]string, error)
	DeleteFAQRecord(tableIdentify, recordID *string) error
}

type faqDAO struct {
	db *gorm.DB
}

func NewFAQDAO(gorm *gorm.DB) FAQDAO {
	return &faqDAO{
		db: gorm,
	}
}

func (f *faqDAO) CreateOrUpdateSheetRecord(m *model.FAQRecord) error {
	if m == nil {
		return errors.New("sheet is nil")
	}

	if m.TableIdentify == nil || m.RecordID == nil {
		return errors.New("missing key fields")
	}

	// 存在时更新，不存在时创建
	err := f.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "table_identify"},
			{Name: "record_id"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"record":           gorm.Expr("VALUES(record)"),
			"resolved_count":   gorm.Expr("VALUES(resolved_count)"),
			"unresolved_count": gorm.Expr("VALUES(unresolved_count)"),
			"updated_at":       gorm.Expr("NOW(3)"),
		}),
	}).Create(m).Error

	return err
}

func (f *faqDAO) GetFAQRecords(tableIdentify *string) ([]model.FAQRecord, error) {
	if tableIdentify == nil {
		return nil, errors.New("missing key fields")
	}

	var faqs []model.FAQRecord

	err := f.db.
		Where("table_identify = ?", *tableIdentify).
		Find(&faqs).Error
	if err != nil {
		return nil, err
	}

	return faqs, nil
}

// GetFAQRecordIDs 获取特定表的所有FAQ记录ID列表
// 和 GetFAQRecords 的区别在于只查询 record_id 字段，不回表
func (f *faqDAO) GetFAQRecordIDs(tableIdentify *string) ([]string, error) {
	if tableIdentify == nil {
		return nil, errors.New("missing key fields")
	}

	var recordIDs []string
	err := f.db.
		Model(&model.FAQRecord{}).
		Where("table_identify = ?", *tableIdentify).
		Pluck("record_id", &recordIDs).Error
	if err != nil {
		return nil, err
	}

	return recordIDs, nil
}

func (f *faqDAO) DeleteFAQRecord(tableIdentify, recordID *string) error {
	if tableIdentify == nil || recordID == nil {
		return errors.New("missing key fields")
	}

	return f.db.
		Where("table_identify = ? AND record_id = ?", *tableIdentify, *recordID).
		Delete(&model.FAQRecord{}).Error
}
