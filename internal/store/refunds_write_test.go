package store_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/order"
	"github.com/magicvr/allinme.core-api/internal/store"
)

func TestCreateRefundIdempotentPersistsReplaysAndLeavesOrderUnchanged(t *testing.T) {
	database := openRefundDemoDatabase(t)
	service := refundServiceFor(t, database, "rfd_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	principal := refundOperator()
	command := order.RefundRequestCommand{Amount: 100, Reason: " customer request ", OrderVersion: 1}
	beforeVersion, beforeUpdatedAt := readOrderVersionAndUpdatedAt(t, database, "ord_00000000000000000000000000000007")
	created, err := service.Create(context.Background(), principal, "ord_00000000000000000000000000000007", "create-key", command)
	if err != nil {
		t.Fatal(err)
	}
	if created.Refund.ID != "rfd_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" || created.Refund.Reason != "customer request" || created.Refund.Status != order.RefundStatusPending {
		t.Fatalf("created refund = %+v", created)
	}
	afterVersion, afterUpdatedAt := readOrderVersionAndUpdatedAt(t, database, "ord_00000000000000000000000000000007")
	if afterVersion != beforeVersion || afterUpdatedAt != beforeUpdatedAt {
		t.Fatalf("create changed order version/time: %d/%s -> %d/%s", beforeVersion, beforeUpdatedAt, afterVersion, afterUpdatedAt)
	}
	var refunds, keys int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM refunds WHERE order_id = 'ord_00000000000000000000000000000007'`).Scan(&refunds); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM refund_idempotency_keys WHERE order_id = 'ord_00000000000000000000000000000007'`).Scan(&keys); err != nil {
		t.Fatal(err)
	}
	if refunds != 1 || keys != 1 {
		t.Fatalf("created rows = refunds %d keys %d", refunds, keys)
	}
	approverID := refundSeedActorID(t, database, "approver")
	if _, err := database.SQL().Exec(`
		UPDATE refunds SET status = 'COMPLETED', version = 2, decided_by = ?, updated_at = '2026-01-03T01:00:00Z', decided_at = '2026-01-03T01:00:00Z'
		WHERE id = 'rfd_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa';
		UPDATE orders SET payment_status = 'PARTIALLY_REFUNDED', version = 2, updated_at = '2026-01-03T01:00:00Z'
		WHERE id = 'ord_00000000000000000000000000000007'
	`, approverID); err != nil {
		t.Fatal(err)
	}
	replayed, err := service.Create(context.Background(), principal, "ord_00000000000000000000000000000007", "create-key", command)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Refund != created.Refund {
		t.Fatalf("replayed snapshot = %+v, want %+v", replayed, created)
	}
	conflict := command
	conflict.Amount++
	if _, err := service.Create(context.Background(), principal, "ord_00000000000000000000000000000007", "create-key", conflict); !errors.Is(err, order.ErrIdempotencyConflict) {
		t.Fatalf("idempotency conflict error = %v", err)
	}
}

