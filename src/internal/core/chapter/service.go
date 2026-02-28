// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package chapter

import (
	"context"
	"log/slog"

	"github.com/taibuivan/yomira/internal/platform/validate"
	"github.com/taibuivan/yomira/pkg/uuid"
)

const (
	FieldComicID       = "comic_id"
	FieldLanguage      = "language"
	FieldChapterNumber = "number"
)

// # Service Layer

// Service orchestrates the business logic for chapters.
type Service struct {
	chapterRepo ChapterRepository
	logger      *slog.Logger
}

// NewService constructs a new [Service] with its required repositories.
func NewService(chapterRepo ChapterRepository, logger *slog.Logger) *Service {
	return &Service{
		chapterRepo: chapterRepo,
		logger:      logger,
	}
}

// # Chapter Operations

/*
ListChapters retrieves serialisation data for a comic.

Parameters:
  - context: context.Context
  - comicID: string (Owner ID)
  - f: ChapterFilter (Language and sorting options)
  - limit: int
  - offset: int

Returns:
  - []*Chapter: Metadata for matched chapters
  - int: Total chapter count for the given comic/filter
  - error: Storage or execution errors
*/
func (service *Service) ListChapters(context context.Context, comicID string, f ChapterFilter, limit, offset int) ([]*Chapter, int, error) {
	return service.chapterRepo.ListByComic(context, comicID, f, limit, offset)
}

/*
GetChapter retrieves metadata for a single chapter by its ID.

Parameters:
  - context: context.Context
  - id: string (UUID)

Returns:
  - *Chapter: The hydrated domain entity
  - error: ErrNotFound if not found
*/
func (service *Service) GetChapter(context context.Context, id string) (*Chapter, error) {
	return service.chapterRepo.FindByID(context, id)
}

/*
CreateChapter initialises a new chapter entry.

Description: Ensures the chapter is linked to a valid comic,
applies basic sanity checks on chapter numbering, and persists
the metadata.

Parameters:
  - context: context.Context
  - chapter: *Chapter (The new chapter data)

Returns:
  - error: Validation or persistence errors
*/
func (service *Service) CreateChapter(context context.Context, chapter *Chapter) error {

	// Identity & Mandatory field generation
	if chapter.ID == "" {
		chapter.ID = uuid.New()
	}

	// Business attribute validation
	validator := &validate.Validator{}
	validator.Required(FieldComicID, chapter.ComicID)
	validator.Required(FieldLanguage, chapter.Language)

	// Negative chapter numbers are logically invalid for serialisation
	validator.Custom(FieldChapterNumber, chapter.Number < 0, "Chapter number cannot be negative")

	if err := validator.Err(); err != nil {
		return err
	}

	// Storage persistence
	if err := service.chapterRepo.Create(context, chapter); err != nil {
		return err
	}

	service.logger.Info("chapter_created",
		slog.String("chapter_id", chapter.ID),
		slog.String("comic_id", chapter.ComicID),
		slog.Float64("number", chapter.Number),
	)

	return nil
}

// # Reader Interactions

/*
MarkChapterAsRead records a user's progress for a specific chapter.

Parameters:
  - context: context.Context
  - chapterID: string (UUID)
  - userID: string (UUID)

Returns:
  - error: Persistence failures
*/
func (service *Service) MarkChapterAsRead(context context.Context, chapterID, userID string) error {
	if err := service.chapterRepo.MarkAsRead(context, chapterID, userID); err != nil {
		return err
	}

	service.logger.Info("chapter_marked_as_read",
		slog.String("chapter_id", chapterID),
		slog.String("user_id", userID),
	)

	return nil
}

// # Internal Helpers

// isUUID returns true if the string matches the standard UUID length.
func isUUID(s string) bool {
	return len(s) == 36
}
