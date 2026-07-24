package handler

import (
	"log/slog"
	"net/http"

	"github.com/magicvr/allinme.core-api/internal/middleware"
	"github.com/magicvr/allinme.core-api/internal/service/auth"
	"github.com/magicvr/allinme.core-api/internal/service/menu"
	"github.com/magicvr/allinme.core-api/internal/service/meta"
)

// Deps are inbound adapter dependencies injected by the composition root.
type Deps struct {
	Logger *slog.Logger
	Meta   *meta.Service
	Auth   *auth.Service
	Menu   *menu.Service
}

// Register mounts all HTTP routes on mux.
// Keep this as the single place Admin and future API modules plug into.
func Register(mux *http.ServeMux, deps Deps) {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}

	// Public
	mux.Handle("GET /healthz", healthz())
	mux.Handle("GET /readyz", readyz(deps.Meta))
	mux.Handle("POST /v1/auth/login", login(deps.Auth))

	// Protected (D-007: default require auth except health/ready/login)
	require := middleware.RequireAuth(deps.Auth)
	mux.Handle("GET /v1/ping", require(ping(deps.Logger)))
	mux.Handle("GET /v1/auth/me", require(me(deps.Auth)))
	mux.Handle("GET /v1/admin/menu", require(adminMenu(deps.Menu)))
}
