/*
Package reference provides the HTTP interface for managing master data.

It orchestrates the retrieval of shared taxonomies (Languages, Tags) and
the management of contributor profiles (Authors, Artists).

# Access Control

  - Public: Discovery of taxonomies and contributor profiles.
  - Moderator: Ability to create and update contributor metadata.
  - Admin: Sensitive operations such as hard/soft deletion of entities.

The handler serves as the bridge between RESTful requests and the [Service] layer.
*/
package reference

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/taibuivan/yomira/internal/platform/middleware"
	requestutil "github.com/taibuivan/yomira/internal/platform/request"
	"github.com/taibuivan/yomira/internal/platform/respond"
	"github.com/taibuivan/yomira/internal/platform/sec"
	"github.com/taibuivan/yomira/pkg/pagination"
)

// Handler implements the HTTP layer for reference and master data.
// It translates web requests into domain service calls.
type Handler struct {
	service *Service
}

// NewHandler constructs a new reference [Handler] with its service dependency.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes returns a [chi.Router] configured with the reference domain's endpoints.
func (handler *Handler) Routes() chi.Router {
	router := chi.NewRouter()

	// # Languages Endpoints
	router.Get("/languages", handler.listLanguages)
	router.Get("/languages/{code}", handler.getLanguage)

	// # Tags Endpoints
	router.Get("/tags", handler.listTags)
	router.Get("/tags/{id}", handler.getTag)
	router.Get("/tags/by-slug/{slug}", handler.getTagBySlug)

	// # Authors and Artists
	router.Route("/authors", func(authorRoute chi.Router) {
		// Public
		authorRoute.Get("/", handler.listAuthors)
		authorRoute.Get("/{id}", handler.getAuthor)

		// Admin/Mod Only
		authorRoute.Group(func(adminRoute chi.Router) {
			adminRoute.Use(middleware.RequireRole(sec.RoleModerator))

			adminRoute.Post("/", handler.createAuthor)
			adminRoute.Patch("/{id}", handler.updateAuthor)

			// Admin strict only
			adminRoute.With(middleware.RequireRole(sec.RoleAdmin)).Delete("/{id}", handler.deleteAuthor)
		})
	})

	router.Route("/artists", func(artistRoute chi.Router) {
		// Public
		artistRoute.Get("/", handler.listArtists)
		artistRoute.Get("/{id}", handler.getArtist)

		// Admin/Mod Only
		artistRoute.Group(func(adminRoute chi.Router) {
			adminRoute.Use(middleware.RequireRole(sec.RoleModerator))

			adminRoute.Post("/", handler.createArtist)
			adminRoute.Patch("/{id}", handler.updateArtist)

			// Admin strict only
			adminRoute.With(middleware.RequireRole(sec.RoleAdmin)).Delete("/{id}", handler.deleteArtist)
		})
	})

	return router
}

/*
GET /api/v1/reference/languages.

Description: Retrieves a complete list of ISO-639-1 languages supported by the platform.

Request:
  - None

Response:
  - 200: []Language: Success
*/
func (handler *Handler) listLanguages(writer http.ResponseWriter, request *http.Request) {

	// Get all languages
	langs, err := handler.service.ListLanguages(request.Context())

	// Handle error
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Structured API Response
	respond.OK(writer, langs)
}

/*
GET /api/v1/reference/languages/{code}.

Description: Retrieves details for a specific language via its ISO code.

Request:
  - code: string (ISO-639-1 code)

Response:
  - 200: Language: Language details
  - 404: ErrNotFound: Language not found
*/
func (handler *Handler) getLanguage(writer http.ResponseWriter, request *http.Request) {

	// Extract code from URL
	code := chi.URLParam(request, "code")

	// Get language
	lang, err := handler.service.GetLanguage(request.Context(), code)

	// Handle error
	if err != nil {
		respond.Error(writer, request, err)
		return
	}
	respond.OK(writer, lang)
}

/*
GET /api/v1/reference/tags.

Description: Retrieves a complete catalogue of content tags and genres, grouped by category.

Request:
  - None

Response:
  - 200: []TagGroup: Success
*/
func (handler *Handler) listTags(writer http.ResponseWriter, request *http.Request) {

	// Get all tags
	groups, err := handler.service.ListTags(request.Context())

	// Handle error
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, groups)
}

