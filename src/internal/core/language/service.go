package language

import (
	"context"
	"log/slog"
)

type Service struct {
	repo   Repository
	logger *slog.Logger
}

func NewService(repo Repository, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

func (service *Service) ListLanguages(context context.Context) ([]*Language, error) {
	return service.repo.ListLanguages(context)
}

func (service *Service) GetLanguage(context context.Context, code string) (*Language, error) {
	return service.repo.GetLanguageByCode(context, code)
}
