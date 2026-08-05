package dao

import (
	"context"
	"errors"

	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"gorm.io/gorm"
)

type IntegrationDAO interface {
	// Transaction 在一个数据库事务中执行 fn。
	Transaction(ctx context.Context, fn func(tx IntegrationDAO) error) error

	// model.FeedbackProject
	CreateProject(ctx context.Context, project *model.FeedbackProject, tx ...*gorm.DB) error
	GetProject(ctx context.Context, projectID string, tx ...*gorm.DB) (*model.FeedbackProject, error)
	ListProjects(ctx context.Context, tx ...*gorm.DB) ([]model.FeedbackProject, error)
	UpdateProject(ctx context.Context, project *model.FeedbackProject, tx ...*gorm.DB) error
	DeleteProject(ctx context.Context, projectID string, tx ...*gorm.DB) error
	RestoreProject(ctx context.Context, projectID string, tx ...*gorm.DB) error

	// mdoel.FeedbackProjectKey
	UpsertProjectKey(ctx context.Context, key *model.FeedbackProjectKey, tx ...*gorm.DB) error
	GetProjectKey(ctx context.Context, projectID, keyID string, tx ...*gorm.DB) (*model.FeedbackProjectKey, error)
	ListProjectKeys(ctx context.Context, projectID string, tx ...*gorm.DB) ([]model.FeedbackProjectKey, error)
	DeleteProjectKey(ctx context.Context, projectID, keyID string, tx ...*gorm.DB) error

	// model.FeedbackProjectTable
	UpsertProjectTable(ctx context.Context, table *model.FeedbackProjectTable, tx ...*gorm.DB) error
	GetProjectTable(ctx context.Context, projectID, tableIdentity string, tx ...*gorm.DB) (*model.FeedbackProjectTable, error)
	ListProjectTables(ctx context.Context, projectID string, tx ...*gorm.DB) ([]model.FeedbackProjectTable, error)
	DeleteProjectTable(ctx context.Context, projectID, tableIdentity string, tx ...*gorm.DB) error

	// model.FeedbackProjectScope
	ListProjectScopes(ctx context.Context, projectID, tableIdentity string, tx ...*gorm.DB) ([]model.FeedbackProjectScope, error)
	ReplaceProjectScopes(ctx context.Context, projectID, tableIdentity string, scopes []model.FeedbackProjectScope, tx ...*gorm.DB) error
}

type integrationDAO struct {
	db            *gorm.DB
	inTransaction bool
}

func NewIntegrationDAO(db *gorm.DB) IntegrationDAO {
	return &integrationDAO{db: db}
}

// Transaction 在一个数据库事务中执行 fn，
// 并将一个事务范围内的 IntegrationDAO 传递给回调函数。
func (d *integrationDAO) Transaction(ctx context.Context, fn func(tx IntegrationDAO) error) error {
	if fn == nil {
		return errors.New("transaction callback is nil")
	}

	db, err := d.getDB(ctx)
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		return fn(&integrationDAO{
			db:            tx,
			inTransaction: true,
		})
	})
}

func (d *integrationDAO) getDB(ctx context.Context, tx ...*gorm.DB) (*gorm.DB, error) {
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

func (d *integrationDAO) CreateProject(ctx context.Context, project *model.FeedbackProject, tx ...*gorm.DB) error {
	if project == nil {
		return errors.New("project is nil")
	}
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return err
	}
	return db.Create(project).Error
}

func (d *integrationDAO) GetProject(ctx context.Context, projectID string, tx ...*gorm.DB) (*model.FeedbackProject, error) {
	if projectID == "" {
		return nil, errors.New("project_id is required")
	}

	var project model.FeedbackProject
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return nil, err
	}
	err = db.Where("project_id = ?", projectID).First(&project).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// ListProjects 期待后续有优化这个地方的时候
func (d *integrationDAO) ListProjects(ctx context.Context, tx ...*gorm.DB) ([]model.FeedbackProject, error) {
	var projects []model.FeedbackProject
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return nil, err
	}
	err = db.Order("id DESC").Find(&projects).Error
	return projects, err
}

