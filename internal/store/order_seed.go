package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/magicvr/allinme.core-api/internal/order"
)

const (
	orderDemoSeedName    = "order_demo"
	orderDemoSeedVersion = 1
)

type OrderSeedResult struct {
	Name        string
	FromVersion int
	ToVersion   int
}

type orderSeedEntry struct {
	orderID       string
	itemID        string
	customerName  string
	status        order.Status
	paymentStatus order.PaymentStatus
	totalAmount   int64
	createdAt     string
	sku           string
	itemName      string
}

var orderDemoEntries = []orderSeedEntry{
	{orderID: "ord_00000000000000000000000000000001", itemID: "itm_00000000000000000000000000000001", customerName: "Draft Demo Customer", status: order.StatusDraft, paymentStatus: order.PaymentStatusUnpaid, totalAmount: 10000, createdAt: "2026-01-01T00:00:00Z", sku: "DEMO-DRAFT", itemName: "Draft Demo Item"},
	{orderID: "ord_00000000000000000000000000000002", itemID: "itm_00000000000000000000000000000002", customerName: "Confirmed Demo Customer", status: order.StatusConfirmed, paymentStatus: order.PaymentStatusUnpaid, totalAmount: 20000, createdAt: "2026-01-01T01:00:00Z", sku: "DEMO-CONFIRMED", itemName: "Confirmed Demo Item"},
	{orderID: "ord_00000000000000000000000000000003", itemID: "itm_00000000000000000000000000000003", customerName: "Fulfilling Demo Customer", status: order.StatusFulfilling, paymentStatus: order.PaymentStatusPaid, totalAmount: 30000, createdAt: "2026-01-01T02:00:00Z", sku: "DEMO-FULFILLING", itemName: "Fulfilling Demo Item"},
	{orderID: "ord_00000000000000000000000000000004", itemID: "itm_00000000000000000000000000000004", customerName: "Shipped Demo Customer", status: order.StatusShipped, paymentStatus: order.PaymentStatusPaid, totalAmount: 40000, createdAt: "2026-01-01T03:00:00Z", sku: "DEMO-SHIPPED", itemName: "Shipped Demo Item"},
	{orderID: "ord_00000000000000000000000000000005", itemID: "itm_00000000000000000000000000000005", customerName: "Completed Demo Customer", status: order.StatusCompleted, paymentStatus: order.PaymentStatusPaid, totalAmount: 50000, createdAt: "2026-01-01T04:00:00Z", sku: "DEMO-COMPLETED", itemName: "Completed Demo Item"},
	{orderID: "ord_00000000000000000000000000000006", itemID: "itm_00000000000000000000000000000006", customerName: "Cancelled Demo Customer", status: order.StatusCancelled, paymentStatus: order.PaymentStatusUnpaid, totalAmount: 60000, createdAt: "2026-01-01T05:00:00Z", sku: "DEMO-CANCELLED", itemName: "Cancelled Demo Item"},
}

func (database *DB) SeedOrderDemo(ctx context.Context, appliedAt time.Time) (OrderSeedResult, error) {
	result := OrderSeedResult{Name: orderDemoSeedName, ToVersion: orderDemoSeedVersion}
	err := database.WithTx(ctx, func(transaction *sql.Tx) error {
		var current int
		err := transaction.QueryRowContext(ctx, `SELECT version FROM seed_versions WHERE name = ?`, orderDemoSeedName).Scan(&current)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("read seed version %q: %w", orderDemoSeedName, err)
		}
		result.FromVersion = current
		if current > orderDemoSeedVersion {
			return fmt.Errorf("seed %q version %d is newer than supported version %d", orderDemoSeedName, current, orderDemoSeedVersion)
		}
		if current == orderDemoSeedVersion {
			if err := validateOrderDemoSeed(ctx, transaction); err != nil {
				return fmt.Errorf("order demo seed is inconsistent; reset is required: %w", err)
			}
			return nil
		}
		for _, entry := range orderDemoEntries {
			if _, err := transaction.ExecContext(ctx, `
				INSERT INTO orders(id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at)
				VALUES (?, ?, ?, ?, 'CNY', ?, 1, ?, ?)
			`, entry.orderID, entry.customerName, entry.status, entry.paymentStatus, entry.totalAmount, entry.createdAt, entry.createdAt); err != nil {
				return fmt.Errorf("insert order demo order: %w", err)
			}
			if _, err := transaction.ExecContext(ctx, `
				INSERT INTO order_items(id, order_id, position, sku, name, quantity, unit_price)
				VALUES (?, ?, 0, ?, ?, 1, ?)
			`, entry.itemID, entry.orderID, entry.sku, entry.itemName, entry.totalAmount); err != nil {
				return fmt.Errorf("insert order demo item: %w", err)
			}
		}
		_, err = transaction.ExecContext(ctx, `INSERT INTO seed_versions(name, version, applied_at) VALUES (?, ?, ?)`, orderDemoSeedName, orderDemoSeedVersion, appliedAt.UTC().Format(time.RFC3339))
		if err != nil {
			return fmt.Errorf("record order demo seed: %w", err)
		}
		return nil
	})
	if err != nil {
		return OrderSeedResult{}, err
	}
	return result, nil
}

func validateOrderDemoSeed(ctx context.Context, transaction *sql.Tx) error {
	for _, entry := range orderDemoEntries {
		var customerName, status, paymentStatus, currency, createdAt, updatedAt string
		var totalAmount, version int64
		err := transaction.QueryRowContext(ctx, `
			SELECT customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at
			FROM orders WHERE id = ?
		`, entry.orderID).Scan(&customerName, &status, &paymentStatus, &currency, &totalAmount, &version, &createdAt, &updatedAt)
		if err != nil || customerName != entry.customerName || status != string(entry.status) || paymentStatus != string(entry.paymentStatus) || currency != "CNY" || totalAmount != entry.totalAmount || version != 1 || createdAt != entry.createdAt || updatedAt != entry.createdAt {
			return fmt.Errorf("order %s differs from seed contract", entry.orderID)
		}
		var itemID, sku, itemName string
		var position, quantity, unitPrice int64
		var itemCount int
		if err := transaction.QueryRowContext(ctx, `SELECT COUNT(*) FROM order_items WHERE order_id = ?`, entry.orderID).Scan(&itemCount); err != nil || itemCount != 1 {
			return fmt.Errorf("order %s has unexpected item count", entry.orderID)
		}
		err = transaction.QueryRowContext(ctx, `
			SELECT id, position, sku, name, quantity, unit_price
			FROM order_items WHERE order_id = ?
		`, entry.orderID).Scan(&itemID, &position, &sku, &itemName, &quantity, &unitPrice)
		if err != nil || itemID != entry.itemID || position != 0 || sku != entry.sku || itemName != entry.itemName || quantity != 1 || unitPrice != entry.totalAmount {
			return fmt.Errorf("item for order %s differs from seed contract", entry.orderID)
		}
	}
	return nil
}
