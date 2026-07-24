package sqlite_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/domain"
	"github.com/magicvr/allinme.core-api/internal/port"
	"github.com/magicvr/allinme.core-api/internal/repository/sqlite"
)

func TestSeedOrdersRollsBackOnInsertFailure(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "seed-rollback.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository := sqlite.NewOrderRepository(db)

	if _, err := db.Exec(`
CREATE TRIGGER fail_order_seed
BEFORE INSERT ON orders
WHEN NEW.order_no = 'ORD-1002'
BEGIN
	SELECT RAISE(ABORT, 'forced seed failure');
END;`); err != nil {
		t.Fatal(err)
	}
	if err := sqlite.SeedOrders(ctx, repository); err == nil {
		t.Fatal("SeedOrders succeeded despite forced insert failure")
	}
	if count, err := repository.Count(ctx); err != nil || count != 0 {
		t.Fatalf("failed seed count = %d, err=%v; want 0", count, err)
	}
	if _, err := db.Exec(`DROP TRIGGER fail_order_seed`); err != nil {
		t.Fatal(err)
	}
	if err := sqlite.SeedOrders(ctx, repository); err != nil {
		t.Fatal(err)
	}
	if count, err := repository.Count(ctx); err != nil || count != 4 {
		t.Fatalf("retry seed count = %d, err=%v; want 4", count, err)
	}
}

