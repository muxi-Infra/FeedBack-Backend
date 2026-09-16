package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	respV3 "github.com/muxi-Infra/FeedBack-Backend/api/response/v3"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/apikey"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/configclock"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/constvar"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"gorm.io/gorm"
)

type V3AdminService interface {
	RegisterProject(context.Context, domain.RegisterProjectInput, domain.ConfigActorV3) (respV3.RegisterProjectResp, error)
	ListProjects(context.Context, *string, *int) ([]model.FeedbackProjectV3, bool, string, error)
	GetProjectConfig(context.Context, string) (model.FeedbackProjectV3, *model.FeedbackProjectKeyV3, []model.FeedbackProjectTableV3, map[string][]string, error)
	UpdateProject(context.Context, string, domain.RegisterProjectInput, domain.ConfigActorV3) (domain.ConfigReceiptV3, error)
	DeleteProject(context.Context, string, domain.ConfigActorV3) (domain.ConfigReceiptV3, error)
	RotateAPIKey(context.Context, string, domain.ConfigActorV3) (respV3.RotateAPIKeyResp, error)
	ConfigStatus(context.Context, string) ProjectConfigStatusV3
	ConfigAudits(context.Context, string, string, uint64, int) ([]model.ConfigAuditV3, error)
}

type v3AdminService struct {
	dao      dao.IntegrationDAOV3
	configs  dao.ConfigDAOV3
	cache    *ProjectConfigCacheV3
	clock    configclock.Clock
	log      logger.Logger
	instance *domain.ConfigInstanceV3
}

func NewV3AdminService(d dao.IntegrationDAOV3, configs dao.ConfigDAOV3, local *ProjectConfigCacheV3, clock configclock.Clock, log logger.Logger, instance *domain.ConfigInstanceV3) V3AdminService {
	return &v3AdminService{dao: d, configs: configs, cache: local, clock: clock, log: log, instance: instance}
}

func (s *v3AdminService) change(ctx context.Context, id, kind, fields string, actor domain.ConfigActorV3, mutation func(*gorm.DB) error) (domain.ConfigReceiptV3, error) {
	event := domain.ConfigEventV3{ProjectID: id, Kind: kind, ChangeID: uuid.NewString()}
	err := s.dao.Transaction(ctx, func(tx *gorm.DB) error {
		var previous uint64
		if kind != "create" {
			p, err := s.configs.LockProject(ctx, tx, id)
			if err != nil {
				return err
			}
			previous = p.ConfigVersion
			if p.DeletedAt != 0 {
				if kind == "delete" {
					event.Version = previous
					event.ChangeID = ""
					return nil
				}
				return gorm.ErrRecordNotFound
			}
			if p.Status != "active" && kind != "delete" {
				return gorm.ErrRecordNotFound
			}
		}
		if previous == ^uint64(0) {
			return errors.New("config version exhausted")
		}
		event.Version = previous + 1
		if err := mutation(tx); err != nil {
			return err
		}
		event.ChangedAt = s.clock.Now()
		return s.configs.RecordChange(ctx, tx, actor, event, previous, fields)
	})
	if err != nil {
		s.log.Warn("config_change_failed", logger.String("project_id", id), logger.Uint64("admin_id", actor.AdminID), logger.String("request_id", actor.RequestID), logger.String("kind", kind), logger.String("error_class", domain.ConfigErrorClass(err)))
		return domain.ConfigReceiptV3{}, errs.V3ProjectDatabaseError(errors.New(domain.ConfigErrorClass(err)))
	}
	s.cache.Notify(event)
	phase := "config_change_committed"
	if event.ChangeID == "" {
		phase = "config_change_unchanged"
	}
	s.log.Info(phase, logger.String("project_id", id), logger.String("change_id", event.ChangeID), logger.Uint64("version", event.Version), logger.Uint64("admin_id", actor.AdminID), logger.String("request_id", actor.RequestID), logger.String("kind", kind))
	return domain.ConfigReceiptV3{Version: event.Version, ChangeID: event.ChangeID}, nil
}

