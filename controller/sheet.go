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
}

func NewSheet(s service.SheetService) *Sheet {
	return &Sheet{
		s: s,
	}
}

// CreatTableRecord 创建多维表格记录
//
//	@Summary		创建多维表格记录
//	@Description	向指定的多维表格应用中添加记录数据
//	@Tags			Sheet
//	@ID				create-table-record
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string						true	"Bearer Token"
//	@Param			request			body		request.CreatTableRecordReg	true	"新增记录请求参数"
//	@Success		200				{object}	response.Response			"成功返回创建记录结果"
//	@Failure		400				{object}	response.Response			"请求参数错误或飞书接口调用失败"
//	@Failure		500				{object}	response.Response			"服务器内部错误"
//	@Router			/api/v1/sheet/records [post]
func (s *Sheet) CreatTableRecord(c *gin.Context, r request.CreatTableRecordReg, uc ijwt.UserClaims) (response.Response, error) {
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
	resp, err := s.s.CreateRecord(record, &tableConfig)
	if err != nil {
		return response.Response{}, err
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp,
	}, nil
}

// GetTableRecordReqByKey 获取多维表格记录
//
//	@Summary		获取多维表格记录
//	@Description	根据指定条件查询多维表格中的记录数据
//	@Tags			Sheet
//	@ID				get-table-record
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string						true	"Bearer Token"
//	@Param			request			query		request.GetTableRecordReq	true	"查询记录请求参数"
//	@Success		200				{object}	response.Response			"成功返回查询结果"
//	@Failure		400				{object}	response.Response			"请求参数错误或飞书接口调用失败"
//	@Failure		500				{object}	response.Response			"服务器内部错误"
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

	resp, err := s.s.GetTableRecordReqByKey(&keyField, r.RecordNames, r.PageToken, &tableConfig)
	if err != nil {
		return response.Response{}, err
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp,
	}, nil
}

// GetFAQResolutionRecord 获取常见问题记录
//
//	@Summary		获取常见问题记录
//	@Description	根据指定条件查询多维表格中的记录数据
//	@Tags			Sheet
//	@ID				get-faq-resolution-record
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string								true	"Bearer Token"
//	@Param			request			query		request.GetFAQProblemTableRecordReg	true	"查询记录请求参数"
//	@Success		200				{object}	response.Response					"成功返回查询结果"
//	@Failure		400				{object}	response.Response					"请求参数错误或飞书接口调用失败"
//	@Failure		500				{object}	response.Response					"服务器内部错误"
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

	resp, err := s.s.GetFAQProblemTableRecord(r.StudentID, r.RecordNames, &tableConfig)
	if err != nil {
		return response.Response{}, err
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp,
	}, nil
}

// UpdateFAQResolutionRecord 更新FAQ解决方案的 已解决/未解决 状态
//
//	@Summary		更新FAQ解决方案的 已解决/未解决 状态
//	@Description	更新FAQ解决方案的 已解决/未解决 状态
//	@Tags			Sheet
//	@ID				update-faq-resolution
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string							true	"Bearer Token"
//	@Param			request			body		request.FAQResolutionUpdateReq	true	"查询记录请求参数"
//	@Success		200				{object}	response.Response				"成功返回查询结果"
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

// GetPhotoUrl 获取截图的临时 url 24小时过期
//
//	@Summary		获取截图临时URL
//	@Description	根据文件Token获取截图的临时下载URL，URL有效期为24小时
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
func (s *Sheet) GetPhotoUrl(c *gin.Context, r request.GetPhotoUrlReq, uc ijwt.UserClaims) (res response.Response, err error) {
	resp, err := s.s.GetPhotoUrl(r.FileTokens)
	if err != nil {
		return response.Response{}, err
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp.Data,
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
