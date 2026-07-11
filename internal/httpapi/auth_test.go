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
	"github.com/magicvr/allinme.core-api/internal/store"
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
		{name: "missing username", method: http.MethodPost, path: "/api/v1/auth/login", body: `{"password":"123456789012"}`, contentType: "application/json", wantStatus: 400, wantCode: "INVALID_REQUEST"},
		{name: "missing password", method: http.MethodPost, path: "/api/v1/auth/login", body: `{"username":"viewer"}`, contentType: "application/json", wantStatus: 400, wantCode: "INVALID_REQUEST"},
		{name: "empty username", method: http.MethodPost, path: "/api/v1/auth/login", body: `{"username":"","password":"123456789012"}`, contentType: "application/json", wantStatus: 400, wantCode: "INVALID_REQUEST"},
		{name: "blank username", method: http.MethodPost, path: "/api/v1/auth/login", body: `{"username":"   ","password":"123456789012"}`, contentType: "application/json", wantStatus: 400, wantCode: "INVALID_REQUEST"},
		{name: "empty password", method: http.MethodPost, path: "/api/v1/auth/login", body: `{"username":"viewer","password":""}`, contentType: "application/json", wantStatus: 400, wantCode: "INVALID_REQUEST"},
		{name: "username type", method: http.MethodPost, path: "/api/v1/auth/login", body: `{"username":1,"password":"123456789012"}`, contentType: "application/json", wantStatus: 400, wantCode: "INVALID_REQUEST"},
		{name: "password type", method: http.MethodPost, path: "/api/v1/auth/login", body: `{"username":"viewer","password":true}`, contentType: "application/json", wantStatus: 400, wantCode: "INVALID_REQUEST"},
		{name: "unknown field", method: http.MethodPost, path: "/api/v1/auth/login", body: `{"username":"viewer","password":"123456789012","extra":true}`, contentType: "application/json", wantStatus: 400, wantCode: "INVALID_REQUEST"},
		{name: "multiple JSON values", method: http.MethodPost, path: "/api/v1/auth/login", body: `{"username":"viewer","password":"123456789012"} {}`, contentType: "application/json", wantStatus: 400, wantCode: "INVALID_REQUEST"},
		{name: "oversized body", method: http.MethodPost, path: "/api/v1/auth/login", body: `{"username":"viewer","password":"` + strings.Repeat("a", 4096) + `"}`, contentType: "application/json", wantStatus: 400, wantCode: "INVALID_REQUEST"},
		{name: "short password", method: http.MethodPost, path: "/api/v1/auth/login", body: `{"username":"viewer","password":"short"}`, contentType: "application/json", wantStatus: 400, wantCode: "INVALID_REQUEST"},
		{name: "missing bearer", method: http.MethodGet, path: "/api/v1/auth/me", wantStatus: 401, wantCode: "UNAUTHENTICATED"},
		{name: "duplicate bearer", method: http.MethodGet, path: "/api/v1/auth/me", authHeaders: []string{"Bearer one", "Bearer two"}, wantStatus: 401, wantCode: "UNAUTHENTICATED"},
		{name: "wrong scheme", method: http.MethodGet, path: "/api/v1/auth/me", authHeaders: []string{"Basic access-token"}, wantStatus: 401, wantCode: "UNAUTHENTICATED"},
		{name: "empty token", method: http.MethodGet, path: "/api/v1/auth/me", authHeaders: []string{"Bearer "}, wantStatus: 401, wantCode: "UNAUTHENTICATED"},
		{name: "extra bearer part", method: http.MethodGet, path: "/api/v1/auth/me", authHeaders: []string{"Bearer access-token extra"}, wantStatus: 401, wantCode: "UNAUTHENTICATED"},
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

