package model

import (
	"time"

	"gorm.io/plugin/soft_delete"
)

type AdminUser struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement"`
	Username     string     `gorm:"column:username;type:varchar(64);not null;uniqueIndex:uk_admin_user_username"`
	DisplayName  string     `gorm:"column:display_name;type:varchar(128);not null"`
	PasswordHash string     `gorm:"column:password_hash;type:varchar(255);not null"`
	Status       string     `gorm:"column:status;type:varchar(32);not null;default:active;index:idx_admin_user_status"`
	LastLoginAt  *time.Time `gorm:"column:last_login_at"`

	CreatedAt time.Time             `gorm:"column:created_at;not null"`
	UpdatedAt time.Time             `gorm:"column:updated_at;not null"`
	DeletedAt soft_delete.DeletedAt `gorm:"column:deleted_at;softDelete:nano;index"`
}

func (AdminUser) TableName() string {
	return "admin_users"
}
