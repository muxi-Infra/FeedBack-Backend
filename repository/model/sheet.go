package model

import (
	"time"
)

type Sheet struct {
	ID            uint64  `gorm:"primaryKey;autoIncrement;index:idx_user_table_id,priority:3"`
	TableIdentify *string `gorm:"column:table_identify;not null;type:varchar(32);uniqueIndex:idx_user_record,priority:1;index:idx_table_sync,priority:1;index:idx_user_table_id,priority:1;index:idx_table_notice,priority:1"`
	RecordID      *string `gorm:"column:record_id;not null;type:varchar(32);uniqueIndex:idx_user_record,priority:3;index:idx_table_sync,priority:3"`
	UserID        *string `gorm:"column:user_id;not null;type:varchar(32);uniqueIndex:idx_user_record,priority:2;index:idx_user_table_id,priority:2"`

	Record   map[string]any `gorm:"column:record;not null;type:json;serializer:json"`
	ShareUrl *string        `gorm:"column:share_url;type:varchar(255);"`

	IsNoticed bool `gorm:"type:tinyint(1);column:is_noticed;not null;default:false;index:idx_table_notice,priority:3"`
	IsSynced  bool `gorm:"type:tinyint(1);column:is_synced;not null;default:false;index:idx_table_sync,priority:2;index:idx_table_notice,priority:2"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Sheet) TableName() string {
	return "sheet"
}
