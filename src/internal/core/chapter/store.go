// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package chapter

import "context"

// # Chapter & Page Data Access

// ChapterRepository defines the data access contract for chapters and pages.
type ChapterRepository interface {

	/*
		ListByComic returns all chapters for a comic, ordered by chapter number.

		Parameters:
		  - context: context.Context
		  - comicID: string (Owner ID)
		  - f: ChapterFilter
		  - limit: int
		  - offset: int

		Returns:
		  - []*Chapter: List of hydrated chapters
		  - int: Total matching chapters
		  - error: Storage failures
	*/
	ListByComic(context context.Context, comicID string, filter ChapterFilter, limit, offset int) ([]*Chapter, int, error)

	/*
		FindByID returns the chapter with the given ID.

		Parameters:
		  - context: context.Context
		  - id: string (UUID)

		Returns:
		  - *Chapter: Hydrated metadata
		  - error: ErrNotFound if missing
	*/
	FindByID(context context.Context, id string) (*Chapter, error)

	/*
		Create persists a new chapter to the store.

		Parameters:
		  - context: context.Context
		  - c: *Chapter

		Returns:
		  - error: Storage failure
	*/
	Create(context context.Context, chapter *Chapter) error

	/*
		Update persists changes to existing chapter metadata.

		Parameters:
		  - context: context.Context
		  - c: *Chapter

		Returns:
		  - error: Update failure
	*/
	Update(context context.Context, chapter *Chapter) error

	/*
		SoftDelete marks a chapter as deleted without physical row removal.

		Parameters:
		  - context: context.Context
		  - id: string (UUID)

		Returns:
		  - error: Removal failure
	*/
	SoftDelete(context context.Context, id string) error

	/*
		ListPages returns all pages for a Chapter ordered by page number.

		Parameters:
		  - context: context.Context
		  - chapterID: string (UUID)

		Returns:
		  - []*Page: List of image metadata
		  - error: Retrieval failure
	*/
	ListPages(context context.Context, chapterID string) ([]*Page, error)

	/*
		CreatePages bulk-inserts pages for a Chapter.

		Parameters:
		  - context: context.Context
		  - pages: []*Page

		Returns:
		  - error: Batch failure
	*/
	CreatePages(context context.Context, pages []*Page) error

	/*
		IncrementViewCount atomically increments the view counter on a chapter.

		Parameters:
		  - context: context.Context
		  - id: string (UUID)
		  - delta: int64

		Returns:
		  - error: Atomic update failure
	*/
	IncrementViewCount(context context.Context, id string, delta int64) error

	/*
		MarkAsRead records that a user has completed a chapter.

		Parameters:
		  - context: context.Context
		  - chapterID: string (Target)
		  - userID: string (Actor)

		Returns:
		  - error: Record failure
	*/
	MarkAsRead(context context.Context, chapterID, userID string) error
}
