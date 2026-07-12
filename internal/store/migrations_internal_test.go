package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"strconv"
	"testing"
)

func TestFailedMigrationRollsBackSchemaAndVersion(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, filepath.Join(t.TempDir(), "allinme.db"), OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	err = database.WithTx(ctx, func(transaction *sql.Tx) error {
		if _, err := transaction.ExecContext(ctx, "CREATE TABLE partial_change (id INTEGER PRIMARY KEY)"); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "THIS IS NOT SQL"); err != nil {
			return err
		}
		_, err := transaction.ExecContext(ctx, "PRAGMA user_version = 1")
		return err
	})
	if err == nil {
		t.Fatal("migration error = nil")
	}
	version, err := database.SchemaVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if version != 0 {
		t.Fatalf("schema version = %d, want 0", version)
	}
	var tableCount int
	if err := database.SQL().QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'partial_change'").Scan(&tableCount); err != nil {
		t.Fatal(err)
	}
	if tableCount != 0 {
		t.Fatalf("partial table count = %d, want 0", tableCount)
	}
}

func TestOrderMigrationFailureKeepsVersionTwoAndRollsBackPartialSchema(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, filepath.Join(t.TempDir(), "allinme.db"), OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations[:2] {
		if err := database.WithTx(ctx, func(transaction *sql.Tx) error {
			if _, err := transaction.ExecContext(ctx, migration.sql); err != nil {
				return err
			}
			_, err := transaction.ExecContext(ctx, "PRAGMA user_version = "+strconv.Itoa(migration.version))
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := database.SQL().ExecContext(ctx, `CREATE TABLE orders (id TEXT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Migrate(ctx); err == nil {
		t.Fatal("v3 migration error = nil")
	}
	version, err := database.SchemaVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if version != 2 {
		t.Fatalf("schema version = %d, want 2", version)
	}
	var itemTables int
	if err := database.SQL().QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'order_items'`).Scan(&itemTables); err != nil {
		t.Fatal(err)
	}
	if itemTables != 0 {
		t.Fatalf("partial order_items tables = %d", itemTables)
	}
}

func TestIdempotencyMigrationFailureKeepsVersionThreeAndRollsBackPartialSchema(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, filepath.Join(t.TempDir(), "allinme.db"), OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations[:3] {
		if err := database.WithTx(ctx, func(transaction *sql.Tx) error {
			if _, err := transaction.ExecContext(ctx, migration.sql); err != nil {
				return err
			}
			_, err := transaction.ExecContext(ctx, "PRAGMA user_version = "+strconv.Itoa(migration.version))
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := database.SQL().ExecContext(ctx, `
		CREATE TABLE migration_collision (id TEXT PRIMARY KEY);
		CREATE INDEX idempotency_keys_order_id_idx ON migration_collision(id);
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Migrate(ctx); err == nil {
		t.Fatal("v4 migration error = nil")
	}
	version, err := database.SchemaVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if version != 3 {
		t.Fatalf("schema version = %d, want 3", version)
	}
	var idempotencyTables int
	if err := database.SQL().QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'idempotency_keys'`).Scan(&idempotencyTables); err != nil {
		t.Fatal(err)
	}
	if idempotencyTables != 0 {
		t.Fatalf("partial idempotency tables = %d", idempotencyTables)
	}
}
