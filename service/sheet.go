package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkbitable "github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/lark"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/repository/cache"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"golang.org/x/sync/errgroup"
)

const (
	StatusNotSelected = "未选择"
	StatusResolved    = "已解决"
	StatusUnresolved  = "未解决"
	queueBatchSize    = 100
	pageSize          = 100 // 数据库分页大小
)

//go:generate mockgen -destination=./mock/sheet_mock.go -package=mocks github.com/muxi-Infra/FeedBack-Backend/service SheetService
type SheetService interface {
	CreateLarkRecord(record *domain.TableRecord, tableConfig *domain.TableConfig) (*string, error)
	CreateDBRecord(recordID, shareUrl *string, recordData map[string]any, tableConfig domain.TableConfig) error
	UpdateDBRecord(recordID, shareUrl *string, recordData map[string]any, tableConfig domain.TableConfig) error
	GetTableRecordReqByKey(keyField *domain.TableField, fieldNames []string, pageToken *string, tableConfig *domain.TableConfig) (*domain.TableRecords, error)
	GetTableRecordReqByUser(userID, pageToken *string, limitSize int, tableConfig *domain.TableConfig) (*domain.TableRecords, error)
	GetTableRecordReqByRecordID(recordID *string, tableConfig *domain.TableConfig) (map[string]any, *string, error)
	GetFAQProblemTableRecord(studentID *string, fieldNames []string, tableConfig *domain.TableConfig) (*domain.FAQTableRecords, error)
	UpdateFAQResolutionRecord(resolution *domain.FAQResolution, tableConfig *domain.TableConfig) error
	GetPhotoUrl(fileTokens []string) ([]domain.File, error)
	SyncUnsyncedTableRecords(tableConfig *domain.TableConfig) ([]string, int, bool, error)
	ForceSyncUserTableRecords(studentID *string, tableConfig *domain.TableConfig) ([]string, int, bool, error)
	ForceSyncTableRecords(tableConfig *domain.TableConfig) ([]string, int, bool, error)
	GetFAQResolutionRecord(studentID *string, tableConfig *domain.TableConfig) ([]domain.FAQTableRecord, error)
	UpdateFAQResolutionRecordV2(resolution *domain.FAQResolutionV2, tableConfig *domain.TableConfig) error
	SyncFAQRecord(tableConfig *domain.TableConfig) error
}

type SheetServiceImpl struct {
	c             lark.Client
	log           logger.Logger
	resolutionDAO dao.FAQResolutionDAO
	sheetDao      dao.SheetDAO
	faqDAO        dao.FAQDAO
	cache         cache.FAQResolutionStateCache
}

func NewSheetService(c lark.Client, log logger.Logger, resolutionDAO dao.FAQResolutionDAO, sheetDAO dao.SheetDAO, faqDAO dao.FAQDAO, cache cache.FAQResolutionStateCache) SheetService {
	s := &SheetServiceImpl{
		c:             c,
		log:           log,
		resolutionDAO: resolutionDAO,
		sheetDao:      sheetDAO,
		faqDAO:        faqDAO,
		cache:         cache,
	}

	// 消费者，异步同步未同步的记录到数据库
	go func() {
		for {
			select {
			case msg := <-syncCh:
				// 同步反馈记录
				// 飞书 -> 数据库
				// 这是对 case table := <-syncTableCh 的聚合处理，减少 API 的使用量
				err := s.SyncLarkRecords(msg.RecordIDs, msg.TableConfig)
				if err != nil {
					s.log.Error("SyncLarkRecords 同步记录到飞书表格失败",
						logger.String("error", err.Error()),
						logger.String("table_identity", *msg.TableConfig.TableIdentity),
					)
				}
			case table := <-syncTableCh:
				// 获取带同步表格标识，区分常见问题表格和反馈记录表格
				if bytes.Contains([]byte(*table.TableIdentity), []byte("-faq")) {
					// 同步常见问题中 解决/未解决 数量
					// redis -> 飞书
					err := s.SyncFAQRecord(&table)
					if err != nil {
						s.log.Error("SyncFAQResolutionCount 同步 FAQ 记录到飞书表格失败",
							logger.String("error", err.Error()),
							logger.String("table_identity", *table.TableIdentity),
						)
					}
				} else {
					// 同步反馈记录
					// 飞书 -> 数据库
					_, _, _, err := s.SyncUnsyncedTableRecords(&table)
					if err != nil {
						s.log.Error("SyncUnsyncedTableRecords 同步未同步记录到飞书表格失败",
							logger.String("error", err.Error()),
							logger.String("table_identity", *table.TableIdentity),
						)
					}
				}
			}
		}
	}()

	return s
}