func TestAuthHTTPMapsInvalidAndInternalAuthenticationSafely(t *testing.T) {
	for _, test := range []struct {
		name          string
		authError     error
		wantStatus    int
		wantCode      string
		wantChallenge bool
	}{
		{name: "tampered or expired token", authError: auth.ErrUnauthenticated, wantStatus: http.StatusUnauthorized, wantCode: "UNAUTHENTICATED", wantChallenge: true},
		{name: "canceled context", authError: context.Canceled, wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		{name: "store internal error", authError: errors.New("SQL C:\\sensitive\\allinme.db token-id-secret"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var logs bytes.Buffer
			service := &fakeAuthService{authenticateError: test.authError}
			handler := httpapi.NewHandler(httpapi.Dependencies{Logger: slog.New(slog.NewJSONHandler(&logs, nil)), Auth: service})
			request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
			request.Header.Set("Authorization", "Bearer sensitive-access-token")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus || !strings.Contains(response.Body.String(), `"code":"`+test.wantCode+`"`) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
			if test.wantChallenge != (response.Header().Get("WWW-Authenticate") == "Bearer") {
				t.Fatalf("WWW-Authenticate = %q", response.Header().Get("WWW-Authenticate"))
			}
			combined := response.Body.String() + logs.String()
			for _, secret := range []string{"sensitive-access-token", "token-id-secret", "sensitive\\allinme.db", "SQL"} {
				if strings.Contains(combined, secret) {
					t.Fatalf("response/logs contain %q: %s", secret, combined)
				}
			}
		})
	}
}

func TestAuthenticationMiddlewareChainAndPublicRoutes(t *testing.T) {
	var logs bytes.Buffer
	service := &fakeAuthService{}
	protected := httpapi.RequireAuthentication(service)(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	}))
	handler := httpapi.NewHandler(httpapi.Dependencies{
		Logger: slog.New(slog.NewJSONHandler(&logs, nil)), Auth: service,
		Readiness: probeFunc(func(context.Context) store.ReadinessStatus { return store.Ready }), Fallback: protected,
	})
	for _, path := range []string{"/healthz", "/readyz"} {
		response := serve(handler, http.MethodGet, path, "public-request")
		if response.Code != http.StatusOK {
			t.Fatalf("%s status = %d", path, response.Code)
		}
	}
	if service.authenticateCount != 0 {
		t.Fatalf("public routes authenticated %d times", service.authenticateCount)
	}

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer access-token")
	request.Header.Set("X-Request-ID", "chain-request")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || service.authenticateCount != 1 || response.Header().Get("X-Request-ID") != "chain-request" {
		t.Fatalf("protected response = %d, auth calls = %d, request ID = %q", response.Code, service.authenticateCount, response.Header().Get("X-Request-ID"))
	}
	for _, expected := range []string{`"request_id":"chain-request"`, `"path":"/protected"`, `"status":204`} {
		if !strings.Contains(logs.String(), expected) {
			t.Fatalf("access log missing %s: %s", expected, logs.String())
		}
	}

	service.authenticateError = auth.ErrUnauthenticated
	logs.Reset()
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || !strings.Contains(logs.String(), `"status":401`) {
		t.Fatalf("auth failure = %d, logs = %s", response.Code, logs.String())
	}
}

func TestProtectedPanicIsObservedByRecoveryAndAccessLog(t *testing.T) {
	var logs bytes.Buffer
	service := &fakeAuthService{}
	protected := httpapi.RequireAuthentication(service)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("protected panic")
	}))
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: slog.New(slog.NewJSONHandler(&logs, nil)), Auth: service, Fallback: protected})
	request := httptest.NewRequest(http.MethodGet, "/protected-panic", nil)
	request.Header.Set("Authorization", "Bearer access-token")
	request.Header.Set("X-Request-ID", "protected-panic-request")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError || service.authenticateCount != 1 {
		t.Fatalf("response = %d, auth calls = %d", response.Code, service.authenticateCount)
	}
	for _, expected := range []string{`"msg":"http panic recovered"`, `"request_id":"protected-panic-request"`, `"status":500`} {
		if !strings.Contains(logs.String(), expected) {
			t.Fatalf("logs missing %s: %s", expected, logs.String())
		}
	}
}