/*
GET /api/v1/reference/tags/{id}.

Description: Retrieves a specific content tag by its numeric identifier.

Request:
  - id: int (Tag ID)

Response:
  - 200: Tag: Success
  - 400: 400: ErrInvalidID/Validation: Invalid ID format
  - 404: 404: ErrNotFound: Tag missing
*/
func (handler *Handler) getTag(writer http.ResponseWriter, request *http.Request) {
	// Extract ID from URL
	idStr := requestutil.ID(request, "id")
	tagID, err := strconv.Atoi(idStr)

	// Validate ID
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Domain Logic Execution
	tag, err := handler.service.GetTag(request.Context(), tagID)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, tag)
}

/*
GET /api/v1/reference/tags/by-slug/{slug}.

Description: Resolves a unique SEO-friendly slug back into a specific tag record.

Request:
  - slug: string (Tag slug)

Response:
  - 200: Tag: Success
  - 404: 404: ErrNotFound: Tag missing
*/
func (handler *Handler) getTagBySlug(writer http.ResponseWriter, request *http.Request) {
	// Extract slug from URL
	slug := chi.URLParam(request, "slug")

	// Domain Logic Execution
	tag, err := handler.service.GetTagBySlug(request.Context(), slug)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Structured API Response
	respond.OK(writer, tag)
}

/*
GET /api/v1/reference/authors.

Description: Provides a paginated list of catalogued authors.
Supports full-text search by name or alias.

Request:
  - q: string (Search query)
  - limit: int
  - page: int

Response:
  - 200: []Author: Paginated list
*/
func (handler *Handler) listAuthors(writer http.ResponseWriter, request *http.Request) {
	// Extract standardized pagination
	paginationParams := pagination.FromRequest(request)

	// Build domain filter
	filter := ContributorFilter{
		Query: request.URL.Query().Get("q"),
	}

	// Domain Logic Execution
	authors, total, err := handler.service.ListAuthors(request.Context(), filter, paginationParams.Limit, paginationParams.Offset())
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Structured API Response
	respond.Paginated(writer, authors, pagination.NewMeta(paginationParams.Page, paginationParams.Limit, total))
}

/*
GET /api/v1/reference/authors/{id}.

Description: Retrieves detailed metadata for a specific author.

Request:
  - id: int (Author ID)

Response:
  - 200: Author: Success
  - 400: 400: ErrInvalidID/Validation: Invalid ID format
  - 404: 404: ErrNotFound: Missing author
*/
func (handler *Handler) getAuthor(writer http.ResponseWriter, request *http.Request) {
	idStr := requestutil.ID(request, "id")
	authorID, err := strconv.Atoi(idStr)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	author, err := handler.service.GetAuthor(request.Context(), authorID)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, author)
}

