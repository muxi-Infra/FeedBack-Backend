package dao

import (
	"context"
	"errors"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"gorm.io/gorm"
)

// IntegrationDAOV3 提供 V3 项目认证和表格配置的只读查询。
type IntegrationDAOV3 interface {
	// todo 后续需要将开启事物的方式统一到外面层级
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	RegisterProject(ctx context.Context, project model.FeedbackProjectV3, key model.FeedbackProjectKeyV3, tables []model.FeedbackProjectTableV3, scopes []model.FeedbackProjectScopeV3, tx ...*gorm.DB) error
	ListProjectsByPage(ctx context.Context, lastID uint64, limit int, tx ...*gorm.DB) ([]model.FeedbackProjectV3, error)
	GetProjectConfig(ctx context.Context, projectID string, tx ...*gorm.DB) (model.FeedbackProjectV3, *model.FeedbackProjectKeyV3, []model.FeedbackProjectTableV3, map[string][]string, error)
	UpdateProject(ctx context.Context, projectID string, project model.FeedbackProjectV3, tables []model.FeedbackProjectTableV3, scopes []model.FeedbackProjectScopeV3, tx ...*gorm.DB) error
	DeleteProject(ctx context.Context, projectID string, tx ...*gorm.DB) error
	RotateAPIKey(ctx context.Context, projectID string, key model.FeedbackProjectKeyV3, tx ...*gorm.DB) error
	GetProject(ctx context.Context, projectID string, tx ...*gorm.DB) (model.FeedbackProjectV3, error)
	GetKey(ctx context.Context, projectID, keyID string, tx ...*gorm.DB) (model.FeedbackProjectKeyV3, error)
	GetTable(ctx context.Context, projectID, tableType string, tx ...*gorm.DB) (model.FeedbackProjectTableV3, error)
	ListScopes(ctx context.Context, projectID, tableIdentity string, tx ...*gorm.DB) ([]string, error)
}

type integrationDAOV3 struct {
	db *gorm.DB
}

func NewIntegrationDAOV3(db *gorm.DB) IntegrationDAOV3 {
	return &integrationDAOV3{
		db: db,
	}
}

// Transaction 由 Service 决定事务边界，DAO 只负责提供事务对象。
func (d *integrationDAOV3) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	if fn == nil {
		return errors.New("transaction callback is nil")
	}
	return d.db.WithContext(ctx).Transaction(fn)
}

func (d *integrationDAOV3) getDB(ctx context.Context, tx ...*gorm.DB) (*gorm.DB, error) {
	if len(tx) > 1 {
		return nil, errors.New("only one transaction is allowed")
	}
	db := d.db
	if len(tx) == 1 && tx[0] != nil {
		db = tx[0]
	}
	return db.WithContext(ctx), nil
}

// ListProjectsByPage 按主键倒序查询启用中的项目。lastID 为上一页最后一条记录的主键。
func (d *integrationDAOV3) ListProjectsByPage(ctx context.Context, lastID uint64, limit int, tx ...*gorm.DB) ([]model.FeedbackProjectV3, error) {
	var projects []model.FeedbackProjectV3
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return nil, err
	}
	db = db.Where("status = ?", "active")
	if lastID > 0 {
		db = db.Where("id < ?", lastID)
	}
	err = db.Order("id DESC").Limit(limit).Find(&projects).Error

	return projects, err
}

func (d *integrationDAOV3) GetProjectConfig(ctx context.Context, projectID string, tx ...*gorm.DB) (model.FeedbackProjectV3, *model.FeedbackProjectKeyV3, []model.FeedbackProjectTableV3, map[string][]string, error) {
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return model.FeedbackProjectV3{}, nil, nil, nil, err
	}

	var project model.FeedbackProjectV3
	if err := db.Where("project_id = ? AND status = ?", projectID, "active").
		First(&project).Error; err != nil {
		return project, nil, nil, nil, err
	}

	var key model.FeedbackProjectKeyV3
	if err := db.Where("project_id = ? AND status = ?", projectID, "active").
		Order("id DESC").First(&key).Error; err != nil {
		return project, nil, nil, nil, err
	}

	var tables []model.FeedbackProjectTableV3
	if err := db.Where("project_id = ? AND status = ?", projectID, "active").
		Order("id ASC").Find(&tables).Error; err != nil {
		return project, nil, nil, nil, err
	}

	var scopes []model.FeedbackProjectScopeV3
	if err := db.Where("project_id = ?", projectID).Find(&scopes).Error; err != nil {
		return project, nil, nil, nil, err
	}

	scopeMap := make(map[string][]string)
	for _, scope := range scopes {
		scopeMap[scope.TableIdentity] = append(scopeMap[scope.TableIdentity], scope.Scope)
	}

	return project, &key, tables, scopeMap, nil
}

