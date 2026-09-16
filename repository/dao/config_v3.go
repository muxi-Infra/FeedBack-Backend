package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ConfigDAOV3 interface {
	Snapshot(context.Context, string) (domain.ProjectSnapshotV3, error)
	Metadata(context.Context, string) (model.FeedbackProjectV3, error)
	ListMetadata(context.Context, uint64, int) ([]model.FeedbackProjectV3, error)
	LockProject(context.Context, *gorm.DB, string) (model.FeedbackProjectV3, error)
	RecordChange(context.Context, *gorm.DB, domain.ConfigActorV3, domain.ConfigEventV3, uint64, string) error
	ListAudits(context.Context, string, string, uint64, int) ([]model.ConfigAuditV3, error)
	ClaimOutbox(context.Context, string, time.Time, time.Duration, int) ([]model.ConfigOutboxV3, error)
	FinishOutbox(context.Context, model.ConfigOutboxV3, string, time.Time) error
	RetryOutbox(context.Context, model.ConfigOutboxV3, time.Time, string) error
	OutboxStats(context.Context) (int64, *time.Time, error)
	PruneOutbox(context.Context, time.Time) error
}
type configDAOV3 struct{ db *gorm.DB }

func NewConfigDAOV3(db *gorm.DB) ConfigDAOV3 { return &configDAOV3{db: db} }

