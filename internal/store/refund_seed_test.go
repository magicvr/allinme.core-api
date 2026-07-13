package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/store"
)

func TestRefundDemoSeedCreatesAndValidatesFixedFixtures(t *testing.T) {
	database := openMigrated(t)
	insertRefundSeedActors(t, database)
	appliedAt := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	if _, err := database.SeedOrderDemo(context.Background(), appliedAt); err != nil {
		t.Fatal(err)
	}
	first, err := database.SeedRefundDemo(context.Background(), appliedAt)
	if err != nil {
		t.Fatal(err)
	}
	if first.FromVersion != 0 || first.ToVersion != 1 {
		t.Fatalf("first seed = %+v", first)
	}
	second, err := database.SeedRefundDemo(context.Background(), appliedAt.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if second.FromVersion != 1 || second.ToVersion != 1 {
		t.Fatalf("second seed = %+v", second)
	}
	assertRefundSeedCounts(t, database, 10, 10, 5)

	var requestedBy, decidedBy string
	if err := database.SQL().QueryRow(`
		SELECT requester.username, decider.username
		FROM refunds
		JOIN users requester ON requester.id = refunds.requested_by
		JOIN users decider ON decider.id = refunds.decided_by
		WHERE refunds.id = 'rfd_00000000000000000000000000000004'
	`).Scan(&requestedBy, &decidedBy); err != nil {
		t.Fatal(err)
	}
	if requestedBy != "operator" || decidedBy != "approver" {
		t.Fatalf("refund actors = %s -> %s", requestedBy, decidedBy)
	}
	var gross, completed int64
	if err := database.SQL().QueryRow(`
		SELECT COALESCE(SUM(total_amount), 0) FROM orders
		WHERE id BETWEEN 'ord_00000000000000000000000000000001' AND 'ord_0000000000000000000000000000000a'
		AND payment_status IN ('PAID', 'PARTIALLY_REFUNDED', 'REFUNDED')
	`).Scan(&gross); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRow(`
		SELECT COALESCE(SUM(amount), 0) FROM refunds
		WHERE id BETWEEN 'rfd_00000000000000000000000000000001' AND 'rfd_00000000000000000000000000000005'
		AND status = 'COMPLETED'
	`).Scan(&completed); err != nil {
		t.Fatal(err)
	}
	if gross != 460000 || completed != 120000 || gross-completed != 340000 {
		t.Fatalf("fixture arithmetic = gross %d completed %d net %d", gross, completed, gross-completed)
	}

	operatorID := refundSeedActorID(t, database, "operator")
	if _, err := database.SQL().Exec(`
		INSERT INTO orders(id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at)
		VALUES ('ord_ffffffffffffffffffffffffffffffff', 'Additional Customer', 'DRAFT', 'PAID', 'CNY', 1, 1, '2026-03-01T00:00:00Z', '2026-03-01T00:00:00Z');
		INSERT INTO order_items(id, order_id, position, sku, name, quantity, unit_price)
		VALUES ('itm_ffffffffffffffffffffffffffffffff', 'ord_ffffffffffffffffffffffffffffffff', 0, 'EXTRA', 'Extra Item', 1, 1);
		INSERT INTO refunds(id, order_id, amount, reason, status, version, requested_by, created_at, updated_at)
		VALUES ('rfd_ffffffffffffffffffffffffffffffff', 'ord_ffffffffffffffffffffffffffffffff', 1, 'extra refund', 'PENDING', 1, ?, '2026-03-02T00:00:00Z', '2026-03-02T00:00:00Z');
	`, operatorID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SeedRefundDemo(context.Background(), appliedAt.Add(2*time.Hour)); err != nil {
		t.Fatalf("repeat seed with additional business data: %v", err)
	}
	assertRefundSeedCounts(t, database, 11, 11, 6)
	var storedAppliedAt string
	if err := database.SQL().QueryRow(`SELECT applied_at FROM seed_versions WHERE name = 'refund_demo'`).Scan(&storedAppliedAt); err != nil {
		t.Fatal(err)
	}
	if storedAppliedAt != appliedAt.Format(time.RFC3339) {
		t.Fatalf("refund applied_at = %q", storedAppliedAt)
	}
}

func TestRefundDemoSeedRejectsInconsistentStateWithoutRepair(t *testing.T) {
	mutations := []string{
		`DELETE FROM refunds WHERE id = 'rfd_00000000000000000000000000000001'`,
		`UPDATE refunds SET reason = 'tampered' WHERE id = 'rfd_00000000000000000000000000000002'`,
		`UPDATE orders SET version = 3 WHERE id = 'ord_00000000000000000000000000000009'`,
		`DELETE FROM order_items WHERE order_id = 'ord_0000000000000000000000000000000a'`,
	}
	for _, mutation := range mutations {
		t.Run(mutation, func(t *testing.T) {
			database := openMigrated(t)
			insertRefundSeedActors(t, database)
			appliedAt := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
			if _, err := database.SeedOrderDemo(context.Background(), appliedAt); err != nil {
				t.Fatal(err)
			}
			if _, err := database.SeedRefundDemo(context.Background(), appliedAt); err != nil {
				t.Fatal(err)
			}
			if _, err := database.SQL().Exec(mutation); err != nil {
				t.Fatal(err)
			}
			if _, err := database.SeedRefundDemo(context.Background(), appliedAt.Add(time.Hour)); err == nil {
				t.Fatal("inconsistent repeat seed error = nil")
			}
			var stored string
			if err := database.SQL().QueryRow(`SELECT applied_at FROM seed_versions WHERE name = 'refund_demo'`).Scan(&stored); err != nil {
				t.Fatal(err)
			}
			if stored != appliedAt.Format(time.RFC3339) {
				t.Fatalf("repeat seed changed applied_at to %q", stored)
			}
		})
	}
}

func TestRefundDemoSeedRequiresAuthAndOrderPrerequisites(t *testing.T) {
	t.Run("missing actors", func(t *testing.T) {
		database := openMigrated(t)
		if _, err := database.SeedOrderDemo(context.Background(), time.Now()); err != nil {
			t.Fatal(err)
		}
		if _, err := database.SeedRefundDemo(context.Background(), time.Now()); err == nil {
			t.Fatal("missing actor seed error = nil")
		}
		assertRefundSeedCounts(t, database, 6, 6, 0)
	})
	t.Run("missing orders", func(t *testing.T) {
		database := openMigrated(t)
		insertRefundSeedActors(t, database)
		if _, err := database.SeedRefundDemo(context.Background(), time.Now()); err == nil {
			t.Fatal("missing order seed error = nil")
		}
		assertRefundSeedCounts(t, database, 0, 0, 0)
	})
}

func TestRefundDemoSeedConflictRollsBackWholeGroup(t *testing.T) {
	database := openMigrated(t)
	insertRefundSeedActors(t, database)
	if _, err := database.SeedOrderDemo(context.Background(), time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SQL().Exec(`
		INSERT INTO orders(id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at)
		VALUES ('ord_00000000000000000000000000000007', 'conflict', 'DRAFT', 'UNPAID', 'CNY', 1, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SeedRefundDemo(context.Background(), time.Now()); err == nil {
		t.Fatal("conflicting refund seed error = nil")
	}
	var fixedNewOrders, refunds, versions int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM orders WHERE id BETWEEN 'ord_00000000000000000000000000000008' AND 'ord_0000000000000000000000000000000a'`).Scan(&fixedNewOrders); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM refunds`).Scan(&refunds); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM seed_versions WHERE name = 'refund_demo'`).Scan(&versions); err != nil {
		t.Fatal(err)
	}
	if fixedNewOrders != 0 || refunds != 0 || versions != 0 {
		t.Fatalf("partial refund seed = orders %d refunds %d versions %d", fixedNewOrders, refunds, versions)
	}
}

func insertRefundSeedActors(t *testing.T, database *store.DB) {
	t.Helper()
	if _, err := database.SQL().Exec(`
		INSERT INTO users(id, username, password_hash, role, created_at, updated_at) VALUES
			('user-viewer', 'viewer', 'hash', 'viewer', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('user-operator', 'operator', 'hash', 'operator', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('user-approver', 'approver', 'hash', 'approver', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('user-admin', 'admin', 'hash', 'admin', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')
	`); err != nil {
		t.Fatal(err)
	}
}

func assertRefundSeedCounts(t *testing.T, database *store.DB, wantOrders, wantItems, wantRefunds int) {
	t.Helper()
	var orders, items, refunds int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&orders); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM order_items`).Scan(&items); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM refunds`).Scan(&refunds); err != nil {
		t.Fatal(err)
	}
	if orders != wantOrders || items != wantItems || refunds != wantRefunds {
		t.Fatalf("seed counts = orders %d items %d refunds %d, want %d/%d/%d", orders, items, refunds, wantOrders, wantItems, wantRefunds)
	}
}

func refundSeedActorID(t *testing.T, database *store.DB, username string) string {
	t.Helper()
	var id string
	if err := database.SQL().QueryRow(`SELECT id FROM users WHERE username = ?`, username).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}
