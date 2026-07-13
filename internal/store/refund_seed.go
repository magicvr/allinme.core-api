package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/order"
)

const (
	refundDemoSeedName    = "refund_demo"
	refundDemoSeedVersion = 1
)

type RefundSeedResult struct {
	Name        string
	FromVersion int
	ToVersion   int
}

type refundOrderSeedEntry struct {
	orderID       string
	itemID        string
	customerName  string
	status        order.Status
	paymentStatus order.PaymentStatus
	totalAmount   int64
	version       int64
	createdAt     string
	updatedAt     string
	sku           string
	itemName      string
}

type refundSeedEntry struct {
	refundID            string
	orderID             string
	amount              int64
	reason              string
	status              order.RefundStatus
	version             int64
	requestedByUsername string
	decidedByUsername   string
	createdAt           string
	updatedAt           string
	decidedAt           string
}

var refundDemoOrderEntries = []refundOrderSeedEntry{
	{orderID: "ord_00000000000000000000000000000007", itemID: "itm_00000000000000000000000000000007", customerName: "Refund Available Demo Customer", status: order.StatusFulfilling, paymentStatus: order.PaymentStatusPaid, totalAmount: 70000, version: 1, createdAt: "2026-01-01T06:00:00Z", updatedAt: "2026-01-01T06:00:00Z", sku: "DEMO-REFUND-AVAILABLE", itemName: "Refund Available Demo Item"},
	{orderID: "ord_00000000000000000000000000000008", itemID: "itm_00000000000000000000000000000008", customerName: "Cancelled Paid Demo Customer", status: order.StatusCancelled, paymentStatus: order.PaymentStatusPaid, totalAmount: 80000, version: 1, createdAt: "2026-01-01T07:00:00Z", updatedAt: "2026-01-01T07:00:00Z", sku: "DEMO-CANCELLED-PAID", itemName: "Cancelled Paid Demo Item"},
	{orderID: "ord_00000000000000000000000000000009", itemID: "itm_00000000000000000000000000000009", customerName: "Partially Refunded Demo Customer", status: order.StatusShipped, paymentStatus: order.PaymentStatusPartiallyRefunded, totalAmount: 90000, version: 2, createdAt: "2026-01-01T08:00:00Z", updatedAt: "2026-01-02T05:00:00Z", sku: "DEMO-PARTIAL-REFUND", itemName: "Partially Refunded Demo Item"},
	{orderID: "ord_0000000000000000000000000000000a", itemID: "itm_0000000000000000000000000000000a", customerName: "Fully Refunded Demo Customer", status: order.StatusCompleted, paymentStatus: order.PaymentStatusRefunded, totalAmount: 100000, version: 2, createdAt: "2026-01-01T09:00:00Z", updatedAt: "2026-01-02T07:00:00Z", sku: "DEMO-FULL-REFUND", itemName: "Fully Refunded Demo Item"},
}

var refundDemoEntries = []refundSeedEntry{
	{refundID: "rfd_00000000000000000000000000000001", orderID: "ord_00000000000000000000000000000008", amount: 10000, reason: "pending primary", status: order.RefundStatusPending, version: 1, requestedByUsername: "operator", createdAt: "2026-01-02T00:00:00Z", updatedAt: "2026-01-02T00:00:00Z"},
	{refundID: "rfd_00000000000000000000000000000002", orderID: "ord_00000000000000000000000000000008", amount: 5000, reason: "pending secondary", status: order.RefundStatusPending, version: 1, requestedByUsername: "admin", createdAt: "2026-01-02T01:00:00Z", updatedAt: "2026-01-02T01:00:00Z"},
	{refundID: "rfd_00000000000000000000000000000003", orderID: "ord_00000000000000000000000000000008", amount: 5000, reason: "rejected request", status: order.RefundStatusRejected, version: 2, requestedByUsername: "operator", decidedByUsername: "admin", createdAt: "2026-01-02T02:00:00Z", updatedAt: "2026-01-02T03:00:00Z", decidedAt: "2026-01-02T03:00:00Z"},
	{refundID: "rfd_00000000000000000000000000000004", orderID: "ord_00000000000000000000000000000009", amount: 20000, reason: "partial refund", status: order.RefundStatusCompleted, version: 2, requestedByUsername: "operator", decidedByUsername: "approver", createdAt: "2026-01-02T04:00:00Z", updatedAt: "2026-01-02T05:00:00Z", decidedAt: "2026-01-02T05:00:00Z"},
	{refundID: "rfd_00000000000000000000000000000005", orderID: "ord_0000000000000000000000000000000a", amount: 100000, reason: "full refund", status: order.RefundStatusCompleted, version: 2, requestedByUsername: "admin", decidedByUsername: "approver", createdAt: "2026-01-02T06:00:00Z", updatedAt: "2026-01-02T07:00:00Z", decidedAt: "2026-01-02T07:00:00Z"},
}

