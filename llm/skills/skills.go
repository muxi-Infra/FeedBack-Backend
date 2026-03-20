package skills

import (
	"os"
	"path/filepath"
	"strings"
)

// LoadSkills 加载所有 skill.md
func LoadSkills(skillsDir string) (string, error) {
	var builder strings.Builder

	builder.WriteString("\n\n# Available Skills\n")

	err := filepath.Walk(skillsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只读取 skill.md
		if !info.IsDir() && info.Name() == "skill.md" {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			builder.WriteString("\n\n=== SKILL START ===\n")
			builder.WriteString(string(data))
			builder.WriteString("\n=== SKILL END ===\n")
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	return builder.String(), nil
}
