package controller

import (
	"github.com/gin-gonic/gin"
	reqV2 "github.com/muxi-Infra/FeedBack-Backend/api/request/v2"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	respV2 "github.com/muxi-Infra/FeedBack-Backend/api/response/v2"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
	"github.com/muxi-Infra/FeedBack-Backend/service"
)

type SheetV2Handler interface {
	GetTableRecordReqByUser(c *gin.Context, r reqV2.GetTableRecordByUserReq, uc ijwt.UserClaims) (response.Response, error)
	SyncUnsyncedTableRecords(c *gin.Context, r reqV2.SyncUnsyncedTableRecordsReq, uc ijwt.UserClaims) (response.Response, error)
	ForceSyncUserTableRecords(c *gin.Context, r reqV2.ForceSyncUserTableRecordsReq, uc ijwt.UserClaims) (response.Response, error)
	ForceSyncTableRecords(c *gin.Context, r reqV2.ForceSyncTableRecordsReq, uc ijwt.UserClaims) (response.Response, error)
}

type SheetV2 struct {
	s service.SheetService
	m service.MessageService
}

func NewSheetV2(s service.SheetService, m service.MessageService) SheetV2Handler {
	sheet := &SheetV2{
		s: s,
		m: m,
	}

	return sheet
}

// GetTableRecordReqByUser 获取用户历史反馈记录
//
//	@Summary		查询用户历史反馈记录
//	@Description	根据学号查询用户的历史反馈记录，支持分页查询，用于查看用户历史反馈内容。
//	@Tags			SheetV2
//	@ID				get-table-record-by-user
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string												true	"Bearer Token"
//	@Param			request			query		reqV2.GetTableRecordByUserReq						true	"查询记录请求参数"
//	@Success		200				{object}	response.Response{data=respV2.GetTableRecordResp}	"成功返回查询结果"
//	@Failure		400				{object}	response.Response									"请求参数错误"
//	@Failure		500				{object}	response.Response									"服务器内部错误"
//	@Router			/api/v2/sheet/records [get]
func (s *SheetV2) GetTableRecordReqByUser(c *gin.Context, r reqV2.GetTableRecordByUserReq, uc ijwt.UserClaims) (response.Response, error) {
	err := validateTableIdentify(*r.TableIdentify, uc.TableIdentity)
	if err != nil {
		return response.Response{}, err
	}

	// 组装参数
	tableConfig := domain.TableConfig{
		TableIdentity: &uc.TableIdentity,
		TableName:     &uc.TableName,
		TableToken:    &uc.TableToken,
		TableID:       &uc.TableId,
		ViewID:        &uc.ViewId,
	}

	serviceResult, err := s.s.GetTableRecordReqByUser(r.StudentID, r.PageToken, *r.LimitSize, &tableConfig)
	if err != nil {
		return response.Response{}, err
	}

	resp := respV2.GetTableRecordResp{
		Records:   make([]domain.TableRecord, 0),
		HasMore:   false,
		PageToken: "",
	}
	if serviceResult.Records != nil {
		resp.Records = serviceResult.Records
	}
	if serviceResult.PageToken != nil {
		resp.PageToken = *serviceResult.PageToken
	}
	if serviceResult.HasMore != nil {
		resp.HasMore = *serviceResult.HasMore
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp,
	}, nil
}

// SyncUnsyncedTableRecords 同步指定表格下所有未同步记录
//
//	@Summary		同步未同步记录
//	@Description	同步指定表格下 is_synced = false 的所有记录，用于后台增量同步。
//	@Tags			SheetV2
//	@ID				sync-unsynced-table-records
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string														true	"Bearer Token"
//	@Param			request			body		reqV2.SyncUnsyncedTableRecordsReq							true	"同步请求参数"
//	@Success		200				{object}	response.Response{data=respV2.SyncUnsyncedTableRecordsResp}	"同步成功"
//	@Failure		400				{object}	response.Response											"请求参数错误"
//	@Failure		500				{object}	response.Response											"服务器内部错误"
//	@Router			/api/v2/sheet/sync [post]
func (s *SheetV2) SyncUnsyncedTableRecords(c *gin.Context, r reqV2.SyncUnsyncedTableRecordsReq, uc ijwt.UserClaims) (response.Response, error) {
	// 校验表权限
	if err := validateTableIdentify(*r.TableIdentify, uc.TableIdentity); err != nil {
		return response.Response{}, err
	}

	tableConfig := domain.TableConfig{
		TableIdentity: &uc.TableIdentity,
		TableName:     &uc.TableName,
		TableToken:    &uc.TableToken,
		TableID:       &uc.TableId,
		ViewID:        &uc.ViewId,
	}

	// 调用 service 层
	recordIDs, total, full, err := s.s.SyncUnsyncedTableRecords(&tableConfig)
	if err != nil {
		return response.Response{}, err
	}

	resp := respV2.SyncUnsyncedTableRecordsResp{
		RecordIDs: make([]string, 0),
		QueueFull: full,
		Total:     total,
	}
	if recordIDs != nil {
		resp.RecordIDs = recordIDs
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp,
	}, nil
}

