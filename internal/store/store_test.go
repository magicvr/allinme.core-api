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
	if first.FromVersion != 0 || first.ToVersion != 1 {
		t.Fatalf("first migration = %+v", first)
	}
	second, err := database.Migrate(ctx)
	if err != nil {
		t.Fatalf("second Migrate() error = %v", err)
	}
	if second.FromVersion != 1 || second.ToVersion != 1 {
		t.Fatalf("second migration = %+v", second)
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
