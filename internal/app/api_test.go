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
	if _, err := database.SeedAuthDemo(ctx, passwords, "123456789012", time.Now(), auth.RandomID); err != nil {
		database.Close()
		t.Fatal(err)
	}
	database.Close()

	configuration := config.APIConfig{Config: base, JWTSigningKey: []byte("12345678901234567890123456789012")}
	application, err := app.NewAuthenticatedAPI(configuration, slog.New(slog.NewJSONHandler(io.Discard, nil)))
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
		if role == "admin" {
			if err := json.NewDecoder(response.Body).Decode(&login); err != nil {
				t.Fatal(err)
			}
		}
	}

	application.Close()
	reopened, err := app.NewAuthenticatedAPI(configuration, slog.New(slog.NewJSONHandler(io.Discard, nil)))
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
	reopened, err = app.NewAuthenticatedAPI(otherKey, slog.New(slog.NewJSONHandler(io.Discard, nil)))
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
