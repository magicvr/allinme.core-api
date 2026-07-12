package store_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/order"
	"github.com/magicvr/allinme.core-api/internal/store"
)

func TestCreateOrderAndUpdateDraftTransactions(t *testing.T) {
	ctx := context.Background()
	database, err := store.Open(ctx, filepath.Join(t.TempDir(), "orders.db"), store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	created, err := database.CreateOrder(ctx, order.CreatePersistence{
		ID: "ord_0000000000000000000000000000000a", CustomerName: "Customer", Currency: "CNY", TotalAmount: 500, CreatedAt: "2026-07-12T00:00:00Z",
		Items: []order.PersistenceItem{{ID: "itm_0000000000000000000000000000000a", Position: 0, SKU: "SKU", Name: "Item", Quantity: 2, UnitPrice: 250}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != order.StatusDraft || created.PaymentStatus != order.PaymentStatusUnpaid || created.Version != 1 || len(created.Items) != 1 {
		t.Fatalf("created = %+v", created)
	}
	updated, err := database.UpdateDraft(ctx, order.UpdateDraftPersistence{
		ID: created.ID, CustomerName: "Updated", Currency: "CNY", TotalAmount: 900, Version: 1, UpdatedAt: "2026-07-12T01:00:00Z",
		Items: []order.PersistenceItem{{ID: "itm_0000000000000000000000000000000b", Position: 0, SKU: "NEW", Name: "New", Quantity: 3, UnitPrice: 300}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.CustomerName != "Updated" || updated.Version != 2 || updated.TotalAmount != 900 || len(updated.Items) != 1 || updated.Items[0].SKU != "NEW" || updated.CreatedAt.Format("2006-01-02T15:04:05Z") != "2026-07-12T00:00:00Z" {
		t.Fatalf("updated = %+v", updated)
	}
	if _, err := database.UpdateDraft(ctx, order.UpdateDraftPersistence{ID: created.ID, CustomerName: "Conflict", Currency: "CNY", TotalAmount: 1, Version: 1, UpdatedAt: "2026-07-12T02:00:00Z", Items: []order.PersistenceItem{{ID: "itm_0000000000000000000000000000000c", SKU: "X", Name: "X", Quantity: 1, UnitPrice: 1}}}); !errors.Is(err, order.ErrVersionConflict) {
		t.Fatalf("stale UpdateDraft() error = %v", err)
	}
}

func TestCreateOrderRollsBackWholeAggregate(t *testing.T) {
	ctx := context.Background()
	database, err := store.Open(ctx, filepath.Join(t.TempDir(), "rollback.db"), store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	_, err = database.CreateOrder(ctx, order.CreatePersistence{
		ID: "ord_0000000000000000000000000000000a", CustomerName: "Customer", Currency: "CNY", TotalAmount: 1, CreatedAt: "2026-07-12T00:00:00Z",
		Items: []order.PersistenceItem{{ID: "bad", Position: 0, SKU: "SKU", Name: "Item", Quantity: 1, UnitPrice: 1}},
	})
	if err == nil {
		t.Fatal("CreateOrder() error = nil")
	}
	var orders, items int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&orders); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM order_items`).Scan(&items); err != nil {
		t.Fatal(err)
	}
	if orders != 0 || items != 0 {
		t.Fatalf("orders=%d items=%d", orders, items)
	}
}

func TestUpdateDraftClassifiesMissingStateAndRollsBackItems(t *testing.T) {
	database := openSeededOrderDatabaseForWrite(t)
	ctx := context.Background()
	if _, err := database.UpdateDraft(ctx, order.UpdateDraftPersistence{ID: "ord_ffffffffffffffffffffffffffffffff", Version: 1}); !errors.Is(err, order.ErrNotFound) {
		t.Fatalf("missing UpdateDraft() error = %v", err)
	}
	confirmedID := "ord_00000000000000000000000000000002"
	if _, err := database.UpdateDraft(ctx, order.UpdateDraftPersistence{ID: confirmedID, Version: 1}); !errors.Is(err, order.ErrStateConflict) {
		t.Fatalf("confirmed UpdateDraft() error = %v", err)
	}
	draftID := "ord_00000000000000000000000000000001"
	_, err := database.UpdateDraft(ctx, order.UpdateDraftPersistence{ID: draftID, CustomerName: "Should Roll Back", Currency: "CNY", TotalAmount: 1, Version: 1, UpdatedAt: "2026-07-12T03:00:00Z", Items: []order.PersistenceItem{{ID: "bad", SKU: "X", Name: "X", Quantity: 1, UnitPrice: 1}}})
	if err == nil {
		t.Fatal("invalid item UpdateDraft() error = nil")
	}
	got, found, err := database.GetOrder(ctx, draftID)
	if err != nil || !found {
		t.Fatal(err)
	}
	if got.CustomerName != "Draft Demo Customer" || got.Version != 1 || got.Items[0].SKU != "DEMO-DRAFT" || got.UpdatedAt.Format("2006-01-02T15:04:05Z") != "2026-01-01T00:00:00Z" {
		t.Fatalf("rolled back order = %+v", got)
	}
}

func openSeededOrderDatabaseForWrite(t *testing.T) *store.DB {
	t.Helper()
	database, err := store.Open(context.Background(), filepath.Join(t.TempDir(), "seeded.db"), store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	if _, err := database.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SeedOrderDemo(context.Background(), testTime()); err != nil {
		t.Fatal(err)
	}
	return database
}

func testTime() (value time.Time) {
	return time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)
}
