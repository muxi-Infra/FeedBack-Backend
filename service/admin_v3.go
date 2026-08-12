package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	respV3 "github.com/muxi-Infra/FeedBack-Backend/api/response/v3"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/apikey"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/constvar"
	"github.com/muxi-Infra/FeedBack-Backend/repository/cache"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"gorm.io/gorm"
)

type V3AdminService interface {
	RegisterProject(ctx context.Context, input domain.RegisterProjectInput) (respV3.RegisterProjectResp, error)
	ListProjects(ctx context.Context) ([]model.FeedbackProjectV3, error)
	GetProjectConfig(ctx context.Context, projectID string) (model.FeedbackProjectV3, *model.FeedbackProjectKeyV3, []model.FeedbackProjectTableV3, map[string][]string, error)
	UpdateProject(ctx context.Context, projectID string, input domain.RegisterProjectInput) error
	DeleteProject(ctx context.Context, projectID string) error
	RotateAPIKey(ctx context.Context, projectID string) (string, string, error)
}

type v3AdminService struct {
	dao    dao.IntegrationDAOV3
	events cache.ProjectConfigEventBusV3
}

func NewV3AdminService(d dao.IntegrationDAOV3, events cache.ProjectConfigEventBusV3) V3AdminService {
	return &v3AdminService{
		dao:    d,
		events: events,
	}
}

func (s *v3AdminService) RegisterProject(ctx context.Context, input domain.RegisterProjectInput) (respV3.RegisterProjectResp, error) {
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
	if err := s.dao.Transaction(ctx, func(tx *gorm.DB) error {
		return s.dao.RegisterProject(ctx, project, key, tables, scopes, tx)
	}); err != nil {
		return respV3.RegisterProjectResp{}, errs.V3ProjectDatabaseError(err)
	}

	if err := s.events.PublishProjectChanged(ctx, projectID); err != nil {
		return respV3.RegisterProjectResp{}, errs.V3ConfigPublishError(err)
	}
	return respV3.RegisterProjectResp{
		ProjectID: projectID,
		KeyID:     keyID,
		APIKey:    plainKey,
	}, nil
}

func (s *v3AdminService) ListProjects(ctx context.Context) ([]model.FeedbackProjectV3, error) {
	projects, err := s.dao.ListProjects(ctx)
	if err != nil {
		return nil, errs.V3ProjectDatabaseError(err)
	}
	return projects, nil
}

func (s *v3AdminService) GetProjectConfig(ctx context.Context, projectID string) (model.FeedbackProjectV3, *model.FeedbackProjectKeyV3, []model.FeedbackProjectTableV3, map[string][]string, error) {
	project, key, tables, scopes, err := s.dao.GetProjectConfig(ctx, projectID)
	if err != nil {
		return project, key, tables, scopes, errs.V3ProjectDatabaseError(err)
	}
	return project, key, tables, scopes, nil
}

func (s *v3AdminService) UpdateProject(ctx context.Context, projectID string, input domain.RegisterProjectInput) error {
	if strings.TrimSpace(projectID) == "" {
		return errs.V3InvalidInputError(errors.New("项目 ID 不能为空"))
	}
	if err := validateProjectInput(input); err != nil {
		return err
	}
	project := model.FeedbackProjectV3{
		ProjectID:   projectID,
		ProjectName: strings.TrimSpace(input.ProjectName),
		School:      strings.TrimSpace(input.School),
		Status:      "active",
	}
	tables, scopes, err := buildProjectTables(projectID, input.Tables)
	if err != nil {
		return err
	}
	if err := s.dao.Transaction(ctx, func(tx *gorm.DB) error {
		return s.dao.UpdateProject(ctx, projectID, project, tables, scopes, tx)
	}); err != nil {
		return errs.V3ProjectDatabaseError(err)
	}
	if err := s.events.PublishProjectChanged(ctx, projectID); err != nil {
		return errs.V3ConfigPublishError(err)
	}
	return nil
}

func (s *v3AdminService) DeleteProject(ctx context.Context, projectID string) error {
	if projectID == "" {
		return errs.V3InvalidInputError(errors.New("project_id 不能为空"))
	}
	if err := s.dao.Transaction(ctx, func(tx *gorm.DB) error {
		return s.dao.DeleteProject(ctx, projectID, tx)
	}); err != nil {
		return errs.V3ProjectDatabaseError(err)
	}
	if err := s.events.PublishProjectChanged(ctx, projectID); err != nil {
		return errs.V3ConfigPublishError(err)
	}
	return nil
}

func (s *v3AdminService) RotateAPIKey(ctx context.Context, projectID string) (string, string, error) {
	if projectID == "" {
		return "", "", errs.V3InvalidInputError(errors.New("project_id 不能为空"))
	}
	plain, err := apikey.Generate()
	if err != nil {
		return "", "", errs.V3APIKeyGenerateError(err)
	}
	keyID := projectID + "-key-" + uuid.NewString()[:12]
	err = s.dao.Transaction(ctx, func(tx *gorm.DB) error {
		return s.dao.RotateAPIKey(ctx, projectID, model.FeedbackProjectKeyV3{
			ProjectID:  projectID,
			KeyID:      keyID,
			APIKeyHash: apikey.Digest(plain),
			Status:     "active",
		}, tx)
	})
	if err != nil {
		return "", "", errs.V3ProjectDatabaseError(err)
	}

	if err := s.events.PublishProjectChanged(ctx, projectID); err != nil {
		return "", "", errs.V3ConfigPublishError(err)
	}
	return keyID, plain, nil
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
