package controller

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	reqV1 "github.com/muxi-Infra/FeedBack-Backend/api/request/v1"
	reqV3 "github.com/muxi-Infra/FeedBack-Backend/api/request/v3"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	respV1 "github.com/muxi-Infra/FeedBack-Backend/api/response/v1"
	respV2 "github.com/muxi-Infra/FeedBack-Backend/api/response/v2"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/constvar"
	"github.com/muxi-Infra/FeedBack-Backend/service"
)

type V3SheetHandler interface {
	CreateFeedback(*gin.Context, reqV3.CreateFeedbackReq, string, string) (response.Response, error)
	GetFeedback(*gin.Context, reqV3.GetFeedbackReq, string, string) (response.Response, error)
	GetRecord(*gin.Context, reqV3.GetRecordReq, string, string) (response.Response, error)
	GetFAQ(*gin.Context, reqV3.GetFAQReq, string, string) (response.Response, error)
	UpdateFAQ(*gin.Context, reqV3.UpdateFAQReq, string, string) (response.Response, error)
	GetPhotoURL(*gin.Context, reqV3.GetPhotoURLReq, string, string) (response.Response, error)
}

type V3Sheet struct {
	// todo 当前复用 V1/V2 共用的 SheetService 同步实现， 后续迁移到 V3 专用同步服务后删除。
	sheet service.SheetService
	// todo 当前复用旧版 MessageService 的通知能力，后续迁移到 V3 通知服务。
	message service.MessageService
	auth    service.V3AuthService
}

func NewV3Sheet(sheet service.SheetService, message service.MessageService, auth service.V3AuthService) V3SheetHandler {
	return &V3Sheet{sheet: sheet, message: message, auth: auth}
}

func v3Forbidden(scope string) error {
	return errs.V3ProjectTokenScopeForbiddenError(scope)
}

func (s *V3Sheet) config(c *gin.Context, projectID, tableType, scope string) (domain.TableConfig, error) {
	cfg, err := s.auth.GetTableConfig(c.Request.Context(), projectID, tableType)
	if err != nil {
		return domain.TableConfig{}, err
	}
	if !cfg.HasScope(scope) {
		return domain.TableConfig{}, v3Forbidden(scope)
	}
	return cfg.Legacy(), nil
}

// CreateFeedback 创建当前登录学生的反馈记录。
//
//	@Summary		创建 V3 反馈记录
//	@Description	学生身份从用户反馈 Token 中解析，前端不得传入 student_id 或表格标识。
//	@Tags			V3Sheet
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string					true	"Bearer 用户反馈 Token"
//	@Param			request			body		reqV3.CreateFeedbackReq	true	"反馈内容"
//	@Success		200				{object}	response.Response{data=respV1.CreatTableRecordResp}
//	@Router			/api/v3/sheet/feedback/records [post]
func (s *V3Sheet) CreateFeedback(c *gin.Context, req reqV3.CreateFeedbackReq, projectID, studentID string) (response.Response, error) {
	// 创建是唯一需要复用 V1 飞书写入能力的 V3 Sheet 操作。
	cfg, err := s.config(c, projectID, constvar.FeedbackTableType, constvar.FeedbackScopeCreate)
	if err != nil {
		return response.Response{}, err
	}

	student := studentID
	r := reqV1.CreatTableRecordReg{
		StudentID:   &student,
		Content:     req.Content,
		Images:      req.Images,
		ContactInfo: req.ContactInfo,
		ExtraRecord: req.ExtraRecord,
	}

	record, err := buildCreateTableRecord(r)
	if err != nil {
		return response.Response{}, err
	}
	record.Record["进度"] = "待处理"
	record.Record["提交时间"] = time.Now().UnixMilli()

	id, err := s.sheet.CreateLarkRecord(record, &cfg)
	if err != nil {
		return response.Response{}, err
	}
	if id == nil {
		return response.Response{
			Code:    0,
			Message: "Success",
			Data:    "",
		}, nil
	}

	// 创建成功后异步补齐本地记录；只有项目配置开启通知时才发送飞书通知。
	go func(recordID, content string, tableConfig domain.TableConfig) {
		recordData, shareURL, getErr := s.sheet.GetTableRecordReqByRecordID(&recordID, &tableConfig)
		if getErr != nil || shareURL == nil {
			return
		}
		if tableConfig.Notice {
			if notifyErr := s.message.SendLarkNotification(*tableConfig.TableName, content, *shareURL); notifyErr != nil {
				return
			}
		}
		_ = s.sheet.CreateDBRecord(&recordID, shareURL, recordData, tableConfig)
	}(*id, *req.Content, cfg)

	return response.Response{
		Code:    0,
		Message: "Success",
		Data: respV1.CreatTableRecordResp{
			RecordID: *id,
		},
	}, nil
}

