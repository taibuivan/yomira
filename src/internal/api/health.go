// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package api implements the observability endpoints for the Yomira platform.

It provides standard Kubernetes-style probes (liveness, readiness) to monitor the
operational health of the application and its critical dependencies.

Architecture:

  - Liveness: Returns 200 OK as long as the process is running.
  - Readiness: Performs shallow pings of Postgres and Redis to verify connectivity.

These handlers ensure that traffic is only routed to "warm" instances that are
fully connected to the data plane.
*/
package api

import (
	"log/slog"
	"net/http"

	"github.com/taibuivan/yomira/internal/platform/constants"
	"github.com/taibuivan/yomira/internal/platform/respond"
)

// # Data Structures

// HealthDependencies holds the injectable dependency checkers for system probes.
type HealthDependencies struct {
	// CheckDatabase performs a shallow ping of the PostgreSQL pool.
	CheckDatabase func() error

	// CheckCache performs a shallow ping of the Redis client.
	CheckCache func() error
}

// healthHandler orchestrates the execution of connectivity checks.
type healthHandler struct {
	dependencies HealthDependencies
	logger       *slog.Logger
}

// # Constructors

// NewHealthHandlers constructs the liveness and readiness [http.HandlerFunc] pair.
func NewHealthHandlers(deps HealthDependencies, logger *slog.Logger) (liveness, readiness http.HandlerFunc) {
	handler := &healthHandler{
		dependencies: deps,
		logger:       logger,
	}
	return handler.liveness, handler.readiness
}

// # Handlers

// liveness handles GET /health.
// It confirms that the HTTP server is alive and accepting connections.
func (handler *healthHandler) liveness(writer http.ResponseWriter, _ *http.Request) {
	respond.OK(writer, map[string]string{
		constants.FieldStatus:  "ok",
		constants.FieldApp:     constants.AppName,
		constants.FieldVersion: constants.AppVersion,
	})
}

// readiness handles GET /ready.
// It verifies that all downstream dependencies (DB, Cache) are reachable.
func (handler *healthHandler) readiness(writer http.ResponseWriter, _ *http.Request) {

	// Inner type for individual check reporting
	type checkResult struct {
		Name  string `json:"name"`
		IsOK  bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}

	results := make([]checkResult, 0, 2)
	isSystemReady := true

	// 1. Check PostgreSQL Connectivity
	if handler.dependencies.CheckDatabase != nil {
		result := checkResult{Name: "postgres", IsOK: true}
		if err := handler.dependencies.CheckDatabase(); err != nil {
			result.IsOK = false
			result.Error = err.Error()
			isSystemReady = false
			handler.logger.Error("readiness_check_failed",
				slog.String("dependency", "postgres"),
				slog.Any("error", err),
			)
		}
		results = append(results, result)
	}

	// 2. Check Redis Connectivity
	if handler.dependencies.CheckCache != nil {
		result := checkResult{Name: "redis", IsOK: true}
		if err := handler.dependencies.CheckCache(); err != nil {
			result.IsOK = false
			result.Error = err.Error()
			isSystemReady = false
			handler.logger.Error("readiness_check_failed",
				slog.String("dependency", "redis"),
				slog.Any("error", err),
			)
		}
		results = append(results, result)
	}

	// 3. Determine Response State
	responseStatus := "ready"
	httpStatus := http.StatusOK

	if !isSystemReady {
		responseStatus = "degraded"
		httpStatus = http.StatusServiceUnavailable

		// Manual header injection for error states to bypass default respond wrappers
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		writer.WriteHeader(httpStatus)
	}

	// 4. Send Response
	respond.OK(writer, map[string]any{
		constants.FieldStatus: responseStatus,
		constants.FieldChecks: results,
	})
}
