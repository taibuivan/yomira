// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

// Package api contains the health check handlers for liveness and readiness probes.
package api

import (
	"log/slog"
	"net/http"

	"github.com/taibuivan/yomira/internal/platform/respond"
)

// HealthDependencies holds the injectable dependency checkers for the /ready endpoint.
type HealthDependencies struct {
	// CheckDatabase pings the PostgreSQL pool.
	CheckDatabase func() error

	// CheckCache pings the Redis client.
	CheckCache func() error
}

type healthHandler struct {
	dependencies HealthDependencies
	logger       *slog.Logger
}

// NewHealthHandlers creates the /health and /ready http.HandlerFuncs.
func NewHealthHandlers(deps HealthDependencies, logger *slog.Logger) (liveness, readiness http.HandlerFunc) {
	handler := &healthHandler{dependencies: deps, logger: logger}
	return handler.liveness, handler.readiness
}

// liveness handles GET /health (Liveness probe).
func (handler *healthHandler) liveness(writer http.ResponseWriter, request *http.Request) {
	respond.OK(writer, map[string]string{"status": "ok"})
}

// readiness handles GET /ready (Readiness probe).
func (handler *healthHandler) readiness(writer http.ResponseWriter, request *http.Request) {
	type checkResult struct {
		Name  string `json:"name"`
		IsOK  bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}

	results := make([]checkResult, 0, 2)
	isSystemReady := true

	// Check PostgreSQL
	if handler.dependencies.CheckDatabase != nil {
		result := checkResult{Name: "postgres", IsOK: true}
		if err := handler.dependencies.CheckDatabase(); err != nil {
			result.IsOK = false
			result.Error = err.Error()
			isSystemReady = false
			handler.logger.Error("readiness_check_failed", slog.String("dependency", "postgres"), slog.Any("error", err))
		}
		results = append(results, result)
	}

	// Check Redis
	if handler.dependencies.CheckCache != nil {
		result := checkResult{Name: "redis", IsOK: true}
		if err := handler.dependencies.CheckCache(); err != nil {
			result.IsOK = false
			result.Error = err.Error()
			isSystemReady = false
			handler.logger.Error("readiness_check_failed", slog.String("dependency", "redis"), slog.Any("error", err))
		}
		results = append(results, result)
	}

	responseStatus := "ready"
	httpStatus := http.StatusOK

	if !isSystemReady {
		responseStatus = "degraded"
		httpStatus = http.StatusServiceUnavailable
		// We use writeHeader manually because respond.OK always sends 200
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		writer.WriteHeader(httpStatus)
	}

	respond.OK(writer, map[string]any{
		"status": responseStatus,
		"checks": results,
	})
}
