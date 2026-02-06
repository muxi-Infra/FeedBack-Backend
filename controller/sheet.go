package controller

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/muxi-Infra/FeedBack-Backend/api/request"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
	"github.com/muxi-Infra/FeedBack-Backend/service"
)

type Sheet struct {
	s service.SheetService
	m service.MessageService
}

func NewSheet(s service.SheetService, m service.MessageService) *Sheet {
	sheet := &Sheet{
		s: s,
		m: m,
	}

	return sheet
}

// CreateTableRecord 创建多维表格记录
//
//	@Summary		创建反馈记录
//	@Description	向指定的多维表格应用中添加用户反馈记录，支持文本内容、截图附件和联系方式。创建成功后会异步发送通知消息。
//	@Tags			Sheet
//	@ID				create-table-record
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string													true	"Bearer Token"
//	@Param			request			body		request.CreatTableRecordReg								true	"新增记录请求参数"
//	@Success		200				{object}	response.Response{data=response.CreatTableRecordResp}	"成功返回创建记录结果"
//	@Failure		400				{object}	response.Response										"请求参数错误或飞书接口调用失败"
//	@Failure		500				{object}	response.Response										"服务器内部错误"
//	@Router			/api/v1/sheet/records [post]
func (s *Sheet) CreateTableRecord(c *gin.Context, r request.CreatTableRecordReg, uc ijwt.UserClaims) (response.Response, error) {
	err := validateTableIdentify(*r.TableIdentify, uc.TableIdentity)
	if err != nil {
		return response.Response{}, err
	}

	// 组装参数
	record, err := buildCreateTableRecord(r)
	if err != nil {
		return response.Response{}, err
	}
	tableConfig := domain.TableConfig{
		TableIdentity: &uc.TableIdentity,
		TableName:     &uc.TableName,
		TableToken:    &uc.TableToken,
		TableID:       &uc.TableId,
		ViewID:        &uc.ViewId,
	}

	// 发起请求
	createdRecordID, err := s.s.CreateRecord(record, &tableConfig)
	if err != nil {
		return response.Response{}, err
	}

	if createdRecordID == nil {
		return response.Response{
			Code:    0,
			Message: "Success",
			Data:    "",
		}, nil
	}

	// TODO 后续想改成 kafka 异步处理
	go func(recordID, content string, tc domain.TableConfig) {
		// 发送消息通知
		_, url, err := s.s.GetTableRecordReqByRecordID(&recordID, &tc)
		if err != nil || url == nil {
			return
		}
		err = s.m.SendLarkNotification(*tc.TableName, content, *url)
		if err != nil {
			return
		}
	}(*createdRecordID, *r.Content, tableConfig)

	resp := response.CreatTableRecordResp{
		RecordID: *createdRecordID,
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp,
	}, nil
}

