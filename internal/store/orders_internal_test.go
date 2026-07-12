package store

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/order"
)

func TestOrderRepositoryListUsesTwoQueriesAndStableFiltering(t *testing.T) {
	database := openSeededOrderDatabase(t)
	queries := 0
	database.queryObserver = func() { queries++ }
	from := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)
	page, err := database.ListOrders(context.Background(), order.ListQuery{Keyword: "demo-%_", CreatedFrom: &from, Page: 1, PageSize: 20, Sort: "createdAt", Descending: true})
	if err != nil {
		t.Fatal(err)
	}
	if queries != 2 {
		t.Fatalf("query count = %d, want 2", queries)
	}
	if page.Total != 0 || len(page.Items) != 0 {
		t.Fatalf("page = %+v", page)
	}
	queries = 0
	page, err = database.ListOrders(context.Background(), order.ListQuery{Keyword: "fulfilling", Page: 1, PageSize: 20, Sort: "createdAt"})
	if err != nil {
		t.Fatal(err)
	}
	if queries != 2 || page.Total != 1 || len(page.Items) != 1 || page.Items[0].Items != nil {
		t.Fatalf("page = %+v queries=%d", page, queries)
	}
	for index := 0; index < 300; index++ {
		id := fmt.Sprintf("ord_%032x", index+1000)
		if _, err := database.SQL().Exec(`INSERT INTO orders(id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at) VALUES (?, ?, 'DRAFT', 'UNPAID', 'CNY', 1, 1, '2026-02-01T00:00:00Z', '2026-02-01T00:00:00Z')`, id, fmt.Sprintf("Bulk %03d", index)); err != nil {
			t.Fatal(err)
		}
	}
	queries = 0
	page, err = database.ListOrders(context.Background(), order.ListQuery{Page: 2, PageSize: 100, Sort: "customerName"})
	if err != nil {
		t.Fatal(err)
	}
	if queries != 2 || page.Total != 306 || len(page.Items) != 100 {
		t.Fatalf("bulk page total=%d items=%d queries=%d", page.Total, len(page.Items), queries)
	}
}

func TestOrderRepositoryDetailAndCorruptData(t *testing.T) {
	database := openSeededOrderDatabase(t)
	result, found, err := database.GetOrder(context.Background(), "ord_00000000000000000000000000000001")
	if err != nil || !found || len(result.Items) != 1 {
		t.Fatalf("GetOrder() = %+v %v %v", result, found, err)
	}
	if _, found, err := database.GetOrder(context.Background(), "ord_ffffffffffffffffffffffffffffffff"); err != nil || found {
		t.Fatalf("missing = %v %v", found, err)
	}
	if _, err := database.SQL().Exec(`PRAGMA ignore_check_constraints = ON; UPDATE orders SET status = 'BROKEN' WHERE id = 'ord_00000000000000000000000000000001'; PRAGMA ignore_check_constraints = OFF;`); err != nil {
		t.Fatal(err)
	}
	if _, _, err := database.GetOrder(context.Background(), "ord_00000000000000000000000000000001"); err == nil {
		t.Fatal("corrupt order returned no error")
	}
	database.Close()
	if _, _, err := database.GetOrder(context.Background(), "ord_00000000000000000000000000000002"); err == nil {
		t.Fatal("closed database returned no error")
	}
}

func TestScanOrderFailureInjection(t *testing.T) {
	scanFailure := fmt.Errorf("injected scan failure")
	if _, err := scanOrder(scannerStub{err: scanFailure}); err != scanFailure {
		t.Fatalf("scanOrder() error = %v", err)
	}
	if _, found, err := scanOrderFound(scannerStub{err: scanFailure}); found || err != scanFailure {
		t.Fatalf("scanOrderFound() = %v %v", found, err)
	}
}

type scannerStub struct{ err error }

func (stub scannerStub) Scan(...any) error { return stub.err }

func openSeededOrderDatabase(t *testing.T) *DB {
	t.Helper()
	database, err := Open(context.Background(), filepath.Join(t.TempDir(), "orders.db"), OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Seed(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SeedOrderDemo(context.Background(), time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(fmt.Errorf("seed orders: %w", err))
	}
	return database
}
