package store

import (
	"context"
	"database/sql"
	"path/filepath"
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
