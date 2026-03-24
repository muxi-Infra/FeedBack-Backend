package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/lark"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
)

var (
	errCount atomic.Int32
)

//go:generate mockgen -destination=./mock/message_mock.go -package=mocks github.com/muxi-Infra/FeedBack-Backend/service MessageService
type MessageService interface {
	SendLarkNotification(ctx context.Context, tableName, content, url string) error
	TriggerNotification(ctx context.Context, tableIdentify string) error
	GetPendingNotifications(ctx context.Context, tableConfig *domain.TableConfig) ([]domain.NotificationRecipient, error)
	SendCCNUBoxNotification(ctx context.Context, studentID, recordID *string) error
	MarkRecordNoticed(ctx context.Context, recordID *string, tableConfig *domain.TableConfig) error
}

type MessageServiceImpl struct {
	c        lark.Client
	log      logger.Logger
	lc       *config.LarkMessage
	cc       *config.CCNUBoxMessage
	sheetDao dao.SheetDAO
}

func NewMessageService(c lark.Client, log logger.Logger, lc *config.LarkMessage, cc *config.CCNUBoxMessage, sheetDao dao.SheetDAO) MessageService {
	m := &MessageServiceImpl{
		c:        c,
		log:      log,
		lc:       lc,
		cc:       cc,
		sheetDao: sheetDao,
	}

	// 消费者，监听通知通道，根据表格配置查询待通知的记录，并发送通知
	go func() {
		ctx := context.Background()
		for {
			table := <-noticeCh
			if !table.Notice {
				// 如果表格配置未启用通知，跳过处理
				m.log.Warn("notification skipped for table",
					logger.String("table_identity", *table.TableIdentity),
				)
				continue
			}
			// 获取待通知的记录列表
			recipients, err := m.GetPendingNotifications(ctx, &table)
			if err != nil {
				m.log.Error("get ccnubox pending notifications failed",
					logger.String("error", err.Error()),
				)
				continue
			}

			// 根据表格标识发送不同类型的通知
			switch *table.TableIdentity {
			case "ccnubox":
				for _, recipient := range recipients {
					err = m.SendCCNUBoxNotification(ctx, &recipient.StudentID, &recipient.RecordID)
					if err != nil {
						m.log.Error("send ccnubox notification failed",
							logger.String("student_id", recipient.StudentID),
							logger.String("error", err.Error()),
						)
					}
					err := m.MarkRecordNoticed(ctx, &recipient.RecordID, &table)
					if err != nil {
						m.log.Error("MarkRecordNoticed 更新记录通知状态失败",
							logger.String("error", err.Error()),
							logger.String("record_id", recipient.RecordID),
							logger.String("table_identity", *table.TableIdentity),
						)
					}
				}
			default:
				m.log.Error("unsupported table identity",
					logger.String("table_identity", *table.TableIdentity))
			}
		}
	}()

	return m
}

func (m *MessageServiceImpl) SendLarkNotification(ctx context.Context, tableName, content, url string) error {
	if len(content) > 30 {
		content = content[:30] + "......"
	}

	message := domain.LarkMessage{
		Type: "template",
		Data: domain.LarkMessageData{
			TemplateId:          m.lc.TemplateID,
			TemplateVersionName: "",
			TemplateVariable: map[string]interface{}{
				"table_name":       tableName,
				"feedback_content": content,
				"shared_url":       map[string]string{"url": url},
			},
		},
	}
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return errs.SerializationError(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)

	for _, r := range m.lc.ReceiveIDs {
		r := r // 避免闭包问题
		wg.Add(1)

		go func() {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				m.log.Warn("SendLarkMessage canceled by context",
					logger.String("receive_id", r.ID),
				)
				return
			}

			req := larkim.NewCreateMessageReqBuilder().
				ReceiveIdType(r.Type).
				Body(larkim.NewCreateMessageReqBodyBuilder().
					ReceiveId(r.ID).
					MsgType("interactive").
					Content(string(messageBytes)).
					Build()).
				Build()

			resp, err := m.c.SendNotice(ctx, req)
			// 处理错误
			if err != nil {
				errCount.Add(1)
				m.log.Error("SendLarkMessage failed",
					logger.String("receive_id", r.ID),
					logger.String("error", err.Error()),
				)
				return
			}

			// 服务端错误处理
			if !resp.Success() {
				errCount.Add(1)
				m.log.Error("Lark API error",
					logger.String("receive_id", r.ID),
					logger.String("request_id", resp.RequestId()),
					logger.String("error", larkcore.Prettify(resp.CodeError)),
				)
				return
			}
		}()
	}

	wg.Wait()
	if errCount.Load() > 0 {
		return errs.LarkMessagePartialFailureError(fmt.Errorf("send message failed: %v", errCount.Load()))
	}

	return nil
}

