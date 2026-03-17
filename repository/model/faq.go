package model

import "time"

type FAQRecord struct {
	ID            uint64  `gorm:"primaryKey;autoIncrement"`
	TableIdentify *string `gorm:"column:table_identify;not null;type:varchar(32);uniqueIndex:idx_faq_record,priority:1"`
	RecordID      *string `gorm:"column:record_id;not null;type:varchar(32);uniqueIndex:idx_faq_record,priority:2"`

	Record          map[string]any `gorm:"column:record;not null;type:json;serializer:json"`
	ResolvedCount   int64          `gorm:"column:resolved_count;default:0"`
	UnresolvedCount int64          `gorm:"column:unresolved_count;default:0"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (FAQRecord) TableName() string {
	return "faq_record"
}
