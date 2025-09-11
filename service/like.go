package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"math"
	"time"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkbitable "github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"

	"feedback/model"
	"feedback/pkg/feishu"
	"feedback/pkg/logger"
	"feedback/repository/dao"
)

type LikeService interface {
	AddLikeTask(appToken, tableId, recordID string, userID string, isLike int, action string) error
	GetLikeTask() (*model.LikeMessage, error)
	HandleLikeTask()
	GetRecord(appToken, tableId, recordID string) (*larkbitable.AppTableRecord, error)
	MoveRetry2Pending() error
}

type LikeServiceImpl struct {
	dao dao.Like
	c   feishu.Client
	log logger.Logger
	o   AuthService
}

func NewLikeService(dao dao.Like, c feishu.Client, log logger.Logger, o AuthService) LikeService {
	return &LikeServiceImpl{
		dao: dao,
		c:   c,
		log: log,
		o:   o,
	}
}

func (s *LikeServiceImpl) AddLikeTask(appToken, tableId, recordID string, userID string, isLike int, action string) error {
	// 创建任务
	task := generateLikeMessage(
		&model.LikeData{
			AppToken: appToken,
			TableID:  tableId,
			RecordID: recordID,
			UserID:   userID,
			IsLike:   isLike,
			Action:   action,
		})

	// 序列化
	taskJson, err := json.Marshal(task)
	if err != nil {
		return err
	}
	return s.dao.AddPendingLikeTask(string(taskJson))
}

