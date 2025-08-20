package feishu

import (
	"context"
	"feedback/config"
	"github.com/google/wire"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkbitable "github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

var ProviderSet = wire.NewSet(NewClient)

// 不再暴露client,而是使用接口，方便mock
type Client interface {
	CreateAPP(ctx context.Context, req *larkbitable.CreateAppReq, options ...larkcore.RequestOptionFunc) (*larkbitable.CreateAppResp, error)
	CopyAPP(ctx context.Context, req *larkbitable.CopyAppReq, options ...larkcore.RequestOptionFunc) (*larkbitable.CopyAppResp, error)
	CreateAppTableRecord(ctx context.Context, req *larkbitable.CreateAppTableRecordReq, options ...larkcore.RequestOptionFunc) (*larkbitable.CreateAppTableRecordResp, error)
	GetAppTableRecord(ctx context.Context, req *larkbitable.SearchAppTableRecordReq, options ...larkcore.RequestOptionFunc) (*larkbitable.SearchAppTableRecordResp, error)
	GetPhotoUrl(ctx context.Context, req *larkdrive.BatchGetTmpDownloadUrlMediaReq, options ...larkcore.RequestOptionFunc) (*larkdrive.BatchGetTmpDownloadUrlMediaResp, error)
	SendNotice(ctx context.Context, req *larkim.CreateMessageReq, options ...larkcore.RequestOptionFunc) (*larkim.CreateMessageResp, error)
}

type ClientImpl struct {
	c *lark.Client
}

func NewClient(conf config.ClientConfig) Client {
	return &ClientImpl{
		c: lark.NewClient(conf.AppID, conf.AppSecret),
	}

}

// CreateAPP 创建多维表格
func (c *ClientImpl) CreateAPP(ctx context.Context, req *larkbitable.CreateAppReq, options ...larkcore.RequestOptionFunc) (*larkbitable.CreateAppResp, error) {
	return c.c.Bitable.V1.App.Create(ctx, req, options...)
}

// CopyAPP 复制多维表格
func (c *ClientImpl) CopyAPP(ctx context.Context, req *larkbitable.CopyAppReq, options ...larkcore.RequestOptionFunc) (*larkbitable.CopyAppResp, error) {
	return c.c.Bitable.V1.App.Copy(ctx, req, options...)
}

// CreateAppTableRecord 创建多维表格记录
func (c *ClientImpl) CreateAppTableRecord(ctx context.Context, req *larkbitable.CreateAppTableRecordReq, options ...larkcore.RequestOptionFunc) (*larkbitable.CreateAppTableRecordResp, error) {
	return c.c.Bitable.V1.AppTableRecord.Create(ctx, req, options...)
}

// GetAppTableRecord 获取多维表格记录
func (c *ClientImpl) GetAppTableRecord(ctx context.Context, req *larkbitable.SearchAppTableRecordReq, options ...larkcore.RequestOptionFunc) (*larkbitable.SearchAppTableRecordResp, error) {
	return c.c.Bitable.V1.AppTableRecord.Search(ctx, req, options...)
}

// GetPhotoUrl 获取图片链接
func (c *ClientImpl) GetPhotoUrl(ctx context.Context, req *larkdrive.BatchGetTmpDownloadUrlMediaReq, options ...larkcore.RequestOptionFunc) (*larkdrive.BatchGetTmpDownloadUrlMediaResp, error) {
	return c.c.Drive.V1.Media.BatchGetTmpDownloadUrl(ctx, req, options...)
}

// SendNotice 发送单条消息
func (c *ClientImpl) SendNotice(ctx context.Context, req *larkim.CreateMessageReq, options ...larkcore.RequestOptionFunc) (*larkim.CreateMessageResp, error) {
	return c.c.Im.V1.Message.Create(ctx, req, options...)
}
