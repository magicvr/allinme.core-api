package admin_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	database.Close()
	values := map[string]string{"DATA_DIR": configuration.DataDir, "DEMO_ACCOUNT_PASSWORD": "123456789012"}
	if err := admin.Execute(context.Background(), mapLookup(values), []string{"reset"}, &output, logger); err != nil {
		t.Fatalf("demo reset error = %v", err)
	}
	database, err = store.Open(context.Background(), configuration.DatabasePath, store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var orders, refunds int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&orders); err != nil || orders != 10 {
		t.Fatalf("reset demo orders = %d, error = %v", orders, err)
	}
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM refunds`).Scan(&refunds); err != nil || refunds != 5 {
		t.Fatalf("reset demo refunds = %d, error = %v", refunds, err)
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

func TestExecuteDevelopmentSeedAndResetRequirePasswordBeforeDatabaseAccess(t *testing.T) {
	dataDir := t.TempDir()
	values := map[string]string{"DATA_DIR": dataDir}
	for _, command := range []string{"seed", "reset"} {
		if err := admin.Execute(context.Background(), mapLookup(values), []string{command}, io.Discard, nil); err == nil {
			t.Fatalf("%s without password error = nil", command)
		}
		if _, err := os.Stat(filepath.Join(dataDir, "allinme.db")); !os.IsNotExist(err) {
			t.Fatalf("%s touched database before validation: %v", command, err)
		}
	}
	values["DEMO_ACCOUNT_PASSWORD"] = "123456789012"
	var output bytes.Buffer
	if err := admin.Execute(context.Background(), mapLookup(values), []string{"migrate"}, &output, nil); err != nil {
		t.Fatal(err)
	}
	if err := admin.Execute(context.Background(), mapLookup(values), []string{"seed"}, &output, nil); err != nil {
		t.Fatal(err)
	}
	database, err := store.Open(context.Background(), filepath.Join(dataDir, "allinme.db"), store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var users int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM users`).Scan(&users); err != nil || users != 4 {
		t.Fatalf("users = %d, error = %v", users, err)
	}
	var orders, refunds int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&orders); err != nil || orders != 10 {
		t.Fatalf("orders = %d, error = %v", orders, err)
	}
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM refunds`).Scan(&refunds); err != nil || refunds != 5 {
		t.Fatalf("refunds = %d, error = %v", refunds, err)
	}
}

func TestExecuteSeedReportsCommittedRuntimeWhenAuthGroupFails(t *testing.T) {
	dataDir := t.TempDir()
	values := map[string]string{"DATA_DIR": dataDir, "DEMO_ACCOUNT_PASSWORD": "123456789012"}
	if err := admin.Execute(context.Background(), mapLookup(values), []string{"migrate"}, io.Discard, nil); err != nil {
		t.Fatal(err)
	}
	database, err := store.Open(context.Background(), filepath.Join(dataDir, "allinme.db"), store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := database.SQL().Exec(`
		INSERT INTO users(id, username, password_hash, role, created_at, updated_at)
		VALUES ('conflict', 'viewer', 'invalid', 'viewer', ?, ?)
	`, now, now); err != nil {
		database.Close()
		t.Fatal(err)
	}
	database.Close()
	var output bytes.Buffer
	err = admin.Execute(context.Background(), mapLookup(values), []string{"seed"}, &output, nil)
	if err == nil || !strings.Contains(err.Error(), "runtime seed committed") {
		t.Fatalf("seed error = %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("failed seed reported overall success: %s", output.String())
	}
	database, err = store.Open(context.Background(), filepath.Join(dataDir, "allinme.db"), store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var runtimeVersion int
	if err := database.SQL().QueryRow(`SELECT version FROM seed_versions WHERE name = 'runtime'`).Scan(&runtimeVersion); err != nil || runtimeVersion != 1 {
		t.Fatalf("runtime version = %d, error = %v", runtimeVersion, err)
	}
	var authVersions int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM seed_versions WHERE name = 'auth_demo'`).Scan(&authVersions); err != nil || authVersions != 0 {
		t.Fatalf("auth versions = %d, error = %v", authVersions, err)
	}
}

