package artist

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

func (service *Service) ListArtists(context context.Context, filter Filter, limit, offset int) ([]*Artist, int, error) {
	return service.repo.ListArtists(context, filter, limit, offset)
}

func (service *Service) GetArtist(context context.Context, id int) (*Artist, error) {
	return service.repo.GetArtist(context, id)
}

func (service *Service) CreateArtist(context context.Context, artist *Artist) error {
	validator := &validate.Validator{}
	validator.Required(FieldName, artist.Name).MaxLen(FieldName, artist.Name, 200)

	if artist.ImageURL != nil {
		validator.URL(FieldImageURL, *artist.ImageURL)
	}

	if err := validator.Err(); err != nil {
		return err
	}

	if err := service.repo.CreateArtist(context, artist); err != nil {
		return err
	}

	service.logger.Info("artist_created", slog.String("name", artist.Name))
	return nil
}

func (service *Service) UpdateArtist(context context.Context, id int, artist *Artist) error {
	artist.ID = id
	validator := &validate.Validator{}
	validator.Required(FieldName, artist.Name).MaxLen(FieldName, artist.Name, 200)

	if artist.ImageURL != nil {
		validator.URL(FieldImageURL, *artist.ImageURL)
	}

	if err := validator.Err(); err != nil {
		return err
	}
	if err := service.repo.UpdateArtist(context, artist); err != nil {
		return err
	}

	service.logger.Info("artist_updated", slog.Int("artist_id", artist.ID))
	return nil
}

func (service *Service) DeleteArtist(context context.Context, id int) error {
	if err := service.repo.DeleteArtist(context, id); err != nil {
		return err
	}

	service.logger.Warn("artist_deleted", slog.Int("artist_id", id))
	return nil
}