func TestAuthenticationSensitiveValuesNeverReachResponseOrLogs(t *testing.T) {
	secrets := []string{
		"plain-password-secret", "$2a$11$password-hash-secret", "signing-key-secret-1234567890123456",
		"session-token-id-secret", "authorization-token-secret",
	}
	var logs bytes.Buffer
	service := &fakeAuthService{loginError: errors.New(strings.Join(secrets, " "))}
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: slog.New(slog.NewJSONHandler(&logs, nil)), Auth: service})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"viewer","password":"plain-password-secret"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer authorization-token-secret")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", response.Code)
	}
	combined := response.Body.String() + logs.String()
	for _, secret := range secrets {
		if strings.Contains(combined, secret) {
			t.Fatalf("response/logs contain %q: %s", secret, combined)
		}
	}
}

func TestKnownAndUnknownAuthenticationFailuresAreIndistinguishable(t *testing.T) {
	responses := make([]string, 0, 2)
	for _, username := range []string{"viewer", "unknown"} {
		var logs bytes.Buffer
		service := &fakeAuthService{loginError: auth.ErrAuthenticationFailed}
		handler := httpapi.NewHandler(httpapi.Dependencies{Logger: slog.New(slog.NewJSONHandler(&logs, nil)), Auth: service})
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"`+username+`","password":"123456789012"}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-Request-ID", "same-request")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s status = %d", username, response.Code)
		}
		responses = append(responses, response.Body.String())
		if strings.Contains(logs.String(), username) || strings.Contains(logs.String(), "known") || strings.Contains(logs.String(), "unknown") {
			t.Fatalf("%s failure logs reveal identity category: %s", username, logs.String())
		}
	}
	if responses[0] != responses[1] {
		t.Fatalf("known and unknown responses differ: %q != %q", responses[0], responses[1])
	}
}

func TestLoginPasswordByteBoundaries(t *testing.T) {
	tests := []struct {
		name       string
		password   string
		wantStatus int
		wantCalls  int
	}{
		{name: "ASCII 11 bytes", password: strings.Repeat("a", 11), wantStatus: http.StatusBadRequest},
		{name: "ASCII 12 bytes", password: strings.Repeat("a", 12), wantStatus: http.StatusOK, wantCalls: 1},
		{name: "ASCII 72 bytes", password: strings.Repeat("a", 72), wantStatus: http.StatusOK, wantCalls: 1},
		{name: "ASCII 73 bytes", password: strings.Repeat("a", 73), wantStatus: http.StatusBadRequest},
		{name: "Unicode 11 bytes", password: "密密密ab", wantStatus: http.StatusBadRequest},
		{name: "Unicode 12 bytes", password: "密码密码密码密码", wantStatus: http.StatusOK, wantCalls: 1},
		{name: "Unicode 72 bytes", password: strings.Repeat("密", 24), wantStatus: http.StatusOK, wantCalls: 1},
		{name: "Unicode 73 bytes", password: strings.Repeat("密", 24) + "a", wantStatus: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeAuthService{}
			handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: service})
			request := authRequest(handler, http.MethodPost, "/api/v1/auth/login", `{"username":"viewer","password":"`+test.password+`"}`)
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus || service.loginCount != test.wantCalls {
				t.Fatalf("status = %d, calls = %d", response.Code, service.loginCount)
			}
		})
	}
}

