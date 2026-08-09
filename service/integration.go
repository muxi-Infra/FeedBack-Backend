package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/apikey"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/repository/cache"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
)

const (
	ProjectStatusActive   = "active"
	ProjectStatusDisabled = "disabled"
	ProjectTableFeedback  = "feedback"
	ProjectTableFAQ       = "faq"
)

// projectIDPattern 用于验证稳定的项目标识符：长度为 2 至 64 个字符，
// 以字母或数字开头，后续可包含字母、数字、“-”或“_”。
// 它用作跨服务身份标识，不应包含机密信息或主机名等环境相关值。
var projectIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{1,63}$`)

type IntegrationService interface {
	RegisterProject(ctx context.Context, input domain.RegisterProjectInput) (*domain.ProjectConfig, error)
	GetProject(ctx context.Context, projectID string) (*domain.ProjectConfig, error)
	ListProjects(ctx context.Context) ([]domain.ProjectSummary, error)
	UpdateProject(ctx context.Context, projectID string, input domain.UpdateProjectInput) error
	UpdateProjectConfig(ctx context.Context, input domain.UpdateProjectConfigInput) error
	DeleteProject(ctx context.Context, projectID string) error
}

type integrationService struct {
	dao          dao.IntegrationDAO
	configEvents cache.ProjectConfigEventBus
	log          logger.Logger
}

func NewIntegrationService(integrationDAO dao.IntegrationDAO, configEvents cache.ProjectConfigEventBus, log logger.Logger) IntegrationService {
	return &integrationService{dao: integrationDAO, configEvents: configEvents, log: log}
}

func (s *integrationService) RegisterProject(ctx context.Context, input domain.RegisterProjectInput) (*domain.ProjectConfig, error) {
	if err := validateRegisterProjectInput(input); err != nil {
		return nil, err
	}

	projectID := strings.TrimSpace(input.ProjectID)
	existing, err := s.dao.GetProject(ctx, projectID)
	if err != nil {
		return nil, errs.IntegrationProjectDatabaseError(err)
	}
	if existing != nil {
		return nil, errs.IntegrationProjectAlreadyExistsError(errors.New("project_id already exists"))
	}

	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = ProjectStatusActive
	}

	project := &model.FeedbackProject{
		ProjectID:   strings.TrimSpace(input.ProjectID),
		ProjectName: strings.TrimSpace(input.ProjectName),
		School:      strings.TrimSpace(input.School),
		Status:      status,
	}
	key := &model.FeedbackProjectKey{
		ProjectID:  project.ProjectID,
		KeyID:      strings.TrimSpace(input.Key.KeyID),
		Issuer:     strings.TrimSpace(input.Key.Issuer),
		APIKeyHash: "",
		ExpiresAt:  input.Key.ExpiresAt,
		Status:     ProjectStatusActive,
	}
	apiKeyValue := strings.TrimSpace(input.Key.APIKey)
	if apiKeyValue == "" {
		apiKeyValue, err = apikey.Generate()
		if err != nil {
			return nil, errs.IntegrationProjectDatabaseError(err)
		}
	}
	key.APIKeyHash = apikey.Digest(apiKeyValue)

	err = s.dao.Transaction(ctx, func(tx dao.IntegrationDAO) error {
		if err := tx.CreateProject(ctx, project); err != nil {
			return errs.IntegrationProjectDatabaseError(err)
		}
		if err := tx.UpsertProjectKey(ctx, key); err != nil {
			return errs.IntegrationProjectDatabaseError(err)
		}

		for _, tableReq := range input.Tables {
			table := buildProjectTable(project.ProjectID, tableReq)
			if err := tx.UpsertProjectTable(ctx, table); err != nil {
				return errs.IntegrationProjectDatabaseError(err)
			}

			scopes := buildProjectScopes(project.ProjectID, table.TableIdentity, tableReq.Scopes)
			if err := tx.ReplaceProjectScopes(ctx, project.ProjectID, table.TableIdentity, scopes); err != nil {
				return errs.IntegrationProjectDatabaseError(err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.publishProjectChanged(ctx, project.ProjectID)

	config, err := s.GetProject(ctx, project.ProjectID)
	if err != nil {
		return nil, err
	}
	if len(config.Keys) > 0 {
		config.Keys[0].APIKey = apiKeyValue
	}
	return config, nil
}

func (s *integrationService) GetProject(ctx context.Context, projectID string) (*domain.ProjectConfig, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, invalidProjectError("project_id is required")
	}

	project, err := s.dao.GetProject(ctx, projectID)
	if err != nil {
		return nil, errs.IntegrationProjectDatabaseError(err)
	}
	if project == nil {
		return nil, errs.IntegrationProjectNotFoundError(errors.New("project_id not found"))
	}

	keys, err := s.dao.ListProjectKeys(ctx, projectID)
	if err != nil {
		return nil, errs.IntegrationProjectDatabaseError(err)
	}
	tables, err := s.dao.ListProjectTables(ctx, projectID)
	if err != nil {
		return nil, errs.IntegrationProjectDatabaseError(err)
	}

	config := &domain.ProjectConfig{
		Project: projectSummary(project),
		Keys:    make([]domain.ProjectKeySummary, 0, len(keys)),
		Tables:  make([]domain.ProjectTableConfig, 0, len(tables)),
	}
	for _, key := range keys {
		config.Keys = append(config.Keys, domain.ProjectKeySummary{
			ID:        key.ID,
			ProjectID: key.ProjectID,
			KeyID:     key.KeyID,
			Issuer:    key.Issuer,
			APIKey:    "",
			Status:    key.Status,
			ExpiresAt: key.ExpiresAt,
		})
	}
	for _, table := range tables {
		scopes, err := s.dao.ListProjectScopes(ctx, projectID, table.TableIdentity)
		if err != nil {
			return nil, errs.IntegrationProjectDatabaseError(err)
		}

		scopeNames := make([]string, 0, len(scopes))
		for _, scope := range scopes {
			scopeNames = append(scopeNames, scope.Scope)
		}
		config.Tables = append(config.Tables, domain.ProjectTableConfig{
			ID:            table.ID,
			ProjectID:     table.ProjectID,
			TableIdentity: table.TableIdentity,
			TableName:     table.PhysicalName,
			TableToken:    table.TableToken,
			TableID:       table.TableID,
			ViewID:        table.ViewID,
			TableType:     table.TableType,
			Notice:        table.Notice,
			Status:        table.Status,
			Scopes:        scopeNames,
		})
	}

	return config, nil
}

func (s *integrationService) ListProjects(ctx context.Context) ([]domain.ProjectSummary, error) {
	projects, err := s.dao.ListProjects(ctx)
	if err != nil {
		return nil, errs.IntegrationProjectDatabaseError(err)
	}
	result := make([]domain.ProjectSummary, 0, len(projects))
	for _, project := range projects {
		result = append(result, *projectSummary(&project))
	}
	return result, nil
}

func projectSummary(project *model.FeedbackProject) *domain.ProjectSummary {
	if project == nil {
		return nil
	}
	return &domain.ProjectSummary{
		ID:          project.ID,
		ProjectID:   project.ProjectID,
		ProjectName: project.ProjectName,
		School:      project.School,
		Status:      project.Status,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
	}
}

func (s *integrationService) UpdateProject(ctx context.Context, projectID string, input domain.UpdateProjectInput) error {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return invalidProjectError("project_id is required")
	}
	if input.Status != "" && !isProjectStatus(input.Status) {
		return invalidProjectError("unsupported project status: " + input.Status)
	}

	project, err := s.dao.GetProject(ctx, projectID)
	if err != nil {
		return errs.IntegrationProjectDatabaseError(err)
	}
	if project == nil {
		return errs.IntegrationProjectNotFoundError(errors.New("project_id not found"))
	}
	if strings.TrimSpace(input.ProjectName) != "" {
		project.ProjectName = strings.TrimSpace(input.ProjectName)
	}
	if strings.TrimSpace(input.School) != "" {
		project.School = strings.TrimSpace(input.School)
	}
	if strings.TrimSpace(input.Status) != "" {
		project.Status = strings.TrimSpace(input.Status)
	}

	if err := s.dao.UpdateProject(ctx, project); err != nil {
		return errs.IntegrationProjectDatabaseError(err)
	}
	s.publishProjectChanged(ctx, projectID)
	return nil
}

// UpdateProjectConfig 全量替换项目配置，保证基本信息、API Key、表配置和 Scope 在同一事务中更新。
func (s *integrationService) UpdateProjectConfig(ctx context.Context, input domain.UpdateProjectConfigInput) error {
	registerInput := domain.RegisterProjectInput{
		ProjectID:   input.ProjectID,
		ProjectName: input.ProjectName,
		School:      input.School,
		Status:      input.Status,
		Key:         input.Key,
		Tables:      input.Tables,
	}
	if err := validateRegisterProjectInput(registerInput); err != nil {
		return err
	}

	projectID := strings.TrimSpace(input.ProjectID)
	project, err := s.dao.GetProject(ctx, projectID)
	if err != nil {
		return errs.IntegrationProjectDatabaseError(err)
	}
	if project == nil {
		return errs.IntegrationProjectNotFoundError(errors.New("project_id not found"))
	}

	if err := s.dao.Transaction(ctx, func(tx dao.IntegrationDAO) error {
		project.ProjectName = strings.TrimSpace(input.ProjectName)
		project.School = strings.TrimSpace(input.School)
		project.Status = strings.TrimSpace(input.Status)
		if err := tx.UpdateProject(ctx, project); err != nil {
			return errs.IntegrationProjectDatabaseError(err)
		}

		apiKeyHash := ""
		existingKey, err := tx.GetProjectKey(ctx, projectID, strings.TrimSpace(input.Key.KeyID))
		if err != nil {
			return errs.IntegrationProjectDatabaseError(err)
		}
		if existingKey != nil {
			apiKeyHash = existingKey.APIKeyHash
		}
		if strings.TrimSpace(input.Key.APIKey) != "" {
			apiKeyHash = apikey.Digest(strings.TrimSpace(input.Key.APIKey))
		}
		if apiKeyHash == "" {
			return errs.IntegrationProjectInvalidError(errors.New("api_key is required when rotating the project key"))
		}
		key := &model.FeedbackProjectKey{
			ProjectID:  projectID,
			KeyID:      strings.TrimSpace(input.Key.KeyID),
			Issuer:     strings.TrimSpace(input.Key.Issuer),
			APIKeyHash: apiKeyHash,
			ExpiresAt:  input.Key.ExpiresAt,
			Status:     ProjectStatusActive,
		}
		if err := tx.UpsertProjectKey(ctx, key); err != nil {
			return errs.IntegrationProjectDatabaseError(err)
		}

		existingKeys, err := tx.ListProjectKeys(ctx, projectID)
		if err != nil {
			return errs.IntegrationProjectDatabaseError(err)
		}
		for _, existing := range existingKeys {
			if existing.KeyID != key.KeyID {
				if err := tx.DeleteProjectKey(ctx, projectID, existing.KeyID); err != nil {
					return errs.IntegrationProjectDatabaseError(err)
				}
			}
		}

		existingTables, err := tx.ListProjectTables(ctx, projectID)
		if err != nil {
			return errs.IntegrationProjectDatabaseError(err)
		}
		requestedTables := make(map[string]struct{}, len(input.Tables))
		for _, tableInput := range input.Tables {
			table := buildProjectTable(projectID, tableInput)
			requestedTables[table.TableIdentity] = struct{}{}
			if err := tx.UpsertProjectTable(ctx, table); err != nil {
				return errs.IntegrationProjectDatabaseError(err)
			}
			scopes := buildProjectScopes(projectID, table.TableIdentity, tableInput.Scopes)
			if err := tx.ReplaceProjectScopes(ctx, projectID, table.TableIdentity, scopes); err != nil {
				return errs.IntegrationProjectDatabaseError(err)
			}
		}
		for _, existing := range existingTables {
			if _, ok := requestedTables[existing.TableIdentity]; !ok {
				if err := tx.DeleteProjectTable(ctx, projectID, existing.TableIdentity); err != nil {
					return errs.IntegrationProjectDatabaseError(err)
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}

	s.publishProjectChanged(ctx, projectID)
	return nil
}

func (s *integrationService) DeleteProject(ctx context.Context, projectID string) error {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return invalidProjectError("project_id is required")
	}
	if err := s.dao.Transaction(ctx, func(tx dao.IntegrationDAO) error {
		return tx.DeleteProject(ctx, projectID)
	}); err != nil {
		return errs.IntegrationProjectDatabaseError(err)
	}
	s.publishProjectChanged(ctx, projectID)
	return nil
}

func (s *integrationService) publishProjectChanged(ctx context.Context, projectID string) {
	if s.configEvents == nil {
		return
	}
	if err := s.configEvents.PublishProjectChanged(ctx, projectID); err != nil && s.log != nil {
		s.log.Error("发布项目配置刷新事件失败",
			logger.String("project_id", projectID),
			logger.String("error", err.Error()),
		)
	}
}

func validateRegisterProjectInput(input domain.RegisterProjectInput) error {
	projectID := strings.TrimSpace(input.ProjectID)
	if !projectIDPattern.MatchString(projectID) {
		return invalidProjectError("project_id must be 2-64 characters and contain only letters, numbers, '_' or '-'")
	}
	if strings.TrimSpace(input.ProjectName) == "" {
		return invalidProjectError("project_name is required")
	}
	if strings.TrimSpace(input.School) == "" {
		return invalidProjectError("school is required")
	}
	if input.Status != "" && !isProjectStatus(input.Status) {
		return invalidProjectError("unsupported project status: " + input.Status)
	}
	if strings.TrimSpace(input.Key.KeyID) == "" {
		return invalidProjectError("key_id is required")
	}
	if strings.TrimSpace(input.Key.Issuer) == "" {
		return invalidProjectError("issuer is required")
	}
	if len(input.Tables) == 0 {
		return invalidProjectError("at least one project table is required")
	}

	// 逐个校验项目绑定的飞书表格配置，并确保同一个项目内的表格标识唯一。
	seenTables := make(map[string]struct{}, len(input.Tables))
	for _, table := range input.Tables {
		identity := strings.TrimSpace(table.TableIdentity)
		if identity == "" {
			return invalidProjectError("table_identity is required")
		}

		// table_identity 是项目内识别表格的唯一标识，不能重复绑定。
		if _, exists := seenTables[identity]; exists {
			return invalidProjectError("duplicate table_identity: " + identity)
		}
		seenTables[identity] = struct{}{}

		// 校验飞书表格的基本访问配置是否完整。
		if strings.TrimSpace(table.TableName) == "" || strings.TrimSpace(table.TableToken) == "" ||
			strings.TrimSpace(table.TableID) == "" || strings.TrimSpace(table.ViewID) == "" {
			return invalidProjectError("table " + identity + " has incomplete Feishu configuration")
		}

		// 未填写表格类型时由后续逻辑使用默认的 feedback 类型；填写后只能使用支持的类型。
		if table.TableType != "" && table.TableType != ProjectTableFeedback && table.TableType != ProjectTableFAQ {
			return invalidProjectError("unsupported table_type: " + table.TableType)
		}
		if len(table.Scopes) == 0 {
			return invalidProjectError("table " + identity + " must have at least one scope")
		}

		// 校验当前表格的权限范围，避免出现未定义权限或重复权限。
		seenScopes := make(map[string]struct{}, len(table.Scopes))
		for _, scope := range table.Scopes {
			if !isSupportedScope(scope) {
				return invalidProjectError("unsupported scope: " + scope)
			}
			if _, exists := seenScopes[scope]; exists {
				return invalidProjectError("duplicate scope " + scope + " for table " + identity)
			}
			seenScopes[scope] = struct{}{}
		}
	}
	return nil
}

func invalidProjectError(message string) error {
	return errs.IntegrationProjectInvalidError(errors.New(message))
}

func buildProjectTable(projectID string, input domain.ProjectTableInput) *model.FeedbackProjectTable {
	tableType := strings.TrimSpace(input.TableType)
	if tableType == "" {
		tableType = ProjectTableFeedback
	}
	return &model.FeedbackProjectTable{
		ProjectID:     projectID,
		TableIdentity: strings.TrimSpace(input.TableIdentity),
		PhysicalName:  strings.TrimSpace(input.TableName),
		TableToken:    strings.TrimSpace(input.TableToken),
		TableID:       strings.TrimSpace(input.TableID),
		ViewID:        strings.TrimSpace(input.ViewID),
		TableType:     tableType,
		Notice:        input.Notice,
		Status:        ProjectStatusActive,
	}
}

func buildProjectScopes(projectID, tableIdentity string, values []string) []model.FeedbackProjectScope {
	result := make([]model.FeedbackProjectScope, 0, len(values))
	for _, value := range values {
		result = append(result, model.FeedbackProjectScope{
			ProjectID:     projectID,
			TableIdentity: tableIdentity,
			Scope:         strings.TrimSpace(value),
		})
	}
	return result
}

func isProjectStatus(status string) bool {
	return status == ProjectStatusActive || status == ProjectStatusDisabled
}

func isSupportedScope(scope string) bool {
	switch strings.TrimSpace(scope) {
	case "feedback:create", "feedback:read:self", "feedback:read", "feedback:write", "feedback:sync":
		return true
	default:
		return false
	}
}
