package model

import "time"

type ConfigAuditV3 struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChangeID        string    `gorm:"type:varchar(36);not null;uniqueIndex" json:"change_id"`
	ProjectID       string    `gorm:"type:varchar(64);not null;index" json:"project_id"`
	AdminID         uint64    `gorm:"not null" json:"admin_id"`
	RequestID       string    `gorm:"type:varchar(64);not null;index" json:"request_id"`
	Kind            string    `gorm:"type:varchar(32);not null" json:"kind"`
	PreviousVersion uint64    `gorm:"not null" json:"previous_version"`
	Version         uint64    `gorm:"not null" json:"version"`
	Fields          string    `gorm:"type:varchar(255);not null" json:"fields"`
	Result          string    `gorm:"type:varchar(16);not null" json:"result"`
	CreatedAt       time.Time `json:"created_at"`
}

func (ConfigAuditV3) TableName() string { return "feedback_project_config_audits" }

type ConfigOutboxV3 struct {
	ID          uint64 `gorm:"primaryKey;autoIncrement"`
	ChangeID    string `gorm:"type:varchar(36);not null;uniqueIndex"`
	ProjectID   string `gorm:"type:varchar(64);not null"`
	Version     uint64 `gorm:"not null"`
	Kind        string `gorm:"type:varchar(32);not null"`
	CreatedAt   time.Time
	AvailableAt time.Time  `gorm:"not null;index:idx_v3_outbox_due,priority:2"`
	PublishedAt *time.Time `gorm:"index:idx_v3_outbox_due,priority:1"`
	LeaseOwner  string     `gorm:"type:varchar(128);not null;default:''"`
	Attempts    int        `gorm:"not null;default:0"`
	LastError   string     `gorm:"type:varchar(32);not null;default:''"`
	MessageID   string     `gorm:"type:varchar(64);not null;default:''"`
}

func (ConfigOutboxV3) TableName() string { return "feedback_project_config_outbox" }