// GetFeedback 分页查询当前登录学生的历史反馈记录。
//
//	@Summary	查询当前学生的 V3 反馈记录
//	@Tags		V3Sheet
//	@Produce	json
//	@Param		Authorization	header		string	true	"Bearer 用户反馈 Token"
//	@Param		page_token		query		string	false	"上一页返回的分页 Token"
//	@Param		limit_size		query		int		false	"每页数量，默认 20"
//	@Success	200				{object}	response.Response{data=domain.TableRecords}
//	@Router		/api/v3/sheet/feedback/records [get]
func (s *V3Sheet) GetFeedback(c *gin.Context, req reqV3.GetFeedbackReq, projectID, studentID string) (response.Response, error) {
	// 复用 V2 的数据库分页查询路径。
	cfg, err := s.config(c, projectID, constvar.FeedbackTableType, constvar.FeedbackScopeReadSelf)
	if err != nil {
		return response.Response{}, err
	}
	limit := req.LimitSize
	if limit <= 0 {
		limit = 20
	}

	result, err := s.sheet.GetTableRecordReqByUser(&studentID, req.PageToken, limit, &cfg)
	if err != nil {
		return response.Response{}, err
	}
	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    result,
	}, nil
}

// GetRecord 查询当前登录学生的一条反馈记录。
//
//	@Summary	查询单条 V3 反馈记录
//	@Tags		V3Sheet
//	@Produce	json
//	@Param		Authorization	header		string	true	"Bearer 用户反馈 Token"
//	@Param		record_id		query		string	true	"反馈记录 ID"
//	@Success	200				{object}	response.Response{data=respV1.GetTableRecordByRecordIdResp}
//	@Router		/api/v3/sheet/feedback/record [get]
func (s *V3Sheet) GetRecord(c *gin.Context, req reqV3.GetRecordReq, projectID, studentID string) (response.Response, error) {
	cfg, err := s.config(c, projectID, constvar.FeedbackTableType, constvar.FeedbackScopeReadSelf)
	if err != nil {
		return response.Response{}, err
	}
	// 复用 V2 的本地数据库查询路径，查询条件同时绑定当前学生，避免跨用户读取。
	record, err := s.sheet.GetTableRecordByUserAndRecordID(&studentID, &req.RecordID, &cfg)
	if err != nil {
		return response.Response{}, err
	}
	return response.Response{
		Code:    0,
		Message: "Success",
		Data: respV1.GetTableRecordByRecordIdResp{
			Record: record,
		}}, nil
}

// GetFAQ 查询当前项目的 FAQ。
//
//	@Summary	查询 V3 FAQ 记录
//	@Tags		V3Sheet
//	@Produce	json
//	@Param		Authorization	header		string	true	"Bearer 用户反馈 Token"
//	@Success	200				{object}	response.Response{data=respV2.GetTableRecordByRecordIdResp}
//	@Router		/api/v3/sheet/faq/records [get]
func (s *V3Sheet) GetFAQ(c *gin.Context, _ reqV3.GetFAQReq, projectID, studentID string) (response.Response, error) {
	cfg, err := s.config(c, projectID, constvar.FAQTableType, constvar.FeedbackScopeRead)
	if err != nil {
		return response.Response{}, err
	}
	// 复用 V2 的 FAQ 本地数据库查询路径。
	result, err := s.sheet.GetFAQResolutionRecord(&studentID, &cfg)
	if err != nil {
		return response.Response{}, err
	}
	return response.Response{
		Code:    0,
		Message: "Success",
		Data: respV2.GetTableRecordByRecordIdResp{
			Records: result,
		},
	}, nil
}

