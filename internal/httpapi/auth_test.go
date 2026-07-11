package httpapi_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/httpapi"
)

func TestAuthHTTPFlow(t *testing.T) {
	service := &fakeAuthService{}
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: service, LoginLimiter: httpapi.NewLoginLimiter(nil)})
	login := authRequest(handler, http.MethodPost, "/api/v1/auth/login", `{"username":" VIEWER ","password":"123456789012"}`)
	login.Header.Set("Content-Type", "application/json")
	loginResponse := httptest.NewRecorder()
	handler.ServeHTTP(loginResponse, login)
	if loginResponse.Code != http.StatusOK || !strings.Contains(loginResponse.Body.String(), `"tokenType":"Bearer"`) {
		t.Fatalf("login = %d %s", loginResponse.Code, loginResponse.Body.String())
	}
	if service.loginUsername != "viewer" {
		t.Fatalf("login username = %q", service.loginUsername)
	}

	me := authRequest(handler, http.MethodGet, "/api/v1/auth/me", "")
	me.Header.Set("Authorization", "bearer access-token")
	meResponse := httptest.NewRecorder()
	handler.ServeHTTP(meResponse, me)
	if meResponse.Code != http.StatusOK || !strings.Contains(meResponse.Body.String(), `"username":"viewer"`) {
		t.Fatalf("me = %d %s", meResponse.Code, meResponse.Body.String())
	}

	logout := authRequest(handler, http.MethodPost, "/api/v1/auth/logout", "")
	logout.Header.Set("Authorization", "Bearer access-token")
	logoutResponse := httptest.NewRecorder()
	handler.ServeHTTP(logoutResponse, logout)
	if logoutResponse.Code != http.StatusNoContent || logoutResponse.Body.Len() != 0 || service.logoutCount != 1 {
		t.Fatalf("logout = %d %q count=%d", logoutResponse.Code, logoutResponse.Body.String(), service.logoutCount)
	}
}

func TestAuthHTTPInputAndBearerBoundaries(t *testing.T) {
	service := &fakeAuthService{}
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: service})
	tests := []struct {
		name        string
		method      string
		path        string
		body        string
		contentType string
		authHeaders []string
		wantStatus  int
		wantCode    string
	}{
		{name: "wrong login method", method: http.MethodGet, path: "/api/v1/auth/login", wantStatus: 405, wantCode: "METHOD_NOT_ALLOWED"},
		{name: "wrong me method", method: http.MethodPost, path: "/api/v1/auth/me", wantStatus: 405, wantCode: "METHOD_NOT_ALLOWED"},
		{name: "unsupported media", method: http.MethodPost, path: "/api/v1/auth/login", body: `{}`, contentType: "text/plain", wantStatus: 415, wantCode: "UNSUPPORTED_MEDIA_TYPE"},
		{name: "unknown field", method: http.MethodPost, path: "/api/v1/auth/login", body: `{"username":"viewer","password":"123456789012","extra":true}`, contentType: "application/json", wantStatus: 400, wantCode: "INVALID_REQUEST"},
		{name: "short password", method: http.MethodPost, path: "/api/v1/auth/login", body: `{"username":"viewer","password":"short"}`, contentType: "application/json", wantStatus: 400, wantCode: "INVALID_REQUEST"},
		{name: "missing bearer", method: http.MethodGet, path: "/api/v1/auth/me", wantStatus: 401, wantCode: "UNAUTHENTICATED"},
		{name: "duplicate bearer", method: http.MethodGet, path: "/api/v1/auth/me", authHeaders: []string{"Bearer one", "Bearer two"}, wantStatus: 401, wantCode: "UNAUTHENTICATED"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := authRequest(handler, test.method, test.path, test.body)
			if test.contentType != "" {
				request.Header.Set("Content-Type", test.contentType)
			}
			for _, value := range test.authHeaders {
				request.Header.Add("Authorization", value)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus || !strings.Contains(response.Body.String(), `"code":"`+test.wantCode+`"`) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
			if test.wantStatus == 401 && response.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Fatalf("WWW-Authenticate = %q", response.Header().Get("WWW-Authenticate"))
			}
		})
	}
}

