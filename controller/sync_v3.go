package controller

import (
	"github.com/gin-gonic/gin"
	reqV3 "github.com/muxi-Infra/FeedBack-Backend/api/request/v3"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	respV3 "github.com/muxi-Infra/FeedBack-Backend/api/response/v3"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/constvar"
	"github.com/muxi-Infra/FeedBack-Backend/service"
)

// V3SyncHandler 管理员同步项目数据
type V3SyncHandler interface {
	SyncUnsynced(*gin.Context, reqV3.SyncProjectReq) (response.Response, error)
	ForceSyncUser(*gin.Context, reqV3.SyncProjectUserReq) (response.Response, error)
	ForceSyncAll(*gin.Context, reqV3.SyncProjectReq) (response.Response, error)
	SyncFAQ(*gin.Context, reqV3.SyncProjectReq) (response.Response, error)
}

type V3Sync struct {
	// todo 当前复用旧版 SheetService 的同步实现，后续 V1/V2 下线后迁移到 V3 服务。
	sheets service.SheetService
	auth   service.V3AuthService
}

func NewV3Sync(sheets service.SheetService, auth service.V3AuthService) V3SyncHandler {
	return &V3Sync{
		sheets: sheets,
		auth:   auth,
	}
}

// SyncUnsynced 同步指定项目尚未同步的反馈记录。
//
//	@Summary		同步 V3 项目未同步反馈
//	@Tags		V3AdminSync
//	@Accept		json
//	@Produce	json
//	@Param		request	body	reqV3.SyncProjectReq	true	"项目参数"
//	@Success	200	{object}	response.Response{data=respV3.SyncRecordsResp}
//	@Router		/api/v3/admin/sheet/feedback/sync [post]
func (h *V3Sync) SyncUnsynced(c *gin.Context, req reqV3.SyncProjectReq) (response.Response, error) {
	config, err := h.auth.GetTableConfig(c.Request.Context(), req.ProjectID, constvar.FeedbackTableType)
	if err != nil {
		return response.Response{}, err
	}
	ids, total, full, err := h.sheets.SyncUnsyncedTableRecords(legacyConfig(config))
	if err != nil {
		return response.Response{}, err
	}
	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    syncResponse(ids, total, full),
	}, nil
}

// ForceSyncUser 强制同步指定学生的全部反馈记录。
//
//	@Summary		强制同步 V3 项目指定学生反馈
//	@Tags		V3AdminSync
//	@Accept		json
//	@Produce	json
//	@Param		request	body	reqV3.SyncProjectUserReq	true	"项目和学生参数"
//	@Success	200	{object}	response.Response{data=respV3.SyncRecordsResp}
//	@Router		/api/v3/admin/sheet/feedback/sync/user [post]
func (h *V3Sync) ForceSyncUser(c *gin.Context, req reqV3.SyncProjectUserReq) (response.Response, error) {
	config, err := h.auth.GetTableConfig(c.Request.Context(), req.ProjectID, constvar.FeedbackTableType)
	if err != nil {
		return response.Response{}, err
	}
	ids, total, full, err := h.sheets.ForceSyncUserTableRecords(&req.StudentID, legacyConfig(config))
	if err != nil {
		return response.Response{}, err
	}
	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    syncResponse(ids, total, full),
	}, nil
}

// ForceSyncAll 强制同步指定项目的全部反馈记录。
//
//	@Summary		强制同步 V3 项目全部反馈
//	@Tags		V3AdminSync
//	@Accept		json
//	@Produce	json
//	@Param		request	body	reqV3.SyncProjectReq	true	"项目参数"
//	@Success	200	{object}	response.Response{data=respV3.SyncRecordsResp}
//	@Router		/api/v3/admin/sheet/feedback/sync/force [post]
func (h *V3Sync) ForceSyncAll(c *gin.Context, req reqV3.SyncProjectReq) (response.Response, error) {
	config, err := h.auth.GetTableConfig(c.Request.Context(), req.ProjectID, constvar.FeedbackTableType)
	if err != nil {
		return response.Response{}, err
	}
	ids, total, full, err := h.sheets.ForceSyncTableRecords(legacyConfig(config))
	if err != nil {
		return response.Response{}, err
	}
	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    syncResponse(ids, total, full),
	}, nil
}

// SyncFAQ 同步指定项目 FAQ 的解决状态统计。
//
//	@Summary		同步 V3 项目 FAQ
//	@Tags		V3AdminSync
//	@Accept		json
//	@Produce	json
//	@Param		request	body	reqV3.SyncProjectReq	true	"项目参数"
//	@Success	200	{object}	response.Response
//	@Router		/api/v3/admin/sheet/faq/sync [post]
func (h *V3Sync) SyncFAQ(c *gin.Context, req reqV3.SyncProjectReq) (response.Response, error) {
	config, err := h.auth.GetTableConfig(c.Request.Context(), req.ProjectID, constvar.FAQTableType)
	if err != nil {
		return response.Response{}, err
	}
	if err := h.sheets.SyncFAQRecord(legacyConfig(config)); err != nil {
		return response.Response{}, err
	}
	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    nil,
	}, nil
}

// legacyConfig 将 V3 表配置转换为旧版 SheetService 使用的配置结构。
// todo 后续同步服务迁移到 V3 后删除该兼容转换。
func legacyConfig(config service.V3TableConfig) *domain.TableConfig {
	item := config.Legacy()
	return &item
}

func syncResponse(ids []string, total int, full bool) respV3.SyncRecordsResp {
	if ids == nil {
		ids = []string{}
	}

	return respV3.SyncRecordsResp{
		RecordIDs: ids,
		Total:     total,
		QueueFull: full,
	}
}
