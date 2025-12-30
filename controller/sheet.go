package controller

import (
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
	"github.com/muxi-Infra/FeedBack-Backend/api/request"
	"github.com/muxi-Infra/FeedBack-Backend/api/response"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/service"
)

type Sheet struct {
	log logger.Logger
	s   service.SheetService
}

func NewSheet(log logger.Logger, s service.SheetService) *Sheet {
	return &Sheet{
		log: log,
		s:   s,
	}
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
	// 组装请求参数
	service.FillFields(&r)

	// 发起请求
	resp, err := f.s.CreateRecord(r.IgnoreConsistencyCheck, r.Fields, r.Content,
		r.ProblemType, uc.TableToken, uc.TableId, uc.TableName)
	if err != nil {
		return response.Response{}, err
	}

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
	resp, err := f.s.GetRecord(r.PageToken, r.SortOrders, r.FilterName, r.FilterVal,
		r.FieldNames, r.Desc, uc.TableToken, uc.TableId, uc.ViewId)
	if err != nil {
		return response.Response{}, err
	}

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
	resp, err := f.s.GetNormalRecord(r.PageToken, r.SortOrders, r.FilterName, r.FilterVal,
		r.FieldNames, r.Desc, uc.TableToken, uc.TableId, uc.ViewId)
	if err != nil {
		return response.Response{}, err
	}

	data := resp.Data
	if data == nil {
		return response.Response{
			Code:    0,
			Message: "Success",
			Data:    data,
		}, nil
	}

	// 并发获取点赞情况，限制并发数避免资源耗尽
	const maxConcurrency = 10
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrency)

	// 关联点赞情况
	for _, record := range data.Items {
		// 局部副本，避免闭包捕获循环变量
		rec := record
		if rec.RecordId == nil {
			continue
		}
		if rec.Fields == nil {
			rec.Fields = make(map[string]any)
		}

		sem <- struct{}{}
		wg.Add(1)

		go func(rrec *larkbitable.AppTableRecord) {
			defer wg.Done()
			defer func() { <-sem }()

			val, _ := f.s.GetUserLikeRecord(*rrec.RecordId, r.StudentID)
			rrec.Fields["点赞情况"] = val
		}(rec)
	}

	wg.Wait()

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    data,
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
	resp, err := f.s.GetPhotoUrl(r.FileTokens)
	if err != nil {
		return response.Response{}, err
	}

	return response.Response{
		Code:    0,
		Message: "Success",
		Data:    resp.Data,
	}, nil
}