func TestLoginRateLimitAndSensitiveLogs(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	limiter := httpapi.NewLoginLimiter(func() time.Time { return now })
	service := &fakeAuthService{loginError: auth.ErrAuthenticationFailed}
	var logs bytes.Buffer
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: slog.New(slog.NewJSONHandler(&logs, nil)), Auth: service, LoginLimiter: limiter})
	for attempt := 1; attempt <= 6; attempt++ {
		request := authRequest(handler, http.MethodPost, "/api/v1/auth/login", `{"username":"viewer","password":"sensitive-password"}`)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer secret-token")
		request.RemoteAddr = "192.0.2.1:1234"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		want := http.StatusUnauthorized
		if attempt == 6 {
			want = http.StatusTooManyRequests
		}
		if response.Code != want {
			t.Fatalf("attempt %d status = %d", attempt, response.Code)
		}
	}
	if service.loginCount != 5 {
		t.Fatalf("login calls = %d", service.loginCount)
	}
	for _, secret := range []string{"sensitive-password", "secret-token", "Authorization"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatalf("logs contain %q: %s", secret, logs.String())
		}
	}
	now = now.Add(time.Minute)
	request := authRequest(handler, http.MethodPost, "/api/v1/auth/login", `{"username":"viewer","password":"sensitive-password"}`)
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = "192.0.2.1:4321"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || service.loginCount != 6 {
		t.Fatalf("after window = %d calls=%d", response.Code, service.loginCount)
	}
}

func TestClientIP(t *testing.T) {
	for input, want := range map[string]string{
		"192.0.2.1:1000": "192.0.2.1", "[2001:db8::1]:2000": "2001:db8::1", "bad": "unknown", "bad:port": "unknown",
	} {
		if got := httpapi.ClientIP(input); got != want {
			t.Errorf("ClientIP(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRequireRolesMatrix(t *testing.T) {
	roles := []auth.Role{auth.RoleViewer, auth.RoleOperator, auth.RoleApprover, auth.RoleAdmin}
	allowed := []auth.Role{auth.RoleApprover, auth.RoleAdmin}
	for _, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			service := &fakeAuthService{authenticatedRole: role}
			protected := httpapi.RequireAuthentication(service)(httpapi.RequireRoles(allowed...)(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.WriteHeader(http.StatusNoContent)
			})))
			handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: service, Fallback: protected})
			request := httptest.NewRequest(http.MethodGet, "/test-role", nil)
			request.Header.Set("Authorization", "Bearer access-token")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			want := http.StatusForbidden
			if role == auth.RoleApprover || role == auth.RoleAdmin {
				want = http.StatusNoContent
			}
			if response.Code != want {
				t.Fatalf("status = %d, want %d", response.Code, want)
			}
		})
	}
}

type fakeAuthService struct {
	loginError        error
	loginUsername     string
	loginCount        int
	logoutCount       int
	authenticatedRole auth.Role
}

func (service *fakeAuthService) Login(_ context.Context, username, _ string) (auth.LoginResult, error) {
	service.loginUsername = username
	service.loginCount++
	if service.loginError != nil {
		return auth.LoginResult{}, service.loginError
	}
	return auth.LoginResult{AccessToken: "access-token", ExpiresAt: time.Date(2026, 7, 12, 12, 15, 0, 0, time.UTC), User: auth.User{ID: "user-1", Username: "viewer", Role: auth.RoleViewer}}, nil
}
func (service *fakeAuthService) Authenticate(_ context.Context, token string) (auth.Principal, error) {
	if token != "access-token" {
		return auth.Principal{}, auth.ErrUnauthenticated
	}
	role := service.authenticatedRole
	if role == "" {
		role = auth.RoleViewer
	}
	return auth.Principal{UserID: "user-1", Username: "viewer", Role: role, TokenID: "token-1", TokenExpiresAt: "2026-07-12T12:15:00Z"}, nil
}
func (service *fakeAuthService) Logout(context.Context, auth.Principal) error {
	service.logoutCount++
	return nil
}

func authRequest(_ http.Handler, method, path, body string) *http.Request {
	return httptest.NewRequest(method, path, io.NopCloser(strings.NewReader(body)))
}

var _ = errors.Is
