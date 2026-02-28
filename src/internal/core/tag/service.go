package tag

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

func (service *Service) ListTags(context context.Context) ([]*TagGroup, error) {
	return service.repo.ListTags(context)
}

func (service *Service) GetTag(context context.Context, id int) (*Tag, error) {
	return service.repo.GetTagByID(context, id)
}

func (service *Service) GetTagBySlug(context context.Context, slug string) (*Tag, error) {
	return service.repo.GetTagBySlug(context, slug)
}