func (s *SheetServiceImpl) CreateLarkRecord(record *domain.TableRecord, tableConfig *domain.TableConfig) (*string, error) {
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

	return resp.Data.Record.RecordId, nil
}

func (s *SheetServiceImpl) CreateDBRecord(recordID, shareUrl *string, recordData map[string]any, tableConfig domain.TableConfig) error {
	studentID, ok := recordData["学号"].(string)
	if !ok {
		s.log.Error("SyncLarkRecords 学号字段类型断言失败",
			logger.String("record_id", *recordID),
		)
		return errs.CreateRecordDBError(errors.New("学号字段类型断言失败"))
	}

	m := &model.Sheet{
		TableIdentify: tableConfig.TableIdentity,
		RecordID:      recordID,
		UserID:        &studentID,
		Record:        recordData,
		ShareUrl:      shareUrl,
		IsSynced:      false,
	}

	err := s.sheetDao.CreateSheetRecord(m)
	if err != nil {
		s.log.Error("CreateDBRecord 保存记录到数据库失败",
			logger.String("error", err.Error()),
			logger.String("record_id", *recordID),
		)
		return errs.CreateRecordDBError(err)
	}

	return nil
}

func (s *SheetServiceImpl) UpdateDBRecord(recordID, shareUrl *string, recordData map[string]any, tableConfig domain.TableConfig) error {
	studentID, ok := recordData["学号"].(string)
	if !ok {
		s.log.Error("SyncLarkRecords 学号字段类型断言失败",
			logger.String("record_id", *recordID),
		)
		return errs.UpdateRecordDBError(errors.New("学号字段类型断言失败"))
	}

	synced := false
	finish, ok := recordData["进度"].(string)
	if ok && finish == "已完成" {
		synced = true
	}

	m := &model.Sheet{
		TableIdentify: tableConfig.TableIdentity,
		RecordID:      recordID,
		UserID:        &studentID,
		Record:        recordData,
		ShareUrl:      shareUrl,
		IsSynced:      synced,
	}

	err := s.sheetDao.CreateOrUpdateSheetRecord(m)
	if err != nil {
		s.log.Error("CreateOrUpdateSheetRecord 保存记录到数据库失败",
			logger.String("error", err.Error()),
			logger.String("record_id", *recordID),
		)
		return errs.UpdateRecordDBError(err)
	}

	return nil
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
	}

	return res, nil
}

// GetTableRecordReqByUser 根据用户查询表格记录，支持分页
func (s *SheetServiceImpl) GetTableRecordReqByUser(userID, pageToken *string, limitSize int, tableConfig *domain.TableConfig) (*domain.TableRecords, error) {
	lastId := new(uint64)
	if pageToken != nil && *pageToken != "" {
		ld, err := decodePageToken(*pageToken)
		if err != nil {
			return nil, errs.PageTokenInvalidError(err)
		}
		lastId = ld
	}

	dbRecords, hasMore, err := s.sheetDao.GetSheetRecordByUser(*tableConfig.TableIdentity, *userID, lastId, limitSize)
	if err != nil {
		s.log.Error("GetTableRecordReqByUser 数据库查询失败",
			logger.String("error", err.Error()),
			logger.String("student_id", *userID),
		)
		return nil, err
	}

	ans := make([]domain.TableRecord, 0, len(dbRecords))
	for _, r := range dbRecords {
		ans = append(ans, domain.TableRecord{
			RecordID: r.RecordID,
			Record:   r.Record,
		})
	}

	// 生成 nextPageToken
	var nextToken *string
	if hasMore && len(dbRecords) > 0 {
		last := dbRecords[len(dbRecords)-1].ID
		token, _ := encodePageToken(last)
		nextToken = &token
	}

	// 组装返回值
	return &domain.TableRecords{
		Records:   ans,
		HasMore:   &hasMore,
		PageToken: nextToken,
	}, nil
}