func (d *configDAOV3) Metadata(ctx context.Context, id string) (model.FeedbackProjectV3, error) {
	var p model.FeedbackProjectV3
	err := d.db.WithContext(ctx).Unscoped().Where("project_id = ?", id).Order("id DESC").First(&p).Error
	return p, err
}
func (d *configDAOV3) ListMetadata(ctx context.Context, after uint64, limit int) ([]model.FeedbackProjectV3, error) {
	var rows []model.FeedbackProjectV3
	// Include tombstones so a lost deletion notification is recoverable even for cold projects.
	err := d.db.WithContext(ctx).Unscoped().Where("id > ?", after).Order("id ASC").Limit(limit).Find(&rows).Error
	return rows, err
}
func (d *configDAOV3) Snapshot(ctx context.Context, id string) (domain.ProjectSnapshotV3, error) {
	var s domain.ProjectSnapshotV3
	var opts *sql.TxOptions
	if d.db.Dialector.Name() == "mysql" {
		opts = &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true}
	}
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Unscoped().Where("project_id = ?", id).Order("id DESC").First(&s.Project).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.Project.ProjectID = id
			return nil
		}
		if err != nil || !s.Active() {
			return err
		}
		if err = tx.Where("project_id = ? AND status = ?", id, "active").Order("id ASC").Find(&s.Tables).Error; err != nil {
			return err
		}
		var scopes []model.FeedbackProjectScopeV3
		if err = tx.Where("project_id = ?", id).Find(&scopes).Error; err != nil {
			return err
		}
		s.Scopes = make(map[string][]string)
		for _, scope := range scopes {
			s.Scopes[scope.TableIdentity] = append(s.Scopes[scope.TableIdentity], scope.Scope)
		}
		return nil
	}, opts)
	return s, err
}
func (d *configDAOV3) LockProject(ctx context.Context, tx *gorm.DB, id string) (model.FeedbackProjectV3, error) {
	var p model.FeedbackProjectV3
	err := tx.WithContext(ctx).Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).Where("project_id = ?", id).Order("id DESC").First(&p).Error
	return p, err
}
func (d *configDAOV3) RecordChange(ctx context.Context, tx *gorm.DB, actor domain.ConfigActorV3, event domain.ConfigEventV3, previous uint64, fields string) error {
	if actor.AdminID == 0 || actor.RequestID == "" {
		return errors.New("config change actor required")
	}
	q := tx.WithContext(ctx).Unscoped().Model(&model.FeedbackProjectV3{}).Where("project_id = ? AND config_version = ?", event.ProjectID, previous)
	if previous == 0 {
		q = tx.WithContext(ctx).Unscoped().Model(&model.FeedbackProjectV3{}).Where("project_id = ? AND config_version = ?", event.ProjectID, event.Version)
	}
	r := q.Updates(map[string]any{"config_version": event.Version, "updated_at": event.ChangedAt})
	if r.Error != nil {
		return r.Error
	}
	if previous != 0 && r.RowsAffected != 1 {
		return fmt.Errorf("config version conflict")
	}
	audit := model.ConfigAuditV3{ChangeID: event.ChangeID, ProjectID: event.ProjectID, AdminID: actor.AdminID, RequestID: actor.RequestID, Kind: event.Kind, PreviousVersion: previous, Version: event.Version, Fields: fields, Result: "committed", CreatedAt: event.ChangedAt}
	if err := tx.Create(&audit).Error; err != nil {
		return err
	}
	return tx.Create(&model.ConfigOutboxV3{ChangeID: event.ChangeID, ProjectID: event.ProjectID, Version: event.Version, Kind: event.Kind, CreatedAt: event.ChangedAt, AvailableAt: event.ChangedAt}).Error
}
func (d *configDAOV3) ListAudits(ctx context.Context, project, request string, before uint64, limit int) ([]model.ConfigAuditV3, error) {
	q := d.db.WithContext(ctx)
	if project != "" {
		q = q.Where("project_id = ?", project)
	}
	if request != "" {
		q = q.Where("request_id = ?", request)
	}
	if before > 0 {
		q = q.Where("id < ?", before)
	}
	var rows []model.ConfigAuditV3
	err := q.Order("id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}
func (d *configDAOV3) ClaimOutbox(ctx context.Context, owner string, now time.Time, lease time.Duration, limit int) ([]model.ConfigOutboxV3, error) {
	var candidates, claimed []model.ConfigOutboxV3
	err := d.db.WithContext(ctx).Where("published_at IS NULL AND available_at <= ?", now).Order("id ASC").Limit(limit).Find(&candidates).Error
	if err != nil {
		return nil, err
	}
	for _, row := range candidates {
		r := d.db.WithContext(ctx).Model(&model.ConfigOutboxV3{}).Where("id = ? AND published_at IS NULL AND available_at <= ? AND attempts = ?", row.ID, now, row.Attempts).Updates(map[string]any{"lease_owner": owner, "available_at": now.Add(lease), "attempts": gorm.Expr("attempts + 1")})
		if r.Error != nil {
			return claimed, r.Error
		}
		if r.RowsAffected == 1 {
			row.LeaseOwner = owner
			row.Attempts++
			claimed = append(claimed, row)
		}
	}
	return claimed, nil
}
func (d *configDAOV3) owned(ctx context.Context, row model.ConfigOutboxV3) *gorm.DB {
	return d.db.WithContext(ctx).Model(&model.ConfigOutboxV3{}).Where("id = ? AND lease_owner = ? AND attempts = ? AND published_at IS NULL", row.ID, row.LeaseOwner, row.Attempts)
}
func (d *configDAOV3) FinishOutbox(ctx context.Context, row model.ConfigOutboxV3, message string, at time.Time) error {
	return configLeaseResult(d.owned(ctx, row).Updates(map[string]any{"published_at": at, "message_id": message, "lease_owner": "", "last_error": ""}))
}
func (d *configDAOV3) RetryOutbox(ctx context.Context, row model.ConfigOutboxV3, next time.Time, class string) error {
	return configLeaseResult(d.owned(ctx, row).Updates(map[string]any{"available_at": next, "last_error": class, "lease_owner": ""}))
}
func configLeaseResult(result *gorm.DB) error {
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("config outbox lease lost")
	}
	return nil
}
func (d *configDAOV3) OutboxStats(ctx context.Context) (int64, *time.Time, error) {
	q := d.db.WithContext(ctx).Model(&model.ConfigOutboxV3{}).Where("published_at IS NULL")
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return 0, nil, err
	}
	var first model.ConfigOutboxV3
	err := q.Select("created_at").Order("created_at ASC").Take(&first).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return count, nil, nil
	}
	return count, &first.CreatedAt, err
}
func (d *configDAOV3) PruneOutbox(ctx context.Context, before time.Time) error {
	var ids []uint64
	if err := d.db.WithContext(ctx).Model(&model.ConfigOutboxV3{}).Where("published_at < ?", before).Order("id ASC").Limit(100).Pluck("id", &ids).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	return d.db.WithContext(ctx).Where("id IN ? AND published_at < ?", ids, before).Delete(&model.ConfigOutboxV3{}).Error
}
