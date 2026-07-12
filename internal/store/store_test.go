package store_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/magicvr/allinme.core-api/internal/store"
)

func TestOpenConfiguresPragmasAndMigrate(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "allinme.db")
	database, err := store.Open(ctx, path, store.OpenCreate)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer database.Close()

	var foreignKeys, busyTimeout int
	var journalMode string
	if err := database.SQL().QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 || busyTimeout != 5000 || journalMode != "wal" {
		t.Fatalf("pragmas = %d, %d, %q", foreignKeys, busyTimeout, journalMode)
	}

	first, err := database.Migrate(ctx)
	if err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if first.FromVersion != 0 || first.ToVersion != 3 {
		t.Fatalf("first migration = %+v", first)
	}
	second, err := database.Migrate(ctx)
	if err != nil {
		t.Fatalf("second Migrate() error = %v", err)
	}
	if second.FromVersion != 3 || second.ToVersion != 3 {
		t.Fatalf("second migration = %+v", second)
	}
}

func TestVersionOneDatabaseIsOutdatedAndUpgradesWithoutLosingSeeds(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "allinme.db")
	database, err := store.Open(ctx, path, store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.SQL().ExecContext(ctx, `
		CREATE TABLE seed_versions (name TEXT PRIMARY KEY, version INTEGER NOT NULL, applied_at TEXT NOT NULL);
		INSERT INTO seed_versions(name, version, applied_at) VALUES ('runtime', 1, 'preserved');
		PRAGMA user_version = 1;
	`); err != nil {
		database.Close()
		t.Fatal(err)
	}
	database.Close()

	probe := store.NewProbe(path)
	if status := probe.Check(ctx); status != store.SchemaOutdated {
		t.Fatalf("readiness = %q, want %q", status, store.SchemaOutdated)
	}

	database, err = store.Open(ctx, path, store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	result, err := database.Migrate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.FromVersion != 1 || result.ToVersion != 3 {
		t.Fatalf("migration = %+v", result)
	}
	var appliedAt string
	if err := database.SQL().QueryRowContext(ctx, "SELECT applied_at FROM seed_versions WHERE name = 'runtime'").Scan(&appliedAt); err != nil {
		t.Fatal(err)
	}
	if appliedAt != "preserved" {
		t.Fatalf("applied_at = %q, want preserved", appliedAt)
	}
}

func TestVersionTwoDatabaseUpgradesWithoutLosingAuthenticationData(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "allinme.db")
	database, err := store.Open(ctx, path, store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.SQL().ExecContext(ctx, `
		CREATE TABLE seed_versions (name TEXT PRIMARY KEY, version INTEGER NOT NULL, applied_at TEXT NOT NULL);
		CREATE TABLE users (
			id TEXT PRIMARY KEY, username TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL,
			role TEXT NOT NULL CHECK (role IN ('viewer', 'operator', 'approver', 'admin')),
			disabled_at TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL
		);
		CREATE TABLE sessions (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id), token_id TEXT NOT NULL UNIQUE,
			expires_at TEXT NOT NULL, revoked_at TEXT, created_at TEXT NOT NULL
		);
		INSERT INTO users(id, username, password_hash, role, created_at, updated_at)
		VALUES ('user-preserved', 'viewer', 'hash', 'viewer', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO sessions(id, user_id, token_id, expires_at, created_at)
		VALUES ('session-preserved', 'user-preserved', 'token-preserved', '2027-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		PRAGMA user_version = 2;
	`); err != nil {
		database.Close()
		t.Fatal(err)
	}
	result, err := database.Migrate(ctx)
	if err != nil {
		database.Close()
		t.Fatal(err)
	}
	defer database.Close()
	if result.FromVersion != 2 || result.ToVersion != 3 {
		t.Fatalf("migration = %+v", result)
	}
	var users, sessions, idempotencyTables int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM users WHERE id = 'user-preserved'`).Scan(&users); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM sessions WHERE id = 'session-preserved'`).Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'idempotency_keys'`).Scan(&idempotencyTables); err != nil {
		t.Fatal(err)
	}
	if users != 1 || sessions != 1 || idempotencyTables != 0 {
		t.Fatalf("users = %d, sessions = %d, idempotency tables = %d", users, sessions, idempotencyTables)
	}
}

func TestWithTxRollsBackCallbackFailure(t *testing.T) {
	ctx := context.Background()
	database := openMigrated(t)
	errSentinel := errors.New("callback failed")
	err := database.WithTx(ctx, func(transaction *sql.Tx) error {
		if _, err := transaction.ExecContext(ctx, "INSERT INTO seed_versions(name, version, applied_at) VALUES ('test', 1, 'now')"); err != nil {
			return err
		}
		return errSentinel
	})
	if !errors.Is(err, errSentinel) {
		t.Fatalf("WithTx() error = %v", err)
	}
	var count int
	if err := database.SQL().QueryRowContext(ctx, "SELECT COUNT(*) FROM seed_versions").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("row count = %d, want 0", count)
	}
}

func TestProbeDoesNotCreateMissingDatabaseAndRecovers(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "allinme.db")
	probe := store.NewProbe(path)
	if status := probe.Check(ctx); status != store.DatabaseMissing {
		t.Fatalf("missing status = %q", status)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("database was created: %v", err)
	}
	database := openMigratedAt(t, path)
	database.Close()
	if status := probe.Check(ctx); status != store.Ready {
		t.Fatalf("ready status = %q", status)
	}
	probe.Close()
	if status := probe.Check(ctx); status != store.DatabaseUnavailable {
		t.Fatalf("closed status = %q", status)
	}
}

func TestClassifySchemaVersion(t *testing.T) {
	tests := []struct {
		version int
		latest  int
		want    store.ReadinessStatus
	}{
		{version: 0, latest: 1, want: store.SchemaUninitialized},
		{version: 1, latest: 2, want: store.SchemaOutdated},
		{version: 2, latest: 2, want: store.Ready},
		{version: 3, latest: 2, want: store.SchemaTooNew},
	}
	for _, test := range tests {
		if got := store.ClassifySchemaVersion(test.version, test.latest); got != test.want {
			t.Errorf("ClassifySchemaVersion(%d, %d) = %q, want %q", test.version, test.latest, got, test.want)
		}
	}
}

func openMigrated(t *testing.T) *store.DB {
	return openMigratedAt(t, filepath.Join(t.TempDir(), "allinme.db"))
}

func openMigratedAt(t *testing.T, path string) *store.DB {
	t.Helper()
	database, err := store.Open(context.Background(), path, store.OpenCreate)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if _, err := database.Migrate(context.Background()); err != nil {
		database.Close()
		t.Fatalf("Migrate() error = %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}
