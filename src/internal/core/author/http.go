package author

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

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) RegisterRoutes(router chi.Router) {
	// Public
	router.Get("/", handler.listAuthors)
	router.Get("/{id}", handler.getAuthor)

	// Admin/Mod Only
	router.Group(func(adminRoute chi.Router) {
		adminRoute.Use(middleware.RequireRole(sec.RoleModerator))

		adminRoute.Post("/", handler.createAuthor)
		adminRoute.Patch("/{id}", handler.updateAuthor)

		// Admin strict only
		adminRoute.With(middleware.RequireRole(sec.RoleAdmin)).Delete("/{id}", handler.deleteAuthor)
	})
}

func (handler *Handler) listAuthors(writer http.ResponseWriter, request *http.Request) {
	paginationParams := pagination.FromRequest(request)

	filter := Filter{
		Query: request.URL.Query().Get("q"),
	}

	authors, total, err := handler.service.ListAuthors(request.Context(), filter, paginationParams.Limit, paginationParams.Offset())
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.Paginated(writer, authors, pagination.NewMeta(paginationParams.Page, paginationParams.Limit, total))
}

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

func (handler *Handler) createAuthor(writer http.ResponseWriter, request *http.Request) {
	var input Author
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	if err := handler.service.CreateAuthor(request.Context(), &input); err != nil {
		respond.Error(writer, request, err)
		return
	}
	respond.Created(writer, input)
}

func (handler *Handler) updateAuthor(writer http.ResponseWriter, request *http.Request) {
	idStr := requestutil.ID(request, "id")
	authorID, err := strconv.Atoi(idStr)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	var input Author
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	if err := handler.service.UpdateAuthor(request.Context(), authorID, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}
	respond.OK(writer, input)
}

func (handler *Handler) deleteAuthor(writer http.ResponseWriter, request *http.Request) {
	idStr := requestutil.ID(request, "id")
	authorID, err := strconv.Atoi(idStr)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	if err := handler.service.DeleteAuthor(request.Context(), authorID); err != nil {
		respond.Error(writer, request, err)
		return
	}
	respond.NoContent(writer)
}
