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
	if result.FromVersion != 4 || result.ToVersion != 7 {
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

func TestVersionFiveDatabaseUpgradesWithoutLosingOrdersOrIdempotency(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "allinme.db")
	database, err := Open(ctx, path, OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	migrations, err := loadMigrations()
	if err != nil {
		database.Close()
		t.Fatal(err)
	}
	for _, migration := range migrations[:5] {
		if err := database.WithTx(ctx, func(transaction *sql.Tx) error {
			if _, err := transaction.ExecContext(ctx, migration.sql); err != nil {
				return err
			}
			if migration.version == 5 {
				if err := backfillSnapshotDigests(ctx, transaction); err != nil {
					return err
				}
			}
			_, err := transaction.ExecContext(ctx, "PRAGMA user_version = "+strconv.Itoa(migration.version))
			return err
		}); err != nil {
			database.Close()
			t.Fatal(err)
		}
	}
	requestDigest := sha256.Sum256([]byte("legacy request"))
	snapshotJSON := `{"order":{"id":"ord_00000000000000000000000000000001"}}`
	snapshotDigest := sha256.Sum256([]byte(snapshotJSON))
	if _, err := database.SQL().ExecContext(ctx, `
		INSERT INTO users(id, username, password_hash, role, created_at, updated_at)
		VALUES ('user-preserved', 'preserved', 'hash', 'operator', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO orders(id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at)
		VALUES ('ord_00000000000000000000000000000001', 'Preserved', 'DRAFT', 'UNPAID', 'CNY', 100, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO order_items(id, order_id, position, sku, name, quantity, unit_price)
		VALUES ('itm_00000000000000000000000000000001', 'ord_00000000000000000000000000000001', 0, 'SKU', 'Preserved item', 1, 100);
		INSERT INTO idempotency_keys(
			principal_user_id, method, route, idempotency_key, request_digest, order_id,
			snapshot_version, snapshot_json, snapshot_digest, created_at
		) VALUES (
			'user-preserved', 'POST', '/api/v1/orders', 'preserved-key', ?,
			'ord_00000000000000000000000000000001', 1, ?, ?, '2026-01-01T00:00:00Z'
		);
	`, requestDigest[:], snapshotJSON, snapshotDigest[:]); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	probe := NewProbe(path)
	if status := probe.Check(ctx); status != SchemaOutdated {
		t.Fatalf("v5 readiness = %q, want %q", status, SchemaOutdated)
	}
	database, err = Open(ctx, path, OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	result, err := database.Migrate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.FromVersion != 5 || result.ToVersion != 7 {
		t.Fatalf("migration = %+v", result)
	}
	for table, want := range map[string]int{"orders": 1, "order_items": 1, "idempotency_keys": 1, "refunds": 0, "refund_idempotency_keys": 0} {
		var count int
		if err := database.SQL().QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Errorf("%s count = %d, want %d", table, count, want)
		}
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	if status := probe.Check(ctx); status != Ready {
		t.Fatalf("v6 readiness = %q, want %q", status, Ready)
	}
}

func TestAttachmentMigrationCreatesFreshSchemaAndReplays(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, filepath.Join(t.TempDir(), "allinme.db"), OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	first, err := database.Migrate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if first.FromVersion != 0 || first.ToVersion != 7 {
		t.Fatalf("first migration = %+v", first)
	}
	second, err := database.Migrate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if second.FromVersion != 7 || second.ToVersion != 7 {
		t.Fatalf("second migration = %+v", second)
	}

	for _, table := range []string{"attachments", "order_attachments"} {
		var count int
		if err := database.SQL().QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("%s table count = %d, want 1", table, count)
		}
	}
	for _, index := range []string{"attachments_created_by_status_idx", "attachments_status_expires_at_idx", "order_attachments_order_position_idx"} {
		var count int
		if err := database.SQL().QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?`, index).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("%s index count = %d, want 1", index, count)
		}
	}
}

func TestVersionSixDatabaseUpgradesToAttachmentsWithoutLosingData(t *testing.T) {
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
	for _, migration := range migrations[:6] {
		if err := database.WithTx(ctx, func(transaction *sql.Tx) error {
			if _, err := transaction.ExecContext(ctx, migration.sql); err != nil {
				return err
			}
			if migration.version == 5 {
				if err := backfillSnapshotDigests(ctx, transaction); err != nil {
					return err
				}
			}
			_, err := transaction.ExecContext(ctx, "PRAGMA user_version = "+strconv.Itoa(migration.version))
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := database.SQL().ExecContext(ctx, `
		INSERT INTO users(id, username, password_hash, role, created_at, updated_at)
		VALUES ('user-preserved', 'preserved-v6', 'hash', 'operator', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO orders(id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at)
		VALUES ('ord_00000000000000000000000000000001', 'Preserved v6 order', 'DRAFT', 'UNPAID', 'CNY', 100, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO order_items(id, order_id, position, sku, name, quantity, unit_price)
		VALUES ('itm_00000000000000000000000000000001', 'ord_00000000000000000000000000000001', 0, 'SKU', 'Preserved item', 1, 100);
		INSERT INTO refunds(id, order_id, amount, reason, status, version, requested_by, created_at, updated_at)
		VALUES ('rfd_00000000000000000000000000000001', 'ord_00000000000000000000000000000001', 50, 'Preserved refund', 'PENDING', 1, 'user-preserved', '2026-01-01T00:00:01Z', '2026-01-01T00:00:01Z');
	`); err != nil {
		t.Fatal(err)
	}

	result, err := database.Migrate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.FromVersion != 6 || result.ToVersion != 7 {
		t.Fatalf("migration = %+v", result)
	}
	for table, want := range map[string]int{"users": 1, "orders": 1, "order_items": 1, "refunds": 1, "attachments": 0, "order_attachments": 0} {
		var count int
		if err := database.SQL().QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Errorf("%s count = %d, want %d", table, count, want)
		}
	}
}

func TestAttachmentMigrationFailureKeepsVersionSixAndRollsBackPartialSchema(t *testing.T) {
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
	for _, migration := range migrations[:6] {
		if err := database.WithTx(ctx, func(transaction *sql.Tx) error {
			if _, err := transaction.ExecContext(ctx, migration.sql); err != nil {
				return err
			}
			if migration.version == 5 {
				if err := backfillSnapshotDigests(ctx, transaction); err != nil {
					return err
				}
			}
			_, err := transaction.ExecContext(ctx, "PRAGMA user_version = "+strconv.Itoa(migration.version))
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := database.SQL().ExecContext(ctx, `
		CREATE TABLE migration_collision (created_by TEXT, status TEXT);
		CREATE INDEX attachments_created_by_status_idx ON migration_collision(created_by, status);
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Migrate(ctx); err == nil {
		t.Fatal("v7 migration error = nil")
	}
	version, err := database.SchemaVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if version != 6 {
		t.Fatalf("schema version = %d, want 6", version)
	}
	for _, table := range []string{"attachments", "order_attachments"} {
		var count int
		if err := database.SQL().QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Errorf("partial %s table count = %d, want 0", table, count)
		}
	}
}

func TestAttachmentMigrationConstraints(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, filepath.Join(t.TempDir(), "allinme.db"), OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SQL().ExecContext(ctx, `
		INSERT INTO users(id, username, password_hash, role, created_at, updated_at)
		VALUES ('user-owner', 'attachment-owner', 'hash', 'operator', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO orders(id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at)
		VALUES ('ord_00000000000000000000000000000001', 'Attachment order', 'DRAFT', 'UNPAID', 'CNY', 100, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
	`); err != nil {
		t.Fatal(err)
	}

	insertAttachment := `
		INSERT INTO attachments(
			id, created_by, status, file_name, storage_key, content_type, size_bytes,
			sha256, expires_at, created_at, updated_at
		) VALUES (?, 'user-owner', ?, 'invoice.pdf', ?, 'application/pdf', ?, zeroblob(32),
			'2026-01-02T00:00:00Z', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`
	tests := []struct {
		name   string
		id     string
		status string
		size   int
	}{
		{name: "invalid status", id: "att_00000000000000000000000000000001", status: "STAGING", size: 1},
		{name: "zero size", id: "att_00000000000000000000000000000002", status: "UPLOADED", size: 0},
		{name: "oversize", id: "att_00000000000000000000000000000003", status: "UPLOADED", size: 10485761},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := database.SQL().ExecContext(ctx, insertAttachment, test.id, test.status, test.id, test.size); err == nil {
				t.Fatal("insert error = nil")
			}
		})
	}

	for _, id := range []string{"att_00000000000000000000000000000004", "att_00000000000000000000000000000005"} {
		if _, err := database.SQL().ExecContext(ctx, insertAttachment, id, "UPLOADED", id, 10485760); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := database.SQL().ExecContext(ctx, `
		INSERT INTO order_attachments(attachment_id, order_id, position, bound_at)
		VALUES ('att_00000000000000000000000000000004', 'ord_00000000000000000000000000000001', 0, '2026-01-01T00:00:01Z')
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SQL().ExecContext(ctx, `
		INSERT INTO order_attachments(attachment_id, order_id, position, bound_at)
		VALUES ('att_00000000000000000000000000000005', 'ord_00000000000000000000000000000001', 0, '2026-01-01T00:00:02Z')
	`); err == nil {
		t.Fatal("duplicate order attachment position error = nil")
	}
}

func TestRefundMigrationFailureKeepsVersionFiveAndRollsBackPartialSchema(t *testing.T) {
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
	for _, migration := range migrations[:5] {
		if err := database.WithTx(ctx, func(transaction *sql.Tx) error {
			if _, err := transaction.ExecContext(ctx, migration.sql); err != nil {
				return err
			}
			if migration.version == 5 {
				if err := backfillSnapshotDigests(ctx, transaction); err != nil {
					return err
				}
			}
			_, err := transaction.ExecContext(ctx, "PRAGMA user_version = "+strconv.Itoa(migration.version))
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := database.SQL().ExecContext(ctx, `
		CREATE TABLE migration_collision (order_id TEXT, status TEXT);
		CREATE INDEX refunds_order_status_idx ON migration_collision(order_id, status);
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Migrate(ctx); err == nil {
		t.Fatal("v6 migration error = nil")
	}
	version, err := database.SchemaVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if version != 5 {
		t.Fatalf("schema version = %d, want 5", version)
	}
	for _, table := range []string{"refunds", "refund_idempotency_keys"} {
		var count int
		if err := database.SQL().QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Errorf("partial %s table count = %d, want 0", table, count)
		}
	}
}