func (d *integrationDAO) UpdateProject(ctx context.Context, project *model.FeedbackProject, tx ...*gorm.DB) error {
	if project == nil || project.ID == 0 {
		return errors.New("project with id is required")
	}
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return err
	}
	return db.Model(&model.FeedbackProject{}).
		Where("id = ?", project.ID).
		Updates(project).Error
}

func (d *integrationDAO) DeleteProject(ctx context.Context, projectID string, tx ...*gorm.DB) error {
	if projectID == "" {
		return errors.New("project_id is required")
	}
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return err
	}
	return db.Where("project_id = ?", projectID).Delete(&model.FeedbackProject{}).Error
}

func (d *integrationDAO) RestoreProject(ctx context.Context, projectID string, tx ...*gorm.DB) error {
	if projectID == "" {
		return errors.New("project_id is required")
	}
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return err
	}
	return db.Unscoped().Model(&model.FeedbackProject{}).
		Where("project_id = ?", projectID).
		Update("deleted_at", 0).Error
}

func (d *integrationDAO) UpsertProjectKey(ctx context.Context, key *model.FeedbackProjectKey, tx ...*gorm.DB) error {
	if key == nil {
		return errors.New("project key is nil")
	}
	if key.ProjectID == "" || key.KeyID == "" {
		return errors.New("project_id and key_id are required")
	}
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return err
	}

	var existing model.FeedbackProjectKey
	err = db.Unscoped().
		Where("project_id = ? AND key_id = ?", key.ProjectID, key.KeyID).
		First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(key).Error
	}
	if err != nil {
		return err
	}

	return db.Unscoped().Model(&existing).Updates(map[string]interface{}{
		"issuer":     key.Issuer,
		"public_key": key.PublicKey,
		"status":     key.Status,
		"expires_at": key.ExpiresAt,
		"deleted_at": 0,
	}).Error
}

func (d *integrationDAO) GetProjectKey(ctx context.Context, projectID, keyID string, tx ...*gorm.DB) (*model.FeedbackProjectKey, error) {
	if projectID == "" || keyID == "" {
		return nil, errors.New("project_id and key_id are required")
	}

	var key model.FeedbackProjectKey
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return nil, err
	}
	err = db.Where("project_id = ? AND key_id = ?", projectID, keyID).First(&key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &key, nil
}

// 期待后续有优化这个的地方
func (d *integrationDAO) ListProjectKeys(ctx context.Context, projectID string, tx ...*gorm.DB) ([]model.FeedbackProjectKey, error) {
	if projectID == "" {
		return nil, errors.New("project_id is required")
	}

	var keys []model.FeedbackProjectKey
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return nil, err
	}
	err = db.Where("project_id = ?", projectID).Order("id DESC").Find(&keys).Error
	return keys, err
}

func (d *integrationDAO) DeleteProjectKey(ctx context.Context, projectID, keyID string, tx ...*gorm.DB) error {
	if projectID == "" || keyID == "" {
		return errors.New("project_id and key_id are required")
	}
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return err
	}
	return db.Where("project_id = ? AND key_id = ?", projectID, keyID).
		Delete(&model.FeedbackProjectKey{}).Error
}

func (d *integrationDAO) UpsertProjectTable(ctx context.Context, table *model.FeedbackProjectTable, tx ...*gorm.DB) error {
	if table == nil {
		return errors.New("project table is nil")
	}
	if table.ProjectID == "" || table.TableIdentity == "" {
		return errors.New("project_id and table_identity are required")
	}
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return err
	}

	var existing model.FeedbackProjectTable
	err = db.Unscoped().
		Where("project_id = ? AND table_identity = ?", table.ProjectID, table.TableIdentity).
		First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(table).Error
	}
	if err != nil {
		return err
	}

	return db.Unscoped().Model(&existing).Updates(map[string]interface{}{
		"table_name":  table.PhysicalName,
		"table_token": table.TableToken,
		"table_id":    table.TableID,
		"view_id":     table.ViewID,
		"table_type":  table.TableType,
		"notice":      table.Notice,
		"status":      table.Status,
		"deleted_at":  0,
	}).Error
}

