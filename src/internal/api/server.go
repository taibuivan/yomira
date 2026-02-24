// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package api wires together the HTTP router, middleware chain, and all
domain handlers into a runnable [http.Server].

Architecture:

  - This package is the topmost Presentation layer boundary.
  - It acts as the central composition root for the HTTP transport framework (chi router).
  - Only this package and cmd/api are allowed to import net/http server primitives.
*/
package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/taibuivan/yomira/internal/core/comic"
	"github.com/taibuivan/yomira/internal/core/group"
	"github.com/taibuivan/yomira/internal/core/reference"
	"github.com/taibuivan/yomira/internal/platform/config"
	"github.com/taibuivan/yomira/internal/platform/constants"
	"github.com/taibuivan/yomira/internal/platform/middleware"
	"github.com/taibuivan/yomira/internal/users/auth"
)

// # Server Definitions

// Server wraps the chi router and the [http.Server].
//
// It is constructed once in main.go with all dependencies injected.
type Server struct {
	httpServer *http.Server
	router     *chi.Mux
	log        *slog.Logger
}

// # Handler Registry

// Handlers groups all domain-specific HTTP handler sets.
//
// # Usage
//
// New domains add a field here — no other change to server.go is required.
type Handlers struct {
	// Liveness is the /health handler — always returns 200 if process is alive.
	Liveness http.HandlerFunc

	// Readiness is the /ready handler — returns 200 when all deps are healthy.
	Readiness http.HandlerFunc

	// Auth handles authentication routes (login, register).
	Auth *auth.Handler

	// Comic handles the publication catalogue and discovery.
	Comic *comic.Handler

	// Reference manages languages, tags, authors, and artists.
	Reference *reference.Handler

	// Group manages scanlation groups and memberships.
	Group *group.Handler
}

// # Server Initialization

// NewServer constructs the chi router with the full middleware chain and
// registers all route groups.
func NewServer(context context.Context, cfg *config.Config, log *slog.Logger, verifier middleware.TokenVerifier, h Handlers) *Server {
	r := chi.NewRouter()

	// # Middleware Chain
	// Global middleware applied in order of execution.
	r.Use(middleware.RequestID())
	r.Use(middleware.StructuredLogger(log))
	r.Use(chimw.Timeout(constants.GlobalRequestTimeout))
	r.Use(middleware.RateLimit(context))
	r.Use(middleware.PanicRecovery(log))
	r.Use(middleware.Authenticate(verifier))
	r.Use(middleware.CORS(cfg))
	r.Use(chimw.CleanPath)

	// # Infrastructure Endpoints
	// Unauthenticated health probes for container orchestration.
	r.Get("/health", h.Liveness)
	r.Get("/ready", h.Readiness)

	// # Application API
	// Domain-specific route groups mounted under versioned prefix.
	r.Route("/api/v1", func(api chi.Router) {
		api.Mount("/auth", h.Auth.Routes())
		api.Mount("/comics", h.Comic.Routes())
		api.Mount("/groups", h.Group.Routes())
		api.Mount("/", h.Reference.Routes())
	})

	return &Server{
		router: r,
		log:    log,
		httpServer: &http.Server{
			Addr:              ":" + cfg.ServerPort,
			Handler:           r,
			ReadTimeout:       constants.DefaultReadTimeout,
			WriteTimeout:      constants.DefaultWriteTimeout,
			IdleTimeout:       constants.DefaultIdleTimeout,
			ReadHeaderTimeout: constants.DefaultReadHeaderTimeout,
		},
	}
}

// # Server Lifecycle

// ListenAndServe starts the HTTP server.
//
// It blocks until the server is closed or an error occurs.
func (s *Server) ListenAndServe() error {
	s.log.Info("server starting", slog.String("addr", s.httpServer.Addr))
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server, waiting for in-flight requests.
func (s *Server) Shutdown(timeout time.Duration) error {
	context, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.httpServer.Shutdown(context)
}
