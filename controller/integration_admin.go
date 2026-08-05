package controller

import (
	"github.com/gin-gonic/gin"
	reqV1 "github.com/muxi-Infra/FeedBack-Backend/api/request/v1"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	respV1 "github.com/muxi-Infra/FeedBack-Backend/api/response/v1"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/service"
)

type IntegrationAdminHandler interface {
	RegisterProject(c *gin.Context, req reqV1.RegisterProjectReq) (response.Response, error)
	GetProject(c *gin.Context) (response.Response, error)
	ListProjects(c *gin.Context) (response.Response, error)
	UpdateProject(c *gin.Context, req reqV1.UpdateProjectReq) (response.Response, error)
	DeleteProject(c *gin.Context) (response.Response, error)
	RestoreProject(c *gin.Context) (response.Response, error)
}

type IntegrationAdmin struct {
	s service.IntegrationService
}

func NewIntegrationAdmin(s service.IntegrationService) IntegrationAdminHandler {
	return &IntegrationAdmin{s: s}
}

func (h *IntegrationAdmin) RegisterProject(c *gin.Context, req reqV1.RegisterProjectReq) (response.Response, error) {
	input := domain.RegisterProjectInput{
		ProjectID:   req.ProjectID,
		ProjectName: req.ProjectName,
		School:      req.School,
		Status:      req.Status,
		Key: domain.ProjectKeyInput{
			KeyID:     req.Key.KeyID,
			Issuer:    req.Key.Issuer,
			PublicKey: req.Key.PublicKey,
			ExpiresAt: req.Key.ExpiresAt,
		},
		Tables: make([]domain.ProjectTableInput, 0, len(req.Tables)),
	}
	for _, table := range req.Tables {
		input.Tables = append(input.Tables, domain.ProjectTableInput{
			TableIdentity: table.TableIdentity,
			TableName:     table.TableName,
			TableToken:    table.TableToken,
			TableID:       table.TableID,
			ViewID:        table.ViewID,
			TableType:     table.TableType,
			Notice:        table.Notice,
			Scopes:        table.Scopes,
		})
	}

	project, err := h.s.RegisterProject(c.Request.Context(), input)
	if err != nil {
		return response.Response{}, err
	}
	return response.Response{Code: 0, Message: "Success", Data: toProjectConfigResponse(project)}, nil
}

func (h *IntegrationAdmin) GetProject(c *gin.Context) (response.Response, error) {
	project, err := h.s.GetProject(c.Request.Context(), c.Param("project_id"))
	if err != nil {
		return response.Response{}, err
	}
	return response.Response{Code: 0, Message: "Success", Data: toProjectConfigResponse(project)}, nil
}

func (h *IntegrationAdmin) ListProjects(c *gin.Context) (response.Response, error) {
	projects, err := h.s.ListProjects(c.Request.Context())
	if err != nil {
		return response.Response{}, err
	}

	items := make([]respV1.ProjectResponse, 0, len(projects))
	for _, project := range projects {
		items = append(items, *toProjectResponse(project))
	}
	return response.Response{Code: 0, Message: "Success", Data: items}, nil
}

func (h *IntegrationAdmin) UpdateProject(c *gin.Context, req reqV1.UpdateProjectReq) (response.Response, error) {
	err := h.s.UpdateProject(c.Request.Context(), c.Param("project_id"), domain.UpdateProjectInput{
		ProjectName: req.ProjectName,
		School:      req.School,
		Status:      req.Status,
	})
	if err != nil {
		return response.Response{}, err
	}
	return response.Response{Code: 0, Message: "Success", Data: nil}, nil
}

func (h *IntegrationAdmin) DeleteProject(c *gin.Context) (response.Response, error) {
	if err := h.s.DeleteProject(c.Request.Context(), c.Param("project_id")); err != nil {
		return response.Response{}, err
	}
	return response.Response{Code: 0, Message: "Success", Data: nil}, nil
}

func (h *IntegrationAdmin) RestoreProject(c *gin.Context) (response.Response, error) {
	if err := h.s.RestoreProject(c.Request.Context(), c.Param("project_id")); err != nil {
		return response.Response{}, err
	}
	return response.Response{Code: 0, Message: "Success", Data: nil}, nil
}

func toProjectConfigResponse(config *domain.ProjectConfig) respV1.ProjectConfigResponse {
	project := toProjectResponse(*config.Project)
	result := respV1.ProjectConfigResponse{
		Project: project,
		Keys:    make([]respV1.ProjectKeyResponse, 0, len(config.Keys)),
		Tables:  make([]respV1.ProjectTableResponse, 0, len(config.Tables)),
	}
	for _, key := range config.Keys {
		result.Keys = append(result.Keys, respV1.ProjectKeyResponse{
			ID:        key.ID,
			ProjectID: key.ProjectID,
			KeyID:     key.KeyID,
			Issuer:    key.Issuer,
			Status:    key.Status,
			ExpiresAt: key.ExpiresAt,
		})
	}
	for _, table := range config.Tables {
		result.Tables = append(result.Tables, respV1.ProjectTableResponse{
			ID:            table.ID,
			ProjectID:     table.ProjectID,
			TableIdentity: table.TableIdentity,
			TableName:     table.TableName,
			TableID:       table.TableID,
			ViewID:        table.ViewID,
			TableType:     table.TableType,
			Notice:        table.Notice,
			Status:        table.Status,
			Scopes:        table.Scopes,
		})
	}
	return result
}

func toProjectResponse(project domain.ProjectSummary) *respV1.ProjectResponse {
	result := respV1.ProjectResponse{
		ID:          project.ID,
		ProjectID:   project.ProjectID,
		ProjectName: project.ProjectName,
		School:      project.School,
		Status:      project.Status,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
	}
	return &result
}
