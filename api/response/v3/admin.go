package v3

import (
	"time"
)

type RegisterProjectResp struct {
	ProjectID string `json:"project_id"`
	KeyID     string `json:"key_id"`
	APIKey    string `json:"api_key"`
}

type RotateAPIKeyResp struct {
	KeyID  string `json:"key_id"`
	APIKey string `json:"api_key"`
}

// ProjectConfigResp 管理员查询项目配置的响应。
type ProjectConfigResp struct {
	Project ProjectDetail       `json:"project"`
	Key     *ProjectKeyDetail   `json:"key"`
	Tables  []ProjectTableItem  `json:"tables"`
	Scopes  map[string][]string `json:"scopes"`
}

// ProjectDetail 项目详情
type ProjectDetail struct {
	ID          uint64    `json:"id"`
	ProjectID   string    `json:"project_id"`
	ProjectName string    `json:"project_name"`
	School      string    `json:"school"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProjectKeyDetail 项目 API Key 的元数据；API Key 明文仅会在创建或重新生成时返回一次。
type ProjectKeyDetail struct {
	ID        uint64     `json:"id"`
	ProjectID string     `json:"project_id"`
	KeyID     string     `json:"key_id"`
	Status    string     `json:"status"`
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// ProjectTableItem 项目绑定的飞书表格配置。
type ProjectTableItem struct {
	ID            uint64    `json:"id"`
	ProjectID     string    `json:"project_id"`
	TableIdentity string    `json:"table_identity"`
	TableName     string    `json:"table_name"`
	TableToken    string    `json:"table_token"`
	TableID       string    `json:"table_id"`
	ViewID        string    `json:"view_id"`
	TableType     string    `json:"table_type"`
	Notice        bool      `json:"notice"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ProjectListItem 项目列表项
type ProjectListItem struct {
	ID          uint64    `json:"id"`
	ProjectID   string    `json:"project_id"`
	ProjectName string    `json:"project_name"`
	School      string    `json:"school"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProjectListResp 项目列表的游标分页响应。
type ProjectListResp struct {
	Projects  []ProjectListItem `json:"projects"`
	HasMore   bool              `json:"has_more"`
	PageToken string            `json:"page_token"`
}