// GetLikeTask 获取任务
func (s *LikeServiceImpl) GetLikeTask() (*model.LikeMessage, error) {
	taskJson, err := s.dao.Pending2ProcessingTask()
	if err != nil {
		return nil, err
	}
	var task model.LikeMessage
	err = json.Unmarshal([]byte(taskJson), &task)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// HandleLikeTask 处理点赞任务
func (s *LikeServiceImpl) HandleLikeTask() {
	// 获取任务
	task, err := s.GetLikeTask()
	if err != nil {
		return
	}
	// 获取记录
	record, err := s.GetRecord(task.Data.AppToken, task.Data.TableID, task.Data.RecordID)
	if err != nil {
		s.log.Errorf("get record failed: %v", err)
		return
	}

	var likeCount, dislikeCount int

	switch task.Data.Action {
	case "add": // 添加点赞
		// 先获取
		res, _ := s.dao.GetUserLikeRecord(task.Data.RecordID, task.Data.UserID)
		switch task.Data.IsLike {
		case 1:
			// 添加点赞
			dislikeCount = record.Fields["未解决"].(int)
			if res == 0 {
				// 用户切换点赞
				dislikeCount--
			}
			count := record.Fields["已解决"].(int)
			count++
			err = s.UpdateRecord(task.Data.AppToken, task.Data.TableID, task.Data.RecordID, "已解决", likeCount, "未解决", dislikeCount)
			if err != nil {
				s.log.Errorf("update record failed: %v", err)
				s.moveTask(task)
				return
			}

		case 0:
			// 添加未解决
			likeCount = record.Fields["已解决"].(int)
			if res == 1 {
				likeCount--
			}
			dislikeCount = record.Fields["未解决"].(int)
			dislikeCount++
			err = s.UpdateRecord(task.Data.AppToken, task.Data.TableID, task.Data.RecordID, "已解决", likeCount, "未解决", dislikeCount)
			if err != nil {
				s.log.Errorf("update record failed: %v", err)
				s.moveTask(task)
				return
			}
		}

	case "remove": // 移除点赞
		switch task.Data.IsLike {
		case 1:
			dislikeCount = record.Fields["未解决"].(int)
			likeCount = record.Fields["已解决"].(int)
			likeCount--
			err = s.UpdateRecord(task.Data.AppToken, task.Data.TableID, task.Data.RecordID, "已解决", likeCount, "未解决", dislikeCount)
			if err != nil {
				s.log.Errorf("update record failed: %v", err)
				s.moveTask(task)
				return
			}

		case 0:
			likeCount = record.Fields["已解决"].(int)
			dislikeCount = record.Fields["未解决"].(int)
			dislikeCount--
			err = s.UpdateRecord(task.Data.AppToken, task.Data.TableID, task.Data.RecordID, "已解决", likeCount, "未解决", dislikeCount)
			if err != nil {
				s.log.Errorf("update record failed: %v", err)
				s.moveTask(task)
				return
			}
		}

	default:
		s.log.Errorf("invalid task action: %s", task.Data.Action)
	}

	// 任务成功
	taskJson, _ := json.Marshal(task)
	err = s.dao.AckProcessingTask(string(taskJson))
	if err != nil {
		s.log.Errorf("ack processing task failed: %v", err)
		return
	}

	// 更新用户点赞状态
	if task.Data.Action == "add" {
		err = s.dao.RecordUserLike(task.Data.RecordID, task.Data.UserID, task.Data.IsLike)
		if err != nil {
			s.log.Errorf("record user like failed: %v", err)
			return
		}
	} else if task.Data.Action == "remove" {
		err = s.dao.DeleteUserLike(task.Data.RecordID, task.Data.UserID)
		if err != nil {
			s.log.Errorf("delete user like failed: %v", err)
			return
		}
	}
}

func (s *LikeServiceImpl) GetRecord(appToken, tableId, recordID string) (*larkbitable.AppTableRecord, error) {
	// 创建请求对象
	req := larkbitable.NewBatchGetAppTableRecordReqBuilder().
		AppToken(appToken).
		TableId(tableId).
		Body(larkbitable.NewBatchGetAppTableRecordReqBodyBuilder().
			RecordIds([]string{recordID}).
			UserIdType(`open_id`).
			WithSharedUrl(true).
			AutomaticFields(true).
			Build()).
		Build()

	// 发起请求
	resp, err := s.c.GetRecordByRecordId(context.Background(), req, larkcore.WithUserAccessToken(s.o.GetAccessToken()))

	// 处理错误
	if err != nil {
		s.log.Errorf("error response: \n%v", err)
		return nil, err
	}

	// 服务端错误处理
	if !resp.Success() {
		//fmt.Printf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
		s.log.Errorf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
		return nil, fmt.Errorf("get record failed: %v", larkcore.Prettify(resp.CodeError))
	}

	// 业务处理
	//fmt.Println(larkcore.Prettify(resp))
	if len(resp.Data.Records) == 0 {
		s.log.Errorf("the record of recordId %s not found", recordID)
		return nil, fmt.Errorf("the record of recordId %s not found", recordID)
	}
	record := resp.Data.Records[0]
	return record, nil
}

func (s *LikeServiceImpl) UpdateRecord(appToken, tableId, recordID, likeKey string, likeCount int, dislikeKey string, dislikeCount int) error {
	// 创建请求对象
	req := larkbitable.NewUpdateAppTableRecordReqBuilder().
		AppToken(appToken).
		TableId(tableId).
		RecordId(recordID).
		UserIdType(`open_id`).
		AppTableRecord(larkbitable.NewAppTableRecordBuilder().
			Fields(map[string]interface{}{
				likeKey:    likeCount,
				dislikeKey: dislikeCount,
			}).
			Build()).
		Build()

	// 发起请求
	resp, err := s.c.UpdateRecord(context.Background(), req, larkcore.WithUserAccessToken(s.o.GetAccessToken()))

	// 处理错误
	if err != nil {
		s.log.Errorf("error response: \n%v", err)
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		//fmt.Printf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
		s.log.Errorf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
		return fmt.Errorf("update record failed: %v", larkcore.Prettify(resp.CodeError))
	}

	// 业务处理
	//fmt.Println(larkcore.Prettify(resp))
	return nil
}

// MoveRetry2Pending 将任务从重试队列移动到待处理队列 // 定时任务
func (s *LikeServiceImpl) MoveRetry2Pending() error {
	return s.dao.MoveRetry2Pending()
}

// tools
func generateLikeMessage(data *model.LikeData) *model.LikeMessage {
	return &model.LikeMessage{
		ID:          uuid.New().String(),
		Timestamp:   time.Now().UnixMilli(),
		Attempts:    0,
		MaxAttempts: 5,
		Data:        data,
	}
}

// 将任务迁移至延迟队列或者死信队列
func (s *LikeServiceImpl) moveTask(task *model.LikeMessage) {
	var taskJson []byte
	if task.Attempts < task.MaxAttempts {
		// 延迟队列
		task.Attempts++ // 尝试次数加1
		taskJson, _ = json.Marshal(task)
		err := s.dao.RetryProcessingTask(string(taskJson), time.Duration(math.Pow(2, float64(task.Attempts)))*time.Second) // 指数退避
		if err != nil {
			s.log.Errorf("move task to retry queue failed: %v", err)
			return
		}
	} else {
		// 死信队列
		taskJson, _ = json.Marshal(task)
		err := s.dao.MoveToDLQ(string(taskJson))
		if err != nil {
			s.log.Errorf("move task to DLQ failed: %v", err)
			return
		}
	}
}
