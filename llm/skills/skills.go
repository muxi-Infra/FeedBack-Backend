package skills

import (
	"context"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"

	"github.com/cloudwego/eino/adk/middlewares/skill"
)

// LoadSkills 加载所有 SKILL.md
func LoadSkills(ctx context.Context, skillsDir string) (skill.Backend, error) {
	backend, err := localbk.NewBackend(ctx, &localbk.Config{})
	if err != nil {
		return nil, err
	}

	skillBackend, err := skill.NewBackendFromFilesystem(ctx, &skill.BackendFromFilesystemConfig{
		Backend: backend,
		BaseDir: skillsDir,
	})
	if err != nil {
		return nil, err
	}

	return skillBackend, nil
}
