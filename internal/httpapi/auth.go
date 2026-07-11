package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/magicvr/allinme.core-api/internal/auth"
)

const loginBodyLimit = 4 * 1024

type AuthService interface {
	Login(context.Context, string, string) (auth.LoginResult, error)
	Authenticate(context.Context, string) (auth.Principal, error)
	Logout(context.Context, auth.Principal) error
}

type principalKey struct{}

func PrincipalFromContext(ctx context.Context) (auth.Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(auth.Principal)
	return principal, ok
}

func RequireRoles(allowed ...auth.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			principal, ok := PrincipalFromContext(request.Context())
			if !ok {
				unauthenticated(response, request)
				return
			}
			if !auth.RoleAllowed(principal.Role, allowed...) {
				writeError(response, request, http.StatusForbidden, "FORBIDDEN", "forbidden")
				return
			}
			next.ServeHTTP(response, request)
		})
	}
}

func RequireAuthentication(service AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return authMiddleware(service, next)
	}
}

func registerAuthRoutes(mux *http.ServeMux, service AuthService, limiter *LoginLimiter) {
	if service == nil {
		return
	}
	if limiter == nil {
		limiter = NewLoginLimiter(nil)
	}
	mux.HandleFunc("/api/v1/auth/login", loginHandler(service, limiter))
	mux.Handle("/api/v1/auth/me", requireMethod(http.MethodGet, authMiddleware(service, http.HandlerFunc(meHandler))))
	mux.Handle("/api/v1/auth/logout", requireMethod(http.MethodPost, authMiddleware(service, http.HandlerFunc(logoutHandler(service)))))
}

func loginHandler(service AuthService, limiter *LoginLimiter) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			methodNotAllowed(response, request, http.MethodPost)
			return
		}
		mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
		if err != nil || mediaType != "application/json" {
			writeError(response, request, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "content type must be application/json")
			return
		}
		var input struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, loginBodyLimit))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil || input.Username == "" || input.Password == "" {
			writeError(response, request, http.StatusBadRequest, "INVALID_REQUEST", "invalid login request")
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			writeError(response, request, http.StatusBadRequest, "INVALID_REQUEST", "invalid login request")
			return
		}
		username := auth.NormalizeUsername(input.Username)
		if username == "" || auth.ValidatePassword(input.Password) != nil {
			writeError(response, request, http.StatusBadRequest, "INVALID_REQUEST", "invalid login request")
			return
		}
		ip := ClientIP(request.RemoteAddr)
		allowed, retry := limiter.Check(ip, username)
		if !allowed {
			response.Header().Set("Retry-After", strconv.Itoa(retry))
			writeError(response, request, http.StatusTooManyRequests, "RATE_LIMITED", "too many login attempts")
			return
		}
		result, err := service.Login(request.Context(), username, input.Password)
		if errors.Is(err, auth.ErrAuthenticationFailed) {
			limiter.Failure(ip, username)
			writeError(response, request, http.StatusUnauthorized, "AUTHENTICATION_FAILED", "authentication failed")
			return
		}
		if err != nil {
			limiter.Cancel(ip, username)
			writeError(response, request, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			return
		}
		limiter.Success(ip, username)
		writeJSON(response, http.StatusOK, struct {
			AccessToken string `json:"accessToken"`
			TokenType   string `json:"tokenType"`
			ExpiresAt   string `json:"expiresAt"`
			User        struct {
				ID       string    `json:"id"`
				Username string    `json:"username"`
				Role     auth.Role `json:"role"`
			} `json:"user"`
		}{AccessToken: result.AccessToken, TokenType: "Bearer", ExpiresAt: result.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"), User: struct {
			ID       string    `json:"id"`
			Username string    `json:"username"`
			Role     auth.Role `json:"role"`
		}{ID: result.User.ID, Username: result.User.Username, Role: result.User.Role}})
	}
}

func requireMethod(method string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != method {
			methodNotAllowed(response, request, method)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func authMiddleware(service AuthService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		values := request.Header.Values("Authorization")
		if len(values) != 1 {
			unauthenticated(response, request)
			return
		}
		parts := strings.Fields(values[0])
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			unauthenticated(response, request)
			return
		}
		principal, err := service.Authenticate(request.Context(), parts[1])
		if errors.Is(err, auth.ErrUnauthenticated) {
			unauthenticated(response, request)
			return
		}
		if err != nil {
			writeError(response, request, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			return
		}
		ctx := context.WithValue(request.Context(), principalKey{}, principal)
		next.ServeHTTP(response, request.WithContext(ctx))
	})
}

func meHandler(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		methodNotAllowed(response, request, http.MethodGet)
		return
	}
	principal, ok := PrincipalFromContext(request.Context())
	if !ok {
		unauthenticated(response, request)
		return
	}
	writeJSON(response, http.StatusOK, struct {
		ID             string    `json:"id"`
		Username       string    `json:"username"`
		Role           auth.Role `json:"role"`
		TokenExpiresAt string    `json:"tokenExpiresAt"`
	}{ID: principal.UserID, Username: principal.Username, Role: principal.Role, TokenExpiresAt: principal.TokenExpiresAt})
}

func logoutHandler(service AuthService) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			methodNotAllowed(response, request, http.MethodPost)
			return
		}
		principal, ok := PrincipalFromContext(request.Context())
		if !ok {
			unauthenticated(response, request)
			return
		}
		if err := service.Logout(request.Context(), principal); err != nil {
			writeError(response, request, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			return
		}
		response.WriteHeader(http.StatusNoContent)
	}
}

func unauthenticated(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("WWW-Authenticate", "Bearer")
	writeError(response, request, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
}

func methodNotAllowed(response http.ResponseWriter, request *http.Request, method string) {
	response.Header().Set("Allow", method)
	writeError(response, request, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
}
