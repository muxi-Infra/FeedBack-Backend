package model

import (
	"time"
)

type Sheet struct {
	ID            uint64  `gorm:"primaryKey;autoIncrement"`
	TableIdentify *string `gorm:"column:table_identify;not null;type:varchar(32);index:idx_user_record,priority:1"`
	RecordID      *string `gorm:"column:record_id;not null;type:varchar(32);index:idx_user_record,priority:3"`
	StudentID     *string `gorm:"column:student_id;not null;type:varchar(32);index:idx_user_record,priority:2"`

	Record map[string]any `gorm:"column:record;not null;serializer:json"`

	//Content     *string        `gorm:"column:content;not null;type:text"`
	//Images      []string       `gorm:"column:images;type:json"`
	//ContactInfo *string        `gorm:"column:contact_info;type:varchar(255)"`
	//ExtraRecord map[string]any `gorm:"column:extra_record;type:json"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Sheet) TableName() string {
	return "sheet"
}
