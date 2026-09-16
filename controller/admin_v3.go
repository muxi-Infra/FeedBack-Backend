package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	reqV3 "github.com/muxi-Infra/FeedBack-Backend/api/request/v3"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	respV3 "github.com/muxi-Infra/FeedBack-Backend/api/response/v3"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/constvar"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"github.com/muxi-Infra/FeedBack-Backend/service"
	"strconv"
)

type V3AdminHandler interface {
	RegisterProject(*gin.Context, reqV3.RegisterProjectReq) (response.Response, error)
	ListProjects(*gin.Context, reqV3.ListProjectsReq) (response.Response, error)
	GetProject(*gin.Context) (response.Response, error)
	UpdateProject(*gin.Context, reqV3.RegisterProjectReq) (response.Response, error)
	DeleteProject(*gin.Context) (response.Response, error)
	RotateAPIKey(*gin.Context) (response.Response, error)
	ConfigStatus(*gin.Context) (response.Response, error)
	ConfigAudits(*gin.Context) (response.Response, error)
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
// @Description 成功表示配置、版本、审计和事件 Outbox 已提交；各实例缓存异步应用。
// @Param X-Request-ID header string false "调用方预先保存的 UUID，供提交结果未知时查审计"
// @Header 200 {string} X-Request-ID "请求关联 ID"
// @Header 200 {string} X-Config-Version "已提交配置版本"
// @Header 200 {string} X-Config-Change-ID "配置变更 ID"
// @Header 200 {string} X-Config-Propagation "asynchronous"
//
//	@Summary	注册 V3 接入项目
//	@Tags		V3Admin
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		Authorization	header		string						true	"Bearer 管理员 Token"
//	@Param		request			body		reqV3.RegisterProjectReq	true	"项目配置"
//	@Success	200				{object}	response.Response{data=respV3.RegisterProjectResp}
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
	}, configActor(c))
	if err != nil {
		return response.Response{}, err
	}

	configHeaders(c, result.Receipt)
	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    result,
	}, nil
}

// ListProjects 查询 V3 接入项目列表。
//
//	@Summary	查询 V3 接入项目列表
//	@Tags		V3Admin
//	@Produce	json
//	@Security	BearerAuth
//	@Param		Authorization	header		string	true	"Bearer 管理员 Token"
//	@Param		page_token		query		string	false	"上一页返回的分页 Token"
//	@Param		limit_size		query		int		false	"每页数量，默认 20，最大 100"
//	@Success	200				{object}	response.Response{data=respV3.ProjectListResp}
//	@Router		/api/v3/admin/integrations/projects [get]
func (h *V3Admin) ListProjects(c *gin.Context, req reqV3.ListProjectsReq) (response.Response, error) {
	projects, hasMore, pageToken, err := h.service.ListProjects(c.Request.Context(), req.PageToken, req.LimitSize)
	if err != nil {
		return response.Response{}, err
	}
	items := make([]respV3.ProjectListItem, 0, len(projects))
	for _, project := range projects {
		items = append(items, respV3.ProjectListItem{
			ConfigVersion: project.ConfigVersion,
			ID:            project.ID,
			ProjectID:     project.ProjectID,
			ProjectName:   project.ProjectName,
			School:        project.School,
			Status:        project.Status,
			CreatedAt:     project.CreatedAt,
			UpdatedAt:     project.UpdatedAt,
		})
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data: respV3.ProjectListResp{
			Projects:  items,
			HasMore:   hasMore,
			PageToken: pageToken,
		},
	}, nil
}

// GetProject 查询 V3 项目的完整配置。
//
//	@Summary	查询 V3 接入项目配置
//	@Tags		V3Admin
//	@Produce	json
//	@Security	BearerAuth
//	@Param		Authorization	header		string	true	"Bearer 管理员 Token"
//	@Param		project_id		path		string	true	"项目 ID"
//	@Success	200				{object}	response.Response{data=respV3.ProjectConfigResp}
//	@Router		/api/v3/admin/integrations/projects/{project_id} [get]
func (h *V3Admin) GetProject(c *gin.Context) (response.Response, error) {
	project, key, tables, scopes, err := h.service.GetProjectConfig(c.Request.Context(), c.Param("project_id"))
	if err != nil {
		return response.Response{}, err
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data: respV3.ProjectConfigResp{
			Project: toProjectDetail(project),
			Key:     toProjectKeyDetail(key),
			Tables:  toProjectTableItems(tables),
			Scopes:  scopes,
		},
	}, nil
}

