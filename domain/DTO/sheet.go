package DTO

type TableRecords struct {
	Records   []TableRecord
	HasMore   *bool   // 是否有更多
	PageToken *string // 分页参数
	Total     *int    // 总记录数
}

type TableRecord struct {
	Record map[string]any
}

type TableField struct {
	FieldName string
	Value     any
}

type TableConfig struct {
	TableName  string
	TableToken string
	TableID    string
	ViewID     string
}
