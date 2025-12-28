package service

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkbitable "github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/muxi-Infra/FeedBack-Backend/api/request"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/feishu"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

type SheetService interface {
	CreateAPP(name, token string) (*larkbitable.CreateAppResp, error)
	CopyAPP(appToken, name, folderToken, timeZone string, withoutContent bool) (*larkbitable.CopyAppResp, error)
	GetRecord(pageToken, sortOrders, filterName, filterVal string, fieldNames []string,
		desc bool, table *config.Table) (*larkbitable.SearchAppTableRecordResp, error)
	GetNormalRecord(pageToken, sortOrders, filterName, filterVal string, fieldNames []string,
		desc bool, table *config.Table) (*larkbitable.SearchAppTableRecordResp, error)
	CreateRecord(ignoreConsistencyCheck bool, fields map[string]interface{},
		content, problemType string, uc ijwt.UserClaims, table *config.Table) (*larkbitable.CreateAppTableRecordResp, error)
	GetPhotoUrl(fileTokens []string) (*larkdrive.BatchGetTmpDownloadUrlMediaResp, error)
	GetUserLikeRecord(recordID string, userID string) (int, error)
}

type SheetServiceImpl struct {
	likeDao dao.Like
	c       feishu.Client
	log     logger.Logger
	cfg     *config.AppTable
	bcfg    *config.BatchNoticeConfig
	Testing bool
}

func NewSheetService(likeDao dao.Like, c feishu.Client, log logger.Logger, cfg *config.AppTable, bcfg *config.BatchNoticeConfig) SheetService {
	return &SheetServiceImpl{
		likeDao: likeDao,
		c:       c,
		log:     log,
		cfg:     cfg,
		bcfg:    bcfg,
		Testing: false,
	}
}

func (s *SheetServiceImpl) GetUserLikeRecord(recordID string, userID string) (int, error) {
	return s.likeDao.GetUserLikeRecord(recordID, userID)
}

func (s *SheetServiceImpl) CreateAPP(name, folderToken string) (*larkbitable.CreateAppResp, error) {
	// 创建请求对象
	req := larkbitable.NewCreateAppReqBuilder().
		ReqApp(larkbitable.NewReqAppBuilder().
			Name(name).
			FolderToken(folderToken).
			Build()).
		Build()

	// 发起请求
	ctx := context.Background()
	resp, err := s.c.CreateAPP(ctx, req)

	// 处理错误
	if err != nil {
		s.log.Error("CreateApp 调用失败",
			logger.String("error", err.Error()),
		)
		return resp, errs.FeishuRequestError(err)
	}

	// 服务端错误处理
	if !resp.Success() {
		s.log.Error("CreateApp Lark 接口错误",
			logger.String("request_id", resp.RequestId()),
			logger.String("error", larkcore.Prettify(resp.CodeError)),
		)
		return resp, errs.FeishuResponseError(resp.CodeError)
	}

	return resp, nil
}

func (s *SheetServiceImpl) CopyAPP(appToken, name, folderToken, timeZone string, withoutContent bool) (*larkbitable.CopyAppResp, error) {
	// 创建请求对象
	req := larkbitable.NewCopyAppReqBuilder().
		AppToken(appToken).
		Body(larkbitable.NewCopyAppReqBodyBuilder().
			Name(name).
			FolderToken(folderToken).
			WithoutContent(withoutContent).
			TimeZone(timeZone).
			Build()).
		Build()

	ctx := context.Background()
	resp, err := s.c.CopyAPP(ctx, req)

	// 处理错误
	if err != nil {
		s.log.Error("CopyApp 调用失败",
			logger.String("error", err.Error()),
		)
		return resp, errs.FeishuRequestError(err)
	}

	// 服务端错误处理
	if !resp.Success() {
		s.log.Error("CopyApp Lark 接口错误",
			logger.String("request_id", resp.RequestId()),
			logger.String("error", larkcore.Prettify(resp.CodeError)),
		)
		return resp, errs.FeishuResponseError(resp.CodeError)
	}

	return resp, nil
}