func TestExecuteProductionBootstrapAdmin(t *testing.T) {
	dataDir := t.TempDir()
	values := map[string]string{
		"APP_ENV": "production", "PORT": "8080", "DATA_DIR": dataDir,
		"BOOTSTRAP_ADMIN_USERNAME": " Root ", "BOOTSTRAP_ADMIN_PASSWORD": "123456789012",
	}
	if err := admin.Execute(context.Background(), mapLookup(values), []string{"migrate"}, io.Discard, nil); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := admin.Execute(context.Background(), mapLookup(values), []string{"bootstrap-admin"}, &output, nil); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(output.Bytes(), []byte(`"username":"root"`)) || bytes.Contains(output.Bytes(), []byte(values["BOOTSTRAP_ADMIN_PASSWORD"])) {
		t.Fatalf("bootstrap output = %s", output.String())
	}
	if err := admin.Execute(context.Background(), mapLookup(values), []string{"bootstrap-admin"}, io.Discard, nil); err == nil {
		t.Fatal("repeat bootstrap error = nil")
	}
	values["DEMO_ACCOUNT_PASSWORD"] = "should-not-be-read"
	if err := admin.Execute(context.Background(), mapLookup(values), []string{"seed"}, io.Discard, nil); err != nil {
		t.Fatalf("production seed error = %v", err)
	}
	database, err := store.Open(context.Background(), filepath.Join(dataDir, "allinme.db"), store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var orders int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&orders); err != nil || orders != 0 {
		t.Fatalf("production seed orders = %d, error = %v", orders, err)
	}
	var refunds int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM refunds`).Scan(&refunds); err != nil || refunds != 0 {
		t.Fatalf("production seed refunds = %d, error = %v", refunds, err)
	}
}

func TestExecuteBootstrapAdminRejectsInvalidBoundariesWithoutCreatingUser(t *testing.T) {
	t.Run("development", func(t *testing.T) {
		values := map[string]string{
			"DATA_DIR": t.TempDir(), "BOOTSTRAP_ADMIN_USERNAME": "admin", "BOOTSTRAP_ADMIN_PASSWORD": "123456789012",
		}
		if err := admin.Execute(context.Background(), mapLookup(values), []string{"bootstrap-admin"}, io.Discard, nil); err == nil {
			t.Fatal("development bootstrap error = nil")
		}
		if _, err := os.Stat(filepath.Join(values["DATA_DIR"], "allinme.db")); !os.IsNotExist(err) {
			t.Fatalf("development bootstrap touched database: %v", err)
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		for _, test := range []struct {
			name     string
			username string
			password string
		}{
			{name: "missing username", password: "123456789012"},
			{name: "blank username", username: "   ", password: "123456789012"},
			{name: "short password", username: "admin", password: "short"},
		} {
			t.Run(test.name, func(t *testing.T) {
				dataDir := t.TempDir()
				values := map[string]string{
					"APP_ENV": "production", "PORT": "8080", "DATA_DIR": dataDir,
					"BOOTSTRAP_ADMIN_USERNAME": test.username, "BOOTSTRAP_ADMIN_PASSWORD": test.password,
				}
				if err := admin.Execute(context.Background(), mapLookup(values), []string{"bootstrap-admin"}, io.Discard, nil); err == nil {
					t.Fatal("invalid bootstrap error = nil")
				}
				if _, err := os.Stat(filepath.Join(dataDir, "allinme.db")); !os.IsNotExist(err) {
					t.Fatalf("invalid bootstrap touched database: %v", err)
				}
			})
		}
	})

	t.Run("unmigrated database", func(t *testing.T) {
		dataDir := t.TempDir()
		path := filepath.Join(dataDir, "allinme.db")
		database, err := store.Open(context.Background(), path, store.OpenCreate)
		if err != nil {
			t.Fatal(err)
		}
		database.Close()
		values := productionBootstrapValues(dataDir)
		if err := admin.Execute(context.Background(), mapLookup(values), []string{"bootstrap-admin"}, io.Discard, nil); err == nil {
			t.Fatal("unmigrated bootstrap error = nil")
		}
	})

	t.Run("nonempty users table", func(t *testing.T) {
		dataDir := t.TempDir()
		values := productionBootstrapValues(dataDir)
		if err := admin.Execute(context.Background(), mapLookup(values), []string{"migrate"}, io.Discard, nil); err != nil {
			t.Fatal(err)
		}
		if err := admin.Execute(context.Background(), mapLookup(values), []string{"bootstrap-admin"}, io.Discard, nil); err != nil {
			t.Fatal(err)
		}
		values["BOOTSTRAP_ADMIN_PASSWORD"] = "different-pass"
		if err := admin.Execute(context.Background(), mapLookup(values), []string{"bootstrap-admin"}, io.Discard, nil); err == nil {
			t.Fatal("nonempty bootstrap error = nil")
		}
		database, err := store.Open(context.Background(), filepath.Join(dataDir, "allinme.db"), store.OpenExisting)
		if err != nil {
			t.Fatal(err)
		}
		defer database.Close()
		var users int
		var username string
		if err := database.SQL().QueryRow(`SELECT COUNT(*), MIN(username) FROM users`).Scan(&users, &username); err != nil {
			t.Fatal(err)
		}
		if users != 1 || username != "admin" {
			t.Fatalf("users = %d, username = %q", users, username)
		}
	})
}

func productionBootstrapValues(dataDir string) map[string]string {
	return map[string]string{
		"APP_ENV": "production", "PORT": "8080", "DATA_DIR": dataDir,
		"BOOTSTRAP_ADMIN_USERNAME": "admin", "BOOTSTRAP_ADMIN_PASSWORD": "123456789012",
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
