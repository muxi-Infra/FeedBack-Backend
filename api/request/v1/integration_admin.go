package v1

import "time"

type RegisterProjectReq struct {
	ProjectID   string                    `json:"project_id" binding:"required"`
	ProjectName string                    `json:"project_name" binding:"required"`
	School      string                    `json:"school" binding:"required"`
	Status      string                    `json:"status"`
	Key         RegisterProjectKeyReq     `json:"key" binding:"required"`
	Tables      []RegisterProjectTableReq `json:"tables" binding:"required,min=1"`
}

type RegisterProjectKeyReq struct {
	KeyID     string     `json:"key_id" binding:"required"`
	Issuer    string     `json:"issuer" binding:"required"`
	PublicKey string     `json:"public_key" binding:"required"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type RegisterProjectTableReq struct {
	TableIdentity string   `json:"table_identity" binding:"required"`
	TableName     string   `json:"table_name" binding:"required"`
	TableToken    string   `json:"table_token" binding:"required"`
	TableID       string   `json:"table_id" binding:"required"`
	ViewID        string   `json:"view_id" binding:"required"`
	TableType     string   `json:"table_type"`
	Notice        bool     `json:"notice"`
	Scopes        []string `json:"scopes" binding:"required,min=1"`
}

type UpdateProjectReq struct {
	ProjectName string `json:"project_name"`
	School      string `json:"school"`
	Status      string `json:"status"`
}
