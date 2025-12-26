package dao

import (
	"database/sql"
	"fmt"
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
