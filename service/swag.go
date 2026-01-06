package service

import (
	"os"
	"os/exec"

	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
)

type SwagService interface {
	GenerateOpenAPI() ([]byte, error)
}

type SwagServiceImpl struct {
	log logger.Logger
}

func NewSwagService(log logger.Logger) SwagService {
	return &SwagServiceImpl{
		log: log,
	}
}

func (s *SwagServiceImpl) GenerateOpenAPI() ([]byte, error) {
	cmd := exec.Command("make", "swag")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, errs.SwagMakeFailureError(err)
	}

	filepath := "docs/openapi3.yaml"
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, errs.SwagOpenFailureError(err)
	}

	return content, nil
}
