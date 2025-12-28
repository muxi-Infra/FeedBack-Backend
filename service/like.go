package service

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/model"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/feishu"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"

	"github.com/google/uuid"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkbitable "github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
)

// 这里是 相应字段
const (
	ResolvedString   = "已解决"
	UnresolvedString = "未解决"
	ResolverInt      = 1
	UnresolvedInt    = 0
)

// 这里是行为
const (
	AddBehavior    = "add"
	RemoveBehavior = "remove"
)

type LikeService interface {
	AddLikeTask(appToken, tableId, recordID string, userID string, isLike int, action string) error
	HandleLikeTask()
	MoveRetry2Pending() error
}

type LikeServiceImpl struct {
	dao dao.Like
	c   feishu.Client
	log logger.Logger
}

func NewLikeService(dao dao.Like, c feishu.Client, log logger.Logger) LikeService {
	return &LikeServiceImpl{
		dao: dao,
		c:   c,
		log: log,
	}
}

// AddLikeTask 添加点赞任务到等待队列
// 其中 AppToken, TableID, RecordID 的作用是为了获取记录
// UserID, IsLike, Action 是用户以及用户的行为
// 举例：
// - UserID = "xxx"  IsLike=0 Action="add" 表示用户xxx给某条记录添加点赞（未解决）
// - UserID = "xxx"  IsLike=1 Action="remove" 表示用户xxx给某条记录取消点赞（已解决）
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
		return errs.SerializationError(err)
	}
	err = s.dao.AddPendingLikeTask(string(taskJson))
	if err != nil {
		return errs.AddPendingLikeTaskError(err)
	}
	return nil
}

// GetLikeTask 获取任务
func (s *LikeServiceImpl) GetLikeTask() (*model.LikeMessage, error) {
	taskJson, err := s.dao.Pending2ProcessingTask()
	if err != nil {
		return nil, errs.QueueOperationError(err)
	}

	var task model.LikeMessage
	err = json.Unmarshal([]byte(taskJson), &task)
	if err != nil {
		return nil, errs.DeserializationError(err)
	}
	return &task, nil
}

