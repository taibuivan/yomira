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
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/taibuivan/yomira/internal/platform/middleware"
	requestutil "github.com/taibuivan/yomira/internal/platform/request"
	"github.com/taibuivan/yomira/internal/platform/respond"
	"github.com/taibuivan/yomira/internal/platform/sec"
	"github.com/taibuivan/yomira/internal/platform/validate"
	"github.com/taibuivan/yomira/pkg/pagination"
)

// # Handler Implementation

// Handler implements the HTTP layer for comic management and discovery.
// It translates web requests into domain service calls.
type Handler struct {
	service *Service
}

// NewHandler constructs a new comic [Handler] with its service dependency.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes returns a [chi.Router] configured with the comic domain's endpoints.
//
// # Routing Strategy
//
//   - Discovery (Public): Accessible by all visitors for browsing.
//   - Management (Restricted): Requires [RoleAdmin] for state-mutating operations.
func (handler *Handler) Routes() chi.Router {
	router := chi.NewRouter()

	// ## Public Discovery Endpoints
	router.Get("/", handler.listComics)
	router.Get("/{identifier}", handler.getComic)
	router.Get("/{id}/titles", handler.listTitles)
	router.Get("/{id}/relations", handler.listRelations)

	// ## Content Management (Admin Protected)
	router.Group(func(admin chi.Router) {
		admin.Use(middleware.RequireRole(sec.RoleAdmin))

		admin.Post("/", handler.createComic)
		admin.Patch("/{id}", handler.updateComic)
		admin.Delete("/{id}", handler.deleteComic)

		// Covers
		admin.Post("/{id}/covers", handler.addCover)
		admin.Delete("/{id}/covers/{coverID}", handler.deleteCover)

		// Art Gallery
		admin.Post("/{id}/art", handler.addArt)
		admin.Patch("/{id}/art/{artID}/approve", handler.approveArt)
		admin.Delete("/{id}/art/{artID}", handler.deleteArt)

		// Titles
		admin.Put("/{id}/titles/{lang}", handler.upsertTitle)
		admin.Delete("/{id}/titles/{lang}", handler.deleteTitle)

		// Relations
		admin.Post("/{id}/relations", handler.addRelation)
		admin.Delete("/{id}/relations/{to}/{type}", handler.removeRelation)
	})

	// ## Public Assets
	router.Get("/{id}/covers", handler.listCovers)
	router.Get("/{id}/art", handler.listArt)

	return router
}

// # Comic Endpoints

/*
GET /api/v1/comics.

Description: Retrieves a paginated list of comics from the catalogue.
Supports complex filtering by status, rating, tags, and full-text search.

Request:
  - q: string (Full-text search)
  - status: []string (ongoing, completed, hiatus, cancelled)
  - contentrating: string (safe, suggestive, explicit)
  - demographic: []string (shounen, shoujo, seinen, josei)
  - originlanguage: []string (e.g., ja, ko, zh)
  - year: int
  - includedtags: []int
  - excludedtags: []int
  - sort: string (latest, popular, rating, alphabetic)
  - dir: string (asc, desc)
  - limit: int
  - page: int

Response:
  - 200: []Comic: Paginated list of comics
*/
func (handler *Handler) listComics(writer http.ResponseWriter, request *http.Request) {
	paginationParams := pagination.FromRequest(request)
	queryParams := request.URL.Query()

	filter := Filter{
		Query:             queryParams.Get("q"),
		Status:            parseStatusSlice(queryParams["status"]),
		ContentRating:     parseRatingSlice(queryParams.Get("contentrating")),
		Demographic:       parseDemographicSlice(queryParams["demographic"]),
		OriginLanguage:    queryParams["originlanguage"],
		IncludedTags:      parseIntSlice(queryParams["includedtags"]),
		ExcludedTags:      parseIntSlice(queryParams["excludedtags"]),
		IncludedAuthors:   parseIntSlice(queryParams["includedauthors"]),
		IncludedArtists:   parseIntSlice(queryParams["includedartists"]),
		AvailableLanguage: queryParams.Get("availablelanguage"),
		Sort:              queryParams.Get("sort"),
		SortDir:           queryParams.Get("dir"),
	}

	if year, err := strconv.Atoi(queryParams.Get("year")); err == nil {
		y := int16(year)
		filter.Year = &y
	}

	comics, total, err := handler.service.ListComics(request.Context(), filter, paginationParams.Limit, paginationParams.Offset())
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.Paginated(writer, comics, pagination.NewMeta(paginationParams.Page, paginationParams.Limit, total))
}

/*
GET /api/v1/comics/{identifier}.

Description: Retrieves detailed metadata for a comic using either its UUID or unique title slug.
UUID lookups take precedence.

Request:
  - identifier: string (UUID or Slug)

Response:
  - 200: Comic: Success
  - 400: 400: ErrInvalidID: Invalid identifier format
  - 404: 404: ErrNotFound: Comic not found
*/
func (handler *Handler) getComic(writer http.ResponseWriter, request *http.Request) {
	identifier := requestutil.ID(request, "identifier")

	comic, err := handler.service.GetComic(request.Context(), identifier)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, comic)
}

// # Request Payloads

// # Request Payloads

// createComicRequest defines the inbound JSON schema for comic creation.
type createComicRequest struct {
	Title           string            `json:"title"`
	TitleAlt        []string          `json:"title_alt"`
	Synopsis        string            `json:"synopsis"`
	Status          Status            `json:"status"`
	ContentRating   ContentRating     `json:"content_rating"`
	Demographic     Demographic       `json:"demographic"`
	DefaultReadMode ReadMode          `json:"default_read_mode"`
	OriginLanguage  string            `json:"origin_language"`
	Year            *int16            `json:"year"`
	Links           map[string]string `json:"links"`
	AuthorIDs       []int             `json:"author_ids"`
	ArtistIDs       []int             `json:"artist_ids"`
	TagIDs          []int             `json:"tag_ids"`
}

