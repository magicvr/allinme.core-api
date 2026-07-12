package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/admin"
	"github.com/magicvr/allinme.core-api/internal/app"
	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/config"
	"github.com/magicvr/allinme.core-api/internal/store"
)

func TestAPIReadinessTransitionsWithoutRestart(t *testing.T) {
	dataDir := t.TempDir()
	configuration, err := config.Load(mapLookup(map[string]string{"DATA_DIR": dataDir}))
	if err != nil {
		t.Fatal(err)
	}
	application, err := app.NewAPI(configuration, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(application.Close)

	if status := requestStatus(application.Handler(), "/healthz"); status != http.StatusOK {
		t.Fatalf("health status = %d", status)
	}
	if status := requestStatus(application.Handler(), "/readyz"); status != http.StatusServiceUnavailable {
		t.Fatalf("initial ready status = %d", status)
	}

	database, err := store.Open(context.Background(), filepath.Join(dataDir, "allinme.db"), store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Migrate(context.Background()); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if _, err := database.Seed(context.Background()); err != nil {
		database.Close()
		t.Fatal(err)
	}
	database.Close()

	if status := requestStatus(application.Handler(), "/readyz"); status != http.StatusOK {
		t.Fatalf("ready status after migrate = %d", status)
	}
	application.Close()
	if status := requestStatus(application.Handler(), "/readyz"); status != http.StatusServiceUnavailable {
		t.Fatalf("ready status after close = %d", status)
	}
	reopened, err := app.NewAPI(configuration, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if status := requestStatus(reopened.Handler(), "/readyz"); status != http.StatusOK {
		t.Fatalf("ready status after reopen = %d", status)
	}
}

func TestAuthenticatedAPIFlowWithSQLite(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	dataDir := t.TempDir()
	base, err := config.LoadBase(mapLookup(map[string]string{"DATA_DIR": dataDir}))
	if err != nil {
		t.Fatal(err)
	}
	database, err := store.Open(ctx, base.DatabasePath, store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Migrate(ctx); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if _, err := database.Seed(ctx); err != nil {
		database.Close()
		t.Fatal(err)
	}
	passwords, err := auth.NewPasswords()
	if err != nil {
		database.Close()
		t.Fatal(err)
	}
	if _, err := database.SeedAuthDemo(ctx, passwords, "123456789012", now, auth.RandomID); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if _, err := database.SeedOrderDemo(ctx, now); err != nil {
		database.Close()
		t.Fatal(err)
	}
	database.Close()

	configuration := config.APIConfig{Config: base, JWTSigningKey: []byte("12345678901234567890123456789012")}
	sequence := 0
	dependencies := app.AuthDependencies{
		Clock: func() time.Time { return now }, LimiterClock: func() time.Time { return now },
		NewID: func() (string, error) { sequence++; return "id-" + strconv.Itoa(sequence), nil },
	}
	application, err := app.NewAuthenticatedAPIWithDependencies(configuration, dependencies, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(application.Close)

	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"viewer","password":"123456789012"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginResponse := httptest.NewRecorder()
	application.Handler().ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("login = %d %s", loginResponse.Code, loginResponse.Body.String())
	}
	var login struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.NewDecoder(loginResponse.Body).Decode(&login); err != nil || login.AccessToken == "" {
		t.Fatalf("decode login: %v", err)
	}

	requestWithToken := func(method, path string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(method, path, nil)
		request.Header.Set("Authorization", "Bearer "+login.AccessToken)
		response := httptest.NewRecorder()
		application.Handler().ServeHTTP(response, request)
		return response
	}
	if response := requestWithToken(http.MethodGet, "/api/v1/auth/me"); response.Code != http.StatusOK {
		t.Fatalf("me = %d %s", response.Code, response.Body.String())
	}
	if response := requestWithToken(http.MethodGet, "/api/v1/orders?pageSize=2&status=DRAFT"); response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"total":1`)) || bytes.Contains(response.Body.Bytes(), []byte(`"items":[{"id":"itm_`)) {
		t.Fatalf("orders = %d %s", response.Code, response.Body.String())
	}
	if response := requestWithToken(http.MethodGet, "/api/v1/orders/ord_00000000000000000000000000000001"); response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"items":[{"id":"itm_`)) {
		t.Fatalf("order detail = %d %s", response.Code, response.Body.String())
	}
	if response := requestWithToken(http.MethodHead, "/api/v1/orders"); response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("orders HEAD = %d Allow=%q %s", response.Code, response.Header().Get("Allow"), response.Body.String())
	}
	if response := requestWithToken(http.MethodPost, "/api/v1/auth/logout"); response.Code != http.StatusNoContent {
		t.Fatalf("logout = %d %s", response.Code, response.Body.String())
	}
	if response := requestWithToken(http.MethodGet, "/api/v1/auth/me"); response.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout = %d %s", response.Code, response.Body.String())
	}
	if status := requestStatus(application.Handler(), "/healthz"); status != http.StatusOK {
		t.Fatalf("health = %d", status)
	}
	if status := requestStatus(application.Handler(), "/readyz"); status != http.StatusOK {
		t.Fatalf("readiness = %d", status)
	}
	for _, role := range []string{"viewer", "operator", "approver", "admin"} {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"`+role+`","password":"123456789012"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		application.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"role":"`+role+`"`)) {
			t.Fatalf("%s login = %d %s", role, response.Code, response.Body.String())
		}
		var roleLogin struct {
			AccessToken string `json:"accessToken"`
		}
		if err := json.NewDecoder(response.Body).Decode(&roleLogin); err != nil || roleLogin.AccessToken == "" {
			t.Fatalf("decode %s login: %v", role, err)
		}
		roleRequest := httptest.NewRequest(http.MethodGet, "/api/v1/orders?pageSize=1", nil)
		roleRequest.Header.Set("Authorization", "Bearer "+roleLogin.AccessToken)
		roleResponse := httptest.NewRecorder()
		application.Handler().ServeHTTP(roleResponse, roleRequest)
		if roleResponse.Code != http.StatusOK {
			t.Fatalf("%s order list = %d %s", role, roleResponse.Code, roleResponse.Body.String())
		}
		if role == "admin" {
			login.AccessToken = roleLogin.AccessToken
		}
	}
	for _, requestCase := range []struct{ method, path string }{
		{http.MethodPost, "/api/v1/orders"},
		{http.MethodPatch, "/api/v1/orders/ord_00000000000000000000000000000001"},
	} {
		if response := requestWithToken(requestCase.method, requestCase.path); response.Code != http.StatusNotFound {
			t.Fatalf("disabled order route %s %s = %d %s", requestCase.method, requestCase.path, response.Code, response.Body.String())
		}
	}

	application.Close()
	reopened, err := app.NewAuthenticatedAPIWithDependencies(configuration, dependencies, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	request.Header.Set("Authorization", "Bearer "+login.AccessToken)
	response := httptest.NewRecorder()
	reopened.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("same-key reopen = %d %s", response.Code, response.Body.String())
	}
	reopened.Close()

	otherKey := configuration
	otherKey.JWTSigningKey = []byte("abcdefghijklmnopqrstuvwxyz123456")
	reopened, err = app.NewAuthenticatedAPIWithDependencies(otherKey, dependencies, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	response = httptest.NewRecorder()
	reopened.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("different-key reopen = %d %s", response.Code, response.Body.String())
	}
}

func TestProductionBootstrapToAuthenticatedAPILogin(t *testing.T) {
	dataDir := t.TempDir()
	values := map[string]string{
		"APP_ENV": "production", "PORT": "8080", "DATA_DIR": dataDir,
		"BOOTSTRAP_ADMIN_USERNAME": " Root ", "BOOTSTRAP_ADMIN_PASSWORD": "123456789012",
		"JWT_SIGNING_KEY": "12345678901234567890123456789012",
	}
	if err := admin.Execute(context.Background(), mapLookup(values), []string{"migrate"}, io.Discard, nil); err != nil {
		t.Fatal(err)
	}
	if err := admin.Execute(context.Background(), mapLookup(values), []string{"bootstrap-admin"}, io.Discard, nil); err != nil {
		t.Fatal(err)
	}
	if err := admin.Execute(context.Background(), mapLookup(values), []string{"bootstrap-admin"}, io.Discard, nil); err == nil {
		t.Fatal("repeat bootstrap error = nil")
	}
	configuration, err := config.LoadAPI(mapLookup(values))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	sequence := 0
	application, err := app.NewAuthenticatedAPIWithDependencies(configuration, app.AuthDependencies{
		Clock: func() time.Time { return now }, LimiterClock: func() time.Time { return now },
		NewID: func() (string, error) { sequence++; return "production-id-" + strconv.Itoa(sequence), nil },
	}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()
	response := loginResponse(application.Handler(), "root", "123456789012", "production-login")
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"username":"root"`)) || !bytes.Contains(response.Body.Bytes(), []byte(`"role":"admin"`)) {
		t.Fatalf("production login = %d %s", response.Code, response.Body.String())
	}
}

func TestAuthenticatedAPIRejectsNegativeIdentityAndSessionStates(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	clock := now
	dataDir := t.TempDir()
	base, err := config.LoadBase(mapLookup(map[string]string{"DATA_DIR": dataDir}))
	if err != nil {
		t.Fatal(err)
	}
	database, err := store.Open(ctx, base.DatabasePath, store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Migrate(ctx); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if _, err := database.Seed(ctx); err != nil {
		database.Close()
		t.Fatal(err)
	}
	passwords, _ := auth.NewPasswords()
	seedSequence := 0
	if _, err := database.SeedAuthDemo(ctx, passwords, "123456789012", now, func() (string, error) {
		seedSequence++
		return "user-" + strconv.Itoa(seedSequence), nil
	}); err != nil {
		database.Close()
		t.Fatal(err)
	}
	database.Close()

	configuration := config.APIConfig{Config: base, JWTSigningKey: []byte("12345678901234567890123456789012")}
	idSequence := 0
	application, err := app.NewAuthenticatedAPIWithDependencies(configuration, app.AuthDependencies{
		Clock: func() time.Time { return clock }, LimiterClock: func() time.Time { return clock },
		NewID: func() (string, error) { idSequence++; return "session-id-" + strconv.Itoa(idSequence), nil },
	}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()

	wrong := loginResponse(application.Handler(), "viewer", "wrong-password", "same-failure")
	unknown := loginResponse(application.Handler(), "unknown", "123456789012", "same-failure")
	if wrong.Code != http.StatusUnauthorized || unknown.Code != http.StatusUnauthorized || wrong.Body.String() != unknown.Body.String() {
		t.Fatalf("login failures differ: %d %s / %d %s", wrong.Code, wrong.Body.String(), unknown.Code, unknown.Body.String())
	}

	login := loginResponse(application.Handler(), "viewer", "123456789012", "valid-login")
	if login.Code != http.StatusOK {
		t.Fatalf("login = %d %s", login.Code, login.Body.String())
	}
	var body struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.NewDecoder(login.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	requestMe := func(token string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		application.Handler().ServeHTTP(response, request)
		return response
	}
	if response := requestMe(body.AccessToken + "tampered"); response.Code != http.StatusUnauthorized {
		t.Fatalf("tampered token status = %d", response.Code)
	}

	mutator, err := store.Open(ctx, base.DatabasePath, store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	defer mutator.Close()
	if _, err := mutator.SQL().Exec(`UPDATE sessions SET user_id = (SELECT id FROM users WHERE username = 'operator') WHERE user_id = (SELECT id FROM users WHERE username = 'viewer')`); err != nil {
		t.Fatal(err)
	}
	if response := requestMe(body.AccessToken); response.Code != http.StatusUnauthorized {
		t.Fatalf("subject mismatch status = %d", response.Code)
	}
	if _, err := mutator.SQL().Exec(`UPDATE sessions SET user_id = (SELECT id FROM users WHERE username = 'viewer')`); err != nil {
		t.Fatal(err)
	}
	if _, err := mutator.SQL().Exec(`UPDATE users SET disabled_at = ? WHERE username = 'viewer'`, now.Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	if response := requestMe(body.AccessToken); response.Code != http.StatusUnauthorized {
		t.Fatalf("disabled user status = %d", response.Code)
	}
	if _, err := mutator.SQL().Exec(`UPDATE users SET disabled_at = NULL WHERE username = 'viewer'`); err != nil {
		t.Fatal(err)
	}
	clock = now.Add(auth.TokenTTL + auth.ClockLeeway)
	if response := requestMe(body.AccessToken); response.Code != http.StatusUnauthorized {
		t.Fatalf("expired token status = %d", response.Code)
	}
}

func loginResponse(handler http.Handler, username, password, requestID string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"`+username+`","password":"`+password+`"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Request-ID", requestID)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestAPINotReadyDatabaseStatesKeepHealthLive(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*testing.T, config.Config)
	}{
		{name: "missing"},
		{name: "uninitialized", prepare: func(t *testing.T, configuration config.Config) {
			database, err := store.Open(context.Background(), configuration.DatabasePath, store.OpenCreate)
			if err != nil {
				t.Fatal(err)
			}
			database.Close()
		}},
		{name: "corrupt", prepare: func(t *testing.T, configuration config.Config) {
			if err := os.WriteFile(configuration.DatabasePath, []byte("not a database"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "too new", prepare: func(t *testing.T, configuration config.Config) {
			database, err := store.Open(context.Background(), configuration.DatabasePath, store.OpenCreate)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := database.SQL().Exec("PRAGMA user_version = " + strconv.Itoa(store.LatestSchemaVersion()+1)); err != nil {
				database.Close()
				t.Fatal(err)
			}
			database.Close()
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configuration, err := config.Load(mapLookup(map[string]string{"DATA_DIR": t.TempDir()}))
			if err != nil {
				t.Fatal(err)
			}
			if test.prepare != nil {
				test.prepare(t, configuration)
			}
			application, err := app.NewAPI(configuration, slog.New(slog.NewJSONHandler(io.Discard, nil)))
			if err != nil {
				t.Fatal(err)
			}
			defer application.Close()
			if status := requestStatus(application.Handler(), "/healthz"); status != http.StatusOK {
				t.Fatalf("health status = %d", status)
			}
			if status := requestStatus(application.Handler(), "/readyz"); status != http.StatusServiceUnavailable {
				t.Fatalf("ready status = %d", status)
			}
		})
	}
}

func requestStatus(handler http.Handler, path string) int {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response.Code
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
