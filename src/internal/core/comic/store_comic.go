// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package comic

import "context"

// # Comic Data Access

// ComicRepository defines the data access contract for the comic domain.
type ComicRepository interface {
	ComicRelationRepository
	ComicAssetRepository

	/*
		List returns a filtered, paginated slice of comics and the total count.

		Parameters:
		  - context: context.Context
		  - filter: Filter (Criteria for status, tags, search, etc.)
		  - limit: int
		  - offset: int

		Returns:
		  - []*Comic: Slice of matching publication records
		  - int: Total count of records matching the filter
		  - error: Database retrieval failures
	*/
	List(context context.Context, filter Filter, limit, offset int) ([]*Comic, int, error)

	/*
		FindByID returns the comic with the given ID.

		Parameters:
		  - context: context.Context
		  - id: string (UUID)

		Returns:
		  - *Comic: The hydrated domain entity
		  - error: ErrNotFound if missing or soft-deleted
	*/
	FindByID(context context.Context, id string) (*Comic, error)

	/*
		FindBySlug returns the comic matching the unique SEO identifier.

		Parameters:
		  - context: context.Context
		  - slug: string

		Returns:
		  - *Comic: The hydrated domain entity
		  - error: ErrNotFound if missing
	*/
	FindBySlug(context context.Context, slug string) (*Comic, error)

	/*
		Create persists a new comic to the store.

		Parameters:
		  - context: context.Context
		  - comic: *Comic (Metadata and initial state)

		Returns:
		  - error: Storage or constraint failures
	*/
	Create(context context.Context, comic *Comic) error

	/*
		Update persists changes to an existing comic's mutable fields.

		Parameters:
		  - context: context.Context
		  - comic: *Comic (Target ID and modified attributes)

		Returns:
		  - error: Storage or validation failures
	*/
	Update(context context.Context, comic *Comic) error

	/*
		SoftDelete marks a comic as deleted without physical row removal.

		Parameters:
		  - context: context.Context
		  - id: string (UUID)

		Returns:
		  - error: State update failures
	*/
	SoftDelete(context context.Context, id string) error

	/*
		IncrementViewCount atomically increments the view counter on a comic.

		Parameters:
		  - context: context.Context
		  - id: string (UUID)
		  - delta: int64 (Amount to add)

		Returns:
		  - error: Atomic jump failure
	*/
	IncrementViewCount(context context.Context, id string, delta int64) error
}
