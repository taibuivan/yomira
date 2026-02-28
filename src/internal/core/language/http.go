package language

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/taibuivan/yomira/internal/platform/respond"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) RegisterRoutes(router chi.Router) {
	router.Get("/", handler.listLanguages)
	router.Get("/{code}", handler.getLanguage)
}

func (handler *Handler) listLanguages(writer http.ResponseWriter, request *http.Request) {
	langs, err := handler.service.ListLanguages(request.Context())
	if err != nil {
		respond.Error(writer, request, err)
		return
	}
	respond.OK(writer, langs)
}

func (handler *Handler) getLanguage(writer http.ResponseWriter, request *http.Request) {
	code := chi.URLParam(request, "code")

	lang, err := handler.service.GetLanguage(request.Context(), code)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}
	respond.OK(writer, lang)
}
