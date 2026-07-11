package admin_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/magicvr/allinme.core-api/internal/admin"
	"github.com/magicvr/allinme.core-api/internal/config"
	"github.com/magicvr/allinme.core-api/internal/store"
)

func TestMigrateSeedAndReset(t *testing.T) {
	configuration := developmentConfig(t)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	var output bytes.Buffer
	for _, command := range []string{"migrate", "migrate", "seed", "seed"} {
		output.Reset()
		if err := admin.Run(context.Background(), configuration, []string{command}, &output, logger); err != nil {
			t.Fatalf("%s error = %v", command, err)
		}
		if output.Len() == 0 {
			t.Fatalf("%s produced no summary", command)
		}
	}

	unrelated := filepath.Join(configuration.DataDir, "keep.txt")
	if err := os.WriteFile(unrelated, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := admin.Run(context.Background(), configuration, []string{"reset"}, &output, logger); err != nil {
		t.Fatalf("reset error = %v", err)
	}
	if _, err := os.Stat(unrelated); err != nil {
		t.Fatalf("unrelated file removed: %v", err)
	}
	database, err := store.Open(context.Background(), configuration.DatabasePath, store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var version int
	if err := database.SQL().QueryRow("SELECT version FROM seed_versions WHERE name = 'runtime'").Scan(&version); err != nil || version != 1 {
		t.Fatalf("runtime seed version = %d, error = %v", version, err)
	}
}

func TestProductionResetIsRejectedBeforeDeletion(t *testing.T) {
	dataDir := t.TempDir()
	configuration, err := config.Load(mapLookup(map[string]string{"APP_ENV": "production", "PORT": "8080", "DATA_DIR": dataDir}))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configuration.DatabasePath, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := admin.Run(context.Background(), configuration, []string{"reset"}, io.Discard, nil); err == nil {
		t.Fatal("reset error = nil")
	}
	contents, err := os.ReadFile(configuration.DatabasePath)
	if err != nil || string(contents) != "preserve" {
		t.Fatalf("database changed: %q, %v", contents, err)
	}
}

func TestSeedRejectsNewerVersionWithoutModification(t *testing.T) {
	configuration := developmentConfig(t)
	if err := admin.Run(context.Background(), configuration, []string{"migrate"}, io.Discard, nil); err != nil {
		t.Fatal(err)
	}
	database, err := store.Open(context.Background(), configuration.DatabasePath, store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.SQL().Exec("INSERT INTO seed_versions(name, version, applied_at) VALUES ('runtime', 2, 'future')"); err != nil {
		database.Close()
		t.Fatal(err)
	}
	database.Close()

	if err := admin.Run(context.Background(), configuration, []string{"seed"}, io.Discard, nil); err == nil {
		t.Fatal("seed error = nil")
	}
	database, err = store.Open(context.Background(), configuration.DatabasePath, store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var version int
	var appliedAt string
	if err := database.SQL().QueryRow("SELECT version, applied_at FROM seed_versions WHERE name = 'runtime'").Scan(&version, &appliedAt); err != nil {
		t.Fatal(err)
	}
	if version != 2 || appliedAt != "future" {
		t.Fatalf("seed changed to version=%d applied_at=%q", version, appliedAt)
	}
}

func TestUnknownCommandIsRejected(t *testing.T) {
	if err := admin.Run(context.Background(), developmentConfig(t), []string{"unknown"}, io.Discard, nil); err == nil {
		t.Fatal("Run() error = nil")
	}
}

func developmentConfig(t *testing.T) config.Config {
	t.Helper()
	configuration, err := config.Load(mapLookup(map[string]string{"DATA_DIR": t.TempDir()}))
	if err != nil {
		t.Fatal(err)
	}
	return configuration
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