// UpdateProject 全量更新 V3 项目的表格配置，不会修改 API Key。
// @Description 成功表示事务已提交；各实例缓存异步应用，Scope 撤销受配置缓存生效窗口约束。
// @Param X-Request-ID header string false "请求关联 UUID"
// @Header 200 {string} X-Request-ID "请求关联 ID"
// @Header 200 {string} X-Config-Version "已提交配置版本"
// @Header 200 {string} X-Config-Change-ID "配置变更 ID"
// @Header 200 {string} X-Config-Propagation "asynchronous"
//
//	@Summary	更新 V3 接入项目配置
//	@Tags		V3Admin
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		Authorization	header		string						true	"Bearer 管理员 Token"
//	@Param		project_id		path		string						true	"项目 ID"
//	@Param		request			body		reqV3.RegisterProjectReq	true	"项目配置"
//	@Success	200				{object}	response.Response
//	@Router		/api/v3/admin/integrations/projects/{project_id} [put]
func (h *V3Admin) UpdateProject(c *gin.Context, req reqV3.RegisterProjectReq) (response.Response, error) {
	input := toRegisterProjectInput(req)
	receipt, err := h.service.UpdateProject(c.Request.Context(), c.Param("project_id"), input, configActor(c))
	if err != nil {
		return response.Response{}, err
	}

	configHeaders(c, receipt)
	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    nil,
	}, nil
}

// DeleteProject 删除 V3 项目及其 API Key、表格和权限配置。
// @Description 成功表示删除事务已提交；已签发 JWT 的后续访问受配置缓存撤销窗口约束。
// @Param X-Request-ID header string false "请求关联 UUID"
// @Header 200 {string} X-Request-ID "请求关联 ID"
// @Header 200 {string} X-Config-Version "删除墓碑版本"
// @Header 200 {string} X-Config-Change-ID "配置变更 ID，重复删除为空"
// @Header 200 {string} X-Config-Propagation "asynchronous"
//
//	@Summary	删除 V3 接入项目
//	@Tags		V3Admin
//	@Produce	json
//	@Security	BearerAuth
//	@Param		Authorization	header		string	true	"Bearer 管理员 Token"
//	@Param		project_id		path		string	true	"项目 ID"
//	@Success	200				{object}	response.Response
//	@Router		/api/v3/admin/integrations/projects/{project_id} [delete]
func (h *V3Admin) DeleteProject(c *gin.Context) (response.Response, error) {
	receipt, err := h.service.DeleteProject(c.Request.Context(), c.Param("project_id"), configActor(c))
	if err != nil {
		return response.Response{}, err
	}
	configHeaders(c, receipt)
	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    nil,
	}, nil
}

