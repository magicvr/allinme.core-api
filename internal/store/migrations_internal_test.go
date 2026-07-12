package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/order"
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

func TestVersionFourIdempotencyRowsUpgradeAndReplay(t *testing.T) {
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
	for _, migration := range migrations[:4] {
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
	requestJSON := `{"operation":"POST /api/v1/orders","customerName":"Legacy","currency":"CNY","items":[{"sku":"SKU","name":"Item","quantity":1,"unitPrice":100}]}`
	requestDigest := sha256.Sum256([]byte(requestJSON))
	snapshotJSON := `{"order":{"id":"ord_00000000000000000000000000000001","customerName":"Legacy","status":"DRAFT","paymentStatus":"UNPAID","currency":"CNY","totalAmount":100,"version":1,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","items":[{"id":"itm_00000000000000000000000000000001","sku":"SKU","name":"Item","quantity":1,"unitPrice":100}]}}`
	if _, err := database.SQL().ExecContext(ctx, `INSERT INTO idempotency_keys(principal_user_id, method, route, idempotency_key, request_digest, order_id, snapshot_version, snapshot_json, created_at) VALUES ('user-1', 'POST', '/api/v1/orders', 'legacy-key', ?, 'ord_00000000000000000000000000000001', 1, ?, '2026-01-01T00:00:00Z')`, requestDigest[:], snapshotJSON); err != nil {
		t.Fatal(err)
	}
	result, err := database.Migrate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.FromVersion != 4 || result.ToVersion != 5 {
		t.Fatalf("migration = %+v", result)
	}
	var snapshotDigest []byte
	if err := database.SQL().QueryRowContext(ctx, `SELECT snapshot_digest FROM idempotency_keys WHERE idempotency_key = 'legacy-key'`).Scan(&snapshotDigest); err != nil {
		t.Fatal(err)
	}
	expectedSnapshotDigest := sha256.Sum256([]byte(snapshotJSON))
	if string(snapshotDigest) != string(expectedSnapshotDigest[:]) {
		t.Fatalf("legacy snapshot digest = %x, want %x", snapshotDigest, expectedSnapshotDigest)
	}
	service, err := order.NewService(database)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := service.Create(ctx, auth.Principal{UserID: "user-1", Role: auth.RoleOperator}, "legacy-key", order.CreateCommand{CustomerName: "Legacy", Currency: "CNY", Items: []order.ItemCommand{{SKU: "SKU", Name: "Item", Quantity: 1, UnitPrice: 100}}})
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ID != "ord_00000000000000000000000000000001" || replayed.TotalAmount != 100 || len(replayed.Items) != 1 {
		t.Fatalf("legacy replay = %+v", replayed)
	}
}

func TestSnapshotIntegrityMigrationFailureKeepsVersionFour(t *testing.T) {
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
	for _, migration := range migrations[:4] {
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
	if _, err := database.SQL().ExecContext(ctx, `ALTER TABLE idempotency_keys ADD COLUMN snapshot_digest BLOB`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Migrate(ctx); err == nil {
		t.Fatal("v5 migration error = nil")
	}
	version, err := database.SchemaVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if version != 4 {
		t.Fatalf("schema version = %d, want 4", version)
	}
}
