package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/magicvr/allinme.core-api/internal/domain"
	"github.com/magicvr/allinme.core-api/internal/port"
)

// OrderRepository is the SQLite implementation of port.OrderRepository.
type OrderRepository struct {
	db *sql.DB
}

// NewOrderRepository wraps db.
func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

const orderTimestampLayout = "2006-01-02T15:04:05.000000000Z"

func orderTimestamp(value time.Time) string {
	return value.UTC().Format(orderTimestampLayout)
}

// Create implements port.OrderRepository.
func (r *OrderRepository) Create(ctx context.Context, order domain.Order) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO orders (id, order_no, customer_name, status, amount_cents, currency, remark, version, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, order.ID, order.OrderNo, order.CustomerName, order.Status, order.AmountCents, order.Currency, order.Remark, order.Version, orderTimestamp(order.CreatedAt), orderTimestamp(order.UpdatedAt))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: orders.order_no") {
			return port.ErrOrderNoConflict
		}
		return fmt.Errorf("order create: %w", err)
	}
	return nil
}

// Get implements port.OrderRepository.
func (r *OrderRepository) Get(ctx context.Context, id string) (domain.Order, error) {
	row := r.db.QueryRowContext(ctx, orderSelect+` WHERE id = ?`, id)
	return scanOrder(row)
}

// List implements port.OrderRepository.
func (r *OrderRepository) List(ctx context.Context, filter port.OrderListFilter) ([]domain.Order, int, error) {
	maxInt := int(^uint(0) >> 1)
	if filter.Page < 1 || filter.PageSize < 1 || (filter.Page > 1 && filter.Page-1 > maxInt/filter.PageSize) {
		return nil, 0, port.ErrInvalidArgument
	}
	where, args := orderFilter(filter)
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM orders`+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("order list count: %w", err)
	}
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := r.db.QueryContext(ctx, orderSelect+where+` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("order list query: %w", err)
	}
	defer rows.Close()

	orders := make([]domain.Order, 0)
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, 0, err
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("order list rows: %w", err)
	}
	return orders, total, nil
}

// Update implements a pending order compare-and-swap update.
func (r *OrderRepository) Update(ctx context.Context, order domain.Order) error {
	result, err := r.db.ExecContext(ctx, `
UPDATE orders
SET customer_name = ?, amount_cents = ?, currency = ?, remark = ?, version = version + 1, updated_at = ?
WHERE id = ? AND version = ? AND status = ?
`, order.CustomerName, order.AmountCents, order.Currency, order.Remark, orderTimestamp(order.UpdatedAt), order.ID, order.Version, domain.OrderStatusPending)
	if err != nil {
		return fmt.Errorf("order update: %w", err)
	}
	return versionConflict(result)
}

// ChangeStatus implements a pending order compare-and-swap state transition.
func (r *OrderRepository) ChangeStatus(ctx context.Context, id string, version int64, status domain.OrderStatus, updatedAt time.Time) error {
	result, err := r.db.ExecContext(ctx, `
UPDATE orders
SET status = ?, version = version + 1, updated_at = ?
WHERE id = ? AND version = ? AND status = ?
`, status, orderTimestamp(updatedAt), id, version, domain.OrderStatusPending)
	if err != nil {
		return fmt.Errorf("order change status: %w", err)
	}
	return versionConflict(result)
}

// BatchDelete validates all targets then deletes them in one transaction.
func (r *OrderRepository) BatchDelete(ctx context.Context, ids []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("order batch delete begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, id := range ids {
		var status domain.OrderStatus
		err := tx.QueryRowContext(ctx, `SELECT status FROM orders WHERE id = ?`, id).Scan(&status)
		if errors.Is(err, sql.ErrNoRows) {
			return port.ErrOrderNotFound
		}
		if err != nil {
			return fmt.Errorf("order batch delete check: %w", err)
		}
		if status != domain.OrderStatusPending && status != domain.OrderStatusCancelled {
			return port.ErrInvalidState
		}
	}
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, `DELETE FROM orders WHERE id = ?`, id); err != nil {
			return fmt.Errorf("order batch delete delete: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("order batch delete commit: %w", err)
	}
	return nil
}

// Count implements port.OrderRepository.
func (r *OrderRepository) Count(ctx context.Context) (int, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM orders`).Scan(&count); err != nil {
		return 0, fmt.Errorf("order count: %w", err)
	}
	return count, nil
}

const orderSelect = `
SELECT id, order_no, customer_name, status, amount_cents, currency, remark, version, created_at, updated_at FROM orders`

func orderFilter(filter port.OrderListFilter) (string, []any) {
	clauses := make([]string, 0, 2)
	args := make([]any, 0, 3)
	if filter.Status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Query != "" {
		clauses = append(clauses, "(order_no LIKE ? OR customer_name LIKE ?)")
		query := "%" + filter.Query + "%"
		args = append(args, query, query)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

type orderScannable interface {
	Scan(dest ...any) error
}

func scanOrder(row orderScannable) (domain.Order, error) {
	var order domain.Order
	var createdAt, updatedAt string
	err := row.Scan(&order.ID, &order.OrderNo, &order.CustomerName, &order.Status, &order.AmountCents, &order.Currency, &order.Remark, &order.Version, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Order{}, port.ErrOrderNotFound
	}
	if err != nil {
		return domain.Order{}, fmt.Errorf("order scan: %w", err)
	}
	order.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return domain.Order{}, fmt.Errorf("order parse created at: %w", err)
	}
	order.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return domain.Order{}, fmt.Errorf("order parse updated at: %w", err)
	}
	return order, nil
}

func versionConflict(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("order rows affected: %w", err)
	}
	if affected != 1 {
		return port.ErrVersionConflict
	}
	return nil
}

var _ port.OrderRepository = (*OrderRepository)(nil)