// ForceSyncUserTableRecords 强制同步某用户的所有记录
//
//	@Summary		强制同步用户记录
//	@Description	同步指定用户在某表格下的所有记录（不区分是否已同步），用于全量重建或修复数据。
//	@Tags			SheetV2
//	@ID				force-sync-user-table-records
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string															true	"Bearer Token"
//	@Param			request			body		reqV2.ForceSyncUserTableRecordsReq								true	"强制同步请求参数"
//	@Success		200				{object}	response.Response{data=respV2.ForceSyncUserTableRecordsResp}	"同步成功"
//	@Failure		400				{object}	response.Response												"请求参数错误"
//	@Failure		500				{object}	response.Response												"服务器内部错误"
//	@Router			/api/v2/sheet/sync/user [post]
func (s *SheetV2) ForceSyncUserTableRecords(c *gin.Context, r reqV2.ForceSyncUserTableRecordsReq, uc ijwt.UserClaims) (response.Response, error) {
	// 校验表权限
	if err := validateTableIdentify(*r.TableIdentify, uc.TableIdentity); err != nil {
		return response.Response{}, err
	}

	tableConfig := domain.TableConfig{
		TableIdentity: &uc.TableIdentity,
		TableName:     &uc.TableName,
		TableToken:    &uc.TableToken,
		TableID:       &uc.TableId,
		ViewID:        &uc.ViewId,
	}

	// 调用 service 层
	recordIDs, total, full, err := s.s.ForceSyncUserTableRecords(r.StudentID, &tableConfig)
	if err != nil {
		return response.Response{}, err
	}

	resp := respV2.ForceSyncUserTableRecordsResp{
		RecordIDs: make([]string, 0),
		QueueFull: full,
		Total:     total,
	}

	if recordIDs != nil {
		resp.RecordIDs = recordIDs
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp,
	}, nil
}

// ForceSyncTableRecords 强制同步表格的所有记录
//
//	@Summary		强制同步表格所有记录
//	@Description	同步某表格下的所有记录（不区分是否已同步），用于全量重建或修复数据。慎用！！！
//	@Tags			SheetV2
//	@ID				force-sync-table-records
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string														true	"Bearer Token"
//	@Param			request			body		reqV2.ForceSyncTableRecordsReq								true	"强制同步请求参数"
//	@Success		200				{object}	response.Response{data=respV2.ForceSyncTableRecordsResp}	"同步成功"
//	@Failure		400				{object}	response.Response											"请求参数错误"
//	@Failure		500				{object}	response.Response											"服务器内部错误"
//	@Router			/api/v2/sheet/sync/force [post]
func (s *SheetV2) ForceSyncTableRecords(c *gin.Context, r reqV2.ForceSyncTableRecordsReq, uc ijwt.UserClaims) (response.Response, error) {
	// 校验表权限
	if err := validateTableIdentify(*r.TableIdentify, uc.TableIdentity); err != nil {
		return response.Response{}, err
	}

	tableConfig := domain.TableConfig{
		TableIdentity: &uc.TableIdentity,
		TableName:     &uc.TableName,
		TableToken:    &uc.TableToken,
		TableID:       &uc.TableId,
		ViewID:        &uc.ViewId,
	}

	// 调用 service 层
	recordIDs, total, full, err := s.s.ForceSyncTableRecords(&tableConfig)
	if err != nil {
		return response.Response{}, err
	}

	resp := respV2.ForceSyncTableRecordsResp{
		RecordIDs: make([]string, 0),
		QueueFull: full,
		Total:     total,
	}

	if recordIDs != nil {
		resp.RecordIDs = recordIDs
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp,
	}, nil
}
