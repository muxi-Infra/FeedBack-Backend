package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkbitable "github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/feishu"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

const (
	StatusNotSelected = "未选择"
	StatusResolved    = "已解决"
	StatusUnresolved  = "未解决"
)

type SheetService interface {
	CreateRecord(record *domain.TableRecord, tableConfig *domain.TableConfig) (*string, error)
	GetTableRecordReqByKey(keyField *domain.TableField, fieldNames []string, pageToken *string, tableConfig *domain.TableConfig) (*domain.TableRecords, error)
	GetFAQProblemTableRecord(studentID *string, fieldNames []string, tableConfig *domain.TableConfig) (*domain.FAQTableRecords, error)
	UpdateFAQResolutionRecord(resolution *domain.FAQResolution, tableConfig *domain.TableConfig) error
	GetPhotoUrl(fileTokens []string) (*larkdrive.BatchGetTmpDownloadUrlMediaResp, error)
}

type SheetServiceImpl struct {
	c      feishu.Client
	log    logger.Logger
	bc     *config.BatchNoticeConfig
	faqDAO dao.FAQResolutionDAO
}

func NewSheetService(c feishu.Client, log logger.Logger, bc *config.BatchNoticeConfig, faqDAO dao.FAQResolutionDAO) SheetService {
	return &SheetServiceImpl{
		c:      c,
		log:    log,
		bc:     bc,
		faqDAO: faqDAO,
	}
}

func (s *SheetServiceImpl) CreateRecord(record *domain.TableRecord, tableConfig *domain.TableConfig) (*string, error) {
	// 创建请求对象
	req := larkbitable.NewCreateAppTableRecordReqBuilder().
		AppToken(*tableConfig.TableToken).
		TableId(*tableConfig.TableID).
		IgnoreConsistencyCheck(true). // 忽略一致性检查，提高性能，但可能会导致某些节点的数据不同步，出现暂时不一致
		AppTableRecord(larkbitable.NewAppTableRecordBuilder().
			Fields(record.Record).
			Build()).
		Build()

	// 发起请求
	ctx := context.Background()
	resp, err := s.c.CreateAppTableRecord(ctx, req)

	// 处理错误
	if err != nil {
		s.log.Error("CreateAppTableRecord 调用失败",
			logger.String("error", err.Error()),
		)
		return nil, errs.FeishuRequestError(err)
	}

	// 服务端错误处理
	if !resp.Success() {
		s.log.Error("CreateAppTableRecord Lark 接口错误",
			logger.String("request_id", resp.RequestId()),
			logger.String("error", larkcore.Prettify(resp.CodeError)),
		)
		return nil, errs.FeishuResponseError(err)
	}

	// 异步发送批量通知
	go func(r domain.TableRecord, t *domain.TableConfig) {
		// 防止 panic
		defer func() {
			if err := recover(); err != nil {
				s.log.Error("panic recovered",
					logger.Reflect("error", err),
				)
			}
		}()

		// 反馈内容 截取前15个字符
		if fc, ok := r.Record["反馈内容"]; ok {
			if len(fc.(string)) > 15 {
				fc = fc.(string)[0:15] + "..."
			}
			s.bc.Content.Data.TemplateVariable.FeedbackContent = fc.(string)
		}
		// 反馈类型
		if ft, ok := r.Record["问题类型"]; ok {
			s.bc.Content.Data.TemplateVariable.FeedbackType = ft.(string)
		}
		// 反馈来源
		s.bc.Content.Data.TemplateVariable.FeedbackSource = *t.TableName

		contentBytes, err := json.Marshal(s.bc.Content)
		if err != nil {
			s.log.Error("json.Marshal failed",
				logger.String("error", err.Error()),
			)
			return
		}

		// 批量发送 群组通知
		if err := s.SendBatchGroupNotice(string(contentBytes)); err != nil {
			s.log.Error("SendBatchGroupNotice failed",
				logger.String("error", err.Error()),
			)
		}

		//发送个人通知
		//if err := s.SendBatchNotice(string(contentBytes)); err != nil {
		//	s.log.Error("SendBatchNotice failed",
		//		logger.String("error", err.Error()),
		//	)
		//}
	}(*record, tableConfig)

	return resp.Data.Record.RecordId, nil
}

