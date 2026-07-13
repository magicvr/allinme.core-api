package store_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
	if first.FromVersion != 0 || first.ToVersion != store.LatestSchemaVersion() {
		t.Fatalf("first migration = %+v", first)
	}
	second, err := database.Migrate(ctx)
	if err != nil {
		t.Fatalf("second Migrate() error = %v", err)
	}
	if second.FromVersion != store.LatestSchemaVersion() || second.ToVersion != store.LatestSchemaVersion() {
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
	if result.FromVersion != 1 || result.ToVersion != store.LatestSchemaVersion() {
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
	if result.FromVersion != 2 || result.ToVersion != store.LatestSchemaVersion() {
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
	if users != 1 || sessions != 1 || idempotencyTables != 1 {
		t.Fatalf("users = %d, sessions = %d, idempotency tables = %d", users, sessions, idempotencyTables)
	}
}

func TestVersionThreeDatabaseUpgradesWithoutLosingOrders(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "allinme.db")
	database, err := store.Open(ctx, path, store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Migrate(ctx); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if _, err := database.SQL().ExecContext(ctx, `
		DROP TABLE refund_idempotency_keys;
		DROP TABLE refunds;
		DROP INDEX idempotency_keys_created_at_idx;
		DROP INDEX idempotency_keys_order_id_idx;
		DROP TABLE idempotency_keys;
		INSERT INTO orders(id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at)
		VALUES ('ord_00000000000000000000000000000001', 'Preserved', 'DRAFT', 'UNPAID', 'CNY', 100, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO order_items(id, order_id, position, sku, name, quantity, unit_price)
		VALUES ('itm_00000000000000000000000000000001', 'ord_00000000000000000000000000000001', 0, 'SKU-1', 'Preserved item', 1, 100);
		PRAGMA user_version = 3;
	`); err != nil {
		database.Close()
		t.Fatal(err)
	}
	database.Close()
	probe := store.NewProbe(path)
	if status := probe.Check(ctx); status != store.SchemaOutdated {
		t.Fatalf("v3 readiness = %q, want %q", status, store.SchemaOutdated)
	}
	database, err = store.Open(ctx, path, store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	result, err := database.Migrate(ctx)
	if err != nil {
		database.Close()
		t.Fatal(err)
	}
	defer database.Close()
	if result.FromVersion != 3 || result.ToVersion != store.LatestSchemaVersion() {
		t.Fatalf("migration = %+v", result)
	}
	var orders, items, idempotencyTables int
	if err := database.SQL().QueryRowContext(ctx, `SELECT COUNT(*) FROM orders WHERE id = 'ord_00000000000000000000000000000001'`).Scan(&orders); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRowContext(ctx, `SELECT COUNT(*) FROM order_items WHERE id = 'itm_00000000000000000000000000000001'`).Scan(&items); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'idempotency_keys'`).Scan(&idempotencyTables); err != nil {
		t.Fatal(err)
	}
	if orders != 1 || items != 1 || idempotencyTables != 1 {
		t.Fatalf("orders = %d, items = %d, idempotency tables = %d", orders, items, idempotencyTables)
	}
	result, err = database.Migrate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.FromVersion != store.LatestSchemaVersion() || result.ToVersion != store.LatestSchemaVersion() {
		t.Fatalf("repeated migration = %+v", result)
	}
	database.Close()
	if status := probe.Check(ctx); status != store.Ready {
		t.Fatalf("latest readiness = %q, want %q", status, store.Ready)
	}
}

func TestIdempotencySchemaEnforcesScopeAndPayloadShape(t *testing.T) {
	ctx := context.Background()
	database := openMigrated(t)
	insert := `
		INSERT INTO idempotency_keys(
			principal_user_id, method, route, idempotency_key, request_digest,
			order_id, snapshot_version, snapshot_json, snapshot_digest, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	valid := []any{
		"user-1", "POST", "/api/v1/orders", "key-1", make([]byte, 32),
		"ord_00000000000000000000000000000001", 1, `{"order":{"id":"ord_00000000000000000000000000000001"}}`, make([]byte, 32), "2026-01-01T00:00:00Z",
	}
	if _, err := database.SQL().ExecContext(ctx, insert, valid...); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SQL().ExecContext(ctx, insert, valid...); err == nil {
		t.Fatal("duplicate idempotency scope error = nil")
	}
	invalidDigest := append([]any(nil), valid...)
	invalidDigest[3] = "key-2"
	invalidDigest[4] = make([]byte, 31)
	if _, err := database.SQL().ExecContext(ctx, insert, invalidDigest...); err == nil {
		t.Fatal("invalid digest error = nil")
	}
	invalidSnapshot := append([]any(nil), valid...)
	invalidSnapshot[3] = "key-3"
	invalidSnapshot[7] = `{`
	if _, err := database.SQL().ExecContext(ctx, insert, invalidSnapshot...); err == nil {
		t.Fatal("invalid snapshot error = nil")
	}
	invalidSnapshotDigest := append([]any(nil), valid...)
	invalidSnapshotDigest[3] = "key-4"
	invalidSnapshotDigest[8] = make([]byte, 31)
	if _, err := database.SQL().ExecContext(ctx, insert, invalidSnapshotDigest...); err == nil {
		t.Fatal("invalid snapshot digest error = nil")
	}
}

func TestRefundSchemaEnforcesFactsScopeAndIndexes(t *testing.T) {
	ctx := context.Background()
	database := openMigrated(t)
	if _, err := database.SQL().ExecContext(ctx, `
		INSERT INTO users(id, username, password_hash, role, created_at, updated_at)
		VALUES
			('user-operator', 'refund-operator', 'hash', 'operator', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('user-approver', 'refund-approver', 'hash', 'approver', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO orders(id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at)
		VALUES
			('ord_00000000000000000000000000000001', 'Refund one', 'COMPLETED', 'PAID', 'CNY', 1000, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('ord_00000000000000000000000000000002', 'Refund two', 'CANCELLED', 'PAID', 'CNY', 2000, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
	`); err != nil {
		t.Fatal(err)
	}
	insertRefund := `
		INSERT INTO refunds(
			id, order_id, amount, reason, status, version, requested_by, decided_by,
			created_at, updated_at, decided_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	pending := []any{
		"rfd_00000000000000000000000000000001", "ord_00000000000000000000000000000001", int64(100), "customer request",
		"PENDING", 1, "user-operator", nil, "2026-01-02T00:00:00Z", "2026-01-02T00:00:00Z", nil,
	}
	if _, err := database.SQL().ExecContext(ctx, insertRefund, pending...); err != nil {
		t.Fatal(err)
	}
	completed := []any{
		"rfd_00000000000000000000000000000002", "ord_00000000000000000000000000000002", int64(200), "completed request",
		"COMPLETED", 2, "user-operator", "user-approver", "2026-01-02T01:00:00Z", "2026-01-02T02:00:00Z", "2026-01-02T02:00:00Z",
	}
	if _, err := database.SQL().ExecContext(ctx, insertRefund, completed...); err != nil {
		t.Fatal(err)
	}

	invalid := append([]any(nil), pending...)
	invalid[0] = "bad"
	if _, err := database.SQL().ExecContext(ctx, insertRefund, invalid...); err == nil {
		t.Fatal("invalid refund ID error = nil")
	}
	invalid = append([]any(nil), pending...)
	invalid[0] = "rfd_00000000000000000000000000000003"
	invalid[2] = int64(0)
	if _, err := database.SQL().ExecContext(ctx, insertRefund, invalid...); err == nil {
		t.Fatal("invalid refund amount error = nil")
	}
	invalid = append([]any(nil), pending...)
	invalid[0] = "rfd_00000000000000000000000000000004"
	invalid[4] = "COMPLETED"
	if _, err := database.SQL().ExecContext(ctx, insertRefund, invalid...); err == nil {
		t.Fatal("invalid refund state fields error = nil")
	}
	invalid = append([]any(nil), pending...)
	invalid[0] = "rfd_00000000000000000000000000000005"
	invalid[1] = "ord_ffffffffffffffffffffffffffffffff"
	if _, err := database.SQL().ExecContext(ctx, insertRefund, invalid...); err == nil {
		t.Fatal("missing refund order error = nil")
	}
	invalid = append([]any(nil), pending...)
	invalid[0] = "rfd_00000000000000000000000000000006"
	invalid[6] = "user-missing"
	if _, err := database.SQL().ExecContext(ctx, insertRefund, invalid...); err == nil {
		t.Fatal("missing refund actor error = nil")
	}
	invalid = append([]any(nil), pending...)
	invalid[0] = "rfd_00000000000000000000000000000007"
	invalid[9] = "not-utc"
	if _, err := database.SQL().ExecContext(ctx, insertRefund, invalid...); err == nil {
		t.Fatal("invalid refund timestamp error = nil")
	}

	insertKey := `
		INSERT INTO refund_idempotency_keys(
			principal_user_id, method, operation, order_id, idempotency_key, request_digest,
			refund_id, snapshot_version, snapshot_json, snapshot_digest, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	key := []any{
		"user-operator", "POST", "POST /api/v1/orders/{orderId}/refunds", "ord_00000000000000000000000000000001", "refund-key",
		make([]byte, 32), "rfd_00000000000000000000000000000001", 1, `{"refund":{"id":"rfd_00000000000000000000000000000001"}}`, make([]byte, 32), "2026-01-02T00:00:00Z",
	}
	if _, err := database.SQL().ExecContext(ctx, insertKey, key...); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SQL().ExecContext(ctx, insertKey, key...); err == nil {
		t.Fatal("duplicate refund idempotency scope error = nil")
	}
	otherOrderKey := append([]any(nil), key...)
	otherOrderKey[3] = "ord_00000000000000000000000000000002"
	otherOrderKey[6] = "rfd_00000000000000000000000000000002"
	if _, err := database.SQL().ExecContext(ctx, insertKey, otherOrderKey...); err != nil {
		t.Fatalf("same key on another order: %v", err)
	}
	invalidKey := append([]any(nil), key...)
	invalidKey[4] = "invalid-digest"
	invalidKey[5] = make([]byte, 31)
	if _, err := database.SQL().ExecContext(ctx, insertKey, invalidKey...); err == nil {
		t.Fatal("invalid refund request digest error = nil")
	}
	invalidKey = append([]any(nil), key...)
	invalidKey[4] = "invalid-snapshot"
	invalidKey[8] = `{`
	if _, err := database.SQL().ExecContext(ctx, insertKey, invalidKey...); err == nil {
		t.Fatal("invalid refund snapshot error = nil")
	}

	wantIndexes := []string{
		"refunds_order_status_idx",
		"refunds_status_created_at_id_idx",
		"refunds_order_created_at_id_idx",
		"refunds_status_decided_at_idx",
		"refund_idempotency_keys_refund_id_idx",
	}
	for _, name := range wantIndexes {
		var count int
		if err := database.SQL().QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?`, name).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("index %s count = %d, want 1", name, count)
		}
	}
	queryPlans := []struct {
		name  string
		query string
		args  []any
	}{
		{name: "refunds_order_status_idx", query: `SELECT amount FROM refunds WHERE order_id = ? AND status IN ('PENDING', 'COMPLETED')`, args: []any{"ord_00000000000000000000000000000001"}},
		{name: "refunds_status_created_at_id_idx", query: `SELECT id FROM refunds WHERE status = ? ORDER BY created_at DESC, id DESC LIMIT 20`, args: []any{"PENDING"}},
		{name: "refunds_order_created_at_id_idx", query: `SELECT id FROM refunds WHERE order_id = ? ORDER BY created_at DESC, id DESC LIMIT 20`, args: []any{"ord_00000000000000000000000000000001"}},
		{name: "refunds_status_decided_at_idx", query: `SELECT SUM(amount) FROM refunds WHERE status = 'COMPLETED' AND decided_at >= ? AND decided_at < ?`, args: []any{"2026-01-01T00:00:00Z", "2026-02-01T00:00:00Z"}},
	}
	for _, plan := range queryPlans {
		rows, err := database.SQL().QueryContext(ctx, "EXPLAIN QUERY PLAN "+plan.query, plan.args...)
		if err != nil {
			t.Fatal(err)
		}
		var details []string
		for rows.Next() {
			var id, parent, notUsed int
			var detail string
			if err := rows.Scan(&id, &parent, &notUsed, &detail); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			details = append(details, detail)
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(strings.Join(details, "\n"), plan.name) {
			t.Errorf("query plan %q = %v, want index %s", plan.query, details, plan.name)
		}
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