func TestCreateRefundClassificationPriorityUsesVersionThenIntegrityThenState(t *testing.T) {
	t.Run("stale version before corrupt aggregate", func(t *testing.T) {
		database := openRefundDemoDatabase(t)
		if _, err := database.SQL().Exec(`UPDATE order_items SET unit_price = 69999 WHERE order_id = 'ord_00000000000000000000000000000007'`); err != nil {
			t.Fatal(err)
		}
		service := refundServiceFor(t, database, "rfd_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		_, err := service.Create(context.Background(), refundOperator(), "ord_00000000000000000000000000000007", "priority", order.RefundRequestCommand{Amount: 1, Reason: "request", OrderVersion: 2})
		if !errors.Is(err, order.ErrVersionConflict) {
			t.Fatalf("priority error = %v", err)
		}
	})
	t.Run("corrupt aggregate before amount", func(t *testing.T) {
		database := openRefundDemoDatabase(t)
		if _, err := database.SQL().Exec(`UPDATE orders SET payment_status = 'PARTIALLY_REFUNDED' WHERE id = 'ord_00000000000000000000000000000008'`); err != nil {
			t.Fatal(err)
		}
		service := refundServiceFor(t, database, "rfd_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		_, err := service.Create(context.Background(), refundOperator(), "ord_00000000000000000000000000000008", "priority", order.RefundRequestCommand{Amount: 9999999999, Reason: "request", OrderVersion: 1})
		if !errors.Is(err, order.ErrInternal) {
			t.Fatalf("priority error = %v", err)
		}
	})
	t.Run("state before available amount", func(t *testing.T) {
		database := openRefundDemoDatabase(t)
		service := refundServiceFor(t, database, "rfd_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		_, err := service.Create(context.Background(), refundOperator(), "ord_00000000000000000000000000000001", "priority", order.RefundRequestCommand{Amount: 9999999999, Reason: "request", OrderVersion: 1})
		if !errors.Is(err, order.ErrStateConflict) {
			t.Fatalf("priority error = %v", err)
		}
	})
}

func TestRefundIdempotencyScopeIncludesOrderAndIsIndependentFromOrderCreate(t *testing.T) {
	database := openRefundDemoDatabase(t)
	if _, err := database.SQL().Exec(`
		INSERT INTO idempotency_keys(
			principal_user_id, method, route, idempotency_key, request_digest, order_id,
			snapshot_version, snapshot_json, snapshot_digest, created_at
		) VALUES (
			'user-operator', 'POST', '/api/v1/orders', 'shared-key', zeroblob(32),
			'ord_00000000000000000000000000000007', 1, '{}', zeroblob(32), '2026-01-01T00:00:00Z'
		)
	`); err != nil {
		t.Fatal(err)
	}
	ids := []string{"rfd_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "rfd_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	index := 0
	service, err := order.NewRefundServiceWithDependencies(database, func() time.Time { return time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC) }, func() (string, error) {
		id := ids[index]
		index++
		return id, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	principal := refundOperator()
	for _, orderID := range []string{"ord_00000000000000000000000000000007", "ord_00000000000000000000000000000008"} {
		if _, err := service.Create(context.Background(), principal, orderID, "shared-key", order.RefundRequestCommand{Amount: 100, Reason: "shared scope", OrderVersion: 1}); err != nil {
			t.Fatalf("create on %s: %v", orderID, err)
		}
	}
	var refunds, refundKeys, orderKeys int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM refunds WHERE id IN ('rfd_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 'rfd_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb')`).Scan(&refunds); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM refund_idempotency_keys WHERE idempotency_key = 'shared-key'`).Scan(&refundKeys); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM idempotency_keys WHERE idempotency_key = 'shared-key'`).Scan(&orderKeys); err != nil {
		t.Fatal(err)
	}
	if refunds != 2 || refundKeys != 2 || orderKeys != 1 {
		t.Fatalf("shared scope rows = refunds %d refund keys %d order keys %d", refunds, refundKeys, orderKeys)
	}
}

func TestCreateRefundClassifiesMissesAndRollsBack(t *testing.T) {
	tests := []struct {
		name    string
		orderID string
		command order.RefundRequestCommand
		mutate  func(*testing.T, *store.DB)
		want    error
		field   string
	}{
		{name: "missing", orderID: "ord_eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", command: order.RefundRequestCommand{Amount: 1, Reason: "request", OrderVersion: 1}, want: order.ErrNotFound},
		{name: "version", orderID: "ord_00000000000000000000000000000007", command: order.RefundRequestCommand{Amount: 1, Reason: "request", OrderVersion: 2}, want: order.ErrVersionConflict},
		{name: "unpaid", orderID: "ord_00000000000000000000000000000001", command: order.RefundRequestCommand{Amount: 1, Reason: "request", OrderVersion: 1}, want: order.ErrStateConflict},
		{name: "amount", orderID: "ord_00000000000000000000000000000007", command: order.RefundRequestCommand{Amount: 70001, Reason: "request", OrderVersion: 1}, field: "amount"},
		{name: "order aggregate", orderID: "ord_00000000000000000000000000000007", command: order.RefundRequestCommand{Amount: 1, Reason: "request", OrderVersion: 1}, mutate: func(t *testing.T, database *store.DB) {
			if _, err := database.SQL().Exec(`UPDATE order_items SET unit_price = 69999 WHERE order_id = 'ord_00000000000000000000000000000007'`); err != nil {
				t.Fatal(err)
			}
		}, want: order.ErrInternal},
		{name: "payment mapping", orderID: "ord_00000000000000000000000000000008", command: order.RefundRequestCommand{Amount: 1, Reason: "request", OrderVersion: 1}, mutate: func(t *testing.T, database *store.DB) {
			if _, err := database.SQL().Exec(`UPDATE orders SET payment_status = 'PARTIALLY_REFUNDED' WHERE id = 'ord_00000000000000000000000000000008'`); err != nil {
				t.Fatal(err)
			}
		}, want: order.ErrInternal},
		{name: "occupied amount", orderID: "ord_00000000000000000000000000000008", command: order.RefundRequestCommand{Amount: 1, Reason: "request", OrderVersion: 1}, mutate: func(t *testing.T, database *store.DB) {
			if _, err := database.SQL().Exec(`UPDATE refunds SET amount = 80000 WHERE id = 'rfd_00000000000000000000000000000001'`); err != nil {
				t.Fatal(err)
			}
		}, want: order.ErrInternal},
		{name: "decision fields", orderID: "ord_00000000000000000000000000000008", command: order.RefundRequestCommand{Amount: 1, Reason: "request", OrderVersion: 1}, mutate: func(t *testing.T, database *store.DB) {
			if _, err := database.SQL().Exec(`PRAGMA ignore_check_constraints = ON`); err != nil {
				t.Fatal(err)
			}
			if _, err := database.SQL().Exec(`UPDATE refunds SET version = 2 WHERE id = 'rfd_00000000000000000000000000000001'`); err != nil {
				t.Fatal(err)
			}
			if _, err := database.SQL().Exec(`PRAGMA ignore_check_constraints = OFF`); err != nil {
				t.Fatal(err)
			}
		}, want: order.ErrInternal},
		{name: "missing actor", orderID: "ord_00000000000000000000000000000008", command: order.RefundRequestCommand{Amount: 1, Reason: "request", OrderVersion: 1}, mutate: func(t *testing.T, database *store.DB) {
			if _, err := database.SQL().Exec(`PRAGMA foreign_keys = OFF`); err != nil {
				t.Fatal(err)
			}
			if _, err := database.SQL().Exec(`DELETE FROM users WHERE username = 'admin'`); err != nil {
				t.Fatal(err)
			}
			if _, err := database.SQL().Exec(`PRAGMA foreign_keys = ON`); err != nil {
				t.Fatal(err)
			}
		}, want: order.ErrInternal},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := openRefundDemoDatabase(t)
			if test.mutate != nil {
				test.mutate(t, database)
			}
			beforeRefunds, beforeKeys := refundRowCounts(t, database)
			service := refundServiceFor(t, database, "rfd_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
			_, err := service.Create(context.Background(), refundOperator(), test.orderID, "failure-key", test.command)
			if test.field != "" {
				details, ok := order.ValidationDetails(err)
				if !ok || len(details) != 1 || details[0].Field != test.field {
					t.Fatalf("validation error = %v details = %+v", err, details)
				}
			} else if !errors.Is(err, test.want) {
				t.Fatalf("Create() error = %v, want %v", err, test.want)
			}
			afterRefunds, afterKeys := refundRowCounts(t, database)
			if afterRefunds != beforeRefunds || afterKeys != beforeKeys {
				t.Fatalf("failed create wrote rows: refunds %d->%d keys %d->%d", beforeRefunds, afterRefunds, beforeKeys, afterKeys)
			}
		})
	}
}

func TestCreateRefundAcrossTwoDatabasesPreventsOvercommit(t *testing.T) {
	first, second := openSharedRefundDemoDatabases(t)
	services := []*order.RefundService{
		refundServiceFor(t, first, "rfd_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		refundServiceFor(t, second, "rfd_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
	}
	commands := []order.RefundRequestCommand{
		{Amount: 50000, Reason: "first", OrderVersion: 1},
		{Amount: 50000, Reason: "second", OrderVersion: 1},
	}
	results, foundErrors := runConcurrentRefundCreates(t, services, []string{"key-a", "key-b"}, commands)
	_ = results
	for index, err := range foundErrors {
		if errors.Is(err, order.ErrUnavailable) {
			_, foundErrors[index] = services[index].Create(context.Background(), refundOperator(), "ord_00000000000000000000000000000007", []string{"key-a", "key-b"}[index], commands[index])
		}
	}
	successes, validations := 0, 0
	for _, err := range foundErrors {
		if err == nil {
			successes++
		} else if _, ok := order.ValidationDetails(err); ok {
			validations++
		} else {
			t.Fatalf("concurrent create error = %v", err)
		}
	}
	if successes != 1 || validations != 1 {
		t.Fatalf("concurrent results = successes %d validations %d errors %v", successes, validations, foundErrors)
	}
	var count int
	var total int64
	if err := first.SQL().QueryRow(`SELECT COUNT(*), COALESCE(SUM(amount), 0) FROM refunds WHERE order_id = 'ord_00000000000000000000000000000007' AND status = 'PENDING'`).Scan(&count, &total); err != nil {
		t.Fatal(err)
	}
	if count != 1 || total != 50000 {
		t.Fatalf("pending refunds = count %d total %d", count, total)
	}
}

func TestCreateRefundAcrossTwoDatabasesSameKeyConvergesToFirstSnapshot(t *testing.T) {
	first, second := openSharedRefundDemoDatabases(t)
	services := []*order.RefundService{
		refundServiceFor(t, first, "rfd_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		refundServiceFor(t, second, "rfd_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
	}
	command := order.RefundRequestCommand{Amount: 100, Reason: "same", OrderVersion: 1}
	results, foundErrors := runConcurrentRefundCreates(t, services, []string{"same-key", "same-key"}, []order.RefundRequestCommand{command, command})
	for index, err := range foundErrors {
		if errors.Is(err, order.ErrUnavailable) {
			results[index], foundErrors[index] = services[index].Create(context.Background(), refundOperator(), "ord_00000000000000000000000000000007", "same-key", command)
		}
		if foundErrors[index] != nil {
			t.Fatalf("same-key create %d error = %v", index, foundErrors[index])
		}
	}
	if results[0].Refund != results[1].Refund {
		t.Fatalf("same-key results differ: %+v / %+v", results[0], results[1])
	}
	var refunds, keys int
	if err := first.SQL().QueryRow(`SELECT COUNT(*) FROM refunds WHERE order_id = 'ord_00000000000000000000000000000007'`).Scan(&refunds); err != nil {
		t.Fatal(err)
	}
	if err := first.SQL().QueryRow(`SELECT COUNT(*) FROM refund_idempotency_keys WHERE order_id = 'ord_00000000000000000000000000000007' AND idempotency_key = 'same-key'`).Scan(&keys); err != nil {
		t.Fatal(err)
	}
	if refunds != 1 || keys != 1 {
		t.Fatalf("same-key rows = refunds %d keys %d", refunds, keys)
	}
}

func openRefundDemoDatabase(t *testing.T) *store.DB {
	t.Helper()
	database := openMigrated(t)
	insertRefundSeedActors(t, database)
	if _, err := database.SeedOrderDemo(context.Background(), time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SeedRefundDemo(context.Background(), time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	return database
}

func openSharedRefundDemoDatabases(t *testing.T) (*store.DB, *store.DB) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "refund-race.db")
	first, err := store.Open(context.Background(), path, store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { first.Close() })
	if _, err := first.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	insertRefundSeedActors(t, first)
	if _, err := first.SeedOrderDemo(context.Background(), time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := first.SeedRefundDemo(context.Background(), time.Now()); err != nil {
		t.Fatal(err)
	}
	second, err := store.Open(context.Background(), path, store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { second.Close() })
	if _, err := second.SQL().Exec(`PRAGMA busy_timeout = 10`); err != nil {
		t.Fatal(err)
	}
	return first, second
}

func refundServiceFor(t *testing.T, database *store.DB, refundID string) *order.RefundService {
	t.Helper()
	service, err := order.NewRefundServiceWithDependencies(database, func() time.Time { return time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC) }, func() (string, error) { return refundID, nil })
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func refundOperator() auth.Principal {
	return auth.Principal{UserID: "user-operator", Username: "operator", Role: auth.RoleOperator}
}

func readOrderVersionAndUpdatedAt(t *testing.T, database *store.DB, orderID string) (int64, string) {
	t.Helper()
	var version int64
	var updatedAt string
	if err := database.SQL().QueryRow(`SELECT version, updated_at FROM orders WHERE id = ?`, orderID).Scan(&version, &updatedAt); err != nil {
		t.Fatal(err)
	}
	return version, updatedAt
}

func refundRowCounts(t *testing.T, database *store.DB) (int, int) {
	t.Helper()
	var refunds, keys int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM refunds`).Scan(&refunds); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM refund_idempotency_keys`).Scan(&keys); err != nil {
		t.Fatal(err)
	}
	return refunds, keys
}

func runConcurrentRefundCreates(t *testing.T, services []*order.RefundService, keys []string, commands []order.RefundRequestCommand) ([]order.RefundResult, []error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	start := make(chan struct{})
	results := make([]order.RefundResult, len(services))
	foundErrors := make([]error, len(services))
	var wait sync.WaitGroup
	for index := range services {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			results[index], foundErrors[index] = services[index].Create(ctx, refundOperator(), "ord_00000000000000000000000000000007", keys[index], commands[index])
		}(index)
	}
	close(start)
	done := make(chan struct{})
	go func() {
		wait.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatalf("concurrent refund create deadline: %v", ctx.Err())
	}
	return results, foundErrors
}

func ExampleRefundCreatePersistence() {
	fmt.Println(order.RefundCreateOperation)
	// Output: POST /api/v1/orders/{orderId}/refunds
}
