// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package comic

import (
	"context"

	"github.com/taibuivan/yomira/internal/platform/validate"
	"github.com/taibuivan/yomira/pkg/uuid"
)

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
