package domain

import "time"

type RegisterProjectInput struct {
	ProjectID   string
	ProjectName string
	School      string
	Status      string
	Key         ProjectKeyInput
	Tables      []ProjectTableInput
}

type ProjectKeyInput struct {
	KeyID     string
	Issuer    string
	APIKey    string
	ExpiresAt *time.Time
}

type ProjectTableInput struct {
	TableIdentity string
	TableName     string
	TableToken    string
	TableID       string
	ViewID        string
	TableType     string
	Notice        bool
	Scopes        []string
}

type UpdateProjectInput struct {
	ProjectName string
	School      string
	Status      string
}

// UpdateProjectConfigInput 用于全量更新项目基本信息、API Key、飞书表配置和 Scope。
type UpdateProjectConfigInput struct {
	ProjectID   string
	ProjectName string
	School      string
	Status      string
	Key         ProjectKeyInput
	Tables      []ProjectTableInput
}

type ProjectSummary struct {
	ID          uint64
	ProjectID   string
	ProjectName string
	School      string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ProjectKeySummary struct {
	ID        uint64
	ProjectID string
	KeyID     string
	Issuer    string
	APIKey    string
	Status    string
	ExpiresAt *time.Time
}

type ExchangeIntegrationTokenInput struct {
	ProjectID     string
	KeyID         string
	StudentID     string
	TableIdentity string
	Timestamp     int64
	Nonce         string
	Signature     string
}

type ProjectTableConfig struct {
	ID            uint64
	ProjectID     string
	TableIdentity string
	TableName     string
	TableToken    string
	TableID       string
	ViewID        string
	TableType     string
	Notice        bool
	Status        string
	Scopes        []string
}

type ProjectConfig struct {
	Project *ProjectSummary
	Keys    []ProjectKeySummary
	Tables  []ProjectTableConfig
}