func (s *SheetServiceImpl) GetTableRecordReqByKey(keyField *domain.TableField, fieldNames []string, pageToken *string, tableConfig *domain.TableConfig) (*domain.TableRecords, error) {
	// 创建请求对象
	req := larkbitable.NewSearchAppTableRecordReqBuilder().
		AppToken(*tableConfig.TableToken).
		TableId(*tableConfig.TableID).
		PageToken(*pageToken). // 分页参数,第一次不需要
		PageSize(10).          // 分页大小
		Body(larkbitable.NewSearchAppTableRecordReqBodyBuilder().
			ViewId(*tableConfig.ViewID).
			FieldNames(fieldNames).
			Sort([]*larkbitable.Sort{
				larkbitable.NewSortBuilder().
					FieldName(`提交时间`).
					Desc(true).
					Build(),
			}).
			Filter(larkbitable.NewFilterInfoBuilder().
				Conjunction(`and`).
				Conditions([]*larkbitable.Condition{
					larkbitable.NewConditionBuilder().
						FieldName(*keyField.FieldName).
						Operator(`contains`).
						Value([]string{keyField.Value.(string)}).
						Build(),
				}).
				Build()).
			AutomaticFields(false).
			Build()).
		Build()

	// 发起请求
	ctx := context.Background()
	resp, err := s.c.GetAppTableRecord(ctx, req)

	// 处理错误
	if err != nil {
		s.log.Error("GetTableRecordReqByKey 调用失败",
			logger.String("error", err.Error()),
		)
		return nil, errs.FeishuRequestError(err)
	}

	// 服务端错误处理
	if !resp.Success() {
		s.log.Error("GetTableRecordReqByKey Lark 接口错误",
			logger.String("request_id", resp.RequestId()),
			logger.String("error", larkcore.Prettify(resp.CodeError)),
		)
		return nil, errs.FeishuResponseError(err)
	}

	var records []domain.TableRecord
	for _, r := range resp.Data.Items {
		records = append(records, domain.TableRecord{
			RecordID: r.RecordId,
			Record:   simplifyFields(r.Fields),
		})
	}

	// 组装返回值
	res := &domain.TableRecords{
		Records:   records,
		HasMore:   resp.Data.HasMore,
		PageToken: resp.Data.PageToken,
		Total:     resp.Data.Total,
	}

	return res, nil
}

func (s *SheetServiceImpl) GetFAQProblemTableRecord(studentID *string, fieldNames []string, tableConfig *domain.TableConfig) (*domain.FAQTableRecords, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	var resolutionMap map[string]*bool
	var feishuResp *larkbitable.SearchAppTableRecordResp

	// 并发查数据库
	g.Go(func() error {
		if studentID == nil {
			resolutionMap = nil
			s.log.Error("GetFAQProblemTableRecord studentID is nil")
			return nil
		}

		list, err := s.faqDAO.ListResolutionsByUser(studentID)
		if err != nil {
			s.log.Error("GetFAQProblemTableRecord faqDAO.ListResolutionsByUser err",
				logger.String("error", err.Error()))
			return err
		}

		m := make(map[string]*bool, len(list))
		for _, r := range list {
			if r.RecordID != nil {
				m[*r.RecordID] = r.IsResolved
			}
		}
		resolutionMap = m
		return nil
	})

	// 并发查飞书表格
	g.Go(func() error {
		// 创建请求对象
		req := larkbitable.NewSearchAppTableRecordReqBuilder().
			AppToken(*tableConfig.TableToken).
			TableId(*tableConfig.TableID).
			PageToken("").
			PageSize(100). // 分页大小，拿全部，有更大的需求再改大
			Body(larkbitable.NewSearchAppTableRecordReqBodyBuilder().
				ViewId(*tableConfig.ViewID).
				FieldNames(fieldNames).
				Build()).
			Build()

		// 发起请求
		resp, err := s.c.GetAppTableRecord(ctx, req)
		if err != nil {
			s.log.Error("GetFAQProblemTableRecord 调用失败",
				logger.String("error", err.Error()),
			)
			return errs.FeishuRequestError(err)
		}

		// 服务端错误处理
		if !resp.Success() {
			s.log.Error("GetFAQProblemTableRecord Lark 接口错误",
				logger.String("request_id", resp.RequestId()),
				logger.String("error", larkcore.Prettify(resp.CodeError)),
			)
			return errs.FeishuResponseError(err)
		}
		feishuResp = resp
		return nil
	})

	// 任何一个失败，整体失败
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// 组装记录
	var records []domain.FAQTableRecord
	for _, r := range feishuResp.Data.Items {
		var isResolved *bool

		if resolutionMap != nil {
			if val, ok := resolutionMap[*r.RecordId]; ok {
				isResolved = val
			}
		}

		records = append(records, domain.FAQTableRecord{
			RecordID:   r.RecordId,
			Record:     simplifyFields(r.Fields),
			IsResolved: stringIsResolved(isResolved),
		})
	}

	// 组装返回值
	res := &domain.FAQTableRecords{
		Records: records,
		Total:   feishuResp.Data.Total,
	}
	return res, nil
}

