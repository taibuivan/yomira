// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package comic

import "context"

// # Comic Data Access

// ComicRepository defines the data access contract for the comic domain.
type ComicRepository interface {

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

	// # Sub-resource Management

	/*
		ListTitles returns all alternative titles for a comic.

		Parameters:
		  - context: context.Context
		  - comicID: string (UUID)

		Returns:
		  - []*Title: Collection of localised metadata
		  - error: Retrieval failures
	*/
	ListTitles(context context.Context, comicID string) ([]*Title, error)

	/*
		UpsertTitle creates or updates a title for a specific language.

		Parameters:
		  - context: context.Context
		  - comicID: string (UUID)
		  - langCode: string (ISO-639-1)
		  - title: string (Alternative name)

		Returns:
		  - error: Persistence failure
	*/
	UpsertTitle(context context.Context, comicID, langCode, title string) error

	/*
		DeleteTitle removes a title for a specific language.

		Parameters:
		  - context: context.Context
		  - comicID: string (UUID)
		  - langCode: string (ISO-639-1)

		Returns:
		  - error: Removal failure
	*/
	DeleteTitle(context context.Context, comicID, langCode string) error

	/*
		ListRelations returns all linked comics.

		Parameters:
		  - context: context.Context
		  - comicID: string (UUID)

		Returns:
		  - []*Relation: List of prequels, sequels, spinoffs
		  - error: Retrieval failure
	*/
	ListRelations(context context.Context, comicID string) ([]*Relation, error)

	/*
		AddRelation creates a link between two comics.

		Parameters:
		  - context: context.Context
		  - fromID: string (Source UUID)
		  - toID: string (Target UUID)
		  - relType: string (Type of link)

		Returns:
		  - error: Mapping failure
	*/
	AddRelation(context context.Context, fromID, toID, relType string) error

	/*
		RemoveRelation removes a link between comics.

		Parameters:
		  - context: context.Context
		  - fromID: string (Source)
		  - toID: string (Target)
		  - relType: string (Type)

		Returns:
		  - error: Deletion failure
	*/
	RemoveRelation(context context.Context, fromID, toID, relType string) error

	// # Assets & Media

	/*
		ListCovers returns all available cover variants for a comic.

		Parameters:
		  - context: context.Context
		  - comicID: string (UUID)

		Returns:
		  - []*Cover: Volume or variant covers
		  - error: Storage failures
	*/
	ListCovers(context context.Context, comicID string) ([]*Cover, error)

	/*
		AddCover attaches a new volume or variant cover.

		Parameters:
		  - context: context.Context
		  - cover: *Cover

		Returns:
		  - error: Upload/Persistence failures
	*/
	AddCover(context context.Context, cover *Cover) error

	/*
		DeleteCover removes a specific cover by ID.

		Parameters:
		  - context: context.Context
		  - id: string (UUID)

		Returns:
		  - error: Removal failures
	*/
	DeleteCover(context context.Context, id string) error

	/*
		ListArt returns the fanart/gallery images for a comic.

		Parameters:
		  - context: context.Context
		  - comicID: string (UUID)
		  - onlyApproved: bool (Filter for public views)

		Returns:
		  - []*Art: Gallery images metadata
		  - error: Storage failures
	*/
	ListArt(context context.Context, comicID string, onlyApproved bool) ([]*Art, error)

	/*
		AddArt persists a new gallery image.

		Parameters:
		  - context: context.Context
		  - art: *Art

		Returns:
		  - error: Submission failure
	*/
	AddArt(context context.Context, art *Art) error

	/*
		DeleteArt removes a gallery image.

		Parameters:
		  - context: context.Context
		  - id: string (UUID)

		Returns:
		  - error: Removal failure
	*/
	DeleteArt(context context.Context, id string) error

	/*
		ApproveArt toggles the visibility of a gallery image.

		Parameters:
		  - context: context.Context
		  - id: string (UUID)
		  - approved: bool (Moderation status)

		Returns:
		  - error: State jump failure
	*/
	ApproveArt(context context.Context, id string, approved bool) error
}

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
