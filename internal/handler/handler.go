package handler

import (
	"log/slog"
	"net/http"
)

// Register mounts all HTTP routes on mux.
// Keep this as the single place Admin and future API modules plug into.
func Register(mux *http.ServeMux, logger *slog.Logger) {
	mux.Handle("GET /healthz", healthz())
	mux.Handle("GET /readyz", readyz())
	mux.Handle("GET /v1/ping", ping(logger))
}