func (s *v3AdminService) RegisterProject(ctx context.Context, input domain.RegisterProjectInput, actor domain.ConfigActorV3) (respV3.RegisterProjectResp, error) {
	if err := validateProjectInput(input); err != nil {
		return respV3.RegisterProjectResp{}, err
	}

	projectID := "project-" + uuid.NewString()
	keyID := projectID + "-key"
	plainKey, err := apikey.Generate()
	if err != nil {
		return respV3.RegisterProjectResp{}, errs.V3APIKeyGenerateError(err)
	}

	project := model.FeedbackProjectV3{
		ProjectID:   projectID,
		ProjectName: strings.TrimSpace(input.ProjectName),
		School:      strings.TrimSpace(input.School),
		Status:      "active",
	}

	key := model.FeedbackProjectKeyV3{
		ProjectID:  projectID,
		KeyID:      keyID,
		APIKeyHash: apikey.Digest(plainKey),
		Status:     "active",
	}

	tables, scopes, err := buildProjectTables(projectID, input.Tables)
	if err != nil {
		return respV3.RegisterProjectResp{}, err
	}
	receipt, err := s.change(ctx, projectID, "create", "project,tables,scopes,key", actor, func(tx *gorm.DB) error {
		return s.dao.RegisterProject(ctx, project, key, tables, scopes, tx)
	})
	if err != nil {
		return respV3.RegisterProjectResp{}, err
	}
	return respV3.RegisterProjectResp{
		Receipt:   receipt,
		ProjectID: projectID,
		KeyID:     keyID,
		APIKey:    plainKey,
	}, nil
}

const (
	defaultProjectPageSize = 10
	maxProjectPageSize     = 50
)

func (s *v3AdminService) ListProjects(ctx context.Context, pageToken *string, limitSize *int) ([]model.FeedbackProjectV3, bool, string, error) {
	lastID := uint64(0)
	if pageToken != nil && strings.TrimSpace(*pageToken) != "" {
		decodedLastID, err := decodeProjectPageToken(*pageToken)
		if err != nil {
			return nil, false, "", errs.V3InvalidInputError(fmt.Errorf("无效的 page_token: %w", err))
		}
		lastID = decodedLastID
	}

	limit := defaultProjectPageSize
	if limitSize != nil {
		limit = *limitSize
	}
	if limit <= 0 || limit > maxProjectPageSize {
		return nil, false, "", errs.V3InvalidInputError(fmt.Errorf("limit_size 必须在 1 到 %d 之间", maxProjectPageSize))
	}

	projects, err := s.dao.ListProjectsByPage(ctx, lastID, limit+1)
	if err != nil {
		return nil, false, "", errs.V3ProjectDatabaseError(err)
	}

	hasMore := len(projects) > limit
	if hasMore {
		projects = projects[:limit]
	}
	nextToken := ""
	if hasMore && len(projects) > 0 {
		nextToken, err = encodeProjectPageToken(projects[len(projects)-1].ID)
		if err != nil {
			return nil, false, "", errs.V3InvalidInputError(fmt.Errorf("生成 page_token 失败: %w", err))
		}
	}
	return projects, hasMore, nextToken, nil
}

func encodeProjectPageToken(lastID uint64) (string, error) {
	data, err := json.Marshal(domain.PageToken{LastID: lastID})
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func decodeProjectPageToken(token string) (uint64, error) {
	data, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return 0, err
	}
	var pageToken domain.PageToken
	if err := json.Unmarshal(data, &pageToken); err != nil {
		return 0, err
	}
	if pageToken.LastID == 0 {
		return 0, errors.New("last_id 不能为空")
	}
	return pageToken.LastID, nil
}

