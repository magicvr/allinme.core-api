package store_test

import (
	"context"
	"testing"
	"time"
)

func TestOrderDemoSeedCreatesAndValidatesFixedOrders(t *testing.T) {
	database := openMigrated(t)
	appliedAt := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	first, err := database.SeedOrderDemo(context.Background(), appliedAt)
	if err != nil {
		t.Fatal(err)
	}
	if first.FromVersion != 0 || first.ToVersion != 1 {
		t.Fatalf("first seed = %+v", first)
	}
	second, err := database.SeedOrderDemo(context.Background(), appliedAt.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if second.FromVersion != 1 || second.ToVersion != 1 {
		t.Fatalf("second seed = %+v", second)
	}
	var orders, items int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&orders); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM order_items`).Scan(&items); err != nil {
		t.Fatal(err)
	}
	if orders != 6 || items != 6 {
		t.Fatalf("orders = %d, items = %d", orders, items)
	}
	var storedAppliedAt string
	if err := database.SQL().QueryRow(`SELECT applied_at FROM seed_versions WHERE name = 'order_demo'`).Scan(&storedAppliedAt); err != nil {
		t.Fatal(err)
	}
	if storedAppliedAt != appliedAt.Format(time.RFC3339) {
		t.Fatalf("applied_at = %q", storedAppliedAt)
	}
	if _, err := database.SQL().Exec(`
		INSERT INTO orders(id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at)
		VALUES ('ord_ffffffffffffffffffffffffffffffff', 'Additional Customer', 'DRAFT', 'UNPAID', 'CNY', 1, 1, '2026-02-01T00:00:00Z', '2026-02-01T00:00:00Z')
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SeedOrderDemo(context.Background(), appliedAt.Add(2*time.Hour)); err != nil {
		t.Fatalf("repeat seed with additional order error = %v", err)
	}
	rows, err := database.SQL().Query(`
		SELECT id, status, payment_status, created_at, updated_at
		FROM orders
		WHERE id BETWEEN 'ord_00000000000000000000000000000001' AND 'ord_00000000000000000000000000000006'
		ORDER BY id
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	wantStatuses := []string{"DRAFT", "CONFIRMED", "FULFILLING", "SHIPPED", "COMPLETED", "CANCELLED"}
	wantPayments := []string{"UNPAID", "UNPAID", "PAID", "PAID", "PAID", "UNPAID"}
	index := 0
	for rows.Next() {
		var id, status, paymentStatus, createdAt, updatedAt string
		if err := rows.Scan(&id, &status, &paymentStatus, &createdAt, &updatedAt); err != nil {
			t.Fatal(err)
		}
		if status != wantStatuses[index] || paymentStatus != wantPayments[index] || createdAt != updatedAt {
			t.Fatalf("order %s = %s/%s %s/%s", id, status, paymentStatus, createdAt, updatedAt)
		}
		index++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if index != 6 {
		t.Fatalf("seed order rows = %d", index)
	}
}

func TestOrderDemoSeedRejectsInconsistentStateWithoutRepair(t *testing.T) {
	mutations := []string{
		`DELETE FROM order_items WHERE order_id = 'ord_00000000000000000000000000000001'`,
		`INSERT INTO order_items(id, order_id, position, sku, name, quantity, unit_price) VALUES ('itm_ffffffffffffffffffffffffffffffff', 'ord_00000000000000000000000000000001', 1, 'EXTRA', 'Extra Item', 1, 1)`,
		`UPDATE orders SET customer_name = 'tampered' WHERE id = 'ord_00000000000000000000000000000001'`,
		`UPDATE orders SET version = 2 WHERE id = 'ord_00000000000000000000000000000001'`,
	}
	for _, mutation := range mutations {
		t.Run(mutation, func(t *testing.T) {
			database := openMigrated(t)
			appliedAt := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
			if _, err := database.SeedOrderDemo(context.Background(), appliedAt); err != nil {
				t.Fatal(err)
			}
			if _, err := database.SQL().Exec(mutation); err != nil {
				t.Fatal(err)
			}
			if _, err := database.SeedOrderDemo(context.Background(), appliedAt.Add(time.Hour)); err == nil {
				t.Fatal("inconsistent repeat seed error = nil")
			}
			var stored string
			if err := database.SQL().QueryRow(`SELECT applied_at FROM seed_versions WHERE name = 'order_demo'`).Scan(&stored); err != nil {
				t.Fatal(err)
			}
			if stored != appliedAt.Format(time.RFC3339) {
				t.Fatalf("repeat seed changed applied_at to %q", stored)
			}
		})
	}
}

func TestOrderDemoSeedConflictRollsBackGroup(t *testing.T) {
	database := openMigrated(t)
	if _, err := database.SQL().Exec(`
		INSERT INTO orders(id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at)
		VALUES ('ord_00000000000000000000000000000001', 'conflict', 'DRAFT', 'UNPAID', 'CNY', 1, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SeedOrderDemo(context.Background(), time.Now()); err == nil {
		t.Fatal("conflicting seed error = nil")
	}
	var orders, items, versions int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&orders); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM order_items`).Scan(&items); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM seed_versions WHERE name = 'order_demo'`).Scan(&versions); err != nil {
		t.Fatal(err)
	}
	if orders != 1 || items != 0 || versions != 0 {
		t.Fatalf("orders = %d, items = %d, versions = %d", orders, items, versions)
	}
}

func TestOrderSchemaConstraintsAndCascade(t *testing.T) {
	database := openMigrated(t)
	invalidStatements := []string{
		`INSERT INTO orders(id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at) VALUES ('bad', 'customer', 'DRAFT', 'UNPAID', 'CNY', 1, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		`INSERT INTO orders(id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at) VALUES ('ord_00000000000000000000000000000001', 'customer', 'UNKNOWN', 'UNPAID', 'CNY', 1, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		`INSERT INTO orders(id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at) VALUES ('ord_00000000000000000000000000000001', 'customer', 'DRAFT', 'UNPAID', 'USD', 1, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
	}
	for _, statement := range invalidStatements {
		if _, err := database.SQL().Exec(statement); err == nil {
			t.Fatalf("invalid statement succeeded: %s", statement)
		}
	}
	if _, err := database.SeedOrderDemo(context.Background(), time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SQL().Exec(`DELETE FROM orders WHERE id = 'ord_00000000000000000000000000000001'`); err != nil {
		t.Fatal(err)
	}
	var items int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM order_items WHERE order_id = 'ord_00000000000000000000000000000001'`).Scan(&items); err != nil || items != 0 {
		t.Fatalf("cascade items = %d, error = %v", items, err)
	}
}
