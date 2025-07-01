package dao

import (
	"database/sql"
	"fmt"
)

type Table interface {
	CreateAppTable(appID string, tableName string, recordData map[string]interface{}) error
	UpdateAppTable(appID string, tableName string, recordData map[string]interface{}) error
	QueryAppTable(appID string, tableName string) (map[string]interface{}, error)
	DeleteAppTable(appID string, tableName string) error
}

func NewDB() *sql.DB {
	db, err := sql.Open("sqlite3", "test.db")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	return db
}