// HandleLikeTask 处理点赞任务
// 处理逻辑是：
// 1. 先从飞书中获取记录
// 2. 根据任务的行为，获取 redis 中存储的点赞状态
// 3. 根据任务的行为和 redis 存储的点赞状态，判断情况：点赞，取消点赞，切换点赞状态
// 4. 更新飞书记录，将任务放到完成队列中
// 5. 更新 redis 中点赞状态
func (s *LikeServiceImpl) HandleLikeTask() {
	// 获取任务
	task, err := s.GetLikeTask()
	if err != nil {
		return
	}
	taskJson, _ := json.Marshal(task)

	// 获取记录
	record, err := s.GetRecord(task.Data.AppToken, task.Data.TableID, task.Data.RecordID)
	if err != nil {
		s.log.Error("get record failed", logger.String("error", err.Error()))
		return
	}

	var likeCount, dislikeCount float64

	switch task.Data.Action {
	case AddBehavior:
		// 添加点赞

		// 先获取 redis 中的点赞状态
		res, _ := s.dao.GetUserLikeRecord(task.Data.RecordID, task.Data.UserID)

		switch task.Data.IsLike {
		case ResolverInt:
			// 需要已解决的点赞数目 +1

			// 记录未解决数目
			if val, ok := record.Fields[UnresolvedString].(float64); ok {
				dislikeCount = val
			}

			if res == UnresolvedInt {
				// 用户切换点赞
				dislikeCount--
			} else if res == ResolverInt {
				// 用户已经点赞
				s.log.Info("用户已经点赞-已解决",
					logger.String("user_id", task.Data.UserID),
					logger.String("record_id", task.Data.RecordID),
				)
				// ack
				err = s.dao.AckProcessingTask(string(taskJson))
				if err != nil {
					s.log.Error("ack processing task failed", logger.String("error", err.Error()))
					return
				}

				return
			}

			// 获取已解决数目
			if val, ok := record.Fields[ResolvedString].(float64); ok {
				likeCount = val
			}

			// 已解决数目 +1
			likeCount++

			// 更新记录
			err = s.UpdateRecord(task.Data.AppToken, task.Data.TableID, task.Data.RecordID, ResolvedString, likeCount, UnresolvedString, dislikeCount)
			if err != nil {
				s.log.Error("update record failed", logger.String("error", err.Error()))
				s.moveTask(task)
				return
			}

		case UnresolvedInt:
			// 需要未解决点赞数目 +1

			// 记录已解决数目
			if val, ok := record.Fields[ResolvedString].(float64); ok {
				likeCount = val
			}

			if res == ResolverInt {
				// 用户切换点赞——已解决切换至未解决
				likeCount--
			} else if res == UnresolvedInt {
				// 用户已点赞
				s.log.Info("用户已经点赞-未解决",
					logger.String("user_id", task.Data.UserID),
					logger.String("record_id", task.Data.RecordID),
				)
				// ack
				err = s.dao.AckProcessingTask(string(taskJson))
				if err != nil {
					s.log.Error("ack processing task failed", logger.String("error", err.Error()))
					return
				}

				return
			}

			if val, ok := record.Fields[UnresolvedString].(float64); ok {
				dislikeCount = val
			}

			dislikeCount++

			// 更新记录
			err = s.UpdateRecord(task.Data.AppToken, task.Data.TableID, task.Data.RecordID, ResolvedString, likeCount, UnresolvedString, dislikeCount)
			if err != nil {
				s.log.Error("update record failed", logger.String("error", err.Error()))
				s.moveTask(task)
				return
			}
		}

	case RemoveBehavior:
		// 移除点赞

		switch task.Data.IsLike {
		case ResolverInt:
			// 需要已解决点赞数目 -1

			// 记录未解决数目
			if val, ok := record.Fields[UnresolvedString].(float64); ok {
				dislikeCount = val
			}

			// 获取已解决数目
			if val, ok := record.Fields[ResolvedString].(float64); ok {
				likeCount = val
			}

			likeCount--

			// 更新记录
			err = s.UpdateRecord(task.Data.AppToken, task.Data.TableID, task.Data.RecordID, ResolvedString, likeCount, UnresolvedString, dislikeCount)
			if err != nil {
				s.log.Error("update record failed", logger.String("error", err.Error()))
				s.moveTask(task)
				return
			}

		case 0:
			// 需要未解决点赞数目 -1

			// 记录已解决数目
			if val, ok := record.Fields[ResolvedString].(float64); ok {
				likeCount = val
			}

			// 获取未解决数目
			if val, ok := record.Fields[UnresolvedString].(float64); ok {
				dislikeCount = val
			}

			dislikeCount--

			err = s.UpdateRecord(task.Data.AppToken, task.Data.TableID, task.Data.RecordID, ResolvedString, likeCount, UnresolvedString, dislikeCount)
			if err != nil {
				s.log.Error("update record failed", logger.String("error", err.Error()))
				s.moveTask(task)
				return
			}
		}

	default:
		s.log.Error("invalid task action", logger.String("action", task.Data.Action))
	}

	// 任务成功
	taskJson, _ = json.Marshal(task)
	err = s.dao.AckProcessingTask(string(taskJson))
	if err != nil {
		s.log.Error("ack processing task failed", logger.String("error", err.Error()))
		return
	}

	// 更新用户点赞状态
	if task.Data.Action == AddBehavior {
		err = s.dao.RecordUserLike(task.Data.RecordID, task.Data.UserID, task.Data.IsLike)
		s.log.Error("record user like failed", logger.String("error", err.Error()))
		return

	} else if task.Data.Action == RemoveBehavior {
		err = s.dao.DeleteUserLike(task.Data.RecordID, task.Data.UserID)
		if err != nil {
			s.log.Error("delete user like failed", logger.String("error", err.Error()))
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
	resp, err := s.c.GetRecordByRecordId(context.Background(), req)

	// 处理错误
	if err != nil {
		s.log.Error("get record by record id failed", logger.String("error", err.Error()))
		return nil, errs.FeishuRequestError(err)
	}

	// 服务端错误处理
	if !resp.Success() {
		s.log.Error("get record failed",
			logger.String("logId", resp.RequestId()),
			logger.String("error", larkcore.Prettify(resp.CodeError)),
		)
		return nil, errs.FeishuResponseError(err)
	}

	// 业务处理
	if len(resp.Data.Records) == 0 {
		s.log.Error("record not found", logger.String("record_id", recordID))
		return nil, errs.RecordNotFoundError(err)
	}
	record := resp.Data.Records[0]
	return record, nil
}

func (s *LikeServiceImpl) UpdateRecord(appToken, tableId, recordID, likeKey string, likeCount float64, dislikeKey string, dislikeCount float64) error {
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
	resp, err := s.c.UpdateRecord(context.Background(), req)

	// 处理错误
	if err != nil {
		s.log.Error("update record failed", logger.String("error", err.Error()))
		return errs.FeishuRequestError(err)
	}

	// 服务端错误处理
	if !resp.Success() {
		s.log.Error("update record failed",
			logger.String("logId", resp.RequestId()),
			logger.String("error", larkcore.Prettify(resp.CodeError)),
		)
		return errs.FeishuResponseError(err)
	}

	// 业务处理
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
		// 迁移至延迟队列
		// 尝试次数加1
		task.Attempts++
		taskJson, _ = json.Marshal(task)
		err := s.dao.RetryProcessingTask(string(taskJson), time.Duration(math.Pow(2, float64(task.Attempts)))*time.Second)
		if err != nil {
			s.log.Error("move task to retry queue failed", logger.String("error", err.Error()))
			return
		}
	} else {
		// 死信队列
		taskJson, _ = json.Marshal(task)
		err := s.dao.MoveToDLQ(string(taskJson))
		if err != nil {
			s.log.Error("move task to DLQ failed", logger.String("error", err.Error()))
			return
		}
	}
}
