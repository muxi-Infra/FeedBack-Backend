package domain

type TableRecords struct {
	Records   []TableRecord
	HasMore   *bool   // 是否有更多
	PageToken *string // 分页参数
	Total     *int    // 总记录数
}

type TableRecord struct {
	RecordID *string        `json:"record_id"`
	Record   map[string]any `json:"record"`
}

type TableField struct {
	FieldName *string
	Value     any
}

type TableConfig struct {
	TableIdentity *string
	TableName     *string
	TableToken    *string
	TableID       *string
	ViewID        *string
}

// FAQTableRecords 定义多维表格记录及其解决状态的集合
type FAQTableRecords struct {
	Records []FAQTableRecord
	Total   *int // 总记录数
}

// FAQTableRecord 定义多维表格记录及其解决状态
type FAQTableRecord struct {
	RecordID   *string        `json:"record_id"`
	Record     map[string]any `json:"record"`
	IsResolved *string        `json:"is_resolved"`
}

// FAQResolution 用于处理 FAQ 记录的解决状态变更
type FAQResolution struct {
	ResolvedFieldName   *string
	UnresolvedFieldName *string
	RecordID            *string
	UserID              *string
	IsResolved          *bool
}

type File struct {
	FileToken      *string `json:"file_token"`
	TmpDownloadURL *string `json:"tmp_download_url"`
}