func (m *MessageServiceImpl) TriggerNotification(ctx context.Context, tableIdentify string) error {
	table, ok := tableCfg[tableIdentify]
	if !ok {
		return errs.TableIdentifierInvalidError(fmt.Errorf("table identify not found: %s", tableIdentify))
	}
	if !table.Notice {
		return errs.TableNotificationNotConfiguredError(fmt.Errorf("table notification not configured: %s", tableIdentify))
	}

	select {
	case noticeCh <- table:
		return nil
	default:
		return errs.AppNotificationChannelFullError(fmt.Errorf("notification channel is full"))
	}
}

// GetPendingNotifications 根据表格配置查询待通知的记录，并返回通知对象列表
func (m *MessageServiceImpl) GetPendingNotifications(ctx context.Context, tableConfig *domain.TableConfig) ([]domain.NotificationRecipient, error) {
	records, err := m.sheetDao.GetUnNoticedRecordsByTable(ctx, *tableConfig.TableIdentity)
	if err != nil {
		return nil, errs.GetUnNoticedRecordByTableError(err)
	}

	var recipients []domain.NotificationRecipient
	for _, record := range records {
		recipients = append(recipients, domain.NotificationRecipient{
			RecordID:  *record.RecordID,
			StudentID: *record.UserID,
		})
	}

	return recipients, nil
}

func (m *MessageServiceImpl) SendCCNUBoxNotification(ctx context.Context, studentID, recordID *string) error {
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(m.cc.BasicUser+":"+m.cc.BasicPassword))
	message := domain.CCNUBoxFeedMessage{
		Content:   "您的问题已经处理完成，点击查看详情",
		StudentID: *studentID,
		Title:     "反馈处理完成提醒",
		RecordID:  *recordID,
	}

	// 编码请求体
	jsonData, err := json.Marshal(message)
	if err != nil {
		m.log.Error("marshal request body failed",
			logger.String("student_id", *studentID),
			logger.String("error", err.Error()),
		)
		return errs.SerializationError(fmt.Errorf("编码请求体失败: %w", err))
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", m.cc.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		m.log.Error("create http request failed",
			logger.String("student_id", *studentID),
			logger.String("error", err.Error()),
		)
		return errs.HTTPRequestCreationError(fmt.Errorf("创建请求失败: %w", err))
	}
	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", auth)
	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		m.log.Error("send request failed",
			logger.String("student_id", *studentID),
			logger.String("error", err.Error()),
		)
		return errs.CCNUBoxRequestError(fmt.Errorf("发送请求失败: %w", err))
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		m.log.Error("read response failed",
			logger.String("student_id", *studentID),
			logger.String("error", err.Error()),
		)
		return errs.HTTPResponseReadError(fmt.Errorf("读取响应失败: %w", err))
	}

	if resp.StatusCode != http.StatusOK {
		m.log.Error("ccnubox response not ok",
			logger.String("student_id", *studentID),
			logger.String("status", fmt.Sprintf("%d", resp.StatusCode)),
			logger.String("body", string(body)),
		)
		return errs.CCNUBoxResponseError(fmt.Errorf("请求返回异常: %d : %s", resp.StatusCode, string(body)))
	}

	return nil
}

// MarkRecordNoticed 更新通知记录的状态为已完成
func (m *MessageServiceImpl) MarkRecordNoticed(ctx context.Context, recordID *string, tableConfig *domain.TableConfig) error {
	err := m.sheetDao.MarkRecordNoticed(ctx, *tableConfig.TableIdentity, *recordID)
	if err != nil {
		return errs.MarkRecordNoticedError(err)
	}
	return nil
}