func TestLoginLimiterFailureClassification(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	limiter := httpapi.NewLoginLimiter(func() time.Time { return now })
	service := &fakeAuthService{loginErrors: []error{
		auth.ErrAuthenticationFailed,
		auth.ErrAuthenticationFailed,
		context.Canceled,
		context.DeadlineExceeded,
		auth.ErrAuthenticationFailed,
		auth.ErrAuthenticationFailed,
		auth.ErrAuthenticationFailed,
	}}
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: service, LoginLimiter: limiter})

	for _, invalid := range []struct {
		body        string
		contentType string
		want        int
	}{
		{body: `{}`, contentType: "text/plain", want: http.StatusUnsupportedMediaType},
		{body: `{"username":"viewer","password":"short"}`, contentType: "application/json", want: http.StatusBadRequest},
	} {
		request := authRequest(handler, http.MethodPost, "/api/v1/auth/login", invalid.body)
		request.Header.Set("Content-Type", invalid.contentType)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != invalid.want {
			t.Fatalf("invalid request status = %d, want %d", response.Code, invalid.want)
		}
	}
	if service.loginCount != 0 {
		t.Fatalf("invalid requests called service %d times", service.loginCount)
	}

	requestLogin := func() *httptest.ResponseRecorder {
		request := authRequest(handler, http.MethodPost, "/api/v1/auth/login", `{"username":"viewer","password":"123456789012"}`)
		request.Header.Set("Content-Type", "application/json")
		request.RemoteAddr = "192.0.2.10:1234"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}
	for attempt := 0; attempt < 2; attempt++ {
		if response := requestLogin(); response.Code != http.StatusUnauthorized {
			t.Fatalf("failure %d status = %d", attempt, response.Code)
		}
	}
	if response := requestLogin(); response.Code != http.StatusInternalServerError {
		t.Fatalf("internal error status = %d", response.Code)
	}
	if response := requestLogin(); response.Code != http.StatusInternalServerError {
		t.Fatalf("timeout error status = %d", response.Code)
	}
	for attempt := 0; attempt < 3; attempt++ {
		if response := requestLogin(); response.Code != http.StatusUnauthorized {
			t.Fatalf("failure %d after internal error status = %d", attempt, response.Code)
		}
	}
	if response := requestLogin(); response.Code != http.StatusTooManyRequests || service.loginCount != 7 {
		t.Fatalf("rate limit status = %d, service calls = %d", response.Code, service.loginCount)
	}
}

func TestLoginLimiterIgnoresForwardedHeaders(t *testing.T) {
	service := &fakeAuthService{loginError: auth.ErrAuthenticationFailed}
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: service})
	for attempt := 0; attempt < 6; attempt++ {
		request := authRequest(handler, http.MethodPost, "/api/v1/auth/login", `{"username":"viewer","password":"123456789012"}`)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-Forwarded-For", "198.51.100."+string(rune('1'+attempt)))
		request.Header.Set("Forwarded", "for=203.0.113.10")
		request.RemoteAddr = "192.0.2.20:1234"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if attempt == 5 && response.Code != http.StatusTooManyRequests {
			t.Fatalf("sixth status = %d", response.Code)
		}
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
	loginErrors       []error
	loginUsername     string
	loginCount        int
	logoutCount       int
	authenticatedRole auth.Role
	authenticateError error
	authenticateCount int
}

func (service *fakeAuthService) Login(_ context.Context, username, _ string) (auth.LoginResult, error) {
	service.loginUsername = username
	service.loginCount++
	if len(service.loginErrors) > 0 {
		err := service.loginErrors[0]
		service.loginErrors = service.loginErrors[1:]
		if err != nil {
			return auth.LoginResult{}, err
		}
	}
	if service.loginError != nil {
		return auth.LoginResult{}, service.loginError
	}
	return auth.LoginResult{AccessToken: "access-token", ExpiresAt: time.Date(2026, 7, 12, 12, 15, 0, 0, time.UTC), User: auth.User{ID: "user-1", Username: "viewer", Role: auth.RoleViewer}}, nil
}
func (service *fakeAuthService) Authenticate(_ context.Context, token string) (auth.Principal, error) {
	service.authenticateCount++
	if service.authenticateError != nil {
		return auth.Principal{}, service.authenticateError
	}
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
