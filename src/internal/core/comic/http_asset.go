// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package comic provides the HTTP interface for discovery and management of the catalogue.

It exposes endpoints for browsing comics, retrieving chapter lists, and managing
metadata by authorised personnel.

# Routing Strategy

  - Public (v1): Discovery endpoints accessible to all visitors (GET /comics).
  - Restricted (v1): Mutative endpoints requiring Admin or Moderator roles (POST, PATCH, DELETE).

The handler translates between the web/JSON layer and the internal domain [Service].
*/
package comic

import (
	"net/http"

	requestutil "github.com/taibuivan/yomira/internal/platform/request"
	"github.com/taibuivan/yomira/internal/platform/respond"
	"github.com/taibuivan/yomira/internal/platform/validate"
)

// # Assets (Covers & Art)

/*
GET /api/v1/comics/{id}/covers.

Description: Retrieves all volume and variant covers associated with a comic.
Results are typically sorted by volume number.

Request:
  - id: string (UUID)

Response:
  - 200: []ComicCover: List of covers
  - 404: ErrNotFound: Comic not found
*/
func (handler *Handler) listCovers(writer http.ResponseWriter, request *http.Request) {
	comicID := requestutil.ID(request, "id")

	covers, err := handler.service.ListCovers(request.Context(), comicID)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, covers)
}

/*
POST /api/v1/comics/{id}/covers.

Description: Attaches a new volume or variant cover to a comic.

Request:
  - id: string (UUID)
  - body: Cover (JSON)

Response:
  - 201: Cover: Created object
  - 400: 400: ErrInvalidJSON/Validation: Input errors
  - 401: 401: ErrUnauthorized: Authentication required
  - 403: 403: ErrForbidden: Insufficient permissions
  - 404: 404: ErrNotFound: Comic not found
*/
func (handler *Handler) addCover(writer http.ResponseWriter, request *http.Request) {
	comicID := requestutil.ID(request, "id")
	var input Cover

	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	input.ComicID = comicID

	if err := handler.service.AddCover(request.Context(), &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.Created(writer, input)
}

/*
DELETE /api/v1/comics/{id}/covers/{coverID}.

Description: Deletes a specific cover record.

Request:
  - id: string (UUID)
  - coverID: string (UUID)

Response:
  - 204: No Content: Success
  - 401: 401: ErrUnauthorized: Authentication required
  - 403: 403: ErrForbidden: Insufficient permissions
  - 404: 404: ErrNotFound: Cover not found
*/
func (handler *Handler) deleteCover(writer http.ResponseWriter, request *http.Request) {
	coverID := requestutil.ID(request, "coverID")
	if err := handler.service.DeleteCover(request.Context(), coverID); err != nil {
		respond.Error(writer, request, err)
		return
	}
	respond.NoContent(writer)
}

/*
GET /api/v1/comics/{id}/art.

Description: Retrieves fanart and official gallery images for a comic.
Public view defaults to approved images only. Admins can see all submissions.

Request:
  - id: string (UUID)
  - all: bool (Toggle admin view)

Response:
  - 200: []Art: Gallery collection
  - 404: 404: ErrNotFound: Comic not found
*/
func (handler *Handler) listArt(writer http.ResponseWriter, request *http.Request) {
	comicID := requestutil.ID(request, "id")

	// Defaults to public view (approved only)
	onlyApproved := true

	// Escalate visibility for Staff
	if claims := requestutil.Claims(request); claims != nil && claims.IsAdmin() {
		onlyApproved = request.URL.Query().Get("all") != "true"
	}

	art, err := handler.service.ListArt(request.Context(), comicID, onlyApproved)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, art)
}

/*
POST /api/v1/comics/{id}/art.

Description: Submits a new gallery image for moderation.

Request (Body):
  - Art JSON object

Response:
  - 201: Art: Submission confirmation
  - 400: 400: ErrInvalidJSON/Validation: Payload errors
  - 401: 401: ErrUnauthorized: Authentication required
  - 404: 404: ErrNotFound: Comic not found
*/
func (handler *Handler) addArt(writer http.ResponseWriter, request *http.Request) {
	comicID := requestutil.ID(request, "id")

	userID, err := requestutil.RequiredUserID(request)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	var input Art
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	v := &validate.Validator{}
	v.Required("image_url", input.ImageURL).URL("image_url", input.ImageURL)

	if err := v.Err(); err != nil {
		respond.Error(writer, request, err)
		return
	}

	input.ComicID = comicID
	input.UploaderID = userID

	if err := handler.service.AddArt(request.Context(), &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.Created(writer, input)
}

/*
PATCH /api/v1/comics/{id}/art/{artID}/approve.

Description: Approves or rejects a gallery image submission (Admin only).

Request:
  - id: string (UUID)
  - artID: string (UUID)
  - body: { approved: bool }

Response:
  - 200: Message: Success
  - 400: 400: ErrInvalidJSON: Invalid payload
  - 401: 401: ErrUnauthorized: Authentication required
  - 403: 403: ErrForbidden: Insufficient permissions
  - 404: 404: ErrNotFound: Art submission not found
*/
func (handler *Handler) approveArt(writer http.ResponseWriter, request *http.Request) {
	artID := requestutil.ID(request, "artID")
	var input struct {
		Approved bool `json:"approved"`
	}
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	if err := handler.service.ApproveArt(request.Context(), artID, input.Approved); err != nil {
		respond.Error(writer, request, err)
		return
	}
	respond.OK(writer, map[string]string{FieldMessage: "Art status updated"})
}

/*
DELETE /api/v1/comics/{id}/art/{artID}.

Description: Removes a gallery image submission.

Request:
  - id: string (UUID)
  - artID: string (UUID)

Response:
  - 204: No Content: Success
  - 401: 401: ErrUnauthorized: Authentication required
  - 403: 403: ErrForbidden: Insufficient permissions
  - 404: 404: ErrNotFound: Art not found
*/
func (handler *Handler) deleteArt(writer http.ResponseWriter, request *http.Request) {
	artID := requestutil.ID(request, "artID")
	if err := handler.service.DeleteArt(request.Context(), artID); err != nil {
		respond.Error(writer, request, err)
		return
	}
	respond.NoContent(writer)
}