func (s *v3AdminService) GetProjectConfig(ctx context.Context, projectID string) (model.FeedbackProjectV3, *model.FeedbackProjectKeyV3, []model.FeedbackProjectTableV3, map[string][]string, error) {
	project, key, tables, scopes, err := s.dao.GetProjectConfig(ctx, projectID)
	if err != nil {
		return project, key, tables, scopes, errs.V3ProjectDatabaseError(err)
	}
	return project, key, tables, scopes, nil
}

func (s *v3AdminService) UpdateProject(ctx context.Context, projectID string, input domain.RegisterProjectInput, actor domain.ConfigActorV3) (domain.ConfigReceiptV3, error) {
	if strings.TrimSpace(projectID) == "" {
		return domain.ConfigReceiptV3{}, errs.V3InvalidInputError(errors.New("project_id required"))
	}
	if err := validateProjectInput(input); err != nil {
		return domain.ConfigReceiptV3{}, err
	}
	project := model.FeedbackProjectV3{ProjectID: projectID, ProjectName: strings.TrimSpace(input.ProjectName), School: strings.TrimSpace(input.School), Status: "active"}
	tables, scopes, err := buildProjectTables(projectID, input.Tables)
	if err != nil {
		return domain.ConfigReceiptV3{}, err
	}
	return s.change(ctx, projectID, "update", "project,tables,scopes", actor, func(tx *gorm.DB) error {
		return s.dao.UpdateProject(ctx, projectID, project, tables, scopes, tx)
	})
}
func (s *v3AdminService) DeleteProject(ctx context.Context, projectID string, actor domain.ConfigActorV3) (domain.ConfigReceiptV3, error) {
	if strings.TrimSpace(projectID) == "" {
		return domain.ConfigReceiptV3{}, errs.V3InvalidInputError(errors.New("project_id required"))
	}
	return s.change(ctx, projectID, "delete", "project,tables,scopes,key", actor, func(tx *gorm.DB) error {
		return s.dao.DeleteProject(ctx, projectID, tx)
	})
}
func (s *v3AdminService) RotateAPIKey(ctx context.Context, projectID string, actor domain.ConfigActorV3) (respV3.RotateAPIKeyResp, error) {
	if strings.TrimSpace(projectID) == "" {
		return respV3.RotateAPIKeyResp{}, errs.V3InvalidInputError(errors.New("project_id required"))
	}
	plain, err := apikey.Generate()
	if err != nil {
		return respV3.RotateAPIKeyResp{}, errs.V3APIKeyGenerateError(err)
	}
	keyID := projectID + "-key-" + uuid.NewString()[:12]
	receipt, err := s.change(ctx, projectID, "rotate_key", "key", actor, func(tx *gorm.DB) error {
		return s.dao.RotateAPIKey(ctx, projectID, model.FeedbackProjectKeyV3{ProjectID: projectID, KeyID: keyID, APIKeyHash: apikey.Digest(plain), Status: "active"}, tx)
	})
	if err != nil {
		return respV3.RotateAPIKeyResp{}, err
	}
	return respV3.RotateAPIKeyResp{KeyID: keyID, APIKey: plain, Receipt: receipt}, nil
}
func (s *v3AdminService) ConfigStatus(ctx context.Context, id string) ProjectConfigStatusV3 {
	return s.cache.Status(ctx, id, s.instance.ID)
}
func (s *v3AdminService) ConfigAudits(ctx context.Context, project, request string, before uint64, limit int) ([]model.ConfigAuditV3, error) {
	if limit < 1 || limit > 100 {
		return nil, errs.V3InvalidInputError(errors.New("limit must be between 1 and 100"))
	}
	rows, err := s.configs.ListAudits(ctx, project, request, before, limit)
	if err != nil {
		return nil, errs.V3ProjectDatabaseError(errors.New(domain.ConfigErrorClass(err)))
	}
	return rows, nil
}

