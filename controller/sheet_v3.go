package controller

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	reqV1 "github.com/muxi-Infra/FeedBack-Backend/api/request/v1"
	reqV3 "github.com/muxi-Infra/FeedBack-Backend/api/request/v3"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	respV1 "github.com/muxi-Infra/FeedBack-Backend/api/response/v1"
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

func (s *V3Sheet) CreateFeedback(c *gin.Context, req reqV3.CreateFeedbackReq, projectID, studentID string) (response.Response, error) {
	// todo 底层使用的是 V1
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

func (s *V3Sheet) GetFeedback(c *gin.Context, req reqV3.GetFeedbackReq, projectID, studentID string) (response.Response, error) {
	// todo 底层使用的是V2
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

func (s *V3Sheet) GetRecord(c *gin.Context, req reqV3.GetRecordReq, projectID, studentID string) (response.Response, error) {
	cfg, err := s.config(c, projectID, constvar.FeedbackTableType, constvar.FeedbackScopeReadSelf)
	if err != nil {
		return response.Response{}, err
	}
	record, _, err := s.sheet.GetTableRecordReqByRecordID(&req.RecordID, &cfg)
	if err != nil {
		return response.Response{}, err
	}
	student, ok := record["学号"].(string)
	if !ok || student != studentID {
		return response.Response{}, errs.V3FeedbackRecordForbiddenError(errors.New("record does not belong to current student"))
	}
	return response.Response{
		Code:    0,
		Message: "Success",
		Data: respV1.GetTableRecordByRecordIdResp{
			Record: record,
		}}, nil
}

func (s *V3Sheet) GetFAQ(c *gin.Context, req reqV3.GetFAQReq, projectID, studentID string) (response.Response, error) {
	cfg, err := s.config(c, projectID, constvar.FAQTableType, constvar.FeedbackScopeRead)
	if err != nil {
		return response.Response{}, err
	}
	result, err := s.sheet.GetFAQProblemTableRecord(&studentID, req.RecordNames, &cfg)
	if err != nil {
		return response.Response{}, err
	}
	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    result,
	}, nil
}

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
//	@Tags	V3Sheet
//	@Produce	json
//	@Param	Authorization	header	string	true	"Bearer 用户反馈 Token"
//	@Param	record_id	query	string	true	"反馈记录 ID"
//	@Param	file_tokens	query	[]string	true	"飞书文件 Token"
//	@Success	200	{object}	response.Response
//	@Router	/api/v3/sheet/feedback/photos/url [get]
func (s *V3Sheet) GetPhotoURL(c *gin.Context, req reqV3.GetPhotoURLReq, projectID, studentID string) (response.Response, error) {
	cfg, err := s.config(c, projectID, constvar.FeedbackTableType, constvar.FeedbackScopeReadSelf)
	if err != nil {
		return response.Response{}, err
	}

	record, _, err := s.sheet.GetTableRecordReqByRecordID(&req.RecordID, &cfg)
	if err != nil {
		return response.Response{}, err
	}
	owner, ok := record["学号"].(string)
	if !ok || owner != studentID {
		return response.Response{}, errs.V3FeedbackRecordForbiddenError(errors.New("record does not belong to current student"))
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
