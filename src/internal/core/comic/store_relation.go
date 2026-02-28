// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package comic

import "context"

type ComicRelationRepository interface {
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
}
