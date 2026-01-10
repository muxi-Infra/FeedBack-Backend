package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkbitable "github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/lark"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/repository/cache"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"github.com/robfig/cron/v3"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
	"time"
)

const (
	StatusNotSelected = "未选择"
	StatusResolved    = "已解决"
	StatusUnresolved  = "未解决"
)

const (
	FeedbackCountTableIdentityKey = "feedback:count:%s"
)

type SheetService interface {
	CreateRecord(record *domain.TableRecord, tableConfig *domain.TableConfig) (*string, error)
	GetTableRecordReqByKey(keyField *domain.TableField, fieldNames []string, pageToken *string, tableConfig *domain.TableConfig) (*domain.TableRecords, error)
	GetFAQProblemTableRecord(studentID *string, fieldNames []string, tableConfig *domain.TableConfig) (*domain.FAQTableRecords, error)
	UpdateFAQResolutionRecord(resolution *domain.FAQResolution, tableConfig *domain.TableConfig) error
	GetPhotoUrl(fileTokens []string) (*larkdrive.BatchGetTmpDownloadUrlMediaResp, error)
}

type SheetServiceImpl struct {
	c             lark.Client
	bc            *config.BatchNoticeConfig
	log           logger.Logger
	faqDAO        dao.FAQResolutionDAO
	cache         cache.FAQResolutionStateCache
	msgCountCache cache.MessageCountCache
	tableProvider TableConfigProvider
}

func NewSheetService(c lark.Client, log logger.Logger, faqDAO dao.FAQResolutionDAO, cache cache.FAQResolutionStateCache, bc *config.BatchNoticeConfig, msgCountCache cache.MessageCountCache, tableProvider TableConfigProvider) SheetService {
	impl := &SheetServiceImpl{
		c:             c,
		bc:            bc,
		log:           log,
		faqDAO:        faqDAO,
		cache:         cache,
		msgCountCache: msgCountCache,
		tableProvider: tableProvider,
	}

	impl.MessageSchedulerStart()
	//impl.SendMessage()

	return impl
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
		return nil, errs.LarkRequestError(err)
	}

	// 服务端错误处理
	if !resp.Success() {
		s.log.Error("CreateAppTableRecord Lark 接口错误",
			logger.String("request_id", resp.RequestId()),
			logger.String("error", larkcore.Prettify(resp.CodeError)),
		)
		return nil, errs.LarkResponseError(err)
	}

	// 成功之后增加反馈数量
	key := fmt.Sprintf(FeedbackCountTableIdentityKey, *tableConfig.TableIdentity)
	err = s.msgCountCache.Increment(key)
	if err != nil {
		s.log.Error("CreateAppTableRecord 增加反馈数量失败",
			logger.String("error", err.Error()),
		)
	}

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
						Value([]string{*keyField.Value.(*string)}).
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
		return nil, errs.LarkRequestError(err)
	}

	// 服务端错误处理
	if !resp.Success() {
		s.log.Error("GetTableRecordReqByKey Lark 接口错误",
			logger.String("request_id", resp.RequestId()),
			logger.String("error", larkcore.Prettify(resp.CodeError)),
		)
		return nil, errs.LarkResponseError(err)
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
			return errs.LarkRequestError(err)
		}

		// 服务端错误处理
		if !resp.Success() {
			s.log.Error("GetFAQProblemTableRecord Lark 接口错误",
				logger.String("request_id", resp.RequestId()),
				logger.String("error", larkcore.Prettify(resp.CodeError)),
			)
			return errs.LarkResponseError(err)
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
	// 1. 查询用户是否已经对该 FAQ 做过选择
	existingRecord, err := s.faqDAO.GetResolutionByUserAndRecord(resolution.UserID, resolution.RecordID)
	if err != nil {
		s.log.Error("UpdateFAQResolutionRecord faqDAO.GetResolutionByUserAndRecord err",
			logger.String("error", err.Error()))
		return errs.FAQResolutionFindError(err)
	}

	// 2. 判断是否为首次选择
	isFirstChoice := existingRecord == nil

	// 3. 如果是修改状态，检查修改次数限制（最多允许修改 3 次）
	if !isFirstChoice {
		if existingRecord.Frequency != nil && *existingRecord.Frequency >= 3 {
			return errs.FAQResolutionChangeLimitExceededError(errors.New("faq resolution change limit exceeded"))
		}

		// 检查状态是否真的发生变化
		if existingRecord.IsResolved != nil && *existingRecord.IsResolved == *resolution.IsResolved {
			return errs.FAQResolutionExistError(errors.New("faq resolution status exist"))
		}
	}

	// 4. 计算 修改次数
	newFrequency := 1
	if !isFirstChoice {
		newFrequency = *existingRecord.Frequency + 1
	}

	// 5. 生成 Redis 缓存 key
	resolvedKey := fmt.Sprintf("%s-%s-%s", *tableConfig.TableIdentity, *resolution.RecordID, StatusResolved)
	unresolvedKey := fmt.Sprintf("%s-%s-%s", *tableConfig.TableIdentity, *resolution.RecordID, StatusUnresolved)

	// 6. 使用 Lua 脚本原子性更新 Redis 计数器
	var resolvedCount, unresolvedCount uint64
	if isFirstChoice {
		// 首次选择：只增加对应状态的计数
		if *resolution.IsResolved {
			resolvedCount, unresolvedCount, err = s.cache.IncAAndGetB(resolvedKey, unresolvedKey)
		} else {
			unresolvedCount, resolvedCount, err = s.cache.IncAAndGetB(unresolvedKey, resolvedKey)
		}
	} else {
		// 修改状态：新状态计数 +1，旧状态计数 -1
		if *resolution.IsResolved {
			resolvedCount, unresolvedCount, err = s.cache.IncAAndDecB(resolvedKey, unresolvedKey)
		} else {
			unresolvedCount, resolvedCount, err = s.cache.IncAAndDecB(unresolvedKey, resolvedKey)
		}
	}

	if err != nil {
		s.log.Error("UpdateFAQResolutionRecord redis cache update err",
			logger.String("error", err.Error()))
		return errs.FAQResolutionCountGetError(err)
	}

	// 只能保证最终一致性
	go func(rName, uName string, rNum, uNum uint64) {
		// 创建请求对象
		req := larkbitable.NewUpdateAppTableRecordReqBuilder().
			AppToken(*tableConfig.TableToken).
			TableId(*tableConfig.TableID).
			RecordId(*resolution.RecordID).
			AppTableRecord(larkbitable.NewAppTableRecordBuilder().
				Fields(map[string]interface{}{rName: rNum, uName: uNum}).
				Build()).
			Build()

		// 发起请求
		ctx := context.Background()
		resp, err := s.c.UpdateRecord(ctx, req)

		// 处理错误
		if err != nil {
			s.log.Error("UpdateFAQResolutionRecord 调用失败",
				logger.String("error", err.Error()))
		}

		// 服务端错误处理
		if !resp.Success() {
			s.log.Error("UpdateFAQResolutionRecord Lark 接口错误",
				logger.String("request_id", resp.RequestId()),
				logger.String("error", larkcore.Prettify(resp.CodeError)))
		}
	}(*resolution.ResolvedFieldName, *resolution.UnresolvedFieldName, resolvedCount, unresolvedCount)

	// 更新或插入数据库记录
	m := &model.FAQResolution{
		RecordID:   resolution.RecordID,
		UserID:     resolution.UserID,
		IsResolved: resolution.IsResolved,
		Frequency:  &newFrequency,
	}
	err = s.faqDAO.CreateOrUpsertFAQResolution(m)
	if err != nil {
		s.log.Error("UpdateFAQResolutionRecord faqDAO.CreateOrUpsertFAQResolution err",
			logger.String("error", err.Error()))
		return errs.FAQResolutionChangeError(err)
	}
	return nil
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
		return nil, errs.LarkRequestError(err)
	}

	// 服务端错误处理
	if !resp.Success() {
		s.log.Error("GetPhotoUrl Lark 接口错误",
			logger.String("request_id", resp.RequestId()),
			logger.String("error", larkcore.Prettify(resp.CodeError)),
		)
		return resp, errs.LarkResponseError(err)
	}

	return resp, nil
}