func (s *SheetServiceImpl) UpdateFAQResolutionRecord(resolution *domain.FAQResolution, tableConfig *domain.TableConfig) error {
	record, err := s.faqDAO.GetResolutionByUserAndRecord(resolution.UserID, resolution.RecordID)
	if err != nil {
		return errs.FAQResolutionFindError(err)
	}
	if record != nil {
		return errs.FAQResolutionExistError(fmt.Errorf("user %s has already set resolution for record %s", *resolution.UserID, *resolution.RecordID))
	}

	// TODO 这里会有一致性问题，展示没有想到好的方案
	// 创建请求对象
	//var fieldName string
	//var fieldValue int
	//if resolution.IsResolved != nil && *resolution.IsResolved {
	//	fieldName = *resolution.ResolvedFieldName // 已解决字段
	//	fieldValue = 4
	//} else {
	//	fieldName = *resolution.UnresolvedFieldName // 未解决字段
	//	fieldValue = 2
	//}
	//req := larkbitable.NewUpdateAppTableRecordReqBuilder().
	//	AppToken(*tableConfig.TableToken).
	//	TableId(*tableConfig.TableID).
	//	RecordId(*resolution.RecordID).
	//	AppTableRecord(larkbitable.NewAppTableRecordBuilder().
	//		Fields(map[string]interface{}{fieldName: fieldValue}).
	//		Build()).
	//	Build()
	//
	//// 发起请求
	//ctx := context.Background()
	//resp, err := s.c.UpdateRecord(ctx, req)
	//
	//// 处理错误
	//if err != nil {
	//	s.log.Error("UpdateFAQResolutionRecord 调用失败",
	//		logger.String("error", err.Error()))
	//	return errs.FeishuRequestError(err)
	//}
	//
	//// 服务端错误处理
	//if !resp.Success() {
	//	s.log.Error("UpdateFAQResolutionRecord Lark 接口错误",
	//		logger.String("request_id", resp.RequestId()),
	//		logger.String("error", larkcore.Prettify(resp.CodeError)))
	//
	//	return errs.FeishuResponseError(err)
	//}

	err = s.faqDAO.UpsertFAQResolution(resolution)
	if err != nil {
		return errs.FAQResolutionChangeError(err)
	}
	return nil
}

// SendBatchNotice  发送通知
// 批量发送给个人通知
func (s *SheetServiceImpl) SendBatchNotice(content string) error {
	// 发送消息这个接口限速50次/s
	// 创建一个限制器
	limiter := rate.NewLimiter(rate.Every(25*time.Millisecond), 1) // 每25ms一次，即40次/s

	// 创建errgroup 接受错误
	g, ctx := errgroup.WithContext(context.Background())

	// 根据open_id发送消息
	for _, old := range s.bc.OpenIDs {
		name := old.Name
		openId := old.OpenID

		g.Go(func() error {
			// 等待限速
			if err := limiter.Wait(ctx); err != nil {
				return err
			}

			// 创建请求对象
			req := larkim.NewCreateMessageReqBuilder().
				ReceiveIdType(`open_id`).
				Body(larkim.NewCreateMessageReqBodyBuilder().
					ReceiveId(openId).
					MsgType(`interactive`).
					Content(content).
					Build()).
				Build()

			// 发起请求
			resp, err := s.c.SendNotice(context.Background(), req)

			// 处理错误
			if err != nil {
				return fmt.Errorf("send to name [%s] open_id [%s] failed: %w", name, openId, err)
			}

			// 服务端错误处理
			if !resp.Success() {
				s.log.Error("SendBatchNotice Lark 接口错误",
					logger.String("request_id", resp.RequestId()),
					logger.String("name", name),
					logger.String("open_id", openId),
					logger.String("error", larkcore.Prettify(resp.CodeError)),
				)
				return fmt.Errorf("send to name [%s] open_id [%s] failed: %v", name, openId, larkcore.Prettify(resp.CodeError))
			}
			return nil
		})
	}
	return g.Wait()
}

