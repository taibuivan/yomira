// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package comic

import (
	"context"

	"github.com/taibuivan/yomira/internal/platform/validate"
	"github.com/taibuivan/yomira/pkg/slug"
	"github.com/taibuivan/yomira/pkg/uuid"
)

// # Service Layer

// Service orchestrates the business logic for the comic catalogue.
// It acts as the primary entry point for managing content metadata.
type Service struct {
	comicRepo   ComicRepository
	chapterRepo ChapterRepository
}

// NewService constructs a new [Service] with its required repositories.
func NewService(comicRepo ComicRepository, chapterRepo ChapterRepository) *Service {
	return &Service{
		comicRepo:   comicRepo,
		chapterRepo: chapterRepo,
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
	return service.comicRepo.Create(context, comic)
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
	return service.comicRepo.Update(context, comic)
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
	return service.comicRepo.SoftDelete(context, id)
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
	return service.chapterRepo.Create(context, chapter)
}

/*
ListTitles returns all alternative titles and translations for a specific comic.

Parameters:
  - context: context.Context
  - comicID: string (UUID)

Returns:
  - []*Title: List of alternative titles
  - error: Retrieval failures
*/
func (service *Service) ListTitles(context context.Context, comicID string) ([]*Title, error) {
	return service.comicRepo.ListTitles(context, comicID)
}

/*
UpsertTitle manages alternative naming for a comic.

Description: Creates a new title entry or updates an existing one
for the specified language code. This is used for multi-language
SEO and discovery.

Parameters:
  - context: context.Context
  - comicID: string (UUID)
  - langCode: string (ISO-639-1)
  - title: string (The translated/alternative title)

Returns:
  - error: Persistence error
*/
func (service *Service) UpsertTitle(context context.Context, comicID, langCode, title string) error {
	return service.comicRepo.UpsertTitle(context, comicID, langCode, title)
}

/*
DeleteTitle removes an alternative title for a specific language.

Parameters:
  - context: context.Context
  - comicID: string (UUID)
  - langCode: string (ISO-639-1)

Returns:
  - error: Removal failures
*/
func (service *Service) DeleteTitle(context context.Context, comicID, langCode string) error {
	return service.comicRepo.DeleteTitle(context, comicID, langCode)
}

/*
ListRelations retrieves all linked comics (sequels, spinoffs, etc.).

Parameters:
  - context: context.Context
  - comicID: string (UUID)

Returns:
  - []*Relation: List of related comics and their types
  - error: Storage failures
*/
func (service *Service) ListRelations(context context.Context, comicID string) ([]*Relation, error) {
	return service.comicRepo.ListRelations(context, comicID)
}

/*
AddRelation defines a link between two comic entries.

Description: Establishes a directional relationship (e.g., Prequel,
Sequel, Spinoff) between publications.

Parameters:
  - context: context.Context
  - fromID: string (Source UUID)
  - toID: string (Target UUID)
  - relType: string (Relation descriptor)

Returns:
  - error: Persistence error
*/
func (service *Service) AddRelation(context context.Context, fromID, toID, relType string) error {
	return service.comicRepo.AddRelation(context, fromID, toID, relType)
}

/*
RemoveRelation deletes a link between two comics.

Parameters:
  - context: context.Context
  - fromID: string (Source UUID)
  - toID: string (Target UUID)
  - relType: string (Relation type)

Returns:
  - error: Persistence failure
*/
func (service *Service) RemoveRelation(context context.Context, fromID, toID, relType string) error {
	return service.comicRepo.RemoveRelation(context, fromID, toID, relType)
}

// # Assets & Media

/*
ListCovers returns all available cover variants for a comic.

Parameters:
  - context: context.Context
  - comicID: string (UUID)

Returns:
  - []*Cover: List of covers
  - error: Storage failures
*/
func (service *Service) ListCovers(context context.Context, comicID string) ([]*Cover, error) {
	return service.comicRepo.ListCovers(context, comicID)
}

/*
AddCover attaches a new volume or variant cover to a comic.

Parameters:
  - context: context.Context
  - cover: *Cover

Returns:
  - error: Validation or persistence errors
*/
func (service *Service) AddCover(context context.Context, cover *Cover) error {
	if cover.ID == "" {
		cover.ID = uuid.New()
	}

	validator := &validate.Validator{}
	validator.Required(FieldComicID, cover.ComicID)
	validator.Required(FieldImageURL, cover.ImageURL).URL(FieldImageURL, cover.ImageURL)

	if err := validator.Err(); err != nil {
		return err
	}

	return service.comicRepo.AddCover(context, cover)
}

/*
DeleteCover removes a specific cover by ID.

Parameters:
  - context: context.Context
  - id: string (UUID)

Returns:
  - error: Storage failures
*/
func (service *Service) DeleteCover(context context.Context, id string) error {
	return service.comicRepo.DeleteCover(context, id)
}

/*
ListArt retrieves the gallery or fanart images for a comic.

Parameters:
  - context: context.Context
  - comicID: string (UUID)
  - onlyApproved: bool (Filter for public view)

Returns:
  - []*Art: Gallery images
  - error: Storage failures
*/
func (service *Service) ListArt(context context.Context, comicID string, onlyApproved bool) ([]*Art, error) {
	return service.comicRepo.ListArt(context, comicID, onlyApproved)
}

/*
AddArt submits a new gallery image for a comic.

Parameters:
  - context: context.Context
  - art: *Art

Returns:
  - error: Validation or persistence errors
*/
func (service *Service) AddArt(context context.Context, art *Art) error {
	if art.ID == "" {
		art.ID = uuid.New()
	}

	validator := &validate.Validator{}
	validator.Required(FieldComicID, art.ComicID)
	validator.Required(FieldUploaderID, art.UploaderID)
	validator.Required(FieldImageURL, art.ImageURL).URL(FieldImageURL, art.ImageURL)

	if err := validator.Err(); err != nil {
		return err
	}

	return service.comicRepo.AddArt(context, art)
}

/*
DeleteArt removes a gallery image.

Parameters:
  - context: context.Context
  - id: string (UUID)

Returns:
  - error: Storage failures
*/
func (service *Service) DeleteArt(context context.Context, id string) error {
	return service.comicRepo.DeleteArt(context, id)
}

/*
ApproveArt toggles the visibility of a gallery image (Moderation).

Parameters:
  - context: context.Context
  - id: string (UUID)
  - approved: bool

Returns:
  - error: Storage failures
*/
func (service *Service) ApproveArt(context context.Context, id string, approved bool) error {
	return service.comicRepo.ApproveArt(context, id, approved)
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
	return service.chapterRepo.MarkAsRead(context, chapterID, userID)
}

// # Internal Helpers

// isUUID returns true if the string matches the standard UUID length.
func isUUID(s string) bool {
	return len(s) == 36
}
