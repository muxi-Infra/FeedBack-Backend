package model

import "time"

type FAQ struct {
	ID            uint64  `gorm:"primaryKey;autoIncrement"`
	TableIdentify *string `gorm:"column:table_identify;not null;type:varchar(32);index:idx_faq_table_identify,priority:1"`
	RecordID      *string `gorm:"column:record_id;not null;type:varchar(32);index:idx_faq_record_id,priority:1"`

	Record   map[string]any `gorm:"column:record;not null;type:json;serializer:json"`
	ShareUrl *string        `gorm:"column:share_url;type:varchar(255);"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (FAQ) TableName() string {
	return "faq"
}