/*
POST /api/v1/reference/authors.

Description: Creates a new author entry in the catalogue.

Request (Body):
  - Author: JSON object

Response:
  - 201: Author: Created author object
  - 400: 400: ErrInvalidJSON/Validation: Invalid input data
  - 401: 401: ErrUnauthorized: Authentication required
  - 403: 403: ErrForbidden: Insufficient permissions
*/
func (handler *Handler) createAuthor(writer http.ResponseWriter, request *http.Request) {
	var input Author

	// Decode request body
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Create author
	if err := handler.service.CreateAuthor(request.Context(), &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.Created(writer, input)
}

/*
PATCH /api/v1/reference/authors/{id}.

Description: Updates metadata for an existing author.

Request:
  - id: int (Author ID)
  - body: Author (Partial JSON)

Response:
  - 200: Author: Updated target
  - 400: 400: ErrInvalidJSON/Validation: Invalid input data
  - 401: 401: ErrUnauthorized: Authentication required
  - 403: 403: ErrForbidden: Insufficient permissions
  - 404: 404: ErrNotFound: Author not found
*/
func (handler *Handler) updateAuthor(writer http.ResponseWriter, request *http.Request) {

	// Extract ID from URL
	idStr := requestutil.ID(request, "id")
	authorID, err := strconv.Atoi(idStr)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Decode request body
	var input Author
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Update author
	if err := handler.service.UpdateAuthor(request.Context(), authorID, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, input)
}

/*
DELETE /api/v1/reference/authors/{id}.

Description: Removes an author from the catalogue.

Request:
  - id: int (Author ID)

Response:
  - 204: No Content: Success
  - 401: 401: ErrUnauthorized: Authentication required
  - 403: 403: ErrForbidden: Insufficient permissions
  - 404: 404: ErrNotFound: Author not found
*/
func (handler *Handler) deleteAuthor(writer http.ResponseWriter, request *http.Request) {

	// Extract ID from URL
	idStr := requestutil.ID(request, "id")
	authorID, err := strconv.Atoi(idStr)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Delete author
	if err := handler.service.DeleteAuthor(request.Context(), authorID); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.NoContent(writer)
}

/*
GET /api/v1/reference/artists.

Description: Provides a paginated list of catalogued artists.
Supports full-text search by name or alias.

Request:
  - q: string (Search query)
  - limit: int
  - page: int

Response:
  - 200: []Artist: Paginated list
*/
func (handler *Handler) listArtists(writer http.ResponseWriter, request *http.Request) {

	// Extract standardized pagination
	paginationParams := pagination.FromRequest(request)

	// Build domain filter
	filter := ContributorFilter{
		Query: request.URL.Query().Get("q"),
	}

	// Domain Logic Execution
	artists, total, err := handler.service.ListArtists(request.Context(), filter, paginationParams.Limit, paginationParams.Offset())
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Structured API Response
	respond.Paginated(writer, artists, pagination.NewMeta(paginationParams.Page, paginationParams.Limit, total))
}

/*
GET /api/v1/reference/artists/{id}.

Description: Retrieves detailed metadata for a specific artist.

Request:
  - id: int (Artist ID)

Response:
  - 200: Artist: Success
  - 400: 400: ErrInvalidJSON: Invalid ID format
  - 404: 404: ErrNotFound: Missing artist
*/
func (handler *Handler) getArtist(writer http.ResponseWriter, request *http.Request) {
	// Extract ID from URL
	idStr := requestutil.ID(request, "id")
	artistID, err := strconv.Atoi(idStr)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Domain Logic Execution
	artist, err := handler.service.GetArtist(request.Context(), artistID)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, artist)
}

/*
POST /api/v1/reference/artists.

Description: Creates a new artist entry in the platform catalogue.

Request (Body):
  - Artist: JSON object

Response:
  - 201: Artist: Created artist object
  - 400: 400: ErrInvalidJSON/Validation: Invalid input data
  - 401: 401: ErrUnauthorized: Authentication required
  - 403: 403: ErrForbidden: Insufficient permissions
*/
func (handler *Handler) createArtist(writer http.ResponseWriter, request *http.Request) {
	var input Artist

	// Decode request body
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Create artist
	if err := handler.service.CreateArtist(request.Context(), &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.Created(writer, input)
}

/*
PATCH /api/v1/reference/artists/{id}.

Description: Updates metadata for an existing artist.

Request:
  - id: int (Artist ID)
  - body: Artist (Partial JSON)

Response:
  - 200: Artist: Updated target
  - 400: 400: ErrInvalidJSON/Validation: Invalid input data
  - 401: 401: ErrUnauthorized: Authentication required
  - 403: 403: ErrForbidden: Insufficient permissions
  - 404: 404: ErrNotFound: Artist not found
*/
func (handler *Handler) updateArtist(writer http.ResponseWriter, request *http.Request) {

	// Extract ID from URL
	idStr := requestutil.ID(request, "id")
	artistID, err := strconv.Atoi(idStr)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Decode request body
	var input Artist
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Update artist
	if err := handler.service.UpdateArtist(request.Context(), artistID, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, input)
}

/*
DELETE /api/v1/reference/artists/{id}.

Description: Removes an artist from the catalogue.

Request:
  - id: int (Artist ID)

Response:
  - 204: No Content: Success
  - 401: 401: ErrUnauthorized: Authentication required
  - 403: 403: ErrForbidden: Insufficient permissions
  - 404: 404: ErrNotFound: Artist not found
*/
func (handler *Handler) deleteArtist(writer http.ResponseWriter, request *http.Request) {

	// Extract ID from URL
	idStr := requestutil.ID(request, "id")
	artistID, err := strconv.Atoi(idStr)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Delete artist
	if err := handler.service.DeleteArtist(request.Context(), artistID); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.NoContent(writer)
}