// # Mutation Endpoints

/*
POST /api/v1/comics.

Description: Creates a new comic entry in the catalogue.
Slugs are auto-generated from the title if not provided.

Request (Body):
  - createComicRequest: JSON object

Response:
  - 201: Comic: Created comic object
  - 400: 400: ErrInvalidJSON/Validation: Invalid input data
  - 401: 401: ErrUnauthorized: Missing or invalid token
  - 403: 403: ErrForbidden: Insufficient permissions
*/
func (handler *Handler) createComic(writer http.ResponseWriter, request *http.Request) {
	var input createComicRequest

	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	v := &validate.Validator{}
	v.Required("title", input.Title).MaxLen("title", input.Title, 500).
		Required("status", string(input.Status)).
		Required("content_rating", string(input.ContentRating)).
		Required("origin_language", input.OriginLanguage)

	if err := v.Err(); err != nil {
		respond.Error(writer, request, err)
		return
	}

	comicDto := &Comic{
		Title:           input.Title,
		TitleAlt:        input.TitleAlt,
		Synopsis:        input.Synopsis,
		Status:          input.Status,
		ContentRating:   input.ContentRating,
		Demographic:     input.Demographic,
		DefaultReadMode: input.DefaultReadMode,
		OriginLanguage:  input.OriginLanguage,
		Year:            input.Year,
		Links:           input.Links,
		AuthorIDs:       input.AuthorIDs,
		ArtistIDs:       input.ArtistIDs,
		TagIDs:          input.TagIDs,
	}

	if err := handler.service.CreateComic(request.Context(), comicDto); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.Created(writer, comicDto)
}

/*
PATCH /api/v1/comics/{id}.

Description: Applies partial updates to an existing comic record.
Clients should only provide the fields that need to be changed.

Request:
  - id: string (UUID)
  - body: createComicRequest (Partial JSON)

Response:
  - 200: Comic: Updated comic object
  - 400: 400: ErrInvalidJSON/Validation: Invalid input data
  - 401: 401: ErrUnauthorized: Missing or invalid token
  - 403: 403: ErrForbidden: Insufficient permissions
  - 404: 404: ErrNotFound: Comic not found
*/
func (handler *Handler) updateComic(writer http.ResponseWriter, request *http.Request) {
	comicID := requestutil.ID(request, "id")

	var input createComicRequest
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	comicDto := &Comic{
		ID:              comicID,
		Title:           input.Title,
		TitleAlt:        input.TitleAlt,
		Synopsis:        input.Synopsis,
		Status:          input.Status,
		ContentRating:   input.ContentRating,
		Demographic:     input.Demographic,
		DefaultReadMode: input.DefaultReadMode,
		OriginLanguage:  input.OriginLanguage,
		Year:            input.Year,
		Links:           input.Links,
		AuthorIDs:       input.AuthorIDs,
		ArtistIDs:       input.ArtistIDs,
		TagIDs:          input.TagIDs,
	}

	if err := handler.service.UpdateComic(request.Context(), comicDto); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, comicDto)
}

/*
DELETE /api/v1/comics/{id}.

Description: Performs a soft-delete of the comic record.
Deleted records are hidden from discovery but remain in the database for auditing.

Request:
  - id: string (UUID)

Response:
  - 204: No Content: Success
  - 401: 401: ErrUnauthorized: Missing or invalid token
  - 403: 403: ErrForbidden: Insufficient permissions
  - 404: 404: ErrNotFound: Comic not found
*/
func (handler *Handler) deleteComic(writer http.ResponseWriter, request *http.Request) {
	comicID := requestutil.ID(request, "id")

	if err := handler.service.DeleteComic(request.Context(), comicID); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.NoContent(writer)
}

// # Helpers

/*
parseIntSlice converts a slice of strings to a slice of integers.

Parameters:
  - values: A slice of strings to convert.

Returns:
  - A slice of integers.
*/
func parseIntSlice(values []string) []int {
	var result []int
	for _, value := range values {
		if integer, err := strconv.Atoi(value); err == nil {
			result = append(result, integer)
		}
	}
	return result
}

/*
parseStatusSlice converts a slice of strings to a slice of Status.

Parameters:
  - values: A slice of strings to convert.

Returns:
  - A slice of Status.
*/
func parseStatusSlice(values []string) []Status {
	var result []Status
	for _, value := range values {
		status := Status(value)
		if status.IsValid() {
			result = append(result, status)
		}
	}
	return result
}

/*
parseRatingSlice converts a string of comma-separated values to a slice of ContentRating.

Parameters:
  - value: A string of comma-separated values.

Returns:
  - A slice of ContentRating.
*/
func parseRatingSlice(value string) []ContentRating {
	if value == "" {
		return nil
	}
	var result []ContentRating
	for _, segment := range strings.Split(value, ",") {
		rating := ContentRating(strings.TrimSpace(segment))
		result = append(result, rating)
	}
	return result
}

/*
parseDemographicSlice converts a slice of strings to a slice of Demographic.

Parameters:
  - values: A slice of strings to convert.

Returns:
  - A slice of Demographic.
*/
func parseDemographicSlice(values []string) []Demographic {
	var result []Demographic
	for _, value := range values {
		result = append(result, Demographic(value))
	}
	return result
}