// GetTableRecordReqByKey 获取用户历史反馈记录
//
//	@Summary		查询历史反馈记录
//	@Description	根据指定的字段条件查询用户的历史反馈记录，支持分页查询。通常用于查看用户之前提交的反馈内容。
//	@Tags			Sheet
//	@ID				get-table-record
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string												true	"Bearer Token"
//	@Param			request			query		request.GetTableRecordReq							true	"查询记录请求参数"
//	@Success		200				{object}	response.Response{data=response.GetTableRecordResp}	"成功返回查询结果"
//	@Failure		400				{object}	response.Response									"请求参数错误或飞书接口调用失败"
//	@Failure		500				{object}	response.Response									"服务器内部错误"
//	@Router			/api/v1/sheet/records [get]
func (s *Sheet) GetTableRecordReqByKey(c *gin.Context, r request.GetTableRecordReq, uc ijwt.UserClaims) (response.Response, error) {
	err := validateTableIdentify(*r.TableIdentify, uc.TableIdentity)
	if err != nil {
		return response.Response{}, err
	}

	// 组装参数
	keyField := domain.TableField{
		FieldName: r.KeyFieldName,
		Value:     r.KeyFieldValue,
	}
	tableConfig := domain.TableConfig{
		TableIdentity: &uc.TableIdentity,
		TableName:     &uc.TableName,
		TableToken:    &uc.TableToken,
		TableID:       &uc.TableId,
		ViewID:        &uc.ViewId,
	}

	serviceResult, err := s.s.GetTableRecordReqByKey(&keyField, r.RecordNames, r.PageToken, &tableConfig)
	if err != nil {
		return response.Response{}, err
	}

	resp := response.GetTableRecordResp{
		Records:   make([]domain.TableRecord, 0),
		HasMore:   false,
		PageToken: "",
		Total:     0,
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
	if serviceResult.Total != nil {
		resp.Total = *serviceResult.Total
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp,
	}, nil
}

// GetTableRecordReqByRecordID 根据记录 ID 查询用户历史反馈记录
//
//	@Summary		按 RecordID 查询历史反馈记录
//	@Description	根据 record_id 获取单条用户反馈记录的详细内容，返回记录字段的键值对。请求中需包含合法的 table_identify，用于校验权限。
//	@Tags			Sheet
//	@ID				get-table-record-by-id
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string															true	"Bearer Token"
//	@Param			request			query		request.GetTableRecordByRecordIDReq								true	"查询记录请求参数"
//	@Success		200				{object}	response.Response{data=response.GetTableRecordByRecordIdResp}	"成功返回查询结果"
//	@Failure		400				{object}	response.Response												"请求参数错误或飞书接口调用失败"
//	@Failure		500				{object}	response.Response												"服务器内部错误"
//	@Router			/api/v1/sheet/record [get]
func (s *Sheet) GetTableRecordReqByRecordID(c *gin.Context, r request.GetTableRecordByRecordIDReq, uc ijwt.UserClaims) (response.Response, error) {
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

	serviceResult, _, err := s.s.GetTableRecordReqByRecordID(r.RecordID, &tableConfig)
	if err != nil {
		return response.Response{}, err
	}

	resp := response.GetTableRecordByRecordIdResp{
		Record: make(map[string]any),
	}

	if serviceResult != nil {
		resp.Record = serviceResult
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp,
	}, nil
}

// GetFAQResolutionRecord 获取常见问题及解决状态
//
//	@Summary		查询FAQ问题记录
//	@Description	根据学号查询用户相关的常见问题记录及其解决状态。
//	@Tags			Sheet
//	@ID				get-faq-resolution-record
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string															true	"Bearer Token"
//	@Param			request			query		request.GetTableRecordByRecordIDReq								true	"查询记录请求参数，包含 record_id 和 table_identify"
//	@Success		200				{object}	response.Response{data=response.GetTableRecordByRecordIdResp}	"成功返回单条记录的字段键值对"
//	@Failure		400				{object}	response.Response												"请求参数错误或飞书接口调用失败"
//	@Failure		500				{object}	response.Response												"服务器内部错误"
//	@Router			/api/v1/sheet/records/faq [get]
func (s *Sheet) GetFAQResolutionRecord(c *gin.Context, r request.GetFAQProblemTableRecordReg, uc ijwt.UserClaims) (response.Response, error) {
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

	faqServiceResult, err := s.s.GetFAQProblemTableRecord(r.StudentID, r.RecordNames, &tableConfig)
	if err != nil {
		return response.Response{}, err
	}

	resp := response.GetFAQProblemTableRecordResp{
		Records: make([]domain.FAQTableRecord, 0),
		Total:   0,
	}

	if faqServiceResult.Records != nil {
		resp.Records = faqServiceResult.Records
	}
	if faqServiceResult.Total != nil {
		resp.Total = *faqServiceResult.Total
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp,
	}, nil
}

// UpdateFAQResolutionRecord 更新FAQ问题解决状态
//
//	@Summary		标记FAQ问题解决状态
//	@Description	用户更新FAQ问题的解决状态，将问题标记为已解决或未解决。
//	@Tags			Sheet
//	@ID				update-faq-resolution
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string							true	"Bearer Token"
//	@Param			request			body		request.FAQResolutionUpdateReq	true	"更新FAQ解决状态请求参数"
//	@Success		200				{object}	response.Response				"成功更新FAQ解决状态"
//	@Failure		400				{object}	response.Response				"请求参数错误或飞书接口调用失败"
//	@Failure		500				{object}	response.Response				"服务器内部错误"
//	@Router			/api/v1/sheet/records/faq [post]
func (s *Sheet) UpdateFAQResolutionRecord(c *gin.Context, r request.FAQResolutionUpdateReq, uc ijwt.UserClaims) (response.Response, error) {
	err := validateTableIdentify(*r.TableIdentify, uc.TableIdentity)
	if err != nil {
		return response.Response{}, err
	}

	// 组装参数
	FAQResolution := domain.FAQResolution{
		UserID:              r.UserID,
		RecordID:            r.RecordID,
		ResolvedFieldName:   r.ResolvedFieldName,
		UnresolvedFieldName: r.UnresolvedFieldName,
		IsResolved:          r.IsResolved,
	}
	tableConfig := domain.TableConfig{
		TableIdentity: &uc.TableIdentity,
		TableName:     &uc.TableName,
		TableToken:    &uc.TableToken,
		TableID:       &uc.TableId,
		ViewID:        &uc.ViewId,
	}

	err = s.s.UpdateFAQResolutionRecord(&FAQResolution, &tableConfig)
	if err != nil {
		return response.Response{}, err
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    nil,
	}, nil
}

// GetPhotoUrl 获取截图临时下载链接
//
//	@Summary		获取截图临时URL
//	@Description	根据文件Token列表批量获取截图的临时下载URL，用于前端展示或下载图片。
//	@Tags			Sheet
//	@ID				get-photo-url
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string					true	"Bearer Token"
//	@Param			request			query		request.GetPhotoUrlReq	true	"获取截图URL请求参数"
//	@Success		200				{object}	response.Response		"成功返回临时URL信息"
//	@Failure		400				{object}	response.Response		"请求参数错误或飞书接口调用失败"
//	@Failure		500				{object}	response.Response		"服务器内部错误"
//	@Router			/api/v1/sheet/photos/url [get]
func (s *Sheet) GetPhotoUrl(c *gin.Context, r request.GetPhotoUrlReq, uc ijwt.UserClaims) (response.Response, error) {
	photoUrlResult, err := s.s.GetPhotoUrl(r.FileTokens)
	if err != nil {
		return response.Response{}, err
	}

	photoUrlResponse := response.GetPhotoUrlResp{
		Files: photoUrlResult,
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    photoUrlResponse,
	}, nil
}

func validateTableIdentify(a, b string) error {
	if a != b {
		return errs.TableIdentifierInvalidError(errors.New("table identify is not invalid"))
	}
	return nil
}

// buildCreateTableRecord 组装以及校验创建记录的参数
func buildCreateTableRecord(r request.CreatTableRecordReg) (*domain.TableRecord, error) {
	// 拷贝 ExtraRecord，避免修改调用方原始 map
	totalRecord := make(map[string]any, len(r.ExtraRecord)+4)
	for k, v := range r.ExtraRecord {
		totalRecord[k] = v
	}

	// 必填字段校验
	if r.StudentID == nil {
		return nil, errs.CreateRecordEmptyStudentIDError(errors.New("student_id is required"))
	} else if len(*r.StudentID) != 10 { // 学号长度为10，后续可以追加校验真实学号
		return nil, errs.CreateRecordInvalidStudentIDError(errors.New("student_id is invalid, length must be 10"))
	}
	totalRecord["学号"] = *r.StudentID
	if r.Content == nil {
		return nil, errs.CreateRecordEmptyContentError(errors.New("content is required"))
	} else if len(*r.Content) == 0 {
		return nil, errs.CreateRecordEmptyContentError(errors.New("content is required"))
	}
	totalRecord["反馈内容"] = *r.Content

	// 可选字段处理
	// 将图片文件 token 数组转换为 Feishu API 期望的对象数组: [{"file_token": "your_file_token"}]
	fileObjs := make([]map[string]string, 0, len(r.Images))
	for _, t := range r.Images {
		fileObjs = append(fileObjs, map[string]string{"file_token": t})
	}
	totalRecord["截图"] = fileObjs
	if r.ContactInfo != nil {
		totalRecord["联系方式（QQ/邮箱）"] = *r.ContactInfo
	}

	record := &domain.TableRecord{
		Record: totalRecord,
	}
	return record, nil
}
