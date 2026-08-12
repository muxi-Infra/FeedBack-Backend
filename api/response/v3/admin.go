package v3

import (
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
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
	Project model.FeedbackProjectV3        `json:"project"`
	Key     *model.FeedbackProjectKeyV3    `json:"key"`
	Tables  []model.FeedbackProjectTableV3 `json:"tables"`
	Scopes  map[string][]string            `json:"scopes"`
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
