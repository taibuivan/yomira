package author

import (
	"context"
	"log/slog"

	"github.com/taibuivan/yomira/internal/platform/validate"
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

func (service *Service) ListAuthors(context context.Context, filter Filter, limit, offset int) ([]*Author, int, error) {
	return service.repo.ListAuthors(context, filter, limit, offset)
}

func (service *Service) GetAuthor(context context.Context, id int) (*Author, error) {
	return service.repo.GetAuthor(context, id)
}

func (service *Service) CreateAuthor(context context.Context, author *Author) error {
	validator := &validate.Validator{}

	validator.Required(FieldName, author.Name).MaxLen(FieldName, author.Name, 200)
	for _, alt := range author.NameAlt {
		validator.MaxLen(FieldNameAlt, alt, 200)
	}

	if author.ImageURL != nil {
		validator.URL(FieldImageURL, *author.ImageURL)
	}

	if err := validator.Err(); err != nil {
		return err
	}

	if err := service.repo.CreateAuthor(context, author); err != nil {
		return err
	}

	service.logger.Info("author_created", slog.String("name", author.Name))
	return nil
}

func (service *Service) UpdateAuthor(context context.Context, id int, author *Author) error {
	author.ID = id
	validator := &validate.Validator{}

	validator.Required(FieldName, author.Name).MaxLen(FieldName, author.Name, 200)
	if author.ImageURL != nil {
		validator.URL(FieldImageURL, *author.ImageURL)
	}

	if err := validator.Err(); err != nil {
		return err
	}

	if err := service.repo.UpdateAuthor(context, author); err != nil {
		return err
	}

	service.logger.Info("author_updated", slog.Int("author_id", author.ID))
	return nil
}

func (service *Service) DeleteAuthor(context context.Context, id int) error {
	if err := service.repo.DeleteAuthor(context, id); err != nil {
		return err
	}

	service.logger.Warn("author_deleted", slog.Int("author_id", id))
	return nil
}
