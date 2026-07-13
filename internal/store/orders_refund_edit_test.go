package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/order"
	"github.com/magicvr/allinme.core-api/internal/store"
)

func TestEditDraftValidatesRefundOccupancyAndRecomputesPaymentStatus(t *testing.T) {
	t.Run("pending occupancy remains paid", func(t *testing.T) {
		database := openDraftRefundDatabase(t, order.PaymentStatusPaid, 1, "PENDING", 1, 20)
		service := orderServiceForRefundEdit(t, database, func() (string, error) { return "itm_dddddddddddddddddddddddddddddddd", nil })
		result, err := service.Edit(context.Background(), auth.Principal{Role: auth.RoleAdmin}, "ord_cccccccccccccccccccccccccccccccc", order.EditCommand{
			CustomerName: "Updated", Currency: "CNY", Version: 1,
			Items: []order.ItemCommand{{SKU: "NEW", Name: "Updated Item", Quantity: 1, UnitPrice: 80}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.TotalAmount != 80 || result.PaymentStatus != order.PaymentStatusPaid || result.Version != 2 || result.AvailableRefundAmount != 60 {
			t.Fatalf("edited order = %+v", result)
		}
		var refundStatus string
		var refundVersion int64
		if err := database.SQL().QueryRow(`SELECT status, version FROM refunds WHERE id = 'rfd_cccccccccccccccccccccccccccccccc'`).Scan(&refundStatus, &refundVersion); err != nil {
			t.Fatal(err)
		}
		if refundStatus != "PENDING" || refundVersion != 1 {
			t.Fatalf("edit changed refund = %s/%d", refundStatus, refundVersion)
		}
	})
	t.Run("completed amount equal new total becomes refunded", func(t *testing.T) {
		database := openDraftRefundDatabase(t, order.PaymentStatusPartiallyRefunded, 2, "COMPLETED", 2, 20)
		service := orderServiceForRefundEdit(t, database, func() (string, error) { return "itm_dddddddddddddddddddddddddddddddd", nil })
		result, err := service.Edit(context.Background(), auth.Principal{Role: auth.RoleAdmin}, "ord_cccccccccccccccccccccccccccccccc", order.EditCommand{
			CustomerName: "Updated", Currency: "CNY", Version: 2,
			Items: []order.ItemCommand{{SKU: "NEW", Name: "Updated Item", Quantity: 1, UnitPrice: 20}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.PaymentStatus != order.PaymentStatusRefunded || result.Version != 3 || result.AvailableRefundAmount != 0 {
			t.Fatalf("edited refunded order = %+v", result)
		}
	})
}

func TestEditDraftRejectsTotalBelowOccupiedAndRollsBack(t *testing.T) {
	database := openDraftRefundDatabase(t, order.PaymentStatusPaid, 1, "PENDING", 1, 20)
	service := orderServiceForRefundEdit(t, database, func() (string, error) { return "itm_dddddddddddddddddddddddddddddddd", nil })
	_, err := service.Edit(context.Background(), auth.Principal{Role: auth.RoleAdmin}, "ord_cccccccccccccccccccccccccccccccc", order.EditCommand{
		CustomerName: "Too low", Currency: "CNY", Version: 1,
		Items: []order.ItemCommand{{SKU: "LOW", Name: "Low Item", Quantity: 1, UnitPrice: 10}},
	})
	details, ok := order.ValidationDetails(err)
	if !ok || len(details) != 1 || details[0].Field != "items" || details[0].Message != "calculated total must not be less than occupied refund amount" {
		t.Fatalf("edit validation = %v, %+v", err, details)
	}
	var total, version int64
	var customer string
	if err := database.SQL().QueryRow(`SELECT customer_name, total_amount, version FROM orders WHERE id = 'ord_cccccccccccccccccccccccccccccccc'`).Scan(&customer, &total, &version); err != nil {
		t.Fatal(err)
	}
	if customer != "Draft Refund Customer" || total != 100 || version != 1 {
		t.Fatalf("failed edit changed order = %s/%d/%d", customer, total, version)
	}
	var itemID string
	if err := database.SQL().QueryRow(`SELECT id FROM order_items WHERE order_id = 'ord_cccccccccccccccccccccccccccccccc'`).Scan(&itemID); err != nil {
		t.Fatal(err)
	}
	if itemID != "itm_cccccccccccccccccccccccccccccccc" {
		t.Fatalf("failed edit changed item = %s", itemID)
	}
}

func TestEditDraftRejectsCorruptRefundAggregateBeforeGeneratingItems(t *testing.T) {
	database := openDraftRefundDatabase(t, order.PaymentStatusPaid, 1, "PENDING", 1, 20)
	if _, err := database.SQL().Exec(`UPDATE orders SET payment_status = 'PARTIALLY_REFUNDED' WHERE id = 'ord_cccccccccccccccccccccccccccccccc'`); err != nil {
		t.Fatal(err)
	}
	generatorCalls := 0
	service := orderServiceForRefundEdit(t, database, func() (string, error) {
		generatorCalls++
		return "itm_dddddddddddddddddddddddddddddddd", nil
	})
	_, err := service.Edit(context.Background(), auth.Principal{Role: auth.RoleAdmin}, "ord_cccccccccccccccccccccccccccccccc", order.EditCommand{
		CustomerName: "Updated", Currency: "CNY", Version: 1,
		Items: []order.ItemCommand{{SKU: "NEW", Name: "Updated Item", Quantity: 1, UnitPrice: 100}},
	})
	if !errors.Is(err, order.ErrInternal) || generatorCalls != 0 {
		t.Fatalf("corrupt edit error = %v, generator calls = %d", err, generatorCalls)
	}
}

func TestTransitionReturnsCurrentRefundAvailability(t *testing.T) {
	database := openRefundDemoDatabase(t)
	service, err := order.NewService(database)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Transition(context.Background(), auth.Principal{Role: auth.RoleAdmin}, "ord_00000000000000000000000000000007", order.ActionShip, order.TransitionCommand{Version: 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != order.StatusShipped || result.AvailableRefundAmount != 70000 {
		t.Fatalf("transitioned order = %+v", result)
	}
}

func openDraftRefundDatabase(t *testing.T, paymentStatus order.PaymentStatus, orderVersion int64, refundStatus string, refundVersion int64, refundAmount int64) *store.DB {
	t.Helper()
	database := openMigrated(t)
	insertRefundSeedActors(t, database)
	operatorID := refundSeedActorID(t, database, "operator")
	approverID := refundSeedActorID(t, database, "approver")
	createdAt := "2026-01-01T00:00:00Z"
	updatedAt := createdAt
	if refundStatus == "COMPLETED" {
		updatedAt = "2026-01-02T00:00:00Z"
	}
	if _, err := database.SQL().Exec(`
		INSERT INTO orders(id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at)
		VALUES ('ord_cccccccccccccccccccccccccccccccc', 'Draft Refund Customer', 'DRAFT', ?, 'CNY', 100, ?, ?, ?);
		INSERT INTO order_items(id, order_id, position, sku, name, quantity, unit_price)
		VALUES ('itm_cccccccccccccccccccccccccccccccc', 'ord_cccccccccccccccccccccccccccccccc', 0, 'ORIGINAL', 'Original Item', 1, 100);
	`, paymentStatus, orderVersion, createdAt, updatedAt); err != nil {
		t.Fatal(err)
	}
	if refundStatus == "PENDING" {
		if _, err := database.SQL().Exec(`
			INSERT INTO refunds(id, order_id, amount, reason, status, version, requested_by, created_at, updated_at)
			VALUES ('rfd_cccccccccccccccccccccccccccccccc', 'ord_cccccccccccccccccccccccccccccccc', ?, 'pending edit refund', 'PENDING', ?, ?, '2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z')
		`, refundAmount, refundVersion, operatorID); err != nil {
			t.Fatal(err)
		}
	} else {
		if _, err := database.SQL().Exec(`
			INSERT INTO refunds(id, order_id, amount, reason, status, version, requested_by, decided_by, created_at, updated_at, decided_at)
			VALUES ('rfd_cccccccccccccccccccccccccccccccc', 'ord_cccccccccccccccccccccccccccccccc', ?, 'completed edit refund', 'COMPLETED', ?, ?, ?, '2026-01-01T12:00:00Z', '2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z')
		`, refundAmount, refundVersion, operatorID, approverID); err != nil {
			t.Fatal(err)
		}
	}
	return database
}

func orderServiceForRefundEdit(t *testing.T, database *store.DB, newItemID func() (string, error)) *order.Service {
	t.Helper()
	service, err := order.NewServiceWithDependencies(database, func() time.Time { return time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC) }, nil, newItemID)
	if err != nil {
		t.Fatal(err)
	}
	return service
}
