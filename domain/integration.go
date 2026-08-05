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
