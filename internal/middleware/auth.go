package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/magicvr/allinme.core-api/internal/domain"
	"github.com/magicvr/allinme.core-api/internal/response"
	"github.com/magicvr/allinme.core-api/internal/service/auth"
)

type ctxKey int

const userKey ctxKey = 1

// UserFromContext returns the authenticated user if present.
func UserFromContext(ctx context.Context) (domain.User, bool) {
	u, ok := ctx.Value(userKey).(domain.User)
	return u, ok
}

// RequireAuth validates Bearer JWT and injects domain.User into the request context.
func RequireAuth(authSvc *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := r.Header.Get("Authorization")
			if raw == "" || !strings.HasPrefix(strings.ToLower(raw), "bearer ") {
				response.Error(w, http.StatusUnauthorized, "unauthorized", "missing or invalid Authorization header")
				return
			}
			token := strings.TrimSpace(raw[7:])
			claims, err := authSvc.ParseToken(r.Context(), token)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "unauthorized", "invalid token")
				return
			}
			user := claims.ToUser()
			ctx := context.WithValue(r.Context(), userKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRoles ensures the user has at least one of the roles (after RequireAuth).
func RequireRoles(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok || !user.HasAnyRole(roles...) {
				response.Error(w, http.StatusForbidden, "forbidden", "insufficient role")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