// SendBatchGroupNotice 发送群组通知
// 支持批量发送
func (s *SheetServiceImpl) SendBatchGroupNotice(content string) error {
	// 发送消息这个接口限速50次/s
	// 创建一个限制器
	limiter := rate.NewLimiter(rate.Every(25*time.Millisecond), 1) // 每25ms一次，即40次/s

	// 创建errgroup 接受错误
	g, ctx := errgroup.WithContext(context.Background())

	// 根据chat_id发送消息
	for _, old := range s.bc.ChatIDs {
		name := old.Name
		chatId := old.ChatID

		g.Go(func() error {
			// 等待限速
			if err := limiter.Wait(ctx); err != nil {
				return err
			}
			// 创建请求对象
			req := larkim.NewCreateMessageReqBuilder().
				ReceiveIdType(`chat_id`).
				Body(larkim.NewCreateMessageReqBodyBuilder().
					ReceiveId(chatId).
					MsgType(`interactive`).
					Content(content).
					Build()).
				Build()

			// 发起请求
			resp, err := s.c.SendNotice(context.Background(), req)

			if err != nil {
				return fmt.Errorf("send to name [%s] chat_id [%s] failed: %w", name, chatId, err)
			}
			if !resp.Success() {
				s.log.Error("SendBatchGroupNotice Lark 接口错误",
					logger.String("request_id", resp.RequestId()),
					logger.String("name", name),
					logger.String("chat_id", chatId),
					logger.String("error", larkcore.Prettify(resp.CodeError)),
				)
				return fmt.Errorf("send to name [%s] chat_id [%s] failed: %v", name, chatId, larkcore.Prettify(resp.CodeError))
			}
			return nil
		})
	}
	return g.Wait()
}

func (s *SheetServiceImpl) GetPhotoUrl(fileTokens []string) (*larkdrive.BatchGetTmpDownloadUrlMediaResp, error) {
	// 创建请求对象
	req := larkdrive.NewBatchGetTmpDownloadUrlMediaReqBuilder().
		FileTokens(fileTokens).
		Build()

	// 发起请求
	ctx := context.Background()
	resp, err := s.c.GetPhotoUrl(ctx, req)

	// 处理错误
	if err != nil {
		s.log.Error("GetPhotoUrl 调用失败",
			logger.String("error", err.Error()),
		)
		return nil, errs.FeishuRequestError(err)
	}

	// 服务端错误处理
	if !resp.Success() {
		s.log.Error("GetPhotoUrl Lark 接口错误",
			logger.String("request_id", resp.RequestId()),
			logger.String("error", larkcore.Prettify(resp.CodeError)),
		)
		return resp, errs.FeishuResponseError(err)
	}

	return resp, nil
}

func simplifyFields(fields map[string]any) map[string]any {
	result := make(map[string]any, len(fields))

	for key, val := range fields {
		switch v := val.(type) {

		// 情况 1：[]any
		case []any:
			// 空数组
			if len(v) == 0 {
				result[key] = v
				continue
			}

			var fileTokens []string
			var text *string
			for _, item := range v {
				m, ok := item.(map[string]any)
				if !ok {
					break
				}

				// 文本字段
				if t, ok := m["text"].(string); ok {
					text = &t
					break
				}

				// 附件 / 图片字段（只要 file_token）
				if token, ok := m["file_token"].(string); ok {
					fileTokens = append(fileTokens, token)
					continue
				}
			}

			if text != nil {
				result[key] = *text
			} else if len(fileTokens) > 0 {
				result[key] = fileTokens
			} else {
				result[key] = v // 兜底
			}

		// 情况 2：已经是基础类型
		case string, float64, bool, int64, int:
			result[key] = v

		// 情况 3：其他未知结构
		// 尽量不要走到这一步，如果走到，即使增加情况处理
		default:
			result[key] = v
		}
	}

	return result
}

func stringIsResolved(isResolved *bool) *string {
	var status string

	switch {
	case isResolved == nil:
		status = StatusNotSelected
	case *isResolved:
		status = StatusResolved
	default:
		status = StatusUnresolved
	}

	return &status
}
