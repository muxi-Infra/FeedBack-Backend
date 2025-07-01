package dao

import "database/sql"

type TableDAO struct {
	DB *sql.DB
}

func NewDAO(db *sql.DB) *TableDAO {
	return &TableDAO{DB: db}
}

func (dao *TableDAO) CreateAppTable(appID string, tableName string, recordData map[string]interface{}) error {
	return nil
}

func (dao *TableDAO) UpdateAppTable(appID string, tableName string, recordData map[string]interface{}) error {
	return nil
}

func (dao *TableDAO) QueryAppTable(appID string, tableName string) (map[string]interface{}, error) {
	return nil, nil
}

func (dao *TableDAO) DeleteAppTable(appID string, tableName string) error {
	return nil
}
