package v3

// RegisterProjectReq 注册一个接入反馈中台的校园项目。
type RegisterProjectReq struct {
	ProjectName string             `json:"project_name" binding:"required"`
	School      string             `json:"school" binding:"required"`
	Tables      []RegisterTableReq `json:"tables" binding:"required,min=1"`
}

type RegisterTableReq struct {
	TableIdentity string   `json:"table_identity" binding:"required"`
	TableName     string   `json:"table_name" binding:"required"`
	TableToken    string   `json:"table_token" binding:"required"`
	TableID       string   `json:"table_id" binding:"required"`
	ViewID        string   `json:"view_id" binding:"required"`
	TableType     string   `json:"table_type" binding:"required"`
	Notice        bool     `json:"notice"`
	Scopes        []string `json:"scopes" binding:"required,min=1"`
}

// SyncProjectReq 管理员同步指定项目的数据。
type SyncProjectReq struct {
	ProjectID string `json:"project_id" binding:"required"`
}

// SyncProjectUserReq 管理员强制同步指定项目和学生的数据。
type SyncProjectUserReq struct {
	ProjectID string `json:"project_id" binding:"required"`
	StudentID string `json:"student_id" binding:"required"`
}
