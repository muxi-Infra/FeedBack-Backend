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
	PublicKey string
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
	Status    string
	ExpiresAt *time.Time
}

type ProjectTableConfig struct {
	ID            uint64
	ProjectID     string
	TableIdentity string
	TableName     string
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
