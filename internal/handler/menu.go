package handler

import (
	"net/http"

	"github.com/magicvr/allinme.core-api/internal/middleware"
	"github.com/magicvr/allinme.core-api/internal/response"
	"github.com/magicvr/allinme.core-api/internal/service/menu"
)

func adminMenu(menuSvc *menu.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := middleware.UserFromContext(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "not authenticated")
			return
		}
		items := menuSvc.ForUser(r.Context(), user)
		response.OK(w, map[string]any{"items": items})
	})
}