func (database *DB) SeedRefundDemo(ctx context.Context, appliedAt time.Time) (RefundSeedResult, error) {
	result := RefundSeedResult{Name: refundDemoSeedName, ToVersion: refundDemoSeedVersion}
	err := database.WithTx(ctx, func(transaction *sql.Tx) error {
		var current int
		err := transaction.QueryRowContext(ctx, `SELECT version FROM seed_versions WHERE name = ?`, refundDemoSeedName).Scan(&current)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("read seed version %q: %w", refundDemoSeedName, err)
		}
		result.FromVersion = current
		if current > refundDemoSeedVersion {
			return fmt.Errorf("seed %q version %d is newer than supported version %d", refundDemoSeedName, current, refundDemoSeedVersion)
		}
		actors, err := loadRefundSeedActors(ctx, transaction)
		if err != nil {
			return err
		}
		if current == refundDemoSeedVersion {
			if err := validateRefundDemoSeed(ctx, transaction, actors); err != nil {
				return fmt.Errorf("refund demo seed is inconsistent; reset is required: %w", err)
			}
			return nil
		}
		if err := validateOrderDemoSeed(ctx, transaction); err != nil {
			return fmt.Errorf("order demo seed prerequisite is inconsistent: %w", err)
		}
		for _, entry := range refundDemoOrderEntries {
			if _, err := transaction.ExecContext(ctx, `
				INSERT INTO orders(id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at)
				VALUES (?, ?, ?, ?, 'CNY', ?, ?, ?, ?)
			`, entry.orderID, entry.customerName, entry.status, entry.paymentStatus, entry.totalAmount, entry.version, entry.createdAt, entry.updatedAt); err != nil {
				return fmt.Errorf("insert refund demo order: %w", err)
			}
			if _, err := transaction.ExecContext(ctx, `
				INSERT INTO order_items(id, order_id, position, sku, name, quantity, unit_price)
				VALUES (?, ?, 0, ?, ?, 1, ?)
			`, entry.itemID, entry.orderID, entry.sku, entry.itemName, entry.totalAmount); err != nil {
				return fmt.Errorf("insert refund demo item: %w", err)
			}
		}
		for _, entry := range refundDemoEntries {
			requestedBy := actors[entry.requestedByUsername].ID
			var decidedBy any
			var decidedAt any
			if entry.decidedByUsername != "" {
				decidedBy = actors[entry.decidedByUsername].ID
				decidedAt = entry.decidedAt
			}
			if _, err := transaction.ExecContext(ctx, `
				INSERT INTO refunds(id, order_id, amount, reason, status, version, requested_by, decided_by, created_at, updated_at, decided_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, entry.refundID, entry.orderID, entry.amount, entry.reason, entry.status, entry.version, requestedBy, decidedBy, entry.createdAt, entry.updatedAt, decidedAt); err != nil {
				return fmt.Errorf("insert refund demo refund: %w", err)
			}
		}
		if err := validateRefundDemoSeed(ctx, transaction, actors); err != nil {
			return fmt.Errorf("validate newly inserted refund demo seed: %w", err)
		}
		if _, err := transaction.ExecContext(ctx, `INSERT INTO seed_versions(name, version, applied_at) VALUES (?, ?, ?)`, refundDemoSeedName, refundDemoSeedVersion, appliedAt.UTC().Format(time.RFC3339)); err != nil {
			return fmt.Errorf("record refund demo seed: %w", err)
		}
		return nil
	})
	if err != nil {
		return RefundSeedResult{}, err
	}
	return result, nil
}

func loadRefundSeedActors(ctx context.Context, transaction *sql.Tx) (map[string]order.RefundActor, error) {
	wantRoles := map[string]auth.Role{"operator": auth.RoleOperator, "approver": auth.RoleApprover, "admin": auth.RoleAdmin}
	actors := make(map[string]order.RefundActor, len(wantRoles))
	for username, role := range wantRoles {
		var id, storedUsername, storedRole string
		var disabledAt sql.NullString
		if err := transaction.QueryRowContext(ctx, `SELECT id, username, role, disabled_at FROM users WHERE username = ?`, username).Scan(&id, &storedUsername, &storedRole, &disabledAt); err != nil {
			return nil, fmt.Errorf("read refund demo actor %q: %w", username, err)
		}
		if id == "" || storedUsername != username || storedRole != string(role) || disabledAt.Valid {
			return nil, fmt.Errorf("refund demo actor %q differs from seed contract", username)
		}
		actors[username] = order.RefundActor{ID: id, Username: storedUsername}
	}
	return actors, nil
}

func validateRefundDemoSeed(ctx context.Context, transaction *sql.Tx, actors map[string]order.RefundActor) error {
	if err := validateOrderDemoSeed(ctx, transaction); err != nil {
		return err
	}
	for _, entry := range refundDemoOrderEntries {
		var customerName, status, paymentStatus, currency, createdAt, updatedAt string
		var totalAmount, version int64
		if err := transaction.QueryRowContext(ctx, `
			SELECT customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at
			FROM orders WHERE id = ?
		`, entry.orderID).Scan(&customerName, &status, &paymentStatus, &currency, &totalAmount, &version, &createdAt, &updatedAt); err != nil || customerName != entry.customerName || status != string(entry.status) || paymentStatus != string(entry.paymentStatus) || currency != "CNY" || totalAmount != entry.totalAmount || version != entry.version || createdAt != entry.createdAt || updatedAt != entry.updatedAt {
			return fmt.Errorf("order %s differs from refund seed contract", entry.orderID)
		}
		var itemID, sku, itemName string
		var position, quantity, unitPrice, itemCount int64
		if err := transaction.QueryRowContext(ctx, `SELECT COUNT(*) FROM order_items WHERE order_id = ?`, entry.orderID).Scan(&itemCount); err != nil || itemCount != 1 {
			return fmt.Errorf("order %s has unexpected refund seed item count", entry.orderID)
		}
		if err := transaction.QueryRowContext(ctx, `SELECT id, position, sku, name, quantity, unit_price FROM order_items WHERE order_id = ?`, entry.orderID).Scan(&itemID, &position, &sku, &itemName, &quantity, &unitPrice); err != nil || itemID != entry.itemID || position != 0 || sku != entry.sku || itemName != entry.itemName || quantity != 1 || unitPrice != entry.totalAmount {
			return fmt.Errorf("item for order %s differs from refund seed contract", entry.orderID)
		}
	}
	for _, entry := range refundDemoEntries {
		var orderID, reason, status, requestedBy, createdAt, updatedAt string
		var amount, version int64
		var decidedBy, decidedAt sql.NullString
		if err := transaction.QueryRowContext(ctx, `
			SELECT order_id, amount, reason, status, version, requested_by, decided_by, created_at, updated_at, decided_at
			FROM refunds WHERE id = ?
		`, entry.refundID).Scan(&orderID, &amount, &reason, &status, &version, &requestedBy, &decidedBy, &createdAt, &updatedAt, &decidedAt); err != nil {
			return fmt.Errorf("read refund %s: %w", entry.refundID, err)
		}
		wantDecidedBy := ""
		if entry.decidedByUsername != "" {
			wantDecidedBy = actors[entry.decidedByUsername].ID
		}
		if orderID != entry.orderID || amount != entry.amount || reason != entry.reason || status != string(entry.status) || version != entry.version || requestedBy != actors[entry.requestedByUsername].ID || decidedBy.String != wantDecidedBy || decidedBy.Valid != (wantDecidedBy != "") || createdAt != entry.createdAt || updatedAt != entry.updatedAt || decidedAt.String != entry.decidedAt || decidedAt.Valid != (entry.decidedAt != "") {
			return fmt.Errorf("refund %s differs from seed contract", entry.refundID)
		}
	}
	return validateRefundDemoArithmetic()
}

func validateRefundDemoArithmetic() error {
	statusCounts := make(map[order.Status]int)
	var grossAmount int64
	for _, entry := range orderDemoEntries {
		statusCounts[entry.status]++
		if entry.paymentStatus == order.PaymentStatusPaid || entry.paymentStatus == order.PaymentStatusPartiallyRefunded || entry.paymentStatus == order.PaymentStatusRefunded {
			grossAmount += entry.totalAmount
		}
	}
	for _, entry := range refundDemoOrderEntries {
		statusCounts[entry.status]++
		if entry.paymentStatus == order.PaymentStatusPaid || entry.paymentStatus == order.PaymentStatusPartiallyRefunded || entry.paymentStatus == order.PaymentStatusRefunded {
			grossAmount += entry.totalAmount
		}
	}
	var completedRefundAmount int64
	for _, entry := range refundDemoEntries {
		if entry.status == order.RefundStatusCompleted {
			completedRefundAmount += entry.amount
		}
	}
	wantCounts := map[order.Status]int{order.StatusDraft: 1, order.StatusConfirmed: 1, order.StatusFulfilling: 2, order.StatusShipped: 2, order.StatusCompleted: 2, order.StatusCancelled: 2}
	if len(orderDemoEntries)+len(refundDemoOrderEntries) != 10 || grossAmount != 460000 || completedRefundAmount != 120000 || grossAmount-completedRefundAmount != 340000 {
		return fmt.Errorf("refund demo fixture arithmetic differs from contract")
	}
	for status, want := range wantCounts {
		if statusCounts[status] != want {
			return fmt.Errorf("refund demo status %s count = %d, want %d", status, statusCounts[status], want)
		}
	}
	return nil
}
