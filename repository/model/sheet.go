package model

import (
	"time"
)

// FAQResolution 定义与数据库映射的结构体
type FAQResolution struct {
	ID            uint64  `gorm:"primaryKey;autoIncrement"`
	UserID        *string `gorm:"column:user_id;not null;type:varchar(64);index:idx_user_id;index:idx_user_table;uniqueIndex:uk_user_record"`
	TableIdentify *string `gorm:"column:table_identify;not null;type:varchar(64);index:idx_user_table;uniqueIndex:uk_user_record"`
	RecordID      *string `gorm:"column:record_id;not null;type:varchar(64);uniqueIndex:uk_user_record"`
	IsResolved    *bool   `gorm:"column:is_resolved"`
	Frequency     *int    `gorm:"column:frequency"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (FAQResolution) TableName() string {
	return "faq_resolution"
}