func (s *SheetServiceImpl) GetRecord(pageToken, sortOrders, filterName, filterVal string,
	fieldNames []string, desc bool, table *config.Table) (*larkbitable.SearchAppTableRecordResp, error) {
	// 创建请求对象
	req := larkbitable.NewSearchAppTableRecordReqBuilder().
		AppToken(s.cfg.AppToken).
		TableId(table.TableID).
		UserIdType(`open_id`).
		PageToken(pageToken). // 分页参数,第一次不需要
		PageSize(20).         // 分页大小，先默认20
		Body(larkbitable.NewSearchAppTableRecordReqBodyBuilder().
			ViewId(table.ViewID).
			FieldNames(fieldNames).
			Sort([]*larkbitable.Sort{
				larkbitable.NewSortBuilder().
					FieldName(sortOrders).
					Desc(desc).
					Build(),
			}).
			Filter(larkbitable.NewFilterInfoBuilder().
				Conjunction(`and`).
				Conditions([]*larkbitable.Condition{
					larkbitable.NewConditionBuilder().
						FieldName(filterName).
						Operator(`contains`).
						Value([]string{filterVal}).
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
		s.log.Error("GetAppTableRecord 调用失败",
			logger.String("error", err.Error()),
		)
		return resp, errs.FeishuRequestError(err)
	}

	// 服务端错误处理
	if !resp.Success() {
		s.log.Error("GetAppTableRecord Lark 接口错误",
			logger.String("request_id", resp.RequestId()),
			logger.String("error", larkcore.Prettify(resp.CodeError)),
		)
		return resp, errs.FeishuResponseError(err)
	}

	return resp, nil
}

func (s *SheetServiceImpl) GetNormalRecord(pageToken, sortOrders, filterName, filterVal string,
	fieldNames []string, desc bool, table *config.Table) (*larkbitable.SearchAppTableRecordResp, error) {
	bodyBuilder := larkbitable.NewSearchAppTableRecordReqBodyBuilder().
		ViewId(table.ViewID).
		FieldNames(fieldNames).
		Sort([]*larkbitable.Sort{
			larkbitable.NewSortBuilder().
				FieldName(sortOrders).
				Desc(desc).
				Build(),
		}).AutomaticFields(false)
	if filterName != "" && filterVal != "" {
		bodyBuilder.Filter(larkbitable.NewFilterInfoBuilder().
			Conjunction(`and`).
			Conditions([]*larkbitable.Condition{
				larkbitable.NewConditionBuilder().
					FieldName(filterName).
					Operator(`contains`).
					Value([]string{filterVal}).
					Build(),
			}).
			Build())
	}
	// 创建请求对象
	req := larkbitable.NewSearchAppTableRecordReqBuilder().
		AppToken(s.cfg.AppToken).
		TableId(table.TableID).
		UserIdType(`open_id`).
		PageToken(pageToken). // 分页参数,第一次不需要
		PageSize(20).         // 分页大小，先默认20
		Body(bodyBuilder.Build()).
		Build()

	// 发起请求
	ctx := context.Background()
	resp, err := s.c.GetAppTableRecord(ctx, req)

	// 处理错误
	if err != nil {
		s.log.Error("GetNormalRecord 调用失败",
			logger.String("error", err.Error()),
		)
		return resp, errs.FeishuRequestError(err)
	}

	// 服务端错误处理
	if !resp.Success() {
		s.log.Error("GetNormalRecord Lark 接口错误",
			logger.String("request_id", resp.RequestId()),
			logger.String("error", larkcore.Prettify(resp.CodeError)),
		)
		return resp, errs.FeishuResponseError(err)
	}

	return resp, nil
}

func (s *SheetServiceImpl) CreateRecord(ignoreConsistencyCheck bool, fields map[string]interface{},
	content, problemType string, uc ijwt.UserClaims, table *config.Table) (*larkbitable.CreateAppTableRecordResp, error) {
	// 创建请求对象
	req := larkbitable.NewCreateAppTableRecordReqBuilder().
		AppToken(s.cfg.AppToken).
		TableId(table.TableID).
		IgnoreConsistencyCheck(ignoreConsistencyCheck).
		AppTableRecord(larkbitable.NewAppTableRecordBuilder().
			Fields(fields).
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
		return resp, errs.FeishuRequestError(err)
	}

	// 服务端错误处理
	if !resp.Success() {
		s.log.Error("CreateAppTableRecord Lark 接口错误",
			logger.String("request_id", resp.RequestId()),
			logger.String("error", larkcore.Prettify(resp.CodeError)),
		)
		return resp, errs.FeishuResponseError(err)
	}

	// 异步发送批量通知
	if !s.Testing { // 测试环境不发送批量通知
		go func() {
			// 防止panic
			defer func() {
				if err := recover(); err != nil {
					s.log.Error("panic recovered",
						logger.Reflect("error", err),
					)
				}
			}()

			// 生成content
			// 反馈内容
			s.bcfg.Content.Data.TemplateVariable.FeedbackContent = content

			// 反馈类型
			s.bcfg.Content.Data.TemplateVariable.FeedbackType = problemType

			// 反馈来源使用配置表格的名字
			s.bcfg.Content.Data.TemplateVariable.FeedbackSource = s.cfg.Tables[uc.TableCode].Name

			// 构造content
			contentBytes, err := json.Marshal(s.bcfg.Content)
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

			// 批量发送 个人通知
			if err := s.SendBatchNotice(string(contentBytes)); err != nil {
				s.log.Error("SendBatchNotice failed",
					logger.String("error", err.Error()),
				)
			}
		}()
	}

	return resp, nil
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
	for _, old := range s.bcfg.OpenIDs {
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
	for _, old := range s.bcfg.ChatIDs {
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

// FillFields 填充filed的工具函数
// 使用反射生成中文字段
func FillFields(req *request.CreateAppTableRecordReq) {
	// 自动填充的
	var loc, _ = time.LoadLocation("Asia/Shanghai")
	req.SubmitTIme = time.Now().In(loc).UnixMilli() // 日期是需要时间戳的 // 毫秒级别的
	req.Status = "处理中"
	req.Fields = make(map[string]interface{})

	val := reflect.ValueOf(req).Elem()
	valType := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := valType.Field(i)
		// 获取中文字段标签
		feishuKey := field.Tag.Get("feishu")
		if feishuKey == "" {
			continue
		}

		value := val.Field(i).Interface()

		// 可选字段不填不加进去
		if isEmptyValue(val.Field(i)) {
			continue
		}
		req.Fields[feishuKey] = value
	}
}

// 判断字段是否为空值
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String, reflect.Slice, reflect.Map:
		return v.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:
		// 其他类型 int float64 bool……
		// 与其对应的0值进行比较
		zero := reflect.Zero(v.Type())
		return reflect.DeepEqual(v.Interface(), zero.Interface())
	}
}
