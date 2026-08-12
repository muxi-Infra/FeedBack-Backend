package dao

import (
	"context"
	"errors"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"gorm.io/gorm"
)

type AdminUserDAOV3 interface {
	Create(ctx context.Context, user *model.AdminUserV3) error
	GetByUsername(ctx context.Context, username string) (*model.AdminUserV3, error)
	UpdateLoginInfo(ctx context.Context, id uint64, at time.Time) error
}

type adminUserDAOV3 struct {
	db *gorm.DB
}

func NewAdminUserDAOV3(db *gorm.DB) AdminUserDAOV3 {
	return &adminUserDAOV3{
		db: db,
	}
}

func (d *adminUserDAOV3) Create(ctx context.Context, user *model.AdminUserV3) error {
	return d.db.WithContext(ctx).Create(user).Error
}

func (d *adminUserDAOV3) GetByUsername(ctx context.Context, username string) (*model.AdminUserV3, error) {
	var user model.AdminUserV3
	err := d.db.WithContext(ctx).Where("username = ? AND status = ?", username, "active").First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *adminUserDAOV3) UpdateLoginInfo(ctx context.Context, id uint64, at time.Time) error {
	return d.db.WithContext(ctx).Model(&model.AdminUserV3{}).Where("id = ?", id).Update("last_login_at", at).Error
}