// RotateAPIKey 重新生成项目 API Key，旧 Key 无法新兑换，已签发 JWT 保持有效。
// @Param X-Request-ID header string false "请求关联 UUID"
// @Header 200 {string} X-Request-ID "请求关联 ID"
// @Header 200 {string} X-Config-Version "已提交配置版本"
// @Header 200 {string} X-Config-Change-ID "配置变更 ID"
// @Header 200 {string} X-Config-Propagation "asynchronous"
//
//	@Summary		重新生成 V3 项目 API Key
//	@Description	提交后旧 Key 无法新兑换，已签发 JWT 保持有效。新 Key 明文只返回一次；网络断开导致结果未知时先用请求 ID 查审计，不自动重试。缓存异步应用。
//	@Tags			V3Admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			Authorization	header		string	true	"Bearer 管理员 Token"
//	@Param			project_id		path		string	true	"项目 ID"
//	@Success		200				{object}	response.Response{data=respV3.RotateAPIKeyResp}
//	@Router			/api/v3/admin/integrations/projects/{project_id}/keys/rotate [post]
func (h *V3Admin) RotateAPIKey(c *gin.Context) (response.Response, error) {
	result, err := h.service.RotateAPIKey(c.Request.Context(), c.Param("project_id"), configActor(c))
	if err != nil {
		return response.Response{}, err
	}

	configHeaders(c, result.Receipt)
	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    result,
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

func toProjectDetail(project model.FeedbackProjectV3) respV3.ProjectDetail {
	return respV3.ProjectDetail{
		ConfigVersion: project.ConfigVersion,
		ID:            project.ID,
		ProjectID:     project.ProjectID,
		ProjectName:   project.ProjectName,
		School:        project.School,
		Status:        project.Status,
		CreatedAt:     project.CreatedAt,
		UpdatedAt:     project.UpdatedAt,
	}
}

func toProjectKeyDetail(key *model.FeedbackProjectKeyV3) *respV3.ProjectKeyDetail {
	if key == nil {
		return nil
	}
	return &respV3.ProjectKeyDetail{
		ID:        key.ID,
		ProjectID: key.ProjectID,
		KeyID:     key.KeyID,
		Status:    key.Status,
		ExpiresAt: key.ExpiresAt,
		CreatedAt: key.CreatedAt,
		UpdatedAt: key.UpdatedAt,
	}
}

func toProjectTableItems(tables []model.FeedbackProjectTableV3) []respV3.ProjectTableItem {
	items := make([]respV3.ProjectTableItem, 0, len(tables))
	for _, table := range tables {
		items = append(items, respV3.ProjectTableItem{
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
			CreatedAt:     table.CreatedAt,
			UpdatedAt:     table.UpdatedAt,
		})
	}
	return items
}

func configActor(c *gin.Context) domain.ConfigActorV3 {
	actor, _ := c.Get(constvar.AdminIDContextKeyV3)
	id, _ := actor.(uint64)
	requestID := c.GetString("config_request_id")
	if requestID == "" {
		requestID = c.GetHeader("X-Request-ID")
		if _, err := uuid.Parse(requestID); err != nil {
			requestID = uuid.NewString()
		}
		c.Set("config_request_id", requestID)
	}
	c.Header("X-Request-ID", requestID)
	return domain.ConfigActorV3{AdminID: id, RequestID: requestID}
}
func configHeaders(c *gin.Context, r domain.ConfigReceiptV3) {
	c.Header("X-Config-Version", strconv.FormatUint(r.Version, 10))
	c.Header("X-Config-Change-ID", r.ChangeID)
	c.Header("X-Config-Propagation", "asynchronous")
}

// ConfigStatus reports only the instance serving this request and does not refresh it.
// @Summary 查询本实例 V3 配置应用状态
// @Tags V3Admin
// @Produce json
// @Security BearerAuth
// @Param project_id path string true "项目 ID"
// @Success 200 {object} response.Response{data=service.ProjectConfigStatusV3}
// @Router /api/v3/admin/integrations/projects/{project_id}/config-status [get]
func (h *V3Admin) ConfigStatus(c *gin.Context) (response.Response, error) {
	return response.Response{Code: 0, Message: "Success", Data: h.service.ConfigStatus(c.Request.Context(), c.Param("project_id"))}, nil
}

// ConfigAudits returns committed changes without configuration values.
// @Summary 查询 V3 配置变更审计
// @Tags V3Admin
// @Produce json
// @Security BearerAuth
// @Param project_id query string false "项目 ID"
// @Param request_id query string false "请求关联 ID"
// @Param before_id query int false "上一页最后 ID"
// @Param limit query int false "每页数量，默认 20，最大 100"
// @Success 200 {object} response.Response{data=[]model.ConfigAuditV3}
// @Router /api/v3/admin/integrations/config-audits [get]
func (h *V3Admin) ConfigAudits(c *gin.Context) (response.Response, error) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil {
		return response.Response{}, errs.V3InvalidInputError(err)
	}
	before, err := strconv.ParseUint(c.DefaultQuery("before_id", "0"), 10, 64)
	if err != nil {
		return response.Response{}, errs.V3InvalidInputError(err)
	}
	rows, err := h.service.ConfigAudits(c.Request.Context(), c.Query("project_id"), c.Query("request_id"), before, limit)
	return response.Response{Code: 0, Message: "Success", Data: rows}, err
}
