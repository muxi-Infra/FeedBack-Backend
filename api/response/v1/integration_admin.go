package v1

import "time"

type ProjectResponse struct {
	ID          uint64    `json:"id"`
	ProjectID   string    `json:"project_id"`
	ProjectName string    `json:"project_name"`
	School      string    `json:"school"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ProjectKeyResponse struct {
	ID        uint64     `json:"id"`
	ProjectID string     `json:"project_id"`
	KeyID     string     `json:"key_id"`
	Issuer    string     `json:"issuer"`
	Status    string     `json:"status"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type ProjectTableResponse struct {
	ID            uint64   `json:"id"`
	ProjectID     string   `json:"project_id"`
	TableIdentity string   `json:"table_identity"`
	TableName     string   `json:"table_name"`
	TableID       string   `json:"table_id"`
	ViewID        string   `json:"view_id"`
	TableType     string   `json:"table_type"`
	Notice        bool     `json:"notice"`
	Status        string   `json:"status"`
	Scopes        []string `json:"scopes"`
}

type ProjectConfigResponse struct {
	Project *ProjectResponse       `json:"project"`
	Keys    []ProjectKeyResponse   `json:"keys"`
	Tables  []ProjectTableResponse `json:"tables"`
}
