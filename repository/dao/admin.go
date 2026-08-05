package dao

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"gorm.io/gorm"
)

type AdminUserDAO interface {
	Create(ctx context.Context, user *model.AdminUser, tx ...*gorm.DB) error
	GetByID(ctx context.Context, id uint64, tx ...*gorm.DB) (*model.AdminUser, error)
	GetByUsername(ctx context.Context, username string, tx ...*gorm.DB) (*model.AdminUser, error)
	UpdatePassword(ctx context.Context, id uint64, passwordHash string, tx ...*gorm.DB) error
	UpdateLoginInfo(ctx context.Context, id uint64, loginAt time.Time, tx ...*gorm.DB) error
	UpdateStatus(ctx context.Context, id uint64, status string, tx ...*gorm.DB) error
	Delete(ctx context.Context, id uint64, tx ...*gorm.DB) error
	Restore(ctx context.Context, id uint64, tx ...*gorm.DB) error
}

type adminUserDAO struct {
	db *gorm.DB
}

func NewAdminUserDAO(db *gorm.DB) AdminUserDAO {
	return &adminUserDAO{db: db}
}

func (d *adminUserDAO) getDB(ctx context.Context, tx ...*gorm.DB) (*gorm.DB, error) {
	if len(tx) > 1 {
		return nil, errors.New("only one transaction is allowed")
	}

	db := d.db
	if len(tx) == 1 && tx[0] != nil {
		db = tx[0]
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}
	return db, nil
}

func (d *adminUserDAO) Create(ctx context.Context, user *model.AdminUser, tx ...*gorm.DB) error {
	if user == nil {
		return errors.New("admin user is nil")
	}
	if strings.TrimSpace(user.Username) == "" {
		return errors.New("admin username is required")
	}
	if strings.TrimSpace(user.PasswordHash) == "" {
		return errors.New("admin password hash is required")
	}

	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return err
	}
	return db.Create(user).Error
}

func (d *adminUserDAO) GetByID(ctx context.Context, id uint64, tx ...*gorm.DB) (*model.AdminUser, error) {
	if id == 0 {
		return nil, errors.New("admin user id is required")
	}

	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return nil, err
	}

	var user model.AdminUser
	err = db.Where("id = ?", id).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *adminUserDAO) GetByUsername(ctx context.Context, username string, tx ...*gorm.DB) (*model.AdminUser, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, errors.New("admin username is required")
	}

	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return nil, err
	}

	var user model.AdminUser
	err = db.Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *adminUserDAO) UpdatePassword(ctx context.Context, id uint64, passwordHash string, tx ...*gorm.DB) error {
	if id == 0 {
		return errors.New("admin user id is required")
	}
	if strings.TrimSpace(passwordHash) == "" {
		return errors.New("admin password hash is required")
	}
	return d.update(ctx, id, map[string]any{"password_hash": passwordHash}, tx...)
}

func (d *adminUserDAO) UpdateLoginInfo(ctx context.Context, id uint64, loginAt time.Time, tx ...*gorm.DB) error {
	if id == 0 {
		return errors.New("admin user id is required")
	}
	return d.update(ctx, id, map[string]any{"last_login_at": loginAt}, tx...)
}

func (d *adminUserDAO) UpdateStatus(ctx context.Context, id uint64, status string, tx ...*gorm.DB) error {
	if id == 0 {
		return errors.New("admin user id is required")
	}
	if strings.TrimSpace(status) == "" {
		return errors.New("admin user status is required")
	}
	return d.update(ctx, id, map[string]any{"status": status}, tx...)
}

func (d *adminUserDAO) update(ctx context.Context, id uint64, values map[string]any, tx ...*gorm.DB) error {
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return err
	}
	return db.Model(&model.AdminUser{}).Where("id = ?", id).Updates(values).Error
}

func (d *adminUserDAO) Delete(ctx context.Context, id uint64, tx ...*gorm.DB) error {
	if id == 0 {
		return errors.New("admin user id is required")
	}
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return err
	}
	return db.Where("id = ?", id).Delete(&model.AdminUser{}).Error
}

func (d *adminUserDAO) Restore(ctx context.Context, id uint64, tx ...*gorm.DB) error {
	if id == 0 {
		return errors.New("admin user id is required")
	}
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return err
	}
	return db.Unscoped().Model(&model.AdminUser{}).
		Where("id = ?", id).
		Update("deleted_at", 0).Error
}
