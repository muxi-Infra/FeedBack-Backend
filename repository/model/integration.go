package model

import (
	"time"

	"gorm.io/plugin/soft_delete"
)

// FeedbackProject 存储已集成校园项目的基本注册信息。
type FeedbackProject struct {
	ID          uint64 `gorm:"primaryKey;autoIncrement"`
	ProjectID   string `gorm:"column:project_id;type:varchar(64);not null;uniqueIndex:uk_feedback_project_project_id,priority:1"`
	ProjectName string `gorm:"column:project_name;type:varchar(128);not null"`
	// 期望可以出华师
	School string `gorm:"column:school;type:varchar(128);not null"`
	Status string `gorm:"column:status;type:varchar(32);not null;default:active;index:idx_feedback_project_status"`

	CreatedAt time.Time             `gorm:"column:created_at;not null"`
	UpdatedAt time.Time             `gorm:"column:updated_at;not null"`
	DeletedAt soft_delete.DeletedAt `gorm:"column:deleted_at;softDelete:nano;index;uniqueIndex:uk_feedback_project_project_id,priority:2"`
}

func (FeedbackProject) TableName() string {
	return "feedback_projects"
}

// FeedbackProjectKey 存储项目 API Key 的摘要。
// API Key 明文只保存在接入项目后端，不写入反馈中台数据库。
type FeedbackProjectKey struct {
	ID         uint64     `gorm:"primaryKey;autoIncrement"`
	ProjectID  string     `gorm:"column:project_id;type:varchar(64);not null;index:idx_feedback_project_key_project"`
	KeyID      string     `gorm:"column:key_id;type:varchar(128);not null;uniqueIndex:uk_feedback_project_key_id,priority:1"`
	APIKeyHash string     `gorm:"column:api_key_hash;type:char(64);not null"`
	Status     string     `gorm:"column:status;type:varchar(32);not null;default:active;index:idx_feedback_project_key_status"`
	ExpiresAt  *time.Time `gorm:"column:expires_at"`

	CreatedAt time.Time             `gorm:"column:created_at;not null"`
	UpdatedAt time.Time             `gorm:"column:updated_at;not null"`
	DeletedAt soft_delete.DeletedAt `gorm:"column:deleted_at;softDelete:nano;index;uniqueIndex:uk_feedback_project_key_id,priority:2"`
}

func (FeedbackProjectKey) TableName() string {
	return "feedback_project_keys"
}

// FeedbackProjectTable 将项目表标识绑定到实际的飞书表格。
// TableToken 属于敏感信息，进行静态加密存储。
type FeedbackProjectTable struct {
	ID            uint64 `gorm:"primaryKey;autoIncrement"`
	ProjectID     string `gorm:"column:project_id;type:varchar(64);not null;index:idx_feedback_project_table_project;uniqueIndex:uk_feedback_project_table,priority:1"`
	TableIdentity string `gorm:"column:table_identity;type:varchar(128);not null;uniqueIndex:uk_feedback_project_table,priority:2"`
	PhysicalName  string `gorm:"column:table_name;type:varchar(128);not null"`
	TableToken    string `gorm:"column:table_token;type:varchar(512);not null"`
	TableID       string `gorm:"column:table_id;type:varchar(128);not null"`
	ViewID        string `gorm:"column:view_id;type:varchar(128);not null"`
	TableType     string `gorm:"column:table_type;type:varchar(32);not null;default:feedback"`
	Notice        bool   `gorm:"column:notice;type:tinyint(1);not null;default:false"`
	Status        string `gorm:"column:status;type:varchar(32);not null;default:active;index:idx_feedback_project_table_status"`

	CreatedAt time.Time             `gorm:"column:created_at;not null"`
	UpdatedAt time.Time             `gorm:"column:updated_at;not null"`
	DeletedAt soft_delete.DeletedAt `gorm:"column:deleted_at;softDelete:nano;index;uniqueIndex:uk_feedback_project_table,priority:3"`
}

func (FeedbackProjectTable) TableName() string {
	return "feedback_project_tables"
}

// FeedbackProjectScope 存储授予项目的、针对特定表标识的权限。
type FeedbackProjectScope struct {
	ID            uint64 `gorm:"primaryKey;autoIncrement"`
	ProjectID     string `gorm:"column:project_id;type:varchar(64);not null;index:idx_feedback_project_scope_project;uniqueIndex:uk_feedback_project_scope,priority:1"`
	TableIdentity string `gorm:"column:table_identity;type:varchar(128);not null;uniqueIndex:uk_feedback_project_scope,priority:2"`
	Scope         string `gorm:"column:scope;type:varchar(64);not null;uniqueIndex:uk_feedback_project_scope,priority:3"`

	CreatedAt time.Time             `gorm:"column:created_at;not null"`
	UpdatedAt time.Time             `gorm:"column:updated_at;not null"`
	DeletedAt soft_delete.DeletedAt `gorm:"column:deleted_at;softDelete:nano;index;uniqueIndex:uk_feedback_project_scope,priority:4"`
}

func (FeedbackProjectScope) TableName() string {
	return "feedback_project_scopes"
}
