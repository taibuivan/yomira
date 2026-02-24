// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package reference

import "context"

// # Reference Data Access

// Repository defines the data access contract for reference and master data.
type Repository interface {

	// ## Language Data Access

	/*
		ListLanguages retrieves all supported linguistic locales.

		Parameters:
		  - context: context.Context

		Returns:
		  - []*Language: Collection of localized metadata
		  - error: Database retrieval failures
	*/
	ListLanguages(context context.Context) ([]*Language, error)

	/*
		GetLanguageByCode fetches a single locale by its ISO-639-1 code.

		Parameters:
		  - context: context.Context
		  - code: string (ISO-639-1)

		Returns:
		  - *Language: The hydrated locale entity
		  - error: ErrNotFound if missing
	*/
	GetLanguageByCode(context context.Context, code string) (*Language, error)

	// ## Tag Data Access

	/*
		ListTags retrieves all tag groups and their nested tags.

		Parameters:
		  - context: context.Context

		Returns:
		  - []*TagGroup: Hierarchical collection of tags organized by group
		  - error: Database retrieval failures
	*/
	ListTags(context context.Context) ([]*TagGroup, error)

	/*
		GetTagByID retrieves a specific tag by its primary key.

		Parameters:
		  - context: context.Context
		  - id: int identifier

		Returns:
		  - *Tag: Hydrated tag entity with its parent group
		  - error: Database retrieval or not found errors
	*/
	GetTagByID(context context.Context, id int) (*Tag, error)

	/*
		GetTagBySlug retrieves a tag using its URL identifier.

		Parameters:
		  - context: context.Context
		  - slug: string semantic identifier

		Returns:
		  - *Tag: Hydrated tag entity
		  - error: Retrieval failures
	*/
	GetTagBySlug(context context.Context, slug string) (*Tag, error)

	// ## Author Data Access

	/*
		ListAuthors retrieves a filtered and paginated list of catalog authors.

		Parameters:
		  - context: context.Context
		  - f: ContributorFilter (Search parameters)
		  - limit, offset: int (Pagination bounds)

		Returns:
		  - []*Author: Paginated matching results
		  - int: Total matching count for pagination metadata
		  - error: Database execution errors
	*/
	ListAuthors(context context.Context, f ContributorFilter, limit, offset int) ([]*Author, int, error)

	/*
		GetAuthor retrieves a single author profile by its primary key.

		Parameters:
		  - context: context.Context
		  - id: int identifier

		Returns:
		  - *Author: Hydrated profile entity
		  - error: SQL or not found errors
	*/
	GetAuthor(context context.Context, id int) (*Author, error)

	// CreateAuthor persists a new author record.
	CreateAuthor(context context.Context, a *Author) error

	// UpdateAuthor applies modifications to an existing author record.
	UpdateAuthor(context context.Context, a *Author) error

	// DeleteAuthor flags an author as logically destroyed.
	DeleteAuthor(context context.Context, id int) error

	// ## Artist Data Access

	/*
		ListArtists retrieves a filtered and paginated list of catalogue illustrators.

		Parameters:
		  - context: context.Context
		  - f: ContributorFilter
		  - limit, offset: int

		Returns:
		  - []*Artist: Collection of artists
		  - int: Total matching count
		  - error: Database retrieval failures
	*/
	ListArtists(context context.Context, f ContributorFilter, limit, offset int) ([]*Artist, int, error)

	/*
		GetArtist retrieves an illustrator profile by ID.

		Parameters:
		  - context: context.Context
		  - id: int secondary key

		Returns:
		  - *Artist: Domain mapping entity
		  - error: Postgres execution or mapping failures
	*/
	GetArtist(context context.Context, id int) (*Artist, error)

	// CreateArtist persists a new illustrator profile.
	CreateArtist(context context.Context, a *Artist) error

	// UpdateArtist modifies an active artist record.
	UpdateArtist(context context.Context, a *Artist) error

	// DeleteArtist flags an illustrator profile as logically deleted.
	DeleteArtist(context context.Context, id int) error
}
