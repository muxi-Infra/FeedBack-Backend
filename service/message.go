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
	larkbitable "github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/lark"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
)

var (
	wg       sync.WaitGroup
	errCount atomic.Int32
)

//go:generate mockgen -destination=./mock/message_mock.go -package=mocks github.com/muxi-Infra/FeedBack-Backend/service MessageService
type MessageService interface {
	SendLarkNotification(tableName, content, url string) error
	TriggerNotification(tableIdentify string) error
	GetCCNUBoxPendingNotifications(tableConfig *domain.TableConfig) ([]domain.NotificationRecipient, error)
	SendCCNUBoxNotification(studentID, recordID *string) error
}

type MessageServiceImpl struct {
	c   lark.Client
	log logger.Logger
	lc  *config.LarkMessage
	cc  *config.CCNUBoxMessage
}

func NewMessageService(c lark.Client, log logger.Logger, lc *config.LarkMessage, cc *config.CCNUBoxMessage) MessageService {
	m := &MessageServiceImpl{
		c:   c,
		log: log,
		lc:  lc,
		cc:  cc,
	}

	go func() {
		for {
			table := <-noticeCh
			switch *table.TableIdentity {
			case "ccnubox":
				recipients, err := m.GetCCNUBoxPendingNotifications(&table)
				if err != nil {
					m.log.Error("get ccnubox pending notifications failed",
						logger.String("error", err.Error()),
					)
					continue
				}
				for _, recipient := range recipients {
					err = m.SendCCNUBoxNotification(&recipient.StudentID, &recipient.RecordID)
					if err != nil {
						m.log.Error("send ccnubox notification failed",
							logger.String("student_id", recipient.StudentID),
							logger.String("error", err.Error()),
						)
					}

					rid := recipient.RecordID
					tbl := table
					msg := ProgressMsg{RecordID: rid, TableConfig: tbl}

					go func(msg ProgressMsg) {
						progressCh <- msg
					}(msg)
				}
			default:
				m.log.Error("unsupported table identity",
					logger.String("table_identity", *table.TableIdentity))
			}
		}
	}()

	return m
}

func (m MessageServiceImpl) SendLarkNotification(tableName, content, url string) error {
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

func (m MessageServiceImpl) TriggerNotification(tableIdentify string) error {
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

func (m MessageServiceImpl) GetCCNUBoxPendingNotifications(tableConfig *domain.TableConfig) ([]domain.NotificationRecipient, error) {
	if *tableConfig.TableIdentity != m.cc.TableIdentify {
		return nil, errs.TableIdentifierInvalidError(fmt.Errorf("invalid table identity: %s", *tableConfig.TableIdentity))
	}

	filter := larkbitable.NewFilterInfoBuilder().
		Conjunction(`and`).
		Conditions([]*larkbitable.Condition{
			larkbitable.NewConditionBuilder().
				FieldName(`进度`).
				Operator(`is`).
				Value([]string{`待通知`}).
				Build(),
		}).Build()

	// 创建请求对象
	req := larkbitable.NewSearchAppTableRecordReqBuilder().
		AppToken(*tableConfig.TableToken).
		TableId(*tableConfig.TableID).
		PageToken(``).
		PageSize(500).
		Body(larkbitable.NewSearchAppTableRecordReqBodyBuilder().
			ViewId(*tableConfig.ViewID).
			FieldNames([]string{`学号`}).
			Filter(filter).
			Build()).
		Build()

	// 发起请求
	ctx := context.Background()
	resp, err := m.c.GetAppTableRecord(ctx, req)

	// 处理错误
	if err != nil {
		m.log.Error("获取待通知人员名单失败",
			logger.String("error", err.Error()))
		return nil, errs.LarkRequestError(fmt.Errorf("获取待通知人员名单 Lark 请求失败: %v", err))
	}

	// 服务端错误处理
	if !resp.Success() {
		m.log.Error("获取待通知人员名单 Lark 接口错误",
			logger.String("request_id", resp.RequestId()),
			logger.String("error", larkcore.Prettify(resp.CodeError)))
		return nil, errs.LarkRequestError(fmt.Errorf("获取待通知人员名单 Lark 接口错误: %v", err))
	}

	var recipients []domain.NotificationRecipient
	for _, item := range resp.Data.Items {
		var r domain.NotificationRecipient
		if item.RecordId != nil {
			r.RecordID = *item.RecordId
		}
		if item.Fields != nil {
			// 使用 simplifyFields 处理复杂结构
			simplifiedFields := simplifyFields(item.Fields)

			// 从简化后的字段中获取学号
			if val, ok := simplifiedFields["学号"]; ok {
				if studentID, ok := val.(string); ok {
					r.StudentID = studentID
				}
			}
		}
		recipients = append(recipients, r)
	}
	return recipients, nil
}

func (m MessageServiceImpl) SendCCNUBoxNotification(studentID, recordID *string) error {
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(m.cc.BasicUser+":"+m.cc.BasicPassword))
	message := domain.CCNUBoxFeedMessage{
		Content:   "您的问题已经处理完成，点击查看详情",
		StudentID: *studentID,
		Title:     "反馈处理完成提醒",
		Type:      "feed_back",
		URL:       *recordID,
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
