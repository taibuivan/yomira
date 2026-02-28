// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package comic

import (
	"context"
)

// # Titles & Relations

/*
ListTitles returns all alternative titles and translations for a specific comic.
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
