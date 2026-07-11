package app_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/magicvr/allinme.core-api/internal/app"
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
			if _, err := database.SQL().Exec("PRAGMA user_version = 2"); err != nil {
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
