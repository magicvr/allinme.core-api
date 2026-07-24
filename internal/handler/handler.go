package handler

import (
	"log/slog"
	"net/http"

	"github.com/magicvr/allinme.core-api/internal/service/meta"
)

// Deps are inbound adapter dependencies injected by the composition root.
type Deps struct {
	Logger *slog.Logger
	Meta   *meta.Service
}

// Register mounts all HTTP routes on mux.
// Keep this as the single place Admin and future API modules plug into.
func Register(mux *http.ServeMux, deps Deps) {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	mux.Handle("GET /healthz", healthz())
	mux.Handle("GET /readyz", readyz(deps.Meta))
	mux.Handle("GET /v1/ping", ping(deps.Logger))
}
