package controller

import (
	"context"
	"feedback/api/request"
	"feedback/api/response"
	"feedback/config"
	"feedback/pkg/ijwt"
	"feedback/pkg/logger"
	"feedback/service"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/larksuite/oapi-sdk-go/v3"
	"github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	"reflect"
	"time"
)

type Sheet struct {
	c   *lark.Client
	log logger.Logger
	o   service.AuthService
	cfg *config.AppTable
}

func NewSheet(client *lark.Client, log logger.Logger, cfg *config.AppTable) *Sheet {
	return &Sheet{
		c:   client,
		log: log,
		cfg: cfg,
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
	resp, err := f.c.Bitable.V1.App.Create(context.Background(), req, larkcore.WithUserAccessToken(f.o.GetAccessToken()))

	// 处理错误
	if err != nil {
		// TODO: log
		return response.Response{
			Code:    500,
			Message: "Internal Server Error",
			Data:    nil,
		}, err
	}

	// 服务端错误处理
	if !resp.Success() {
		// TODO: log
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
	resp, err := f.c.Bitable.V1.App.Copy(context.Background(), req, larkcore.WithUserAccessToken(f.o.GetAccessToken()))

	// 处理错误
	if err != nil {
		// TODO: log
		// fmt.Println(err)
		return response.Response{
			Code:    500,
			Message: "Internal Server Error",
			Data:    nil,
		}, err
	}

	// 服务端错误处理
	if !resp.Success() {
		// TODO: log
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
	resp, err := f.c.Bitable.V1.AppTableRecord.Create(context.Background(), req, larkcore.WithUserAccessToken(f.o.GetAccessToken()))

	// 处理错误
	if err != nil {
		// TODO: log
		return response.Response{
			Code:    500,
			Message: "Internal Server Error",
			Data:    nil,
		}, err
	}

	// 服务端错误处理
	if !resp.Success() {
		// TODO: log
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
	resp, err := f.c.Bitable.V1.AppTableRecord.Search(context.Background(), req, larkcore.WithUserAccessToken(f.o.GetAccessToken()))

	// 处理错误
	if err != nil {
		// TODO: log
		return response.Response{
			Code:    500,
			Message: "Internal Server Error",
			Data:    nil,
		}, err
	}

	// 服务端错误处理
	if !resp.Success() {
		// TODO: log
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
	resp, err := f.c.Bitable.V1.AppTableRecord.Search(context.Background(), req, larkcore.WithUserAccessToken(f.o.GetAccessToken()))

	// 处理错误
	if err != nil {
		// TODO: log
		return response.Response{
			Code:    500,
			Message: "Internal Server Error",
			Data:    nil,
		}, err
	}

	// 服务端错误处理
	if !resp.Success() {
		// TODO: log
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
	resp, err := f.c.Drive.V1.Media.BatchGetTmpDownloadUrl(context.Background(), req, larkcore.WithUserAccessToken(f.o.GetAccessToken()))

	// 处理错误
	if err != nil {
		// TODO: log
		return response.Response{
			Code:    500,
			Message: "Internal Server Error",
			Data:    nil,
		}, err
	}

	// 服务端错误处理
	if !resp.Success() {
		// Todo log
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