func (d *integrationDAOV3) UpdateProject(ctx context.Context, projectID string, project model.FeedbackProjectV3, tables []model.FeedbackProjectTableV3, scopes []model.FeedbackProjectScopeV3, tx ...*gorm.DB) error {
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return err
	}
	if err := db.Model(&model.FeedbackProjectV3{}).
		Where("project_id = ? AND status = ?", projectID, "active").
		Updates(
			map[string]any{
				"project_name": project.ProjectName,
				"school":       project.School,
				"status":       project.Status,
			}).Error; err != nil {
		return err
	}

	if err := db.Where("project_id = ?", projectID).
		Delete(&model.FeedbackProjectTableV3{}).Error; err != nil {
		return err
	}

	if err := db.Where("project_id = ?", projectID).
		Delete(&model.FeedbackProjectScopeV3{}).Error; err != nil {
		return err
	}

	if len(tables) > 0 {
		if err := db.Create(&tables).Error; err != nil {
			return err
		}
	}
	if len(scopes) > 0 {
		if err := db.Create(&scopes).Error; err != nil {
			return err
		}
	}
	return nil
}

func (d *integrationDAOV3) DeleteProject(ctx context.Context, projectID string, tx ...*gorm.DB) error {
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return err
	}
	// 按 Scope、Table、Key、Project 的顺序删除，避免关联数据残留
	for _, value := range []any{
		&model.FeedbackProjectScopeV3{},
		&model.FeedbackProjectTableV3{},
		&model.FeedbackProjectKeyV3{},
		&model.FeedbackProjectV3{},
	} {
		if err := db.Where("project_id = ?", projectID).Delete(value).Error; err != nil {
			return err
		}
	}
	return nil
}

func (d *integrationDAOV3) RotateAPIKey(ctx context.Context, projectID string, key model.FeedbackProjectKeyV3, tx ...*gorm.DB) error {
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return err
	}
	if err := db.Model(&model.FeedbackProjectKeyV3{}).
		Where("project_id = ? AND status = ?", projectID, "active").
		Update("status", "revoked").Error; err != nil {
		return err
	}
	return db.Create(&key).Error
}

func (d *integrationDAOV3) RegisterProject(ctx context.Context, project model.FeedbackProjectV3, key model.FeedbackProjectKeyV3, tables []model.FeedbackProjectTableV3, scopes []model.FeedbackProjectScopeV3, tx ...*gorm.DB) error {
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return err
	}
	if err := db.Create(&project).Error; err != nil {
		return err
	}
	if err := db.Create(&key).Error; err != nil {
		return err
	}
	if len(tables) > 0 {
		if err := db.Create(&tables).Error; err != nil {
			return err
		}
	}
	if len(scopes) > 0 {
		if err := db.Create(&scopes).Error; err != nil {
			return err
		}
	}
	return nil
}

func (d *integrationDAOV3) GetProject(ctx context.Context, projectID string, tx ...*gorm.DB) (model.FeedbackProjectV3, error) {
	var p model.FeedbackProjectV3
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return p, err
	}
	err = db.Where("project_id = ? AND status = ?", projectID, "active").
		First(&p).Error
	return p, err
}

func (d *integrationDAOV3) GetKey(ctx context.Context, projectID, keyID string, tx ...*gorm.DB) (model.FeedbackProjectKeyV3, error) {
	var k model.FeedbackProjectKeyV3
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return k, err
	}
	err = db.Where("project_id = ? AND key_id = ? AND status = ?", projectID, keyID, "active").First(&k).Error
	if err == nil && k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt) {
		return k, errors.New("api key is expired")
	}
	return k, err
}

func (d *integrationDAOV3) GetTable(ctx context.Context, projectID, tableType string, tx ...*gorm.DB) (model.FeedbackProjectTableV3, error) {
	var t model.FeedbackProjectTableV3
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return t, err
	}
	err = db.Where("project_id = ? AND table_type = ? AND status = ?", projectID, tableType, "active").First(&t).Error
	return t, err
}

func (d *integrationDAOV3) ListScopes(ctx context.Context, projectID, tableIdentity string, tx ...*gorm.DB) ([]string, error) {
	var scopes []string
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return nil, err
	}
	err = db.Model(&model.FeedbackProjectScopeV3{}).
		Where("project_id = ? AND table_identity = ?", projectID, tableIdentity).Pluck("scope", &scopes).Error
	return scopes, err
}
