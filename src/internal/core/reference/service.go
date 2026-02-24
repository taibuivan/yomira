// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package reference

import (
	"context"

	"github.com/taibuivan/yomira/internal/platform/validate"
)

// # Service Layer

// Service orchestrates business rules for reference data.
//
// It acts as the gateway for managing the foundation entities (Languages, Tags)
// and the contributor profiles (Authors, Artists) that populate the catalogue.
type Service struct {
	repo Repository
}

// NewService constructs a new reference [Service].
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// # Language Methods

/*
ListLanguages returns a complete list of all supported languages.

Parameters:
  - context: context.Context

Returns:
  - []*Language: List of hydrated language entities
  - error: Database retrieval failures
*/
func (service *Service) ListLanguages(context context.Context) ([]*Language, error) {
	return service.repo.ListLanguages(context)
}

/*
GetLanguage retrieves a specific language by its ISO-639-1 code.

Parameters:
  - context: context.Context
  - code: string (ISO code)

Returns:
  - *Language: Hydrated language entity
  - error: Not found or storage errors
*/
func (service *Service) GetLanguage(context context.Context, code string) (*Language, error) {
	return service.repo.GetLanguageByCode(context, code)
}

// # Tag Methods

/*
ListTags returns all available content tags organized by their group hierarchy.

Parameters:
  - context: context.Context

Returns:
  - []*TagGroup: Nested tag groups with their child tags
  - error: Retrieval failures
*/
func (service *Service) ListTags(context context.Context) ([]*TagGroup, error) {
	return service.repo.ListTags(context)
}

/*
GetTag retrieves a specific tag by its primary integer identifier.

Parameters:
  - context: context.Context
  - id: int

Returns:
  - *Tag: Hydrated tag entity
  - error: apperr.NotFound or database errors
*/
func (service *Service) GetTag(context context.Context, id int) (*Tag, error) {
	return service.repo.GetTagByID(context, id)
}

/*
GetTagBySlug resolves a human-readable URL slug back into a target tag.

Parameters:
  - context: context.Context
  - slug: string

Returns:
  - *Tag: Resolved tag entity
  - error: Execution failures
*/
func (service *Service) GetTagBySlug(context context.Context, slug string) (*Tag, error) {
	return service.repo.GetTagBySlug(context, slug)
}

// # Author Methods

/*
ListAuthors provides a paginated search for writers in the catalog.

Parameters:
  - context: context.Context
  - filter: ContributorFilter (Search query)
  - limit: int
  - offset: int

Returns:
  - []*Author: Hydrated list of matches
  - int: Total record count for pagination
  - error: Retrieval errors
*/
func (service *Service) ListAuthors(context context.Context, filter ContributorFilter, limit, offset int) ([]*Author, int, error) {
	return service.repo.ListAuthors(context, filter, limit, offset)
}

/*
GetAuthor retrieves detailed structural metadata for a specific author.

Parameters:
  - context: context.Context
  - id: int

Returns:
  - *Author: Target entity
  - error: Not found or execution errors
*/
func (service *Service) GetAuthor(context context.Context, id int) (*Author, error) {
	return service.repo.GetAuthor(context, id)
}

/*
CreateAuthor validates business constraints before persisting a new author record.

Description: Enforces uniqueness and data integrity rules for contributor names
and metadata before storage.

Parameters:
  - context: context.Context
  - author: *Author

Returns:
  - error: Validation failures (apperr.Invalid) or storage errors
*/
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

	return service.repo.CreateAuthor(context, author)
}

/*
UpdateAuthor applies metadata changes to an existing author record.

Parameters:
  - context: context.Context
  - id: int
  - author: *Author

Returns:
  - error: Validation or execution failures
*/
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

	return service.repo.UpdateAuthor(context, author)
}

/*
DeleteAuthor flags the author record as deleted in the system.

Parameters:
  - context: context.Context
  - id: int

Returns:
  - error: Side-effect failures
*/
func (service *Service) DeleteAuthor(context context.Context, id int) error {
	return service.repo.DeleteAuthor(context, id)
}

// # Artist Methods

/*
ListArtists returns a paginated search for illustrators in the catalog.

Parameters:
  - context: context.Context
  - filter: ContributorFilter
  - limit: int
  - offset: int

Returns:
  - []*Artist: Match results
  - int: Total record count
  - error: Resolution errors
*/
func (service *Service) ListArtists(context context.Context, filter ContributorFilter, limit, offset int) ([]*Artist, int, error) {
	return service.repo.ListArtists(context, filter, limit, offset)
}

/*
GetArtist retrieves an illustrator record by its primary identifier.

Parameters:
  - context: context.Context
  - id: int

Returns:
  - *Artist: Hydrated entity
  - error: Failures
*/
func (service *Service) GetArtist(context context.Context, id int) (*Artist, error) {
	return service.repo.GetArtist(context, id)
}

/*
CreateArtist validates business constraints before persisting an illustrator profile.

Parameters:
  - context: context.Context
  - artist: *Artist

Returns:
  - error: Validation or storage failures
*/
func (service *Service) CreateArtist(context context.Context, artist *Artist) error {
	validator := &validate.Validator{}
	validator.Required(FieldName, artist.Name).MaxLen(FieldName, artist.Name, 200)

	if artist.ImageURL != nil {
		validator.URL(FieldImageURL, *artist.ImageURL)
	}

	if err := validator.Err(); err != nil {
		return err
	}

	return service.repo.CreateArtist(context, artist)
}

/*
UpdateArtist modifies an existing illustrator record in persistent storage.

Parameters:
  - context: context.Context
  - id: int
  - artist: *Artist

Returns:
  - error: Update failures
*/
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
	return service.repo.UpdateArtist(context, artist)
}

/*
DeleteArtist flags the illustrator record as deleted.

Parameters:
  - context: context.Context
  - id: int

Returns:
  - error: Execution errors
*/
func (service *Service) DeleteArtist(context context.Context, id int) error {
	return service.repo.DeleteArtist(context, id)
}