func buildProjectTables(projectID string, inputs []domain.RegisterProjectTableInput) ([]model.FeedbackProjectTableV3, []model.FeedbackProjectScopeV3, error) {
	tables := make([]model.FeedbackProjectTableV3, 0, len(inputs))
	scopes := make([]model.FeedbackProjectScopeV3, 0)
	seen := map[string]bool{}
	for _, item := range inputs {
		if seen[item.TableIdentity] {
			return nil, nil, errs.V3InvalidInputError(fmt.Errorf("重复的 table_identity: %s", item.TableIdentity))
		}
		seen[item.TableIdentity] = true
		if item.TableType != constvar.FeedbackTableType && item.TableType != constvar.FAQTableType {
			return nil, nil, errs.V3InvalidInputError(fmt.Errorf("不支持的 table_type: %s", item.TableType))
		}
		tables = append(tables, model.FeedbackProjectTableV3{
			ProjectID:     projectID,
			TableIdentity: strings.TrimSpace(item.TableIdentity),
			PhysicalName:  strings.TrimSpace(item.TableName),
			TableToken:    strings.TrimSpace(item.TableToken),
			TableID:       strings.TrimSpace(item.TableID),
			ViewID:        strings.TrimSpace(item.ViewID),
			TableType:     strings.TrimSpace(item.TableType),
			Notice:        item.Notice,
			Status:        "active",
		})
		for _, scope := range item.Scopes {
			scopes = append(scopes, model.FeedbackProjectScopeV3{
				ProjectID:     projectID,
				TableIdentity: strings.TrimSpace(item.TableIdentity),
				Scope:         strings.TrimSpace(scope),
			})
		}
	}
	return tables, scopes, nil
}

// validateProjectInput 校验 V3 项目及其飞书表配置，避免无效配置进入数据库后才在运行时失败。
func validateProjectInput(input domain.RegisterProjectInput) error {
	if strings.TrimSpace(input.ProjectName) == "" {
		return errs.V3InvalidInputError(errors.New("项目名称不能为空"))
	}
	if strings.TrimSpace(input.School) == "" {
		return errs.V3InvalidInputError(errors.New("学校不能为空"))
	}
	if len(input.Tables) == 0 {
		return errs.V3InvalidInputError(errors.New("至少需要配置一张表"))
	}

	seenTables := make(map[string]struct{}, len(input.Tables))
	for _, table := range input.Tables {
		identity := strings.TrimSpace(table.TableIdentity)
		if identity == "" {
			return errs.V3InvalidInputError(errors.New("table_identity 不能为空"))
		}
		if _, exists := seenTables[identity]; exists {
			return errs.V3InvalidInputError(fmt.Errorf("重复的 table_identity: %s", identity))
		}
		seenTables[identity] = struct{}{}
		if strings.TrimSpace(table.TableName) == "" || strings.TrimSpace(table.TableToken) == "" ||
			strings.TrimSpace(table.TableID) == "" || strings.TrimSpace(table.ViewID) == "" {
			return errs.V3InvalidInputError(fmt.Errorf("表 %s 的飞书配置不完整", identity))
		}
		if table.TableType != constvar.FeedbackTableType && table.TableType != constvar.FAQTableType {
			return errs.V3InvalidInputError(fmt.Errorf("不支持的 table_type: %s", table.TableType))
		}

		seenScopes := make(map[string]struct{}, len(table.Scopes))
		for _, value := range table.Scopes {
			scope := strings.TrimSpace(value)
			if !isSupportedV3Scope(scope) {
				return errs.V3InvalidInputError(fmt.Errorf("不支持的 scope: %s", value))
			}
			if _, exists := seenScopes[scope]; exists {
				return errs.V3InvalidInputError(fmt.Errorf("表 %s 的 scope 重复: %s", identity, scope))
			}
			seenScopes[scope] = struct{}{}
		}
		if len(seenScopes) == 0 {
			return errs.V3InvalidInputError(fmt.Errorf("表 %s 至少需要配置一个 scope", identity))
		}
	}
	return nil
}

func isSupportedV3Scope(scope string) bool {
	_, ok := constvar.SupportedFeedbackScopes()[scope]
	return ok
}
