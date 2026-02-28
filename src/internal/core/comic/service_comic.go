// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package comic

import (
	"context"
	"log/slog"

	"github.com/taibuivan/yomira/internal/platform/validate"
	"github.com/taibuivan/yomira/pkg/slug"
	"github.com/taibuivan/yomira/pkg/uuid"
)

// # Service Layer

// Service orchestrates the business logic for the comic catalogue.
// It acts as the primary entry point for managing content metadata.
type Service struct {
	comicRepo ComicRepository
	logger    *slog.Logger
}

// NewService constructs a new [Service] with its required repositories.
func NewService(comicRepo ComicRepository, logger *slog.Logger) *Service {
	return &Service{
		comicRepo: comicRepo,
		logger:    logger,
	}
}

// # Comic Lookups

/*
ListComics retrieves a paginated and filtered collection of comics.

Description: This method orchestrates the discovery phase of the catalogue.
It passes filter criteria directly to the repository layer for efficient
database-level filtering and sorting.

Parameters:
  - context: context.Context
  - filter: Filter (Criteria for status, tags, search, etc.)
  - limit: int (Max records to return)
  - offset: int (Pagination cursor)

Returns:
  - []*Comic: Slice of matching publication records
  - int: Total count of records matching the filter (for pagination metadata)
  - error: System or repository level errors
*/
func (service *Service) ListComics(context context.Context, filter Filter, limit, offset int) ([]*Comic, int, error) {
	return service.comicRepo.List(context, filter, limit, offset)
}

/*
GetComic fetches a single publication record by UUID or SEO Slug.

Description: The service intelligently determines the lookup strategy.
If the identifier matches the UUID format, it performs a primary key
lookup; otherwise, it resolves via the unique URL slug.

Parameters:
  - context: context.Context
  - identifier: string (UUID or Slug)

Returns:
  - *Comic: The hydrated domain entity
  - error: ErrNotFound if no match is found
*/
func (service *Service) GetComic(context context.Context, identifier string) (*Comic, error) {

	// Identity format detection
	if isUUID(identifier) {
		return service.comicRepo.FindByID(context, identifier)
	}

	// Slug resolution
	return service.comicRepo.FindBySlug(context, identifier)
}

// # Comic Management

/*
CreateComic initialises a new publication record in the system.

Description: Performs deep business validation on the metadata,
generates a stable UUID v7 identity, and creates SEO-friendly
slugs before persisting to the repository.

Parameters:
  - context: context.Context
  - comic: *Comic (The entity to be persisted)

Returns:
  - error: Validation or persistence errors
*/
func (service *Service) CreateComic(context context.Context, comic *Comic) error {

	// Business attribute validation
	validator := &validate.Validator{}
	validator.Required(FieldTitle, comic.Title).MaxLen(FieldTitle, comic.Title, 500)

	// Lifecycle state validation
	validator.Required(FieldStatus, string(comic.Status)).OneOf(FieldStatus, string(comic.Status),
		string(StatusOngoing),
		string(StatusCompleted),
		string(StatusHiatus),
		string(StatusCancelled),
	)

	// Audience rating audit
	validator.Required(FieldContentRating, string(comic.ContentRating)).OneOf(FieldContentRating, string(comic.ContentRating),
		string(ContentRatingSafe),
		string(ContentRatingSuggestive),
		string(ContentRatingExplicit),
	)

	// Identity & Slug generation
	if comic.ID == "" {
		comic.ID = uuid.New()
	}

	// Slug generation
	if comic.Slug == "" {
		comic.Slug = slug.From(comic.Title)
	}

	// Return validation errors if any constraints failed
	if err := validator.Err(); err != nil {
		return err
	}

	// Persistence via Repository
	if err := service.comicRepo.Create(context, comic); err != nil {
		return err
	}

	service.logger.Info("comic_created",
		slog.String("comic_id", comic.ID),
		slog.String("title", comic.Title),
	)

	return nil
}

/*
UpdateComic applies modifications to an existing publication.

Description: Supports partial updates. Non-empty fields in the
input entity will overwrite existing values. Enforces business
rules on the updated attributes.

Parameters:
  - context: context.Context
  - c: *Comic (Updated attributes)

Returns:
  - error: Validation or persistence errors
*/
func (service *Service) UpdateComic(context context.Context, comic *Comic) error {

	// Integrity validation for updated fields
	validator := &validate.Validator{}

	// Business attribute validation
	if comic.Title != "" {
		validator.MaxLen(FieldTitle, comic.Title, 500)
	}

	// Slug generation
	if comic.Slug != "" {
		validator.Slug(FieldSlug, comic.Slug)
	}

	// Lifecycle state validation
	if comic.Status != "" {
		validator.OneOf(FieldStatus, string(comic.Status),
			string(StatusOngoing),
			string(StatusCompleted),
			string(StatusHiatus),
			string(StatusCancelled),
		)
	}

	// Return validation errors if any constraints failed
	if err := validator.Err(); err != nil {
		return err
	}

	// Execute storage update
	if err := service.comicRepo.Update(context, comic); err != nil {
		return err
	}

	service.logger.Info("comic_updated", slog.String("comic_id", comic.ID))

	return nil
}

/*
DeleteComic removes a comic from active discovery.

Description: Implements soft-delete logic. The record remains
in the database but its visibility status is flipped to hidden.

Parameters:
  - context: context.Context
  - id: string (UUID)

Returns:
  - error: Persistence error if removal fails
*/
func (service *Service) DeleteComic(context context.Context, id string) error {
	if err := service.comicRepo.SoftDelete(context, id); err != nil {
		return err
	}

	service.logger.Warn("comic_deleted", slog.String("comic_id", id))

	return nil
}

// # Internal Helpers

// isUUID returns true if the string matches the standard UUID length.
func isUUID(s string) bool {
	return len(s) == 36
}
