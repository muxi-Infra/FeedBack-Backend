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
	// todo 单独一个全量更新耗时有点长，后续根据需要添加单独的更新
	UpdateProjectConfig(c *gin.Context, req reqV1.UpdateProjectConfigReq) (response.Response, error)
	RotateProjectKey(c *gin.Context) (response.Response, error)
	DeleteProject(c *gin.Context) (response.Response, error)
}

type IntegrationAdmin struct {
	s service.IntegrationService
}

func NewIntegrationAdmin(s service.IntegrationService) IntegrationAdminHandler {
	return &IntegrationAdmin{s: s}
}

// RegisterProject 注册一个对接反馈中台的校园项目，并生成项目 API Key。
//
//	@Summary		注册反馈项目
//	@Description	登记项目 API Key、飞书反馈表和表级 scope。接口仅允许管理员访问；新生成的 API Key 只在注册响应中返回一次。
//	@Tags			Integration Admin
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string						true	"Bearer Admin JWT"
//	@Param			request			body		reqV1.RegisterProjectReq	true	"项目配置"
//	@Success		200				{object}	response.Response{data=respV1.ProjectConfigResponse}
//	@Failure		400				{object}	response.Response
//	@Failure		401				{object}	response.Response
//	@Failure		403				{object}	response.Response
//	@Failure		409				{object}	response.Response
//	@Router			/api/v1/integrations/projects [post]
func (h *IntegrationAdmin) RegisterProject(c *gin.Context, req reqV1.RegisterProjectReq) (response.Response, error) {
	input := domain.RegisterProjectInput{
		ProjectID:   req.ProjectID,
		ProjectName: req.ProjectName,
		School:      req.School,
		Status:      req.Status,
		Key: domain.ProjectKeyInput{
			KeyID:     req.Key.KeyID,
			Issuer:    req.Key.Issuer,
			APIKey:    req.Key.APIKey,
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

// GetProject 获取指定项目的配置摘要。
//
//	@Summary	获取项目配置
//	@Tags		Integration Admin
//	@Produce	json
//	@Param		Authorization	header		string	true	"Bearer Admin JWT"
//	@Param		project_id		path		string	true	"项目 ID"
//	@Success	200				{object}	response.Response{data=respV1.ProjectConfigResponse}
//	@Failure	401				{object}	response.Response
//	@Failure	403				{object}	response.Response
//	@Failure	404				{object}	response.Response
//	@Router		/api/v1/integrations/projects/{project_id} [get]
func (h *IntegrationAdmin) GetProject(c *gin.Context) (response.Response, error) {
	project, err := h.s.GetProject(c.Request.Context(), c.Param("project_id"))
	if err != nil {
		return response.Response{}, err
	}
	return response.Response{Code: 0, Message: "Success", Data: toProjectConfigResponse(project)}, nil
}

// ListProjects 获取所有已登记项目。
//
//	@Summary	获取项目列表
//	@Tags		Integration Admin
//	@Produce	json
//	@Param		Authorization	header		string	true	"Bearer Admin JWT"
//	@Success	200				{object}	response.Response{data=[]respV1.ProjectResponse}
//	@Failure	401				{object}	response.Response
//	@Failure	403				{object}	response.Response
//	@Router		/api/v1/integrations/projects [get]
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

// UpdateProject 更新项目基本信息。
//
//	@Summary	更新项目
//	@Tags		Integration Admin
//	@Accept		json
//	@Produce	json
//	@Param		Authorization	header		string					true	"Bearer Admin JWT"
//	@Param		project_id		path		string					true	"项目 ID"
//	@Param		request			body		reqV1.UpdateProjectReq	true	"项目基本信息"
//	@Success	200				{object}	response.Response
//	@Failure	400				{object}	response.Response
//	@Failure	401				{object}	response.Response
//	@Failure	403				{object}	response.Response
//	@Failure	404				{object}	response.Response
//	@Router		/api/v1/integrations/projects/{project_id} [put]
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

// UpdateProjectConfig 全量更新项目基本信息、API Key、飞书表配置和 Scope。
//
//	@Summary		全量更新项目配置
//	@Description	一次性替换项目基本信息、API Key、飞书表配置和表级 Scope。未提交的旧 API Key 或表配置会被软删除。
//	@Tags			Integration Admin
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string							true	"Bearer Admin JWT"
//	@Param			project_id		path		string							true	"项目 ID"
//	@Param			request			body		reqV1.UpdateProjectConfigReq	true	"完整项目配置"
//	@Success		200				{object}	response.Response
//	@Failure		400				{object}	response.Response
//	@Failure		401				{object}	response.Response
//	@Failure		403				{object}	response.Response
//	@Failure		404				{object}	response.Response
//	@Router			/api/v1/integrations/projects/{project_id}/config [put]
func (h *IntegrationAdmin) UpdateProjectConfig(c *gin.Context, req reqV1.UpdateProjectConfigReq) (response.Response, error) {
	input := domain.UpdateProjectConfigInput{
		ProjectID:   c.Param("project_id"),
		ProjectName: req.ProjectName,
		School:      req.School,
		Status:      req.Status,
		Key: domain.ProjectKeyInput{
			KeyID:     req.Key.KeyID,
			Issuer:    req.Key.Issuer,
			APIKey:    req.Key.APIKey,
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

	if err := h.s.UpdateProjectConfig(c.Request.Context(), input); err != nil {
		return response.Response{}, err
	}
	return response.Response{Code: 0, Message: "Success", Data: nil}, nil
}

// RotateProjectKey 重新生成项目 API Key，旧 Key 会立即失效。
//
//	@Summary		轮换项目 API Key
//	@Description	生成新的项目 API Key。明文 Key 只在本次响应中返回，管理员需要立即保存并更新校园项目后端配置。
//	@Tags			Integration Admin
//	@Produce		json
//	@Param			Authorization	header	string	true	"Bearer Admin JWT"
//	@Param			project_id		path	string	true	"项目 ID"
//	@Param			key_id			path	string	true	"Key ID"
//	@Success		200	{object}	response.Response{data=respV1.RotateProjectKeyResponse}
//	@Failure		401	{object}	response.Response
//	@Failure		403	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Router			/api/v1/integrations/projects/{project_id}/keys/{key_id}/rotate [post]
func (h *IntegrationAdmin) RotateProjectKey(c *gin.Context) (response.Response, error) {
	result, err := h.s.RotateProjectKey(c.Request.Context(), c.Param("project_id"), c.Param("key_id"))
	if err != nil {
		return response.Response{}, err
	}
	return response.Response{
		Code:    0,
		Message: "Success",
		Data: respV1.RotateProjectKeyResponse{
			ProjectID: result.ProjectID,
			KeyID:     result.KeyID,
			APIKey:    result.APIKey,
		},
	}, nil
}

// DeleteProject 软删除项目。
//
//	@Summary	删除项目
//	@Tags		Integration Admin
//	@Produce	json
//	@Param		Authorization	header		string	true	"Bearer Admin JWT"
//	@Param		project_id		path		string	true	"项目 ID"
//	@Success	200				{object}	response.Response
//	@Failure	401				{object}	response.Response
//	@Failure	403				{object}	response.Response
//	@Failure	404				{object}	response.Response
//	@Router		/api/v1/integrations/projects/{project_id} [delete]
func (h *IntegrationAdmin) DeleteProject(c *gin.Context) (response.Response, error) {
	if err := h.s.DeleteProject(c.Request.Context(), c.Param("project_id")); err != nil {
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
			APIKey:    key.APIKey,
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
			TableToken:    table.TableToken,
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
