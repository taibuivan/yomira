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
)

// # Titles & Relations

/*
GET /api/v1/comics/{id}/titles.

Description: Lists all alternative titles and translated names for a comic.

Request:
  - id: string (UUID)

Response:
  - 200: []Title: Success
  - 404: 404: ErrNotFound: Comic not found
*/
func (handler *Handler) listTitles(writer http.ResponseWriter, request *http.Request) {
	comicID := requestutil.ID(request, "id")

	titles, err := handler.service.ListTitles(request.Context(), comicID)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, titles)
}

/*
PUT /api/v1/comics/{id}/titles/{lang}.

Description: Adds or updates a title for a specific language code.
Used for multi-language discovery.

Request:
  - id: string (UUID)
  - lang: string (ISO-639-1 code)
  - body: { title: string }

Response:
  - 200: Message: Success
  - 400: 400: ErrInvalidJSON: Invalid body
  - 401: 401: ErrUnauthorized: Authentication required
  - 403: 403: ErrForbidden: Insufficient permissions
  - 404: 404: ErrNotFound: Comic not found
*/
func (handler *Handler) upsertTitle(writer http.ResponseWriter, request *http.Request) {
	// Extract params from URL
	comicID := requestutil.ID(request, "id")
	languageCode := requestutil.ID(request, "lang")

	// Strict JSON decoding
	var input struct {
		Title string `json:"title"`
	}

	// Decode JSON
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Domain Logic Execution
	if err := handler.service.UpsertTitle(request.Context(), comicID, languageCode, input.Title); err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Structured API Response
	respond.OK(writer, map[string]string{FieldMessage: "Title updated"})
}

/*
DELETE /api/v1/comics/{id}/titles/{lang}.

Description: Removes an alternative title for a specific language.

Request:
  - id: string (UUID)
  - lang: string

Response:
  - 204: No Content: Success
  - 401: 401: ErrUnauthorized: Authentication required
  - 403: 403: ErrForbidden: Insufficient permissions
  - 404: 404: ErrNotFound: Comic not found
*/
func (handler *Handler) deleteTitle(writer http.ResponseWriter, request *http.Request) {

	// Extract params from URL
	comicID := requestutil.ID(request, "id")
	languageCode := requestutil.ID(request, "lang")

	// Domain Logic Execution
	if err := handler.service.DeleteTitle(request.Context(), comicID, languageCode); err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Structured API Response
	respond.NoContent(writer)
}

/*
GET /api/v1/comics/{id}/relations.

Description: Lists sequels, prequels, spinoffs, and other related works for a comic.

Request:
  - id: string (UUID)

Response:
  - 200: []Relation: Success
  - 404: 404: ErrNotFound: Comic not found
*/
func (handler *Handler) listRelations(writer http.ResponseWriter, request *http.Request) {
	// Extract ID from URL
	comicID := requestutil.ID(request, "id")

	// Domain Logic Execution
	relations, err := handler.service.ListRelations(request.Context(), comicID)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Structured API Response
	respond.OK(writer, relations)
}

/*
POST /api/v1/comics/{id}/relations.

Description: Defines a directional relationship between two comics.

Request:
  - id: string (UUID)
  - body: { to_id: string, type: string }

Response:
  - 201: Message: Success
  - 400: 400: ErrInvalidJSON: Invalid body
  - 401: 401: ErrUnauthorized: Authentication required
  - 403: 403: ErrForbidden: Insufficient permissions
  - 404: 404: ErrNotFound: Comic not found
*/
func (handler *Handler) addRelation(writer http.ResponseWriter, request *http.Request) {
	// Extract ID from URL
	comicID := requestutil.ID(request, "id")

	// Strict JSON decoding
	var input struct {
		ToID string `json:"to_id"`
		Type string `json:"type"`
	}
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Domain Logic Execution
	if err := handler.service.AddRelation(request.Context(), comicID, input.ToID, input.Type); err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Structured API Response
	respond.Created(writer, map[string]string{FieldMessage: "Relation added"})
}

/*
DELETE /api/v1/comics/{id}/relations/{to}/{type}.

Description: Removes a specific relationship between two comics.

Request:
  - id: string (UUID)
  - to: string (Target UUID)
  - type: string (Relation type)

Response:
  - 204: No Content: Success
  - 401: 401: ErrUnauthorized: Authentication required
  - 403: 403: ErrForbidden: Insufficient permissions
  - 404: 404: ErrNotFound: Relation not found
*/
func (handler *Handler) removeRelation(writer http.ResponseWriter, request *http.Request) {
	comicID := requestutil.ID(request, "id")
	targetID := requestutil.ID(request, "to")
	relationType := requestutil.ID(request, "type")

	if err := handler.service.RemoveRelation(request.Context(), comicID, targetID, relationType); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.NoContent(writer)
}