func (s *SheetServiceImpl) MessageSchedulerStart() {
	// 时区
	loc, _ := time.LoadLocation("Asia/Shanghai")
	c := cron.New(
		cron.WithSeconds(),
		cron.WithLocation(loc),
	)

	_, err := c.AddFunc("0 0 18 * * *", func() {
		s.log.Info("MessageSchedulerStart")

		s.SendMessage()

		s.log.Info("MessageSchedulerEnd")
	})

	if err != nil {
		s.log.Error("MessageSchedulerStart cron.AddFunc err", logger.String("error", err.Error()))
	}
	c.Start()
}

// SendMessage 发送消息
func (s *SheetServiceImpl) SendMessage() {
	// 1. 获取 table 配置
	//fmt.Println("1. 获取 table 配置")
	tables := s.tableProvider.GetTablesByType(TableTypeFeedback)
	if len(tables) == 0 {
		return
	}

	// 2. 根据 table_identity 来获取一天的反馈量
	//fmt.Println("2. 根据 table_identity 来获取一天的反馈量")
	for _, table := range tables {
		key := fmt.Sprintf(FeedbackCountTableIdentityKey, table.Identity)
		count, err := s.msgCountCache.GetAndReset(key)
		if err != nil {
			s.log.Error("GetAndReset failed", logger.String("key", key), logger.String("error", err.Error()))
			continue
		}

		// 3. 构建卡片消息
		//fmt.Println("3. 构建卡片消息")
		s.bc.Content.Data.TemplateVariable.FeedbackSource = table.Name
		s.bc.Content.Data.TemplateVariable.DailyNewCount = int(count)
		// todo 目前是一样的，后续可以优化
		// pc Url
		s.bc.Content.Data.TemplateVariable.TableUrl.PCUrl = table.TableUrl
		// ios Url
		s.bc.Content.Data.TemplateVariable.TableUrl.IOSUrl = table.TableUrl
		// android Url
		s.bc.Content.Data.TemplateVariable.TableUrl.AndroidUrl = table.TableUrl
		// 兜底url
		s.bc.Content.Data.TemplateVariable.TableUrl.Url = table.TableUrl

		contentBytes, err := json.Marshal(s.bc.Content)
		if err != nil {
			s.log.Error("json.Marshal failed", logger.String("error", err.Error()))
			continue
		}

		// 4. 发送卡片消息
		//fmt.Println("4. 发送卡片消息")
		if err := s.sendMessage(string(contentBytes)); err != nil {
			s.log.Error("sendMessage failed", logger.String("error", err.Error()))
			continue
		}
	}
}

// sendMessage 发送卡片
func (s *SheetServiceImpl) sendMessage(content string) error {
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