func (s *SheetServiceImpl) GetTableRecordReqByRecordID(recordID *string, tableConfig *domain.TableConfig) (map[string]any, *string, error) {
	// 创建请求对象
	req := larkbitable.NewBatchGetAppTableRecordReqBuilder().
		AppToken(*tableConfig.TableToken).
		TableId(*tableConfig.TableID).
		Body(larkbitable.NewBatchGetAppTableRecordReqBodyBuilder().
			RecordIds([]string{*recordID}).
			WithSharedUrl(true).
			Build()).
		Build()

	// 发起请求
	ctx := context.Background()
	resp, err := s.c.GetRecordByRecordId(ctx, req)

	// 处理错误
	if err != nil {
		s.log.Error("GetTableRecordReqByID 调用失败",
			logger.String("error", err.Error()),
		)
		return nil, nil, errs.LarkRequestError(err)
	}

	// 服务端错误处理
	if !resp.Success() {
		s.log.Error("GetTableRecordReqByID Lark 接口错误",
			logger.String("request_id", resp.RequestId()),
			logger.String("error", larkcore.Prettify(resp.CodeError)),
		)
		return nil, nil, errs.LarkResponseError(err)
	}

	if len(resp.Data.Records) == 0 {
		s.log.Error("GetTableRecordReqByID no record found",
			logger.String("record_id", *recordID),
		)
		return nil, nil, errs.TableRecordNotFoundError(errors.New("未找到记录"))
	}

	return simplifyFields(resp.Data.Records[0].Fields), resp.Data.Records[0].SharedUrl, nil
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

		list, err := s.resolutionDAO.ListResolutionsByUser(studentID, tableConfig.TableIdentity)
		if err != nil {
			s.log.Error("GetFAQProblemTableRecord resolutionDAO.ListResolutionsByUser err",
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
	// 1. 查询用户是否已经对该 FAQRecord 做过选择
	existingRecord, err := s.resolutionDAO.GetResolutionByUserAndRecord(resolution.UserID, tableConfig.TableIdentity, resolution.RecordID)
	if err != nil {
		s.log.Error("UpdateFAQResolutionRecord resolutionDAO.GetResolutionByUserAndRecord err",
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
		UserID:        resolution.UserID,
		TableIdentify: tableConfig.TableIdentity,
		RecordID:      resolution.RecordID,
		IsResolved:    resolution.IsResolved,
		Frequency:     &newFrequency,
	}
	err = s.resolutionDAO.CreateOrUpsertFAQResolution(m)
	if err != nil {
		s.log.Error("UpdateFAQResolutionRecord resolutionDAO.CreateOrUpsertFAQResolution err",
			logger.String("error", err.Error()))
		return errs.FAQResolutionChangeError(err)
	}
	return nil
}

func (s *SheetServiceImpl) GetPhotoUrl(fileTokens []string) ([]domain.File, error) {
	if len(fileTokens) == 0 {
		s.log.Error("GetPhotoUrl fileTokens is empty")
		return nil, errs.FileTokenInvalidError(errors.New("file token 为空"))
	}

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
		return nil, errs.LarkResponseError(err)
	}

	var files []domain.File
	for _, item := range resp.Data.TmpDownloadUrls {
		files = append(files, domain.File{
			FileToken:      item.FileToken,
			TmpDownloadURL: item.TmpDownloadUrl,
		})
	}

	if len(files) == 0 {
		s.log.Error("GetPhotoUrl fileTokens is all invalid")
		return nil, errs.FileTokenInvalidError(errors.New("file token 全部无效"))
	}

	return files, nil
}

// UpdateRecordProgress 异步更新飞书表格记录进度为已完成
// 2026-03-14 下线这个功能
// 更换为 message.MarkRecordNoticed 更新通知记录的状态为已完成
func (s *SheetServiceImpl) UpdateRecordProgress(recordID *string, tableConfig *domain.TableConfig) error {
	req := larkbitable.NewUpdateAppTableRecordReqBuilder().
		AppToken(*tableConfig.TableToken).
		TableId(*tableConfig.TableID).
		RecordId(*recordID).
		AppTableRecord(larkbitable.NewAppTableRecordBuilder().
			Fields(map[string]interface{}{`进度`: `已完成`}).
			Build()).
		Build()

	// 发起请求
	ctx := context.Background()
	resp, err := s.c.UpdateRecord(ctx, req)

	// 处理错误
	if err != nil {
		s.log.Error("UpdateRecordProgress 调用失败",
			logger.String("error", err.Error()),
		)
		return errs.LarkRequestError(err)
	}

	// 服务端错误处理
	if !resp.Success() {
		s.log.Error("UpdateRecordProgress Lark 接口错误",
			logger.String("request_id", resp.RequestId()),
			logger.String("error", larkcore.Prettify(resp.CodeError)),
		)
		return errs.LarkResponseError(err)
	}

	return nil
}

// SyncUnsyncedTableRecords 同步未同步的记录到飞书表格，返回成功同步的 recordID 列表
func (s *SheetServiceImpl) SyncUnsyncedTableRecords(tableConfig *domain.TableConfig) ([]string, int, bool, error) {
	// 获取未同步 recordID 列表
	recordIDs, err := s.sheetDao.GetUnsyncedRecordsByTable(*tableConfig.TableIdentity)
	if err != nil {
		return nil, 0, false, errs.GetUnsyncedRecordsByTableError(err)
	}

	if len(recordIDs) == 0 {
		return []string{}, 0, false, nil
	}

	// 过滤空 ID
	filtered := make([]string, 0, len(recordIDs))
	for _, rid := range recordIDs {
		if rid != "" {
			filtered = append(filtered, rid)
		}
	}

	if len(filtered) == 0 {
		return []string{}, 0, false, nil
	}

	// 拆分入队
	totalEnqueued := 0
	queueFull := false

	for i := 0; i < len(filtered); i += queueBatchSize {
		end := i + queueBatchSize
		if end > len(filtered) {
			end = len(filtered)
		}

		subBatch := filtered[i:end]

		msg := SyncMsg{
			RecordIDs:   subBatch,
			TableConfig: *tableConfig,
		}

		select {
		case syncCh <- msg:
			totalEnqueued += len(subBatch)
		default:
			queueFull = true
			// 返回已经成功入队的部分
			return filtered[:totalEnqueued], totalEnqueued, true, nil
		}
	}

	return filtered, len(filtered), queueFull, nil
}

func (s *SheetServiceImpl) ForceSyncUserTableRecords(studentID *string, tableConfig *domain.TableConfig) ([]string, int, bool, error) {
	var lastID *uint64

	enqueuedIDs := make([]string, 0)
	queueFull := false

loop:
	for {
		records, hasMore, err := s.sheetDao.GetSheetRecordByUser(*tableConfig.TableIdentity, *studentID, lastID, pageSize)
		if err != nil {
			return nil, 0, false, err
		}

		if len(records) == 0 {
			break
		}
		// 聚合当前页 recordIDs
		batchIDs := make([]string, 0, len(records))
		for _, r := range records {
			if r.RecordID != nil && *r.RecordID != "" {
				batchIDs = append(batchIDs, *r.RecordID)
			}
		}

		if len(batchIDs) > 0 {
			msg := SyncMsg{
				RecordIDs:   batchIDs,
				TableConfig: *tableConfig,
			}

			select {
			case syncCh <- msg:
				enqueuedIDs = append(enqueuedIDs, batchIDs...)
			default:
				queueFull = true
				break loop
			}
		}

		// 更新游标（因为 DESC）
		lastID = &records[len(records)-1].ID

		if !hasMore {
			break
		}
	}

	return enqueuedIDs, len(enqueuedIDs), queueFull, nil
}

func (s *SheetServiceImpl) ForceSyncTableRecords(tableConfig *domain.TableConfig) ([]string, int, bool, error) {
	var allRecordIDs []string
	pageToken := new(string)

	for {
		reqBuilder := larkbitable.NewSearchAppTableRecordReqBuilder().
			AppToken(*tableConfig.TableToken).
			TableId(*tableConfig.TableID).
			PageSize(500) // 建议用飞书允许的最大值

		if pageToken != nil {
			reqBuilder.PageToken(*pageToken)
		}

		req := reqBuilder.
			Body(larkbitable.NewSearchAppTableRecordReqBodyBuilder().
				ViewId(*tableConfig.ViewID).
				AutomaticFields(false).
				Build()).
			Build()

		ctx := context.Background()
		resp, err := s.c.GetAppTableRecord(ctx, req)
		if err != nil {
			s.log.Error("ForceSyncTableRecords 调用失败",
				logger.String("error", err.Error()),
			)
			return nil, 0, false, errs.LarkRequestError(err)
		}

		if !resp.Success() {
			s.log.Error("ForceSyncTableRecords Lark 接口错误",
				logger.String("request_id", resp.RequestId()),
				logger.String("error", larkcore.Prettify(resp.CodeError)),
			)
			return nil, 0, false, errs.LarkResponseError(err)
		}

		// 只提取 RecordID
		for _, r := range resp.Data.Items {
			allRecordIDs = append(allRecordIDs, *r.RecordId)
		}

		if !*resp.Data.HasMore {
			break
		}

		pageToken = resp.Data.PageToken
	}

	// 分批入队
	totalEnqueued := 0
	queueFull := false

	for i := 0; i < len(allRecordIDs); i += queueBatchSize {
		end := i + queueBatchSize
		if end > len(allRecordIDs) {
			end = len(allRecordIDs)
		}

		subBatch := allRecordIDs[i:end]

		msg := SyncMsg{
			RecordIDs:   subBatch,
			TableConfig: *tableConfig,
		}

		select {
		case syncCh <- msg:
			totalEnqueued += len(subBatch)

		default:
			queueFull = true
			return allRecordIDs[:totalEnqueued], totalEnqueued, true, nil
		}
	}

	return allRecordIDs, totalEnqueued, queueFull, nil
}

func (s *SheetServiceImpl) SyncLarkRecords(recordIDs []string, tableConfig domain.TableConfig) error {
	// 创建请求对象
	req := larkbitable.NewBatchGetAppTableRecordReqBuilder().
		AppToken(*tableConfig.TableToken).
		TableId(*tableConfig.TableID).
		Body(larkbitable.NewBatchGetAppTableRecordReqBodyBuilder().
			RecordIds(recordIDs).
			WithSharedUrl(true).
			Build()).
		Build()

	// 发起请求
	ctx := context.Background()
	resp, err := s.c.GetRecordByRecordId(ctx, req)

	// 处理错误
	if err != nil {
		s.log.Error("GetTableRecordReqByID 调用失败",
			logger.String("error", err.Error()),
		)
		return errs.LarkRequestError(err)
	}

	// 服务端错误处理
	if !resp.Success() {
		s.log.Error("GetTableRecordReqByID Lark 接口错误",
			logger.String("request_id", resp.RequestId()),
			logger.String("error", larkcore.Prettify(resp.CodeError)),
		)
		return errs.LarkResponseError(err)
	}

	if len(resp.Data.Records) == 0 {
		s.log.Error("GetTableRecordReqByID no record found")
		return errs.TableRecordNotFoundError(errors.New("未找到记录"))
	}

	for _, r := range resp.Data.Records {
		recordData := simplifyFields(r.Fields)

		err := s.UpdateDBRecord(r.RecordId, r.SharedUrl, recordData, tableConfig)
		if err != nil {
			s.log.Error("SyncLarkRecords 更新数据库记录失败",
				logger.String("error", err.Error()),
				logger.String("record_id", *r.RecordId),
			)
			// 同步失败不返回错误，继续同步其他记录，最终通过定时任务再次尝试同步未同步的记录
		}
	}

	return nil
}

// GetFAQResolutionRecord 获取 FAQ 记录，全部从 DB 获取
func (s *SheetServiceImpl) GetFAQResolutionRecord(studentID *string, tableConfig *domain.TableConfig) ([]domain.FAQTableRecord, error) {
	g, _ := errgroup.WithContext(context.Background())

	var resolutionMap map[string]*bool
	var dbResp map[string]map[string]any

	// 并发查数据库
	g.Go(func() error {
		if studentID == nil {
			resolutionMap = nil
			s.log.Error("GetFAQProblemTableRecord studentID is nil")
			return nil
		}

		list, err := s.resolutionDAO.ListResolutionsByUser(studentID, tableConfig.TableIdentity)
		if err != nil {
			s.log.Error("GetFAQProblemTableRecord resolutionDAO.ListResolutionsByUser err",
				logger.String("error", err.Error()))
			return errs.FAQResolutionFindError(err)
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

	g.Go(func() error {
		records, err := s.faqDAO.GetFAQRecords(tableConfig.TableIdentity)
		if err != nil {
			s.log.Error("GetFAQResolutionRecord faqDAO.GetFAQRecords err",
				logger.String("error", err.Error()),
			)
			return errs.GetFAQRecordByTableError(err)
		}
		dbResp = make(map[string]map[string]any)
		for _, r := range records {
			dbResp[*r.RecordID] = r.Record
		}
		return nil
	})

	// 任何一个失败，整体失败
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// 组装记录
	var records []domain.FAQTableRecord
	for key, record := range dbResp {
		var isResolved *bool
		recordID := key
		if resolutionMap != nil {
			if val, ok := resolutionMap[recordID]; ok {
				isResolved = val
			}
		}

		records = append(records, domain.FAQTableRecord{
			RecordID:   &recordID,
			Record:     record,
			IsResolved: stringIsResolved(isResolved),
		})
	}

	// 组装返回值
	return records, nil
}

// UpdateFAQResolutionRecordV2 更新 FAQ 记录，更新数据库后更新 Redis 计数器
func (s *SheetServiceImpl) UpdateFAQResolutionRecordV2(resolution *domain.FAQResolutionV2, tableConfig *domain.TableConfig) error {
	// 1. 查询用户是否已经对该 FAQRecord 做过选择
	existingRecord, err := s.resolutionDAO.GetResolutionByUserAndRecord(resolution.UserID, tableConfig.TableIdentity, resolution.RecordID)
	if err != nil {
		s.log.Error("UpdateFAQResolutionRecord resolutionDAO.GetResolutionByUserAndRecord err",
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
	if isFirstChoice {
		// 首次选择：只增加对应状态的计数
		if *resolution.IsResolved {
			_, _, err = s.cache.IncAAndGetB(resolvedKey, unresolvedKey)
		} else {
			_, _, err = s.cache.IncAAndGetB(unresolvedKey, resolvedKey)
		}
	} else {
		// 修改状态：新状态计数 +1，旧状态计数 -1
		if *resolution.IsResolved {
			_, _, err = s.cache.IncAAndDecB(resolvedKey, unresolvedKey)
		} else {
			_, _, err = s.cache.IncAAndDecB(unresolvedKey, resolvedKey)
		}
	}

	if err != nil {
		s.log.Error("UpdateFAQResolutionRecord redis cache update err",
			logger.String("error", err.Error()))
		return errs.FAQResolutionCountGetError(err)
	}

	// 更新或插入数据库记录
	m := &model.FAQResolution{
		UserID:        resolution.UserID,
		TableIdentify: tableConfig.TableIdentity,
		RecordID:      resolution.RecordID,
		IsResolved:    resolution.IsResolved,
		Frequency:     &newFrequency,
	}
	err = s.resolutionDAO.CreateOrUpsertFAQResolution(m)
	if err != nil {
		s.log.Error("UpdateFAQResolutionRecord resolutionDAO.CreateOrUpsertFAQResolution err",
			logger.String("error", err.Error()))
		return errs.FAQResolutionChangeError(err)
	}
	return nil
}

// SyncFAQRecord 同步飞书表格和数据库中的 FAQ 记录，保证两者的一致性
// 这个过程需要 redis <-> mysql <-> 飞书表格 三者的配合，保证最终一致性
func (s *SheetServiceImpl) SyncFAQRecord(tableConfig *domain.TableConfig) error {
	g, ctx := errgroup.WithContext(context.Background())

	var larkResp map[string]map[string]interface{}
	var dbResp map[string]struct{}
	// 1 获取飞书数据
	g.Go(func() error {
		// 创建请求对象
		req := larkbitable.NewSearchAppTableRecordReqBuilder().
			AppToken(*tableConfig.TableToken).
			TableId(*tableConfig.TableID).
			PageToken("").
			PageSize(100). // 分页大小，拿全部，有更大的需求再改大
			Body(larkbitable.NewSearchAppTableRecordReqBodyBuilder().
				ViewId(*tableConfig.ViewID).
				FieldNames(nil). //返回所有字段
				Build()).
			Build()

		// 发起请求
		resp, err := s.c.GetAppTableRecord(ctx, req)
		if err != nil {
			s.log.Error("SyncFAQRecord 调用失败",
				logger.String("error", err.Error()),
			)
			return errs.LarkRequestError(err)
		}

		// 服务端错误处理
		if !resp.Success() {
			s.log.Error("SyncFAQRecord Lark 接口错误",
				logger.String("request_id", resp.RequestId()),
				logger.String("error", larkcore.Prettify(resp.CodeError)),
			)
			return errs.LarkResponseError(err)
		}

		larkResp = make(map[string]map[string]interface{})
		for _, r := range resp.Data.Items {
			larkResp[*r.RecordId] = simplifyFields(r.Fields)
		}

		return nil
	})

	// 2 获取数据库数据
	g.Go(func() error {
		records, err := s.faqDAO.GetFAQRecords(tableConfig.TableIdentity)
		if err != nil {
			s.log.Error("SyncFAQRecord 数据库查询失败",
				logger.String("error", err.Error()),
			)
			return errs.GetFAQRecordByTableError(err)
		}
		dbResp = make(map[string]struct{})
		for _, r := range records {
			dbResp[*r.RecordID] = struct{}{}
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return err
	}

	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup
	flag := true

	// 3 同步 record + 更新 Redis 计数器
	for recordID, fields := range larkResp {
		// Redis vote
		resolvedKey := fmt.Sprintf("%s-%s-%s", *tableConfig.TableIdentity, recordID, StatusResolved)
		unresolvedKey := fmt.Sprintf("%s-%s-%s", *tableConfig.TableIdentity, recordID, StatusUnresolved)

		resolvedNum, unresolvedNum, _ := s.cache.GetAAndGetB(resolvedKey, unresolvedKey)

		// MySQL upsert
		m := &model.FAQRecord{
			TableIdentify:   tableConfig.TableIdentity,
			RecordID:        &recordID,
			Record:          fields,
			ResolvedCount:   int64(resolvedNum),
			UnresolvedCount: int64(unresolvedNum),
		}

		err := s.faqDAO.CreateOrUpdateSheetRecord(m)
		if err != nil {
			flag = false
			s.log.Error("mysql upsert err",
				logger.String("record_id", recordID),
				logger.String("error", err.Error()),
			)
		}

		// 飞书更新 已解决/未解决 数量
		sem <- struct{}{}
		wg.Add(1)
		go func(rID string, rNum, uNum uint64) {
			defer func() {
				<-sem
				wg.Done()
			}()

			req := larkbitable.NewUpdateAppTableRecordReqBuilder().
				AppToken(*tableConfig.TableToken).
				TableId(*tableConfig.TableID).
				RecordId(rID).
				AppTableRecord(larkbitable.NewAppTableRecordBuilder().
					Fields(map[string]interface{}{
						StatusResolved:   rNum,
						StatusUnresolved: uNum,
					}).
					Build()).
				Build()
			// 发起请求
			resp, err := s.c.UpdateRecord(context.Background(), req)

			// 处理错误
			if err != nil {
				s.log.Error("SyncFAQResolutionCount UpdateRecord 调用失败",
					logger.String("error", err.Error()),
					logger.String("record_id", rID),
				)
				return
			}

			// 服务端错误处理
			if !resp.Success() {
				s.log.Error("SyncFAQResolutionCount UpdateRecord Lark 接口错误",
					logger.String("request_id", resp.RequestId()),
					logger.String("error", larkcore.Prettify(resp.CodeError)),
					logger.String("record_id", rID),
				)
				return
			}
		}(recordID, resolvedNum, unresolvedNum)
	}

	// 4 删除多余的记录
	var redisKeys []string
	for recordID := range dbResp {
		if _, ok := larkResp[recordID]; !ok {
			// 1 删除 mysql
			err := s.faqDAO.DeleteFAQRecord(tableConfig.TableIdentity, &recordID)
			if err != nil {
				flag = false
				s.log.Error("删除多余 FAQ 记录失败",
					logger.String("record_id", recordID),
					logger.String("error", err.Error()),
				)
			}

			// 2 删除 Redis
			redisKeys = append(redisKeys,
				fmt.Sprintf("%s-%s-%s", *tableConfig.TableIdentity, recordID, StatusResolved),
				fmt.Sprintf("%s-%s-%s", *tableConfig.TableIdentity, recordID, StatusUnresolved),
			)
		}
	}
	if len(redisKeys) > 0 {
		err := s.cache.Delete(redisKeys...)
		if err != nil {
			flag = false
			s.log.Error("批量删除 redis 失败",
				logger.String("error", err.Error()),
			)
		}
	}

	wg.Wait()

	if !flag {
		return errs.SyncFAQRecordPartialFailedError(errors.New("部分 FAQ 记录同步失败"))
	}
	return nil
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

func encodePageToken(id uint64) (string, error) {
	token := domain.PageToken{
		LastID: id,
	}

	b, err := json.Marshal(token)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}

func decodePageToken(token string) (*uint64, error) {
	// TODO 后续加上放篡改
	b, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}

	var pt domain.PageToken
	if err := json.Unmarshal(b, &pt); err != nil {
		return nil, err
	}

	return &pt.LastID, nil
}
