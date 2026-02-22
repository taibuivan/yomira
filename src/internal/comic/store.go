// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package comic

import "context"

// ComicRepository defines the data access contract for the comic domain.
//
// # Architecture
//
// Implementations live in internal/storage/postgres — this interface lives in
// the domain package because the service layer (the consumer) defines what it needs.
// [ComicFilter] it is used to pass filter parameters to the List method.
type ComicRepository interface {
	// List returns a filtered, paginated slice of comics and the total count.
	//
	// Returns:
	//   - []*Comic: The list of comics matching the filter.
	//   - int: Total count for pagination.
	//   - error: Database or connection errors.
	List(ctx context.Context, f ComicFilter, limit, offset int) ([]*Comic, int, error)

	// FindByID returns the comic with the given ID.
	//
	// It returns ErrNotFound if the comic is absent or soft-deleted.
	FindByID(ctx context.Context, id string) (*Comic, error)

	// FindBySlug returns the comic with the given slug.
	//
	// It returns ErrNotFound if no match is found.
	FindBySlug(ctx context.Context, slug string) (*Comic, error)

	// Create persists a new comic to the store.
	//
	// The caller is responsible for generating and setting the ID and Slug
	// before calling this method.
	Create(ctx context.Context, c *Comic) error

	// Update persists changes to an existing comic's mutable fields.
	Update(ctx context.Context, c *Comic) error

	// SoftDelete marks a comic as deleted without removing the row.
	SoftDelete(ctx context.Context, id string) error

	// IncrementViewCount atomically increments the view counter on a comic.
	//
	// Parameters:
	//   - id: The unique identifier of the comic.
	//   - delta: The amount to add to the current view count (e.g., 1).
	IncrementViewCount(ctx context.Context, id string, delta int64) error
}

// ChapterRepository defines the data access contract for chapters and pages.
//
// # Consistency
//
// Pages are managed through this same repository because they are always
// accessed in the context of a [Chapter] and never independently (Aggregate Pattern).
type ChapterRepository interface {
	// ListByComic returns all chapters for a comic, ordered by chapter number.
	//
	// Returns:
	//   - []*Chapter: The list of chapters found.
	//   - int: The total count of chapters (useful for pagination).
	//   - error: Any database-level errors.
	ListByComic(ctx context.Context, comicID string, f ChapterFilter, limit, offset int) ([]*Chapter, int, error)

	// FindByID returns the chapter with the given ID.
	//
	// It returns ErrNotFound if no chapter exists with that ID.
	FindByID(ctx context.Context, id string) (*Chapter, error)

	// Create persists a new chapter to the store.
	Create(ctx context.Context, c *Chapter) error

	// Update persists changes to existing chapter metadata.
	Update(ctx context.Context, c *Chapter) error

	// SoftDelete marks a chapter as deleted without removing the row.
	SoftDelete(ctx context.Context, id string) error

	// ListPages returns all pages for a [Chapter] ordered by page number.
	ListPages(ctx context.Context, chapterID string) ([]*Page, error)

	// CreatePages bulk-inserts pages for a [Chapter].
	// This is typically used by the crawler or during manual uploads.
	CreatePages(ctx context.Context, pages []*Page) error
}
