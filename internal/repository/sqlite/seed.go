package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/magicvr/allinme.core-api/internal/domain"
	"github.com/magicvr/allinme.core-api/internal/port"
)

// SeedUsers inserts demo users when the users table is empty.
// Password for all seeds: Demo@1234 (demo only).
func SeedUsers(ctx context.Context, users port.UserRepository, hasher port.PasswordHasher) error {
	n, err := users.Count(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	hash, err := hasher.Hash("Demo@1234")
	if err != nil {
		return err
	}
	seeds := []domain.User{
		{ID: "usr_admin", Username: "admin", Name: "Admin", PasswordHash: hash, Roles: []string{"admin"}},
		{ID: "usr_operator", Username: "operator", Name: "Operator", PasswordHash: hash, Roles: []string{"operator"}},
		{ID: "usr_viewer", Username: "viewer", Name: "Viewer", PasswordHash: hash, Roles: []string{"viewer"}},
	}
	for _, u := range seeds {
		if err := users.Create(ctx, u); err != nil {
			return fmt.Errorf("seed user %s: %w", u.Username, err)
		}
	}
	return nil
}

// SeedOrders inserts deterministic demo records when the orders table is empty.
// The emptiness check and all inserts use one transaction so a failed seed leaves no partial data.
func SeedOrders(ctx context.Context, orders *OrderRepository) error {
	tx, err := orders.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("seed orders begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM orders`).Scan(&count); err != nil {
		return fmt.Errorf("seed orders count: %w", err)
	}
	if count > 0 {
		return tx.Commit()
	}
	for _, order := range orderSeeds() {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO orders (id, order_no, customer_name, status, amount_cents, currency, remark, version, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, order.ID, order.OrderNo, order.CustomerName, order.Status, order.AmountCents, order.Currency, order.Remark, order.Version, orderTimestamp(order.CreatedAt), orderTimestamp(order.UpdatedAt)); err != nil {
			return fmt.Errorf("seed order %s: %w", order.OrderNo, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("seed orders commit: %w", err)
	}
	return nil
}

func orderSeeds() []domain.Order {
	seededAt := time.Date(2026, time.July, 25, 0, 0, 0, 0, time.UTC)
	return []domain.Order{
		{ID: "ord_seed_pending", OrderNo: "ORD-1001", CustomerName: "Pending Customer", Status: domain.OrderStatusPending, AmountCents: 12800, Currency: "CNY", Remark: "pending demo", Version: 1, CreatedAt: seededAt, UpdatedAt: seededAt},
		{ID: "ord_seed_paid", OrderNo: "ORD-1002", CustomerName: "Paid Customer", Status: domain.OrderStatusPaid, AmountCents: 25600, Currency: "CNY", Remark: "paid demo", Version: 1, CreatedAt: seededAt.Add(time.Minute), UpdatedAt: seededAt.Add(time.Minute)},
		{ID: "ord_seed_cancelled", OrderNo: "ORD-1003", CustomerName: "Cancelled Customer", Status: domain.OrderStatusCancelled, AmountCents: 6400, Currency: "CNY", Remark: "cancelled demo", Version: 1, CreatedAt: seededAt.Add(2 * time.Minute), UpdatedAt: seededAt.Add(2 * time.Minute)},
		{ID: "ord_seed_refunded", OrderNo: "ORD-1004", CustomerName: "Refunded Customer", Status: domain.OrderStatusRefunded, AmountCents: 3200, Currency: "CNY", Remark: "refunded demo", Version: 1, CreatedAt: seededAt.Add(3 * time.Minute), UpdatedAt: seededAt.Add(3 * time.Minute)},
	}
}
