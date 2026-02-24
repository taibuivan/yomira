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
	"github.com/taibuivan/yomira/internal/platform/apperr"
	"github.com/taibuivan/yomira/internal/platform/middleware"
	requestutil "github.com/taibuivan/yomira/internal/platform/request"
	"github.com/taibuivan/yomira/internal/platform/respond"
	"github.com/taibuivan/yomira/internal/platform/sec"
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
	router.Get("/{comicID}/chapters", handler.listChapters)
	router.Get("/{id}/titles", handler.listTitles)
	router.Get("/{id}/relations", handler.listRelations)

	// ## Content Management (Admin Protected)
	router.Group(func(admin chi.Router) {
		admin.Use(middleware.RequireRole(sec.RoleAdmin))

		admin.Post("/", handler.createComic)
		admin.Patch("/{id}", handler.updateComic)
		admin.Delete("/{id}", handler.deleteComic)

		admin.Post("/{comicID}/chapters", handler.createChapter)

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

	// ## User Interactions
	router.Post("/chapters/{id}/read", handler.markAsRead)

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
	// Pagination extraction using pkg/pagination
	paginationParams := pagination.FromRequest(request)

	// Query filter assembly
	queryParams := request.URL.Query()

	// Filter
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

	// Get year from query
	if year, err := strconv.Atoi(queryParams.Get("year")); err == nil {
		y := int16(year)
		filter.Year = &y
	}

	// Domain Logic Execution
	comics, total, err := handler.service.ListComics(request.Context(), filter, paginationParams.Limit, paginationParams.Offset())
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Structured API Response
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
	// Extract identifier from URL
	identifier := requestutil.ID(request, "identifier")

	// Domain Logic Execution
	comic, err := handler.service.GetComic(request.Context(), identifier)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Structured API Response
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

	// Strict JSON decoding
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Map DTO to Domain Entity
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

	// Domain Logic Execution
	if err := handler.service.CreateComic(request.Context(), comicDto); err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Structured API Response
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
	// Extract ID from URL
	comicID := requestutil.ID(request, "id")

	// Strict JSON decoding
	var input createComicRequest
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Map DTO to Domain Entity
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

	// Domain Logic Execution
	if err := handler.service.UpdateComic(request.Context(), comicDto); err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Structured API Response
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
	// Extract ID from URL
	comicID := requestutil.ID(request, "id")

	// Domain Logic Execution
	if err := handler.service.DeleteComic(request.Context(), comicID); err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Structured API Response
	respond.NoContent(writer)
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
func (handler *Handler) listChapters(writer http.ResponseWriter, request *http.Request) {
	// Extract comic ID from URL
	comicID := requestutil.ID(request, "comicID")

	// Pagination extraction using pkg/pagination
	paginationParams := pagination.FromRequest(request)

	// Build filter
	filter := ChapterFilter{
		Language: request.URL.Query().Get("lang"),
		SortDir:  request.URL.Query().Get("dir"),
	}

	// Domain Logic Execution
	chapters, total, err := handler.service.ListChapters(request.Context(), comicID, filter, paginationParams.Limit, paginationParams.Offset())
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Structured API Response
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
func (handler *Handler) createChapter(writer http.ResponseWriter, request *http.Request) {
	// Extract comic ID from URL
	comicID := requestutil.ID(request, "comicID")

	// Strict JSON decoding
	var input createChapterRequest
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Map DTO to Domain Entity
	chapterDto := &Chapter{
		ComicID:  comicID,
		Number:   input.Number,
		Title:    input.Title,
		Language: input.Language,
	}

	// Domain Logic Execution
	if err := handler.service.CreateChapter(request.Context(), chapterDto); err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Structured API Response
	respond.Created(writer, chapterDto)
}

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
	// Extract ID from URL
	comicID := requestutil.ID(request, "id")

	// Domain Logic Execution
	titles, err := handler.service.ListTitles(request.Context(), comicID)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Structured API Response
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
	// Extract params from URL
	comicID := requestutil.ID(request, "id")
	targetID := requestutil.ID(request, "to")
	relationType := requestutil.ID(request, "type")

	// Domain Logic Execution
	if err := handler.service.RemoveRelation(request.Context(), comicID, targetID, relationType); err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Structured API Response
	respond.NoContent(writer)
}

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

	// Variable Extraction
	comicID := requestutil.ID(request, "id")

	// Domain Logic
	covers, err := handler.service.ListCovers(request.Context(), comicID)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Payload Delivery
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

	// Initialisation
	comicID := requestutil.ID(request, "id")
	var input Cover

	// Request Parsing
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Attribute Alignment
	input.ComicID = comicID

	// Service Invocation
	if err := handler.service.AddCover(request.Context(), &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Success Response
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

	// Endpoint Context
	comicID := requestutil.ID(request, "id")

	// Moderation Awareness
	// Defaults to public view (approved only)
	onlyApproved := true

	// Escalate visibility for Staff
	if claims := requestutil.Claims(request); claims != nil && claims.IsAdmin() {
		onlyApproved = request.URL.Query().Get("all") != "true"
	}

	// Logic Execution
	art, err := handler.service.ListArt(request.Context(), comicID, onlyApproved)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Final Response
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

	// Parameters
	comicID := requestutil.ID(request, "id")

	// Identity Verification
	claims := requestutil.Claims(request)
	if claims == nil {
		respond.Error(writer, request, apperr.Unauthorized("Login required to submit art"))
		return
	}

	// Payload Mapping
	userID := claims.UserID
	var input Art
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Normalisation
	input.ComicID = comicID
	input.UploaderID = userID

	// Persistence Call
	if err := handler.service.AddArt(request.Context(), &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Standard Delivery
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
func (handler *Handler) markAsRead(writer http.ResponseWriter, request *http.Request) {

	// Variable targets
	chapterID := requestutil.ID(request, "id")

	// Session Validation
	claims := requestutil.Claims(request)
	if claims == nil {
		respond.Error(writer, request, apperr.Unauthorized("Login required to track reading progress"))
		return
	}

	// Map identities
	userID := claims.UserID

	// Logic Dispatch
	if err := handler.service.MarkChapterAsRead(request.Context(), chapterID, userID); err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Feedback
	respond.OK(writer, map[string]string{FieldMessage: "Chapter marked as read"})
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
