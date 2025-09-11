package dao

import (
	"context"
	"database/sql"
	"feedback/config"
	"fmt"
	"github.com/go-redis/redis/v8"
	"time"
)

type Table interface {
	CreateAppTable(appID string, tableName string, recordData map[string]interface{}) error
	UpdateAppTable(appID string, tableName string, recordData map[string]interface{}) error
	QueryAppTable(appID string, tableName string) (map[string]interface{}, error)
	DeleteAppTable(appID string, tableName string) error
}

type Like interface {
	AddPendingLikeTask(data string) error
	Pending2ProcessingTask() (string, error)
	AckProcessingTask(task string) error
	RetryProcessingTask(task string, delay time.Duration) error
	MoveToDLQ(task string) error
	RecordUserLike(recordID string, userID string, isLike int) error
	DeleteUserLike(recordID string, userID string) error
	GetUserLikeRecord(recordID string, userID string) (int, error)
	MoveRetry2Pending() error
}

func NewDB() *sql.DB {
	db, err := sql.Open("sqlite3", "test.db")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	return db
}

func NewRedisClient(cfg *config.RedisConfig) (*redis.Client, error) {
	// 创建Redis客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,     // redis地址
		Password: cfg.Password, // Redis认证密码(可选)
		DB:       cfg.DB,       // 选择的数据库
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect redis: %v", err)
	}

	return rdb, nil
}
