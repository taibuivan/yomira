package tag

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	requestutil "github.com/taibuivan/yomira/internal/platform/request"
	"github.com/taibuivan/yomira/internal/platform/respond"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) RegisterRoutes(router chi.Router) {
	router.Get("/", handler.listTags)
	router.Get("/{id}", handler.getTag)
	router.Get("/by-slug/{slug}", handler.getTagBySlug)
}

func (handler *Handler) listTags(writer http.ResponseWriter, request *http.Request) {
	groups, err := handler.service.ListTags(request.Context())
	if err != nil {
		respond.Error(writer, request, err)
		return
	}
	respond.OK(writer, groups)
}

func (handler *Handler) getTag(writer http.ResponseWriter, request *http.Request) {
	idStr := requestutil.ID(request, "id")
	tagID, err := strconv.Atoi(idStr)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	tag, err := handler.service.GetTag(request.Context(), tagID)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}
	respond.OK(writer, tag)
}

func (handler *Handler) getTagBySlug(writer http.ResponseWriter, request *http.Request) {
	slug := chi.URLParam(request, "slug")

	tag, err := handler.service.GetTagBySlug(request.Context(), slug)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}
	respond.OK(writer, tag)
}
