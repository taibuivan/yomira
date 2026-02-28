package artist

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
	router.Get("/", handler.listArtists)
	router.Get("/{id}", handler.getArtist)

	// Admin/Mod Only
	router.Group(func(adminRoute chi.Router) {
		adminRoute.Use(middleware.RequireRole(sec.RoleModerator))

		adminRoute.Post("/", handler.createArtist)
		adminRoute.Patch("/{id}", handler.updateArtist)

		// Admin strict only
		adminRoute.With(middleware.RequireRole(sec.RoleAdmin)).Delete("/{id}", handler.deleteArtist)
	})
}

func (handler *Handler) listArtists(writer http.ResponseWriter, request *http.Request) {
	paginationParams := pagination.FromRequest(request)

	filter := Filter{
		Query: request.URL.Query().Get("q"),
	}

	artists, total, err := handler.service.ListArtists(request.Context(), filter, paginationParams.Limit, paginationParams.Offset())
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.Paginated(writer, artists, pagination.NewMeta(paginationParams.Page, paginationParams.Limit, total))
}

func (handler *Handler) getArtist(writer http.ResponseWriter, request *http.Request) {
	idStr := requestutil.ID(request, "id")
	artistID, err := strconv.Atoi(idStr)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	artist, err := handler.service.GetArtist(request.Context(), artistID)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}
	respond.OK(writer, artist)
}

func (handler *Handler) createArtist(writer http.ResponseWriter, request *http.Request) {
	var input Artist

	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	if err := handler.service.CreateArtist(request.Context(), &input); err != nil {
		respond.Error(writer, request, err)
		return
	}
	respond.Created(writer, input)
}

func (handler *Handler) updateArtist(writer http.ResponseWriter, request *http.Request) {
	idStr := requestutil.ID(request, "id")
	artistID, err := strconv.Atoi(idStr)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	var input Artist
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	if err := handler.service.UpdateArtist(request.Context(), artistID, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}
	respond.OK(writer, input)
}

func (handler *Handler) deleteArtist(writer http.ResponseWriter, request *http.Request) {
	idStr := requestutil.ID(request, "id")
	artistID, err := strconv.Atoi(idStr)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	if err := handler.service.DeleteArtist(request.Context(), artistID); err != nil {
		respond.Error(writer, request, err)
		return
	}
	respond.NoContent(writer)
}
