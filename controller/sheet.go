package controller

import (
	"context"
	"encoding/json"
	"feedback/pkg/feishu"
	"fmt"
	"reflect"
	"time"

	"feedback/api/request"
	"feedback/api/response"
	"feedback/config"
	"feedback/pkg/ijwt"
	"feedback/pkg/logger"
	"feedback/service"

	"github.com/gin-gonic/gin"
	"github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

type Sheet struct {
	c       feishu.Client
	log     logger.Logger
	o       service.AuthService
	s       service.SheetService
	cfg     *config.AppTable
	bcfg    *config.BatchNoticeConfig
	Testing bool
}

func NewSheet(client feishu.Client, log logger.Logger, o service.AuthService, s service.SheetService, cfg *config.AppTable, bcfg *config.BatchNoticeConfig) *Sheet {
	return &Sheet{
		c:       client,
		log:     log,
		o:       o,
		s:       s,
		cfg:     cfg,
		bcfg:    bcfg,
		Testing: false,
	}
}

// CreateApp 创建多维表格
//
//	@Summary		创建多维表格
//	@Description	基于给定的名称和文件夹 Token 创建一个新的多维表格应用
//	@Tags			Sheet
//	@ID				create-app
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string					true	"Bearer Token"
//	@Param			request			body		request.CreateAppReq	true	"创建表格请求参数"
//	@Success		200				{object}	response.Response		"成功返回创建结果"
//	@Failure		400				{object}	response.Response		"请求参数错误或飞书接口调用失败"
//	@Failure		500				{object}	response.Response		"服务器内部错误"
//	@Router			/sheet/createapp [post]
func (f *Sheet) CreateApp(c *gin.Context, r request.CreateAppReq, uc ijwt.UserClaims) (response.Response, error) {
	// 创建 Client
	// c := lark.NewClient("YOUR_APP_ID", "YOUR_APP_SECRET")
	// 创建请求对象
	req := larkbitable.NewCreateAppReqBuilder().
		ReqApp(larkbitable.NewReqAppBuilder().
			Name(r.Name).
			FolderToken(r.FolderToken).
			Build()).
		Build()

	// 发起请求
	resp, err := f.c.CreateAPP(context.Background(), req, larkcore.WithUserAccessToken(f.o.GetAccessToken()))

	// 处理错误
	if err != nil {
		f.log.Errorf("[CreateApp] 调用失败: %v", err)
		return response.Response{
			Code:    500,
			Message: "Internal Server Error",
			Data:    nil,
		}, err
	}

	// 服务端错误处理
	if !resp.Success() {
		f.log.Errorf("[CreateApp] Lark 接口错误, rid=%s, err=%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
		return response.Response{
			Code:    400,
			Message: "Bad Request",
			Data:    resp.CodeError,
		}, fmt.Errorf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
	}

	// 业务处理
	//fmt.Println(larkcore.Prettify(resp))
	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp.Data,
	}, nil
}

// CopyApp 从模版复制创建多维表格
//
//	@Summary		从模版复制创建多维表格
//	@Description	基于已有的模板 AppToken 复制创建一个新的多维表格应用
//	@Tags			Sheet
//	@ID				copy-app
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string				true	"Bearer Token"
//	@Param			request			body		request.CopyAppReq	true	"复制表格请求参数"
//	@Success		200				{object}	response.Response	"成功返回复制结果"
//	@Failure		400				{object}	response.Response	"请求参数错误或飞书接口调用失败"
//	@Failure		500				{object}	response.Response	"服务器内部错误"
//	@Router			/sheet/copyapp [post]
func (f *Sheet) CopyApp(c *gin.Context, r request.CopyAppReq, uc ijwt.UserClaims) (response.Response, error) {
	// 创建 Client
	// c:= lark.NewClient("YOUR_APP_ID", "YOUR_APP_SECRET")
	// 创建请求对象
	req := larkbitable.NewCopyAppReqBuilder().
		AppToken(r.AppToken).
		Body(larkbitable.NewCopyAppReqBodyBuilder().
			Name(r.Name).
			FolderToken(r.FolderToken).
			WithoutContent(r.WithoutContent).
			TimeZone(r.TimeZone).
			Build()).
		Build()

	// 发起请求
	resp, err := f.c.CopyAPP(context.Background(), req, larkcore.WithUserAccessToken(f.o.GetAccessToken()))

	// 处理错误
	if err != nil {
		f.log.Errorf("[CopyApp] 调用失败: %v", err)
		// fmt.Println(err)
		return response.Response{
			Code:    500,
			Message: "Internal Server Error",
			Data:    nil,
		}, err
	}

	// 服务端错误处理
	if !resp.Success() {
		f.log.Errorf("[CopyApp] Lark 接口错误, rid=%s, err=%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
		return response.Response{
			Code:    400,
			Message: "Bad Request",
			Data:    resp.CodeError,
		}, fmt.Errorf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
	}

	// 业务处理
	//fmt.Println(larkcore.Prettify(resp))
	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp.Data,
	}, nil
}

// CreateAppTableRecord 创建多维表格记录
//
//	@Summary		创建多维表格记录
//	@Description	向指定的多维表格应用中添加记录数据
//	@Tags			Sheet
//	@ID				create-app-table-record
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string							true	"Bearer Token"
//	@Param			request			body		request.CreateAppTableRecordReq	true	"新增记录请求参数"
//	@Success		200				{object}	response.Response				"成功返回创建记录结果"
//	@Failure		400				{object}	response.Response				"请求参数错误或飞书接口调用失败"
//	@Failure		500				{object}	response.Response				"服务器内部错误"
//	@Router			/sheet/createrecord [post]
func (f *Sheet) CreateAppTableRecord(c *gin.Context, r request.CreateAppTableRecordReq, uc ijwt.UserClaims) (response.Response, error) {
	// 获取表ID
	tabel, ok := f.cfg.Tables[uc.TableID]
	if !ok {
		return response.Response{
			Code:    400,
			Message: "Bad Request",
			Data:    nil,
		}, fmt.Errorf("table id %s not found", uc.TableID)
	}

	// 填充fields
	if r.Fields == nil {
		r.Fields = make(map[string]interface{})
	}
	fillFields(&r)

	// 创建请求对象
	req := larkbitable.NewCreateAppTableRecordReqBuilder().
		AppToken(f.cfg.AppToken).
		TableId(tabel.TableID).
		IgnoreConsistencyCheck(r.IgnoreConsistencyCheck).
		AppTableRecord(larkbitable.NewAppTableRecordBuilder().
			Fields(r.Fields).
			Build()).
		Build()

	// 发起请求
	resp, err := f.c.CreateAppTableRecord(context.Background(), req, larkcore.WithUserAccessToken(f.o.GetAccessToken()))

	// 处理错误
	if err != nil {
		f.log.Errorf("[CreateAppTableRecord] 调用失败: %v", err)
		return response.Response{
			Code:    500,
			Message: "Internal Server Error",
			Data:    nil,
		}, err
	}

	// 服务端错误处理
	if !resp.Success() {
		f.log.Errorf("[CreateAppTableRecord] Lark 接口错误, rid=%s, err=%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
		return response.Response{
			Code:    400,
			Message: "Bad Request",
			Data:    resp.CodeError,
		}, fmt.Errorf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
	}

	// 异步发送批量通知
	if !f.Testing { // 测试环境不发送批量通知
		go func() {
			// 防止panic
			defer func() {
				if err := recover(); err != nil {
					f.log.Errorf("panic: %v", err)
				}
			}()

			// 生成content
			// 反馈内容
			f.bcfg.Content.Data.TemplateVariable.FeedbackContent = r.Content

			// 反馈类型
			f.bcfg.Content.Data.TemplateVariable.FeedbackType = r.ProblemType

			// 反馈来源使用配置表格的名字
			f.bcfg.Content.Data.TemplateVariable.FeedbackSource = f.cfg.Tables[uc.TableID].Name

			// 构造content
			contentBytes, err := json.Marshal(f.bcfg.Content)
			if err != nil {
				f.log.Errorf("json.Marshal error: %v", err)
				return
			}

			// 批量发送 群组通知
			if err := f.SendBatchGroupNotice(context.Background(), string(contentBytes)); err != nil {
				f.log.Errorf("SendBatchGroupNotice error: %v", err)
			}

			// 批量发送 个人通知
			if err := f.SendBatchNotice(context.Background(), string(contentBytes)); err != nil {
				f.log.Errorf("SendBatchNotice error: %v", err)
			}
		}()
	}

	// 业务处理
	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp.Data,
	}, nil
}

// GetAppTableRecord 获取多维表格记录
//
//	@Summary		获取多维表格记录
//	@Description	根据指定条件查询多维表格中的记录数据
//	@Tags			Sheet
//	@ID				get-app-table-record
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string							true	"Bearer Token"
//	@Param			request			body		request.GetAppTableRecordReq	true	"查询记录请求参数"
//	@Success		200				{object}	response.Response				"成功返回查询结果"
//	@Failure		400				{object}	response.Response				"请求参数错误或飞书接口调用失败"
//	@Failure		500				{object}	response.Response				"服务器内部错误"
//	@Router			/sheet/getrecord [post]
func (f *Sheet) GetAppTableRecord(c *gin.Context, r request.GetAppTableRecordReq, uc ijwt.UserClaims) (response.Response, error) {
	// 获取表ID
	table, ok := f.cfg.Tables[uc.TableID]
	if !ok {
		return response.Response{
			Code:    400,
			Message: "Bad Request",
			Data:    nil,
		}, fmt.Errorf("table id %s not found", uc.TableID)
	}
	// 创建请求对象
	req := larkbitable.NewSearchAppTableRecordReqBuilder().
		AppToken(f.cfg.AppToken).
		TableId(table.TableID).
		UserIdType(`open_id`).
		PageToken(r.PageToken). // 分页参数,第一次不需要
		PageSize(20).           // 分页大小，先默认20
		Body(larkbitable.NewSearchAppTableRecordReqBodyBuilder().
			ViewId(table.ViewID).
			FieldNames(r.FieldNames).
			Sort([]*larkbitable.Sort{
				larkbitable.NewSortBuilder().
					FieldName(r.SortOrders).
					Desc(r.Desc).
					Build(),
			}).
			Filter(larkbitable.NewFilterInfoBuilder().
				Conjunction(`and`).
				Conditions([]*larkbitable.Condition{
					larkbitable.NewConditionBuilder().
						FieldName(r.FilterName).
						Operator(`contains`).
						Value([]string{r.FilterVal}).
						Build(),
				}).
				Build()).
			AutomaticFields(false).
			Build()).
		Build()

	// 发起请求
	resp, err := f.c.GetAppTableRecord(context.Background(), req, larkcore.WithUserAccessToken(f.o.GetAccessToken()))

	// 处理错误
	if err != nil {
		f.log.Errorf("[GetAppTableRecord] 调用失败: %v", err)
		return response.Response{
			Code:    500,
			Message: "Internal Server Error",
			Data:    nil,
		}, err
	}

	// 服务端错误处理
	if !resp.Success() {
		f.log.Errorf("[GetAppTableRecord] Lark 接口错误, rid=%s, err=%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
		return response.Response{
			Code:    400,
			Message: "Bad Request",
			Data:    resp.CodeError,
		}, fmt.Errorf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
	}

	// 业务处理
	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp.Data,
	}, nil
}

// GetNormalRecord 获取常见问题记录
//
//	@Summary		获取常见问题记录
//	@Description	根据指定条件查询多维表格中的记录数据
//	@Tags			Sheet
//	@ID				get-app-table-normal-problem-record
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string							true	"Bearer Token"
//	@Param			request			body		request.GetAppTableRecordReq	true	"查询记录请求参数"
//	@Success		200				{object}	response.Response				"成功返回查询结果"
//	@Failure		400				{object}	response.Response				"请求参数错误或飞书接口调用失败"
//	@Failure		500				{object}	response.Response				"服务器内部错误"
//	@Router			/sheet/getnormal [post]
func (f *Sheet) GetNormalRecord(c *gin.Context, r request.GetAppTableRecordReq, uc ijwt.UserClaims) (response.Response, error) {
	// 获取表ID
	table, ok := f.cfg.Tables[uc.NormalTableID]
	if !ok {
		return response.Response{
			Code:    400,
			Message: "Bad Request",
			Data:    nil,
		}, fmt.Errorf("normal problem table %s id  not found", uc.NormalTableID)
	}

	bodyBuilder := larkbitable.NewSearchAppTableRecordReqBodyBuilder().
		ViewId(table.ViewID).
		FieldNames(r.FieldNames).
		Sort([]*larkbitable.Sort{
			larkbitable.NewSortBuilder().
				FieldName(r.SortOrders).
				Desc(r.Desc).
				Build(),
		}).AutomaticFields(false)
	if r.FilterName != "" && r.FilterVal != "" {
		bodyBuilder.Filter(larkbitable.NewFilterInfoBuilder().
			Conjunction(`and`).
			Conditions([]*larkbitable.Condition{
				larkbitable.NewConditionBuilder().
					FieldName(r.FilterName).
					Operator(`contains`).
					Value([]string{r.FilterVal}).
					Build(),
			}).
			Build())
	}
	// 创建请求对象
	req := larkbitable.NewSearchAppTableRecordReqBuilder().
		AppToken(f.cfg.AppToken).
		TableId(table.TableID).
		UserIdType(`open_id`).
		PageToken(r.PageToken). // 分页参数,第一次不需要
		PageSize(20).           // 分页大小，先默认20
		Body(bodyBuilder.Build()).
		Build()

	// 发起请求
	resp, err := f.c.GetAppTableRecord(context.Background(), req, larkcore.WithUserAccessToken(f.o.GetAccessToken()))

	// 处理错误
	if err != nil {
		f.log.Errorf("[GetNormalRecord] 调用失败: %v", err)
		return response.Response{
			Code:    500,
			Message: "Internal Server Error",
			Data:    nil,
		}, err
	}

	// 服务端错误处理
	if !resp.Success() {
		f.log.Errorf("[GetNormalRecord] Lark 接口错误, rid=%s, err=%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
		return response.Response{
			Code:    400,
			Message: "Bad Request",
			Data:    resp.CodeError,
		}, fmt.Errorf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
	}

	// 业务处理
	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    f.BandingLike(resp.Data, r.StudentID),
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
//	@Param			request			body		request.GetPhotoUrlReq	true	"获取截图URL请求参数"
//	@Success		200				{object}	response.Response		"成功返回临时URL信息"
//	@Failure		400				{object}	response.Response		"请求参数错误或飞书接口调用失败"
//	@Failure		500				{object}	response.Response		"服务器内部错误"
//	@Router			/sheet/getphotourl [post]
func (f *Sheet) GetPhotoUrl(c *gin.Context, r request.GetPhotoUrlReq, uc ijwt.UserClaims) (res response.Response, err error) {

	// 创建请求对象
	req := larkdrive.NewBatchGetTmpDownloadUrlMediaReqBuilder().
		FileTokens(r.FileTokens).
		Build()

	// 发起请求
	resp, err := f.c.GetPhotoUrl(context.Background(), req, larkcore.WithUserAccessToken(f.o.GetAccessToken()))

	// 处理错误
	if err != nil {
		f.log.Errorf("[GetPhotoUrl] 调用失败: %v", err)
		return response.Response{
			Code:    500,
			Message: "Internal Server Error",
			Data:    nil,
		}, err
	}

	// 服务端错误处理
	if !resp.Success() {
		f.log.Errorf("[GetPhotoUrl] Lark 接口错误, rid=%s, err=%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
		return response.Response{
			Code:    400,
			Message: "Bad Request",
			Data:    resp.CodeError,
		}, fmt.Errorf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
	}

	// 业务处理
	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp.Data,
	}, nil
}

// SendBatchNotice  发送通知
// 批量发送给个人通知
func (f *Sheet) SendBatchNotice(c context.Context, content string) error {
	// 发送消息这个接口限速50次/s
	// 创建一个限制器
	limiter := rate.NewLimiter(rate.Every(25*time.Millisecond), 1) // 每25ms一次，即40次/s

	// 创建errgroup 接受错误
	g, ctx := errgroup.WithContext(c)

	// 根据open_id发送消息
	for _, old := range f.bcfg.OpenIDs {
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
			resp, err := f.c.SendNotice(context.Background(), req, larkcore.WithUserAccessToken(f.o.GetAccessToken()))

			// 处理错误
			if err != nil {
				return fmt.Errorf("send to name [%s] open_id [%s] failed: %w", name, openId, err)
			}

			// 服务端错误处理
			if !resp.Success() {
				f.log.Errorf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
				return fmt.Errorf("send to name [%s] open_id [%s] failed: %v", name, openId, larkcore.Prettify(resp.CodeError))
			}
			return nil
		})
	}
	return g.Wait()
}

// SendBatchGroupNotice 发送群组通知
// 支持批量发送
func (f *Sheet) SendBatchGroupNotice(c context.Context, content string) error {
	// 发送消息这个接口限速50次/s
	// 创建一个限制器
	limiter := rate.NewLimiter(rate.Every(25*time.Millisecond), 1) // 每25ms一次，即40次/s

	// 创建errgroup 接受错误
	g, ctx := errgroup.WithContext(c)

	// 根据chat_id发送消息
	for _, old := range f.bcfg.ChatIDs {
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
			resp, err := f.c.SendNotice(context.Background(), req, larkcore.WithUserAccessToken(f.o.GetAccessToken()))

			if err != nil {
				return fmt.Errorf("send to name [%s] chat_id [%s] failed: %w", name, chatId, err)
			}
			if !resp.Success() {
				f.log.Errorf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
				return fmt.Errorf("send to name [%s] chat_id [%s] failed: %v", name, chatId, larkcore.Prettify(resp.CodeError))
			}
			return nil
		})
	}
	return g.Wait()
}

// 填充filed的工具函数
// 使用反射生成中文字段
func fillFields(req *request.CreateAppTableRecordReq) {

	// 自动填充的
	var loc, _ = time.LoadLocation("Asia/Shanghai")
	req.SubmitTIme = time.Now().In(loc).UnixMilli() // 日期是需要时间戳的 // 毫秒级别的
	req.Status = "处理中"

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

func (f *Sheet) BandingLike(data *larkbitable.SearchAppTableRecordRespData, studentID string) *larkbitable.SearchAppTableRecordRespData {
	if data == nil {
		return nil
	}

	for _, record := range data.Items {
		if record.RecordId == nil {
			continue
		}
		if record.Fields == nil {
			record.Fields = make(map[string]any)
		}

		if f.s != nil {
			val, _ := f.s.GetUserLikeRecord(*record.RecordId, studentID)
			record.Fields["点赞情况"] = val
		}
	}
	return data
}