// UpdateFAQ 更新当前学生对 FAQ 的解决状态。
//
//	@Summary	更新 V3 FAQ 解决状态
//	@Tags		V3Sheet
//	@Accept		json
//	@Produce	json
//	@Param		Authorization	header		string				true	"Bearer 用户反馈 Token"
//	@Param		request			body		reqV3.UpdateFAQReq	true	"FAQ 解决状态"
//	@Success	200				{object}	response.Response
//	@Router		/api/v3/sheet/faq/records [post]
func (s *V3Sheet) UpdateFAQ(c *gin.Context, req reqV3.UpdateFAQReq, projectID, studentID string) (response.Response, error) {
	cfg, err := s.config(c, projectID, constvar.FAQTableType, constvar.FeedbackScopeWrite)
	if err != nil {
		return response.Response{}, err
	}
	err = s.sheet.UpdateFAQResolutionRecordV2(&domain.FAQResolutionV2{
		RecordID:   req.RecordID,
		UserID:     &studentID,
		IsResolved: req.IsResolved,
	}, &cfg)
	if err != nil {
		return response.Response{}, err
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    nil,
	}, nil
}

// GetPhotoURL 获取当前反馈项目中的图片临时访问地址。
//
//	@Summary	获取反馈图片临时 URL
//	@Tags		V3Sheet
//	@Produce	json
//	@Param		Authorization	header		string		true	"Bearer 用户反馈 Token"
//	@Param		record_id		query		string		true	"反馈记录 ID"
//	@Param		file_tokens		query		[]string	true	"飞书文件 Token"
//	@Success	200				{object}	response.Response
//	@Router		/api/v3/sheet/feedback/photos/url [get]
func (s *V3Sheet) GetPhotoURL(c *gin.Context, req reqV3.GetPhotoURLReq, projectID, studentID string) (response.Response, error) {
	cfg, err := s.config(c, projectID, constvar.FeedbackTableType, constvar.FeedbackScopeReadSelf)
	if err != nil {
		return response.Response{}, err
	}

	// 复用 V2 的本地数据库查询路径校验记录归属；飞书只用于换取图片临时 URL。
	record, err := s.sheet.GetTableRecordByUserAndRecordID(&studentID, &req.RecordID, &cfg)
	if err != nil {
		return response.Response{}, err
	}
	if !photoTokensBelongToRecord(record["截图"], req.FileTokens) {
		return response.Response{}, errs.V3FeedbackPhotoForbiddenError(errors.New("photo token does not belong to record"))
	}

	files, err := s.sheet.GetPhotoUrl(req.FileTokens)
	if err != nil {
		return response.Response{}, err
	}
	return response.Response{
		Code:    0,
		Message: "Success",
		Data: respV1.GetPhotoUrlResp{
			Files: files,
		},
	}, nil
}

// photoTokensBelongToRecord 确保请求的每个文件 Token 都来自当前反馈记录。
func photoTokensBelongToRecord(value any, requested []string) bool {
	available := make(map[string]struct{})
	switch items := value.(type) {
	case []string:
		for _, token := range items {
			available[token] = struct{}{}
		}
	case []any:
		for _, item := range items {
			if token, ok := item.(string); ok {
				available[token] = struct{}{}
			}
		}
	case string:
		available[items] = struct{}{}
	}
	if len(available) == 0 {
		return false
	}
	for _, token := range requested {
		if _, ok := available[token]; !ok {
			return false
		}
	}
	return true
}
