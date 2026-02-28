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
package chapter

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/taibuivan/yomira/internal/platform/middleware"
	requestutil "github.com/taibuivan/yomira/internal/platform/request"
	"github.com/taibuivan/yomira/internal/platform/respond"
	"github.com/taibuivan/yomira/internal/platform/sec"
	"github.com/taibuivan/yomira/internal/platform/validate"
	"github.com/taibuivan/yomira/pkg/pagination"
)

const (
	FieldItems   = "items"
	FieldTotal   = "total"
	FieldMessage = "message"
)

// # Handler Implementation

// Handler implements the HTTP layer for chapter management.
type Handler struct {
	service *Service
}

// NewHandler constructs a new chapter [Handler].
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes attaches chapter and page-related endpoints to the root API router.
// Chapter endpoints span both /comics/{id}/... and /chapters/... prefixes.
func (handler *Handler) RegisterRoutes(api chi.Router) {
	// Discovery endpoints
	api.Get("/comics/{comicID}/chapters", handler.ListChapters)

	// Admin protected endpoints
	api.Group(func(admin chi.Router) {
		admin.Use(middleware.RequireRole(sec.RoleAdmin))
		admin.Post("/comics/{comicID}/chapters", handler.CreateChapter)
	})

	// User interactions (Require authentication)
	api.Group(func(user chi.Router) {
		user.Use(middleware.RequireAuth)
		user.Post("/chapters/{id}/read", handler.MarkAsRead)
	})
}

// # Chapter Retrieval

/*
GET /api/v1/comics/{comicID}/chapters.

Description: Returns a paginated roster of chapters for a specific comic.

Request:
  - comicID: string (UUID)
  - lang: string (Filter by language code)
  - dir: string (asc, desc)
  - limit: int
  - page: int

Response:
  - 200: []Chapter: Paginated list
  - 404: 404: ErrNotFound: Comic not found
*/
func (handler *Handler) ListChapters(writer http.ResponseWriter, request *http.Request) {
	comicID := requestutil.ID(request, "comicID")

	paginationParams := pagination.FromRequest(request)

	filter := ChapterFilter{
		Language: request.URL.Query().Get("lang"),
		SortDir:  request.URL.Query().Get("dir"),
	}

	chapters, total, err := handler.service.ListChapters(request.Context(), comicID, filter, paginationParams.Limit, paginationParams.Offset())
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, map[string]any{
		FieldItems: chapters,
		FieldTotal: total,
	})
}

// # Chapter Creation

// createChapterRequest defines the inbound JSON schema for individual uploads.
type createChapterRequest struct {
	Number   float64 `json:"number"`
	Title    string  `json:"title"`
	Language string  `json:"language"`
}

/*
POST /api/v1/comics/{comicID}/chapters.

Description: Creates a new chapter record for a comic.

Request:
  - comicID: string (UUID)
  - body: createChapterRequest

Response:
  - 201: Chapter: Created chapter object
  - 400: 400: ErrInvalidJSON/Validation: Invalid payload
  - 401: 401: ErrUnauthorized: Authentication required
  - 403: 403: ErrForbidden: Insufficient permissions
*/
func (handler *Handler) CreateChapter(writer http.ResponseWriter, request *http.Request) {
	comicID := requestutil.ID(request, "comicID")

	var input createChapterRequest
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	v := &validate.Validator{}
	v.Required("language", input.Language)
	v.Custom("number", input.Number < 0, "Chapter number cannot be negative")

	if err := v.Err(); err != nil {
		respond.Error(writer, request, err)
		return
	}

	chapterDto := &Chapter{
		ComicID:  comicID,
		Number:   input.Number,
		Title:    input.Title,
		Language: input.Language,
	}

	if err := handler.service.CreateChapter(request.Context(), chapterDto); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.Created(writer, chapterDto)
}

// # Reader Interaction

/*
POST /api/v1/comics/chapters/{id}/read.

Description: Records that the current user has completed reading a chapter.
Used for synchronising reading progress across devices.

Request:
  - id: string (Chapter UUID)

Response:
  - 200: Message: Success
  - 401: 401: ErrUnauthorized: Login required to track reading progress
  - 404: 404: ErrNotFound: Chapter not found
*/
func (handler *Handler) MarkAsRead(writer http.ResponseWriter, request *http.Request) {
	chapterID := requestutil.ID(request, "id")

	userID, err := requestutil.RequiredUserID(request)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	if err := handler.service.MarkChapterAsRead(request.Context(), chapterID, userID); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, map[string]string{FieldMessage: "Chapter marked as read"})
}
