package model

import (
	"time"

	"gorm.io/plugin/soft_delete"
)

// AdminUserV3 保存管理后台账号。
type AdminUserV3 struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement"`
	Username     string     `gorm:"column:username;type:varchar(64);not null;uniqueIndex:uk_v3_admin_username,priority:1"`
	DisplayName  string     `gorm:"column:display_name;type:varchar(128);not null"`
	PasswordHash string     `gorm:"column:password_hash;type:varchar(255);not null"`
	Status       string     `gorm:"column:status;type:varchar(32);not null;default:active;index"`
	LastLoginAt  *time.Time `gorm:"column:last_login_at"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    soft_delete.DeletedAt `gorm:"column:deleted_at;softDelete:nano;index;uniqueIndex:uk_v3_admin_username,priority:2"`
}

func (AdminUserV3) TableName() string {
	return "admin_users"
}

// FeedbackProjectV3 保存 V3 接入项目的基本信息。
type FeedbackProjectV3 struct {
	ID          uint64 `gorm:"primaryKey;autoIncrement"`
	ProjectID   string `gorm:"column:project_id;type:varchar(64);not null;uniqueIndex:uk_v3_project,priority:1"`
	ProjectName string `gorm:"column:project_name;type:varchar(128);not null"`
	School      string `gorm:"column:school;type:varchar(128);not null"`
	Status      string `gorm:"column:status;type:varchar(32);not null;default:active;index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   soft_delete.DeletedAt `gorm:"column:deleted_at;softDelete:nano;index;uniqueIndex:uk_v3_project,priority:2"`
}

func (FeedbackProjectV3) TableName() string {
	return "feedback_projects"
}

// FeedbackProjectKeyV3 保存项目 API Key 的摘要，不保存 API Key 明文。
type FeedbackProjectKeyV3 struct {
	ID         uint64     `gorm:"primaryKey;autoIncrement"`
	ProjectID  string     `gorm:"column:project_id;type:varchar(64);not null;index"`
	KeyID      string     `gorm:"column:key_id;type:varchar(128);not null;uniqueIndex:uk_v3_key,priority:1"`
	APIKeyHash string     `json:"-" gorm:"column:api_key_hash;type:char(64);not null"`
	Status     string     `gorm:"column:status;type:varchar(32);not null;default:active;index"`
	ExpiresAt  *time.Time `gorm:"column:expires_at"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  soft_delete.DeletedAt `gorm:"column:deleted_at;softDelete:nano;index;uniqueIndex:uk_v3_key,priority:2"`
}

func (FeedbackProjectKeyV3) TableName() string {
	return "feedback_project_keys"
}

// FeedbackProjectTableV3 将项目的逻辑表类型绑定到实际飞书表格。
type FeedbackProjectTableV3 struct {
	ID            uint64 `gorm:"primaryKey;autoIncrement"`
	ProjectID     string `gorm:"column:project_id;type:varchar(64);not null;index;uniqueIndex:uk_v3_table,priority:1"`
	TableIdentity string `gorm:"column:table_identity;type:varchar(128);not null;uniqueIndex:uk_v3_table,priority:2"`
	PhysicalName  string `gorm:"column:table_name;type:varchar(128);not null"`
	TableToken    string `gorm:"column:table_token;type:varchar(512);not null"`
	TableID       string `gorm:"column:table_id;type:varchar(128);not null"`
	ViewID        string `gorm:"column:view_id;type:varchar(128);not null"`
	TableType     string `gorm:"column:table_type;type:varchar(32);not null;default:feedback;index"`
	Notice        bool   `gorm:"column:notice;not null;default:false"`
	Status        string `gorm:"column:status;type:varchar(32);not null;default:active;index"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     soft_delete.DeletedAt `gorm:"column:deleted_at;softDelete:nano;index;uniqueIndex:uk_v3_table,priority:3"`
}

func (FeedbackProjectTableV3) TableName() string {
	return "feedback_project_tables"
}

// FeedbackProjectScopeV3 保存项目针对单张表的权限。
type FeedbackProjectScopeV3 struct {
	ID            uint64 `gorm:"primaryKey;autoIncrement"`
	ProjectID     string `gorm:"column:project_id;type:varchar(64);not null;index;uniqueIndex:uk_v3_scope,priority:1"`
	TableIdentity string `gorm:"column:table_identity;type:varchar(128);not null;uniqueIndex:uk_v3_scope,priority:2"`
	Scope         string `gorm:"column:scope;type:varchar(64);not null;uniqueIndex:uk_v3_scope,priority:3"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     soft_delete.DeletedAt `gorm:"column:deleted_at;softDelete:nano;index;uniqueIndex:uk_v3_scope,priority:4"`
}

func (FeedbackProjectScopeV3) TableName() string {
	return "feedback_project_scopes"
}
