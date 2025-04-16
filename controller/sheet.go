package controller

import (
	"context"
	"feedback/api/request"
	"feedback/api/response"
	"feedback/pkg/ijwt"
	"feedback/pkg/logger"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/larksuite/oapi-sdk-go/v3"
	"github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
)

type Sheet struct {
	c   *lark.Client
	log logger.Logger
}

func NewSheet(client *lark.Client, log logger.Logger) *Sheet {
	return &Sheet{
		c:   client,
		log: log,
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
//	@Router			/sheet/createapp [post]
func (f Sheet) CreateApp(c *gin.Context, r request.CreateAppReq, uc ijwt.UserClaims) (response.Response, error) {
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
	resp, err := f.c.Bitable.V1.App.Create(context.Background(), req, larkcore.WithUserAccessToken(uc.Token))

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
			Message: "Internal Server Error",
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
func (f Sheet) CopyApp(c *gin.Context, r request.CopyAppReq, uc ijwt.UserClaims) (response.Response, error) {
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
	resp, err := f.c.Bitable.V1.App.Copy(context.Background(), req, larkcore.WithUserAccessToken(uc.Token))

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
			Message: "Internal Server Error",
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
func (f Sheet) CreateAppTableRecord(c *gin.Context, r request.CreateAppTableRecordReq, uc ijwt.UserClaims) (response.Response, error) {
	// 创建 Client
	// c := lark.NewClient("YOUR_APP_ID", "YOUR_APP_SECRET")
	// 创建请求对象
	req := larkbitable.NewCreateAppTableRecordReqBuilder().
		AppToken(r.AppToken).
		TableId(r.TableId).
		IgnoreConsistencyCheck(r.IgnoreConsistencyCheck).
		AppTableRecord(larkbitable.NewAppTableRecordBuilder().
			Fields(r.Fields).
			Build()).
		Build()

	// 发起请求
	resp, err := f.c.Bitable.V1.AppTableRecord.Create(context.Background(), req, larkcore.WithUserAccessToken(uc.Token))

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
			Message: "Internal Server Error",
			Data:    resp.CodeError,
		}, fmt.Errorf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
	}

	// 业务处理
	// fmt.Println(larkcore.Prettify(resp))
	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp.Data,
	}, nil
}
