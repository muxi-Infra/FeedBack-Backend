package dao

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	pendingQueue    = "like_task_pending_queue"    // 待处理队列
	processingQueue = "like_task_processing_queue" // 处理中队列
	retryQueue      = "like_task_retry_queue"      // 重试队列
	dlqQueue        = "like_task_dlq_queue"        // 死信队列
	userLikeHash    = "user_like_record"           // 用户点赞记录
)

type LikeDAO struct {
	client *redis.Client
}

func NewLike(client *redis.Client) Like {
	return &LikeDAO{client: client}
}

// AddPendingLikeTask 向待处理队列中添加点赞任务 需要 record_id user_id(学号) 使用 0 和 1 来表示 未解决 已解决
func (dao *LikeDAO) AddPendingLikeTask(data string) error {
	return dao.client.LPush(context.Background(), pendingQueue, data).Err()
}

// Pending2ProcessingTask 待处理任务 -> 处理中队列
// 这样做是为了避免处理任务时出现异常导致任务丢失
func (dao *LikeDAO) Pending2ProcessingTask() (string, error) {
	task, err := dao.client.BRPopLPush(context.Background(),
		pendingQueue, processingQueue, 0).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil
		}
		return "", err
	}
	return task, nil
}

// AckProcessingTask 确认任务执行完成，删除处理中队列中的任务
func (dao *LikeDAO) AckProcessingTask(task string) error {
	return dao.client.LRem(context.Background(), processingQueue, 1, task).Err()
}

// RetryProcessingTask 任务失败 -> retry 队列中
// 使用 ZSET 实现延迟队列，分数是 当前时间 + 延迟时间
func (dao *LikeDAO) RetryProcessingTask(task string, delay time.Duration) error {
	return dao.client.ZAdd(context.Background(), retryQueue, &redis.Z{
		Score:  float64(time.Now().Add(delay).Unix()),
		Member: task,
	}).Err()
}

// MoveToDLQ 移动任务到死信队列
func (dao *LikeDAO) MoveToDLQ(task string) error {
	return dao.client.LPush(context.Background(), dlqQueue, task).Err()
}

// RecordUserLike 记录用户点赞
func (dao *LikeDAO) RecordUserLike(recordID string, userID string, isLike int) error {
	key := recordID + "_" + userID
	return dao.client.HSet(context.Background(), userLikeHash, key, isLike).Err()
}

// DeleteUserLike 删除用户点赞情况
func (dao *LikeDAO) DeleteUserLike(recordID string, userID string) error {
	key := recordID + "_" + userID
	return dao.client.HDel(context.Background(), userLikeHash, key).Err()
}

// GetUserLikeRecord 获取用户点赞情况
func (dao *LikeDAO) GetUserLikeRecord(recordID string, userID string) (int, error) {
	key := recordID + "_" + userID
	res, err := dao.client.HGet(context.Background(), userLikeHash, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return -1, nil // 用户没有点赞
		}
		return -1, err
	}

	val, err := strconv.Atoi(res)
	if err != nil {
		return -1, err
	}
	return val, nil
}

func (dao *LikeDAO) MoveRetry2Pending() error {
	now := time.Now().Unix()
	tasks, err := dao.client.ZRangeByScore(context.Background(), retryQueue, &redis.ZRangeBy{
		Min: "-inf",
		Max: strconv.Itoa(int(now)),
	}).Result()
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		return nil
	}

	// 使用 watch 保证原子性
	ctx := context.Background()
	for _, task := range tasks {
		err = dao.client.Watch(ctx, func(tx *redis.Tx) error {
			_, err = tx.TxPipelined(ctx, func(pipeliner redis.Pipeliner) error {
				pipeliner.LPush(ctx, pendingQueue, task)
				pipeliner.ZRem(ctx, retryQueue, task)
				return nil
			})
			return err
		}, retryQueue)
		if err != nil {
			// 跳过该任务并继续
			fmt.Println("move retry task to pending error: ", err)
			continue
		}
	}
	return err
}
