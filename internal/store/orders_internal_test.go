package store

import (
	"context"
	"errors"
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

func TestOrderRepositoryListCountAndPageShareSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orders.db")
	reader, err := Open(context.Background(), path, OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reader.Close() })
	if _, err := reader.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.Seed(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.SeedOrderDemo(context.Background(), time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	writer, err := Open(context.Background(), path, OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = writer.Close() })

	countComplete := make(chan struct{})
	allowPage := make(chan struct{})
	queries := 0
	reader.queryObserver = func() {
		queries++
		if queries == 1 {
			close(countComplete)
			<-allowPage
		}
	}
	type listResult struct {
		page order.Page
		err  error
	}
	result := make(chan listResult, 1)
	go func() {
		page, err := reader.ListOrders(context.Background(), order.ListQuery{Page: 1, PageSize: 20, Sort: "createdAt"})
		result <- listResult{page: page, err: err}
	}()

	select {
	case <-countComplete:
	case <-time.After(time.Second):
		t.Fatal("COUNT query did not reach coordination point")
	}
	if _, err := writer.SQL().Exec(`INSERT INTO orders(id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at) VALUES ('ord_ffffffffffffffffffffffffffffffff', 'Concurrent', 'DRAFT', 'UNPAID', 'CNY', 1, 1, '2026-03-01T00:00:00Z', '2026-03-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	close(allowPage)
	select {
	case listed := <-result:
		if listed.err != nil {
			t.Fatal(listed.err)
		}
		if queries != 2 || listed.page.Total != 6 || len(listed.page.Items) != 6 {
			t.Fatalf("snapshot page total=%d items=%d queries=%d", listed.page.Total, len(listed.page.Items), queries)
		}
	case <-time.After(time.Second):
		t.Fatal("list did not complete")
	}
	var persisted int
	if err := writer.SQL().QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&persisted); err != nil {
		t.Fatal(err)
	}
	if persisted != 7 {
		t.Fatalf("persisted orders = %d, want 7", persisted)
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
	if _, _, err := database.GetOrder(context.Background(), "ord_00000000000000000000000000000001"); !errors.Is(err, order.ErrInternal) {
		t.Fatalf("corrupt order error = %v", err)
	}
	database.Close()
	if _, _, err := database.GetOrder(context.Background(), "ord_00000000000000000000000000000002"); !errors.Is(err, order.ErrInternal) {
		t.Fatalf("closed database detail error = %v", err)
	}
	if _, err := database.ListOrders(context.Background(), order.ListQuery{Page: 1, PageSize: 20, Sort: "createdAt"}); !errors.Is(err, order.ErrInternal) {
		t.Fatalf("closed database list error = %v", err)
	}
}

func TestOrderRepositoryRejectsNonCanonicalTimes(t *testing.T) {
	for _, column := range []string{"created_at", "updated_at"} {
		t.Run(column, func(t *testing.T) {
			database := openSeededOrderDatabase(t)
			if _, err := database.SQL().Exec(`PRAGMA ignore_check_constraints = ON; UPDATE orders SET ` + column + ` = '2026-01-01T08:00:00+08:00' WHERE id = 'ord_00000000000000000000000000000001'; PRAGMA ignore_check_constraints = OFF;`); err != nil {
				t.Fatal(err)
			}
			if _, _, err := database.GetOrder(context.Background(), "ord_00000000000000000000000000000001"); !errors.Is(err, order.ErrInternal) {
				t.Fatalf("detail corrupt time error = %v", err)
			}
			if _, err := database.ListOrders(context.Background(), order.ListQuery{Page: 1, PageSize: 20, Sort: "createdAt"}); !errors.Is(err, order.ErrInternal) {
				t.Fatalf("list corrupt time error = %v", err)
			}
		})
	}
}

func TestOrderRepositorySearchFoldsOnlyASCII(t *testing.T) {
	database := openSeededOrderDatabase(t)
	fixtures := []struct {
		id, customer string
	}{
		{"ord_0000000000000000000000000000000a", "Änne"},
		{"ord_0000000000000000000000000000000b", "上海客户"},
	}
	for _, fixture := range fixtures {
		if _, err := database.SQL().Exec(`INSERT INTO orders(id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at) VALUES (?, ?, 'DRAFT', 'UNPAID', 'CNY', 1, 1, '2026-02-01T00:00:00Z', '2026-02-01T00:00:00Z')`, fixture.id, fixture.customer); err != nil {
			t.Fatal(err)
		}
	}
	for _, keyword := range []string{"Änne", "ÄNNE", "上海客户"} {
		page, err := database.ListOrders(context.Background(), order.ListQuery{Keyword: keyword, Page: 1, PageSize: 20, Sort: "createdAt"})
		if err != nil {
			t.Fatal(err)
		}
		if page.Total != 1 || len(page.Items) != 1 {
			t.Fatalf("keyword %q page = %+v", keyword, page)
		}
	}
	page, err := database.ListOrders(context.Background(), order.ListQuery{Keyword: "änne", Page: 1, PageSize: 20, Sort: "createdAt"})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 0 {
		t.Fatalf("non-ASCII folded unexpectedly: %+v", page)
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
