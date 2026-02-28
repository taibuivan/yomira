// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package comic

import "context"

type ComicAssetRepository interface {
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
