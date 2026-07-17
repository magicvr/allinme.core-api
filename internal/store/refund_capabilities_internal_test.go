package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/order"
)

func TestOrderRefundCapabilitiesBatchLoadAndCorruptionClassification(t *testing.T) {
	ctx := context.Background()
	database := openRefundCapabilityDatabase(t)
	queries := 0
	database.queryObserver = func() { queries++ }
	page, err := database.ListOrders(ctx, order.ListQuery{Page: 1, PageSize: 20, Sort: "createdAt", Descending: true})
	if err != nil {
		t.Fatal(err)
	}
	if queries != 5 || len(page.Items) != 10 {
		t.Fatalf("batch query count/items = %d/%d", queries, len(page.Items))
	}
	wantAvailable := map[string]int64{
		"ord_00000000000000000000000000000007": 70000,
		"ord_00000000000000000000000000000008": 65000,
		"ord_00000000000000000000000000000009": 70000,
		"ord_0000000000000000000000000000000a": 0,
	}
	for _, value := range page.Items {
		if want, ok := wantAvailable[value.ID]; ok && value.AvailableRefundAmount != want {
			t.Errorf("order %s available = %d, want %d", value.ID, value.AvailableRefundAmount, want)
		}
	}
	detail, found, err := database.GetOrder(ctx, "ord_00000000000000000000000000000008")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("detail order not found")
	}
	if detail.AvailableRefundAmount != 65000 {
		t.Fatalf("detail available = %d", detail.AvailableRefundAmount)
	}
}

func TestOrderRefundCapabilitiesRejectCorruptRefundAggregates(t *testing.T) {
	const orderID = "ord_00000000000000000000000000000008"
	tests := []struct {
		name   string
		mutate func(*testing.T, *DB)
	}{
		{name: "status", mutate: func(t *testing.T, database *DB) {
			execRefundCapabilityMutation(t, database, `PRAGMA ignore_check_constraints = ON; UPDATE refunds SET status = 'BROKEN' WHERE id = 'rfd_00000000000000000000000000000001'; PRAGMA ignore_check_constraints = OFF;`)
		}},
		{name: "version", mutate: func(t *testing.T, database *DB) {
			execRefundCapabilityMutation(t, database, `PRAGMA ignore_check_constraints = ON; UPDATE refunds SET version = 2 WHERE id = 'rfd_00000000000000000000000000000001'; PRAGMA ignore_check_constraints = OFF;`)
		}},
		{name: "actor", mutate: func(t *testing.T, database *DB) {
			execRefundCapabilityMutation(t, database, `UPDATE users SET username = CAST(X'80' AS TEXT) WHERE id = 'user-operator'`)
		}},
		{name: "decision fields", mutate: func(t *testing.T, database *DB) {
			execRefundCapabilityMutation(t, database, `PRAGMA ignore_check_constraints = ON; UPDATE refunds SET decided_by = 'user-admin' WHERE id = 'rfd_00000000000000000000000000000001'; PRAGMA ignore_check_constraints = OFF;`)
		}},
		{name: "occupied amount", mutate: func(t *testing.T, database *DB) {
			execRefundCapabilityMutation(t, database, `UPDATE refunds SET amount = 76000 WHERE id = 'rfd_00000000000000000000000000000001'`)
		}},
		{name: "payment mapping", mutate: func(t *testing.T, database *DB) {
			execRefundCapabilityMutation(t, database, `UPDATE orders SET payment_status = 'PARTIALLY_REFUNDED' WHERE id = 'ord_00000000000000000000000000000008'`)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := openRefundCapabilityDatabase(t)
			test.mutate(t, database)
			if _, err := database.ListOrders(context.Background(), order.ListQuery{Page: 1, PageSize: 20, Sort: "createdAt"}); !errors.Is(err, order.ErrInternal) {
				t.Fatalf("list error = %v", err)
			}
			if _, _, err := database.GetOrder(context.Background(), orderID); !errors.Is(err, order.ErrInternal) {
				t.Fatalf("detail error = %v", err)
			}
		})
	}
}

func TestListRefundsUsesSnapshotAndBatchedActorHydration(t *testing.T) {
	database := openRefundCapabilityDatabase(t)
	queries := 0
	database.queryObserver = func() { queries++ }
	page, err := database.ListRefunds(context.Background(), order.RefundListQuery{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if queries != 3 || page.Total != 5 || len(page.Items) != 2 || page.Page != 1 || page.PageSize != 2 {
		t.Fatalf("page = %+v queries=%d", page, queries)
	}
	if page.Items[0].ID != "rfd_00000000000000000000000000000005" || page.Items[0].Status != order.RefundStatusCompleted || page.Items[0].DecidedBy == nil || page.Items[0].DecidedBy.Username != "approver" {
		t.Fatalf("first refund = %+v", page.Items[0])
	}
	completed, err := database.ListRefunds(context.Background(), order.RefundListQuery{Status: order.RefundStatusCompleted, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if completed.Total != 2 || len(completed.Items) != 2 {
		t.Fatalf("completed page = %+v", completed)
	}
	filtered, err := database.ListRefunds(context.Background(), order.RefundListQuery{OrderID: "ord_00000000000000000000000000000009", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Total != 1 || len(filtered.Items) != 1 || filtered.Items[0].OrderID != "ord_00000000000000000000000000000009" {
		t.Fatalf("filtered page = %+v", filtered)
	}
}

func TestListRefundsRejectsMissingActorAsInternal(t *testing.T) {
	database := openRefundCapabilityDatabase(t)
	if _, err := database.SQL().Exec(`PRAGMA foreign_keys = OFF; DELETE FROM users WHERE id = 'user-operator'; PRAGMA foreign_keys = ON;`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ListRefunds(context.Background(), order.RefundListQuery{Page: 1, PageSize: 20}); !errors.Is(err, order.ErrInternal) {
		t.Fatalf("missing actor error = %v", err)
	}
}

func openRefundCapabilityDatabase(t *testing.T) *DB {
	t.Helper()
	return openRefundCapabilityDatabaseAt(t, filepath.Join(t.TempDir(), "refund-capabilities.db"))
}

func openRefundCapabilityDatabaseAt(t *testing.T, path string) *DB {
	t.Helper()
	ctx := context.Background()
	database, err := Open(ctx, path, OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SQL().ExecContext(ctx, `
		INSERT INTO users(id, username, password_hash, role, created_at, updated_at) VALUES
			('user-viewer', 'viewer', 'hash', 'viewer', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('user-operator', 'operator', 'hash', 'operator', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('user-approver', 'approver', 'hash', 'approver', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('user-admin', 'admin', 'hash', 'admin', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SeedOrderDemo(ctx, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SeedRefundDemo(ctx, time.Now()); err != nil {
		t.Fatal(err)
	}
	return database
}

func execRefundCapabilityMutation(t *testing.T, database *DB, statement string) {
	t.Helper()
	if _, err := database.SQL().Exec(statement); err != nil {
		t.Fatal(err)
	}
}
