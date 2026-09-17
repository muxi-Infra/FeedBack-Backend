package domain

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"os"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
)

type ConfigActorV3 struct {
	AdminID   uint64
	RequestID string
}
type ConfigInstanceV3 struct{ ID string }

func NewConfigInstanceV3() *ConfigInstanceV3 {
	host, _ := os.Hostname()
	if len(host) > 40 {
		host = host[:40]
	}
	return &ConfigInstanceV3{ID: fmt.Sprintf("%s-%d-%s", host, os.Getpid(), uuid.NewString())}
}

type ConfigReceiptV3 struct {
	Version  uint64
	ChangeID string
}
type ConfigEventV3 struct {
	ProjectID string
	Version   uint64
	ChangeID  string
	Kind      string
	ChangedAt time.Time
	MessageID string
}
type ProjectSnapshotV3 struct {
	Project model.FeedbackProjectV3
	Tables  []model.FeedbackProjectTableV3
	Scopes  map[string][]string
}

func (s ProjectSnapshotV3) Active() bool {
	return s.Project.ID != 0 && s.Project.DeletedAt == 0 && s.Project.Status == "active"
}

var ErrConfigVersionBehind = errors.New("configuration snapshot is below the required version")

// ConfigErrorClass never includes driver messages or configuration values.
func ConfigErrorClass(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, ErrConfigVersionBehind) {
		return "version_behind"
	}
	return "dependency"
}