func TestOrderRepositoryListCASBatchAndSeed(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "orders.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository := sqlite.NewOrderRepository(db)

	if err := sqlite.SeedOrders(ctx, repository); err != nil {
		t.Fatal(err)
	}
	if err := sqlite.SeedOrders(ctx, repository); err != nil {
		t.Fatal(err)
	}
	if count, err := repository.Count(ctx); err != nil || count != 4 {
		t.Fatalf("seed count = %d, %v; want 4", count, err)
	}
	allSeeded, seededTotal, err := repository.List(ctx, port.OrderListFilter{Page: 1, PageSize: 20})
	if err != nil || seededTotal != 4 {
		t.Fatalf("seed list total = %d, err=%v", seededTotal, err)
	}
	seededStates := make(map[domain.OrderStatus]bool)
	for _, order := range allSeeded {
		seededStates[order.Status] = true
	}
	for _, status := range []domain.OrderStatus{domain.OrderStatusPending, domain.OrderStatusPaid, domain.OrderStatusCancelled, domain.OrderStatusRefunded} {
		if !seededStates[status] {
			t.Fatalf("missing seeded %s order", status)
		}
	}

	pending, total, err := repository.List(ctx, port.OrderListFilter{Status: domain.OrderStatusPending, Page: 1, PageSize: 20})
	if err != nil || total != 1 || len(pending) != 1 {
		t.Fatalf("pending list = %+v total=%d err=%v", pending, total, err)
	}
	matched, matchedTotal, err := repository.List(ctx, port.OrderListFilter{Query: "Paid Customer", Page: 1, PageSize: 20})
	if err != nil || matchedTotal != 1 || len(matched) != 1 || matched[0].Status != domain.OrderStatusPaid {
		t.Fatalf("q filter = %+v total=%d err=%v", matched, matchedTotal, err)
	}

	base := time.Date(2026, time.July, 25, 1, 0, 0, 0, time.UTC)
	for i := 0; i < 2; i++ {
		order := domain.Order{ID: "ord_extra_" + string(rune('a'+i)), OrderNo: "ORD-EXTRA-" + string(rune('A'+i)), CustomerName: "Pager", Status: domain.OrderStatusPending, AmountCents: 100, Currency: "CNY", Version: 1, CreatedAt: base.Add(time.Duration(i) * time.Minute), UpdatedAt: base.Add(time.Duration(i) * time.Minute)}
		if err := repository.Create(ctx, order); err != nil {
			t.Fatal(err)
		}
	}
	pageOne, total, err := repository.List(ctx, port.OrderListFilter{Page: 1, PageSize: 1})
	if err != nil || total != 6 || len(pageOne) != 1 {
		t.Fatalf("page one len=%d total=%d err=%v", len(pageOne), total, err)
	}
	pageTwo, _, err := repository.List(ctx, port.OrderListFilter{Page: 2, PageSize: 1})
	if err != nil || len(pageTwo) != 1 || pageOne[0].ID == pageTwo[0].ID {
		t.Fatalf("page two = %+v err=%v", pageTwo, err)
	}
	second := time.Date(2026, time.July, 25, 3, 0, 0, 0, time.UTC)
	for _, order := range []domain.Order{
		{ID: "ord_same_second_zero", OrderNo: "ORD-SAME-0", CustomerName: "SameSecond", Status: domain.OrderStatusPending, AmountCents: 100, Currency: "CNY", Version: 1, CreatedAt: second, UpdatedAt: second},
		{ID: "ord_same_second_fraction", OrderNo: "ORD-SAME-100MS", CustomerName: "SameSecond", Status: domain.OrderStatusPending, AmountCents: 100, Currency: "CNY", Version: 1, CreatedAt: second.Add(100 * time.Millisecond), UpdatedAt: second.Add(100 * time.Millisecond)},
	} {
		if err := repository.Create(ctx, order); err != nil {
			t.Fatal(err)
		}
	}
	sameSecond, sameSecondTotal, err := repository.List(ctx, port.OrderListFilter{Query: "SameSecond", Page: 1, PageSize: 20})
	if err != nil || sameSecondTotal != 2 || len(sameSecond) != 2 || sameSecond[0].ID != "ord_same_second_fraction" || sameSecond[1].ID != "ord_same_second_zero" {
		t.Fatalf("same-second timestamp order = %+v total=%d err=%v", sameSecond, sameSecondTotal, err)
	}
	var storedTimestamp string
	if err := db.QueryRow(`SELECT created_at FROM orders WHERE id = 'ord_same_second_zero'`).Scan(&storedTimestamp); err != nil {
		t.Fatal(err)
	}
	if storedTimestamp != "2026-07-25T03:00:00.000000000Z" {
		t.Fatalf("stored timestamp = %q", storedTimestamp)
	}
	duplicate := pending[0]
	duplicate.ID = "ord_duplicate"
	if err := repository.Create(ctx, duplicate); !errors.Is(err, port.ErrOrderNoConflict) {
		t.Fatalf("duplicate order no error = %v", err)
	}

	updated := pending[0]
	updated.CustomerName = "Changed"
	updated.UpdatedAt = base.Add(time.Hour)
	if err := repository.Update(ctx, updated); err != nil {
		t.Fatal(err)
	}
	if err := repository.Update(ctx, updated); !errors.Is(err, port.ErrVersionConflict) {
		t.Fatalf("stale update error = %v", err)
	}
	if err := repository.ChangeStatus(ctx, updated.ID, 2, domain.OrderStatusPaid, base.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := repository.ChangeStatus(ctx, updated.ID, 2, domain.OrderStatusCancelled, base.Add(3*time.Hour)); !errors.Is(err, port.ErrVersionConflict) {
		t.Fatalf("stale transition error = %v", err)
	}

	cancelled, _, err := repository.List(ctx, port.OrderListFilter{Status: domain.OrderStatusCancelled, Page: 1, PageSize: 20})
	if err != nil || len(cancelled) != 1 {
		t.Fatalf("cancelled lookup = %+v err=%v", cancelled, err)
	}
	if err := repository.BatchDelete(ctx, []string{cancelled[0].ID, "ord_seed_paid"}); !errors.Is(err, port.ErrInvalidState) {
		t.Fatalf("mixed batch error = %v", err)
	}
	if _, err := repository.Get(ctx, cancelled[0].ID); err != nil {
		t.Fatalf("cancelled order missing after rollback: %v", err)
	}
	if err := repository.BatchDelete(ctx, []string{cancelled[0].ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Get(ctx, cancelled[0].ID); !errors.Is(err, port.ErrOrderNotFound) {
		t.Fatalf("deleted order lookup error = %v", err)
	}
}
