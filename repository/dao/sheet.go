package dao

import (
	"errors"

	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"gorm.io/gorm"
)

type SheetDAO interface {
	CreateSheetRecord(m *model.Sheet) error
	UpdateSheetRecord(m *model.Sheet) error
	GetSheetRecordByStudent(tableIdentify, studentID string) ([]*model.Sheet, error)
	GetSheetRecordByRecordID(tableIdentify, studentID, recordID string) (*model.Sheet, error)
}

type sheetDAO struct {
	db *gorm.DB
}

func NewSheetDAO(gorm *gorm.DB) SheetDAO {
	return &sheetDAO{
		db: gorm,
	}
}

func (s *sheetDAO) CreateSheetRecord(m *model.Sheet) error {
	if m == nil {
		return errors.New("sheet is nil")
	}

	if m.TableIdentify == nil || m.StudentID == nil || m.RecordID == nil {
		return errors.New("missing key fields for update")
	}

	return s.db.Create(m).Error
}

func (s *sheetDAO) UpdateSheetRecord(m *model.Sheet) error {
	if m == nil {
		return errors.New("sheet is nil")
	}

	if m.TableIdentify == nil || m.StudentID == nil || m.RecordID == nil {
		return errors.New("missing key fields for update")
	}

	return s.db.Model(&model.Sheet{}).
		Where("table_identify = ? AND student_id = ? AND record_id = ?",
			*m.TableIdentify,
			*m.StudentID,
			*m.RecordID,
		).
		Updates(m).Error
}

func (s *sheetDAO) CreateOrUpdateSheetRecord(m *model.Sheet) error {
	panic("implement me")
}

func (s *sheetDAO) GetSheetRecordByStudent(tableIdentify, studentID string) ([]*model.Sheet, error) {
	var records []*model.Sheet

	err := s.db.
		Where("table_identify = ? AND student_id = ?", tableIdentify, studentID).
		Order("created_at DESC").
		Find(&records).Error

	if err != nil {
		return nil, err
	}

	return records, nil
}

func (s *sheetDAO) GetSheetRecordByRecordID(tableIdentify, studentID, recordID string) (*model.Sheet, error) {
	var record model.Sheet

	err := s.db.
		Where(
			"table_identify = ? AND student_id = ? AND record_id = ?",
			tableIdentify,
			studentID,
			recordID,
		).
		Take(&record).Error

	if err != nil {
		return nil, err
	}

	return &record, nil
}
