package controller

import (
	"github.com/gin-gonic/gin"
	reqV3 "github.com/muxi-Infra/FeedBack-Backend/api/request/v3"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	respV3 "github.com/muxi-Infra/FeedBack-Backend/api/response/v3"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/service"
)

type V3AdminHandler interface {
	RegisterProject(*gin.Context, reqV3.RegisterProjectReq) (response.Response, error)
	ListProjects(*gin.Context) (response.Response, error)
	GetProject(*gin.Context) (response.Response, error)
	UpdateProject(*gin.Context, reqV3.RegisterProjectReq) (response.Response, error)
	DeleteProject(*gin.Context) (response.Response, error)
	RotateAPIKey(*gin.Context) (response.Response, error)
}

type V3Admin struct {
	service service.V3AdminService
}

func NewV3Admin(s service.V3AdminService) V3AdminHandler {
	return &V3Admin{
		service: s,
	}
}

// RegisterProject 注册项目并生成一次性 API Key。
//
//	@Summary	注册 V3 接入项目
//	@Tags		V3Admin
//	@Accept		json
//	@Produce	json
//	@Security	BasicAuth
//	@Param		request	body		reqV3.RegisterProjectReq	true	"项目配置"
//	@Success	200		{object}	response.Response{data=respV3.RegisterProjectResp}
//	@Router		/api/v3/admin/integrations/projects [post]
func (h *V3Admin) RegisterProject(c *gin.Context, req reqV3.RegisterProjectReq) (response.Response, error) {
	tables := make([]domain.RegisterProjectTableInput, 0, len(req.Tables))
	for _, item := range req.Tables {
		tables = append(tables, domain.RegisterProjectTableInput{
			TableIdentity: item.TableIdentity,
			TableName:     item.TableName,
			TableToken:    item.TableToken,
			TableID:       item.TableID,
			ViewID:        item.ViewID,
			TableType:     item.TableType,
			Notice:        item.Notice,
			Scopes:        item.Scopes,
		})
	}
	result, err := h.service.RegisterProject(c.Request.Context(), domain.RegisterProjectInput{
		ProjectName: req.ProjectName,
		School:      req.School,
		Tables:      tables,
	})
	if err != nil {
		return response.Response{}, err
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    result,
	}, nil
}

func (h *V3Admin) ListProjects(c *gin.Context) (response.Response, error) {
	projects, err := h.service.ListProjects(c.Request.Context())
	if err != nil {
		return response.Response{}, err
	}
	items := make([]respV3.ProjectListItem, 0, len(projects))
	for _, project := range projects {
		items = append(items, respV3.ProjectListItem{
			ID:          project.ID,
			ProjectID:   project.ProjectID,
			ProjectName: project.ProjectName,
			School:      project.School,
			Status:      project.Status,
			CreatedAt:   project.CreatedAt,
			UpdatedAt:   project.UpdatedAt,
		})
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    items,
	}, nil
}

func (h *V3Admin) GetProject(c *gin.Context) (response.Response, error) {
	project, key, tables, scopes, err := h.service.GetProjectConfig(c.Request.Context(), c.Param("project_id"))
	if err != nil {
		return response.Response{}, err
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data: respV3.ProjectConfigResp{
			Project: project,
			Key:     key,
			Tables:  tables,
			Scopes:  scopes,
		},
	}, nil
}

func (h *V3Admin) UpdateProject(c *gin.Context, req reqV3.RegisterProjectReq) (response.Response, error) {
	input := toRegisterProjectInput(req)
	if err := h.service.UpdateProject(c.Request.Context(), c.Param("project_id"), input); err != nil {
		return response.Response{}, err
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    nil,
	}, nil
}

func (h *V3Admin) DeleteProject(c *gin.Context) (response.Response, error) {
	if err := h.service.DeleteProject(c.Request.Context(), c.Param("project_id")); err != nil {
		return response.Response{}, err
	}
	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    nil,
	}, nil
}

func (h *V3Admin) RotateAPIKey(c *gin.Context) (response.Response, error) {
	keyID, apiKey, err := h.service.RotateAPIKey(c.Request.Context(), c.Param("project_id"))
	if err != nil {
		return response.Response{}, err
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data: respV3.RotateAPIKeyResp{
			KeyID:  keyID,
			APIKey: apiKey,
		},
	}, nil
}

func toRegisterProjectInput(req reqV3.RegisterProjectReq) domain.RegisterProjectInput {
	tables := make([]domain.RegisterProjectTableInput, 0, len(req.Tables))
	for _, item := range req.Tables {
		tables = append(tables, domain.RegisterProjectTableInput{
			TableIdentity: item.TableIdentity,
			TableName:     item.TableName,
			TableToken:    item.TableToken,
			TableID:       item.TableID,
			ViewID:        item.ViewID,
			TableType:     item.TableType,
			Notice:        item.Notice,
			Scopes:        item.Scopes,
		})
	}
	return domain.RegisterProjectInput{
		ProjectName: req.ProjectName,
		School:      req.School,
		Tables:      tables,
	}
}
