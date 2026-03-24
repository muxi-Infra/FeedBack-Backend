package dao

import (
	"context"
	"errors"

	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	DefaultLimit = 10
	MaxLimit     = 100
)

type SheetDAO interface {
	CreateSheetRecord(ctx context.Context, m *model.Sheet) error
	CreateOrUpdateSheetRecord(ctx context.Context, m *model.Sheet) error
	CountSheetRecordByUser(ctx context.Context, tableIdentify, userID string) (uint64, error)
	GetSheetRecordByUser(ctx context.Context, tableIdentify, userID string, lastID *uint64, limit int) ([]*model.Sheet, bool, error)
	GetSheetRecordByRecordID(ctx context.Context, tableIdentify, userID, recordID string) (*model.Sheet, error)
	ResetIsSyncedByUser(ctx context.Context, tableIdentify, userID string) error
	GetUnsyncedRecordsByTable(ctx context.Context, tableIdentify string) ([]string, error)
	GetUnNoticedRecordsByTable(ctx context.Context, tableIdentify string) ([]model.Sheet, error)
	MarkRecordNoticed(ctx context.Context, tableIdentify, recordID string) error
}

type sheetDAO struct {
	db *gorm.DB
}

func NewSheetDAO(gorm *gorm.DB) SheetDAO {
	return &sheetDAO{
		db: gorm,
	}
}

func (s *sheetDAO) CreateSheetRecord(ctx context.Context, m *model.Sheet) error {
	if m == nil {
		return errors.New("sheet is nil")
	}

	if m.TableIdentify == nil || m.UserID == nil || m.RecordID == nil {
		return errors.New("missing key fields for update")
	}

	return s.db.WithContext(ctx).
		Create(m).Error
}

func (s *sheetDAO) UpdateSheetRecord(ctx context.Context, m *model.Sheet) error {
	if m == nil {
		return errors.New("sheet is nil")
	}

	if m.TableIdentify == nil || m.UserID == nil || m.RecordID == nil {
		return errors.New("missing key fields for update")
	}

	return s.db.WithContext(ctx).
		Model(&model.Sheet{}).
		Where("table_identify = ? AND user_id = ? AND record_id = ?",
			*m.TableIdentify, *m.UserID, *m.RecordID,
		).
		Updates(m).Error
}

func (s *sheetDAO) CreateOrUpdateSheetRecord(ctx context.Context, m *model.Sheet) error {
	if m == nil {
		return errors.New("sheet is nil")
	}

	if m.TableIdentify == nil || m.UserID == nil || m.RecordID == nil {
		return errors.New("missing key fields")
	}

	// 存在时更新，不存在时创建
	err := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "table_identify"},
				{Name: "user_id"},
				{Name: "record_id"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"record":     gorm.Expr("VALUES(record)"),
				"share_url":  gorm.Expr("VALUES(share_url)"),
				"is_synced":  gorm.Expr("VALUES(is_synced)"),
				"updated_at": gorm.Expr("NOW(3)"),
			}),
		}).Create(m).Error

	return err
}

// CountSheetRecordByUser 统计指定用户在指定表格下的记录总数
func (s *sheetDAO) CountSheetRecordByUser(ctx context.Context, tableIdentify, userID string) (uint64, error) {
	var total int64

	err := s.db.WithContext(ctx).
		Model(&model.Sheet{}).
		Where("table_identify = ? AND user_id = ?", tableIdentify, userID).
		Count(&total).Error

	if err != nil {
		return 0, err
	}

	return uint64(total), nil
}

// GetSheetRecordByUser 根据 tableIdentify 和 userID 获取该用户在该表格下的记录列表，支持分页（lastID + limit）
// 返回值中 hasMore 表示是否有下一页
// service 层中 lastID := records[len(records)-1].ID
func (s *sheetDAO) GetSheetRecordByUser(ctx context.Context, tableIdentify, userID string, lastID *uint64, limit int) ([]*model.Sheet, bool, error) {
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	var records []*model.Sheet

	query := s.db.WithContext(ctx).
		Model(&model.Sheet{}).
		Where("table_identify = ? AND user_id = ?", tableIdentify, userID)

	// 游标条件（因为 ORDER BY id DESC）
	if lastID != nil && *lastID > 0 {
		query = query.Where("id < ?", *lastID)
	}

	// 多查一条判断是否有下一页
	err := query.WithContext(ctx).
		Order("id DESC").
		Limit(limit + 1).
		Find(&records).Error
	if err != nil {
		return nil, false, err
	}

	hasMore := false
	if len(records) > limit {
		hasMore = true
		records = records[:limit]
	}

	return records, hasMore, nil
}

// GetSheetRecordByRecordID 根据 tableIdentify、userID 和 recordID 获取单条记录
func (s *sheetDAO) GetSheetRecordByRecordID(ctx context.Context, tableIdentify, userID, recordID string) (*model.Sheet, error) {
	var record model.Sheet

	err := s.db.WithContext(ctx).
		Where(
			"table_identify = ? AND user_id = ? AND record_id = ?",
			tableIdentify, userID, recordID,
		).
		Take(&record).Error

	if err != nil {
		return nil, err
	}

	return &record, nil
}

// ResetIsSyncedByUser 重置指定用户在指定表格下的所有记录的 is_synced 字段为 0（未同步状态）
func (s *sheetDAO) ResetIsSyncedByUser(ctx context.Context, tableIdentify, userID string) error {
	if tableIdentify == "" || userID == "" {
		return errors.New("missing tableIdentify or userID")
	}

	return s.db.WithContext(ctx).
		Model(&model.Sheet{}).
		Where("table_identify = ? AND user_id = ?", tableIdentify, userID).
		Update("is_synced", 0).Error
}

// GetUnsyncedRecordsByTable 获取指定表格下所有未同步的记录ID列表（不区分用户）
func (s *sheetDAO) GetUnsyncedRecordsByTable(ctx context.Context, tableIdentify string) ([]string, error) {
	var recordIDs []string

	err := s.db.WithContext(ctx).
		Model(&model.Sheet{}).
		Where("table_identify = ? AND is_synced = 0", tableIdentify).
		Pluck("record_id", &recordIDs).Error

	if err != nil {
		return nil, err
	}

	return recordIDs, nil
}

// GetUnNoticedRecordsByTable 获取指定表格下所有未通知的记录ID列表
func (s *sheetDAO) GetUnNoticedRecordsByTable(ctx context.Context, tableIdentify string) ([]model.Sheet, error) {
	var records []model.Sheet

	err := s.db.WithContext(ctx).
		Model(&model.Sheet{}).
		Select([]string{"record_id", "user_id"}).
		Where("table_identify = ? AND is_noticed = 0", tableIdentify).
		Find(&records).Error

	if err != nil {
		return nil, err
	}

	return records, nil
}

// MarkRecordNoticed 将指定表格下的特定记录的 is_noticed 字段更新为 1（已通知状态）
func (s *sheetDAO) MarkRecordNoticed(ctx context.Context, tableIdentify, recordID string) error {
	return s.db.WithContext(ctx).
		Model(&model.Sheet{}).
		Where("table_identify = ? AND record_id = ?", tableIdentify, recordID).
		Update("is_noticed", 1).Error
}