func (d *integrationDAO) GetProjectTable(ctx context.Context, projectID, tableIdentity string, tx ...*gorm.DB) (*model.FeedbackProjectTable, error) {
	if projectID == "" || tableIdentity == "" {
		return nil, errors.New("project_id and table_identity are required")
	}

	var table model.FeedbackProjectTable
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return nil, err
	}
	err = db.Where("project_id = ? AND table_identity = ?", projectID, tableIdentity).
		First(&table).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &table, nil
}

func (d *integrationDAO) ListProjectTables(ctx context.Context, projectID string, tx ...*gorm.DB) ([]model.FeedbackProjectTable, error) {
	if projectID == "" {
		return nil, errors.New("project_id is required")
	}

	var tables []model.FeedbackProjectTable
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return nil, err
	}
	err = db.Where("project_id = ?", projectID).Order("id DESC").Find(&tables).Error
	return tables, err
}

func (d *integrationDAO) DeleteProjectTable(ctx context.Context, projectID, tableIdentity string, tx ...*gorm.DB) error {
	if projectID == "" || tableIdentity == "" {
		return errors.New("project_id and table_identity are required")
	}
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return err
	}
	return db.Where("project_id = ? AND table_identity = ?", projectID, tableIdentity).
		Delete(&model.FeedbackProjectTable{}).Error
}

func (d *integrationDAO) ListProjectScopes(ctx context.Context, projectID, tableIdentity string, tx ...*gorm.DB) ([]model.FeedbackProjectScope, error) {
	if projectID == "" || tableIdentity == "" {
		return nil, errors.New("project_id and table_identity are required")
	}

	var scopes []model.FeedbackProjectScope
	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return nil, err
	}
	err = db.Where("project_id = ? AND table_identity = ?", projectID, tableIdentity).
		Order("id ASC").Find(&scopes).Error
	return scopes, err
}

// ReplaceProjectScopes 替换项目表标识的全部权限范围。
// 如果提供了事务，则该替换操作会参与调用方的事务；
// 否则，DAO 会为此操作创建一个事务。
func (d *integrationDAO) ReplaceProjectScopes(ctx context.Context, projectID, tableIdentity string, scopes []model.FeedbackProjectScope, tx ...*gorm.DB) error {
	if projectID == "" || tableIdentity == "" {
		return errors.New("project_id and table_identity are required")
	}

	replace := func(db *gorm.DB) error {
		var existing []model.FeedbackProjectScope
		if err := db.Unscoped().
			Where("project_id = ? AND table_identity = ?", projectID, tableIdentity).
			Find(&existing).Error; err != nil {
			return err
		}

		requested := make(map[string]struct{}, len(scopes))
		for _, scope := range scopes {
			if scope.Scope == "" {
				return errors.New("scope is required")
			}
			requested[scope.Scope] = struct{}{}
		}

		for _, current := range existing {
			if _, ok := requested[current.Scope]; ok {
				if current.DeletedAt != 0 {
					if err := db.Unscoped().Model(&current).Update("deleted_at", 0).Error; err != nil {
						return err
					}
				}
				delete(requested, current.Scope)
				continue
			}
			if current.DeletedAt == 0 {
				if err := db.Delete(&current).Error; err != nil {
					return err
				}
			}
		}

		for scope := range requested {
			item := &model.FeedbackProjectScope{
				ProjectID:     projectID,
				TableIdentity: tableIdentity,
				Scope:         scope,
			}
			if err := db.Create(item).Error; err != nil {
				return err
			}
		}
		return nil
	}

	db, err := d.getDB(ctx, tx...)
	if err != nil {
		return err
	}
	if d.inTransaction || (len(tx) > 0 && tx[0] != nil) {
		return replace(db)
	}
	return db.Transaction(replace)
}
