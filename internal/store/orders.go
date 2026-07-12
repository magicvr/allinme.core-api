package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/magicvr/allinme.core-api/internal/order"
)

func (database *DB) ListOrders(ctx context.Context, query order.ListQuery) (page order.Page, resultErr error) {
	transaction, err := database.sql.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return order.Page{}, fmt.Errorf("begin order list transaction: %w", err)
	}
	defer func() {
		if resultErr != nil {
			_ = transaction.Rollback()
		}
	}()

	where, arguments := orderWhere(query)
	database.observeQuery()
	if err := transaction.QueryRowContext(ctx, "SELECT COUNT(*) FROM orders o "+where, arguments...).Scan(&page.Total); err != nil {
		return order.Page{}, fmt.Errorf("count orders: %w", err)
	}
	sortColumn := map[string]string{
		"createdAt": "o.created_at", "updatedAt": "o.updated_at", "totalAmount": "o.total_amount",
		"customerName": "o.customer_name", "status": "o.status",
	}[query.Sort]
	if sortColumn == "" {
		sortColumn = "o.created_at"
	}
	direction := "ASC"
	if query.Descending {
		direction = "DESC"
	}
	offset := (query.Page - 1) * query.PageSize
	pageArguments := append(append([]any{}, arguments...), query.PageSize, offset)
	database.observeQuery()
	rows, err := transaction.QueryContext(ctx, `SELECT o.id, o.customer_name, o.status, o.payment_status, o.currency, o.total_amount, o.version, o.created_at, o.updated_at FROM orders o `+where+` ORDER BY `+sortColumn+` `+direction+`, o.id `+direction+` LIMIT ? OFFSET ?`, pageArguments...)
	if err != nil {
		return order.Page{}, fmt.Errorf("query orders: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		result, scanErr := scanOrder(rows)
		if scanErr != nil {
			return order.Page{}, scanErr
		}
		page.Items = append(page.Items, result)
	}
	if err := rows.Err(); err != nil {
		return order.Page{}, fmt.Errorf("iterate orders: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return order.Page{}, fmt.Errorf("commit order list transaction: %w", err)
	}
	page.Page, page.PageSize = query.Page, query.PageSize
	if page.Items == nil {
		page.Items = []order.Order{}
	}
	return page, nil
}

func orderWhere(query order.ListQuery) (string, []any) {
	conditions := []string{"1 = 1"}
	arguments := []any{}
	if query.Keyword != "" {
		keyword := "%" + escapeLike(strings.ToLower(query.Keyword)) + "%"
		conditions = append(conditions, `(lower(o.id) LIKE ? ESCAPE '\' OR lower(o.customer_name) LIKE ? ESCAPE '\' OR EXISTS (SELECT 1 FROM order_items oi WHERE oi.order_id = o.id AND (lower(oi.sku) LIKE ? ESCAPE '\' OR lower(oi.name) LIKE ? ESCAPE '\')))`)
		arguments = append(arguments, keyword, keyword, keyword, keyword)
	}
	if query.Status != "" {
		conditions = append(conditions, "o.status = ?")
		arguments = append(arguments, query.Status)
	}
	if query.PaymentStatus != "" {
		conditions = append(conditions, "o.payment_status = ?")
		arguments = append(arguments, query.PaymentStatus)
	}
	if query.CreatedFrom != nil {
		conditions = append(conditions, "o.created_at >= ?")
		arguments = append(arguments, query.CreatedFrom.UTC().Format(time.RFC3339))
	}
	if query.CreatedTo != nil {
		conditions = append(conditions, "o.created_at <= ?")
		arguments = append(arguments, query.CreatedTo.UTC().Format(time.RFC3339))
	}
	return "WHERE " + strings.Join(conditions, " AND "), arguments
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return strings.ReplaceAll(value, `_`, `\_`)
}

func (database *DB) GetOrder(ctx context.Context, id string) (order.Order, bool, error) {
	transaction, err := database.sql.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return order.Order{}, false, fmt.Errorf("begin order detail transaction: %w", err)
	}
	defer transaction.Rollback()
	result, found, err := getOrderTx(ctx, transaction, id)
	if err != nil || !found {
		return order.Order{}, found, err
	}
	if err := transaction.Commit(); err != nil {
		return order.Order{}, false, fmt.Errorf("commit order detail transaction: %w", err)
	}
	return result, true, nil
}

func (database *DB) CreateOrder(ctx context.Context, persistence order.CreatePersistence) (result order.Order, resultErr error) {
	transaction, err := database.sql.BeginTx(ctx, nil)
	if err != nil {
		return order.Order{}, fmt.Errorf("begin create order transaction: %w", err)
	}
	defer func() {
		if resultErr != nil {
			_ = transaction.Rollback()
		}
	}()
	if _, err := transaction.ExecContext(ctx, `INSERT INTO orders(id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at) VALUES (?, ?, 'DRAFT', 'UNPAID', ?, ?, 1, ?, ?)`, persistence.ID, persistence.CustomerName, persistence.Currency, persistence.TotalAmount, persistence.CreatedAt, persistence.CreatedAt); err != nil {
		return order.Order{}, fmt.Errorf("insert order: %w", err)
	}
	if err := insertOrderItems(ctx, transaction, persistence.ID, persistence.Items); err != nil {
		return order.Order{}, err
	}
	result, found, err := getOrderTx(ctx, transaction, persistence.ID)
	if err != nil {
		return order.Order{}, err
	}
	if !found {
		return order.Order{}, errors.New("created order is missing")
	}
	if err := transaction.Commit(); err != nil {
		return order.Order{}, fmt.Errorf("commit create order transaction: %w", err)
	}
	return result, nil
}

func (database *DB) UpdateDraft(ctx context.Context, persistence order.UpdateDraftPersistence) (result order.Order, resultErr error) {
	transaction, err := database.sql.BeginTx(ctx, nil)
	if err != nil {
		return order.Order{}, fmt.Errorf("begin edit order transaction: %w", err)
	}
	defer func() {
		if resultErr != nil {
			_ = transaction.Rollback()
		}
	}()
	current, found, err := getOrderTx(ctx, transaction, persistence.ID)
	if err != nil {
		return order.Order{}, err
	}
	if !found {
		return order.Order{}, order.ErrNotFound
	}
	if current.Version != persistence.Version {
		return order.Order{}, order.ErrVersionConflict
	}
	if current.Status != order.StatusDraft {
		return order.Order{}, order.ErrStateConflict
	}
	if current.Version == math.MaxInt64 {
		return order.Order{}, errors.New("order version exhausted")
	}
	update, err := transaction.ExecContext(ctx, `UPDATE orders SET customer_name = ?, currency = ?, total_amount = ?, version = version + 1, updated_at = ? WHERE id = ? AND status = 'DRAFT' AND version = ?`, persistence.CustomerName, persistence.Currency, persistence.TotalAmount, persistence.UpdatedAt, persistence.ID, persistence.Version)
	if err != nil {
		return order.Order{}, fmt.Errorf("update draft order: %w", err)
	}
	affected, err := update.RowsAffected()
	if err != nil {
		return order.Order{}, fmt.Errorf("read updated order rows: %w", err)
	}
	if affected != 1 {
		return order.Order{}, classifyUpdateMiss(ctx, transaction, persistence.ID, persistence.Version)
	}
	if _, err := transaction.ExecContext(ctx, `DELETE FROM order_items WHERE order_id = ?`, persistence.ID); err != nil {
		return order.Order{}, fmt.Errorf("delete draft order items: %w", err)
	}
	if err := insertOrderItems(ctx, transaction, persistence.ID, persistence.Items); err != nil {
		return order.Order{}, err
	}
	result, found, err = getOrderTx(ctx, transaction, persistence.ID)
	if err != nil {
		return order.Order{}, err
	}
	if !found {
		return order.Order{}, errors.New("updated order is missing")
	}
	if err := transaction.Commit(); err != nil {
		return order.Order{}, fmt.Errorf("commit edit order transaction: %w", err)
	}
	return result, nil
}

func classifyUpdateMiss(ctx context.Context, transaction *sql.Tx, id string, version int64) error {
	current, found, err := getOrderTx(ctx, transaction, id)
	if err != nil {
		return err
	}
	if !found {
		return order.ErrNotFound
	}
	if current.Version != version {
		return order.ErrVersionConflict
	}
	return order.ErrStateConflict
}

func insertOrderItems(ctx context.Context, transaction *sql.Tx, orderID string, items []order.PersistenceItem) error {
	for _, item := range items {
		if _, err := transaction.ExecContext(ctx, `INSERT INTO order_items(id, order_id, position, sku, name, quantity, unit_price) VALUES (?, ?, ?, ?, ?, ?, ?)`, item.ID, orderID, item.Position, item.SKU, item.Name, item.Quantity, item.UnitPrice); err != nil {
			return fmt.Errorf("insert order item: %w", err)
		}
	}
	return nil
}

func getOrderTx(ctx context.Context, transaction *sql.Tx, id string) (order.Order, bool, error) {
	result, found, err := scanOrderFound(transaction.QueryRowContext(ctx, `SELECT id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at FROM orders WHERE id = ?`, id))
	if err != nil || !found {
		return order.Order{}, found, err
	}
	rows, err := transaction.QueryContext(ctx, `SELECT id, sku, name, quantity, unit_price FROM order_items WHERE order_id = ? ORDER BY position`, id)
	if err != nil {
		return order.Order{}, false, fmt.Errorf("query order items: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item order.Item
		if err := rows.Scan(&item.ID, &item.SKU, &item.Name, &item.Quantity, &item.UnitPrice); err != nil {
			return order.Order{}, false, fmt.Errorf("scan order item: %w", err)
		}
		if !order.ValidItemID(item.ID) || item.SKU == "" || item.Name == "" || item.Quantity < 1 || item.Quantity > order.MaxQuantity || item.UnitPrice < 1 || item.UnitPrice > order.MaxAmount {
			return order.Order{}, false, errors.New("invalid order item data")
		}
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return order.Order{}, false, fmt.Errorf("iterate order items: %w", err)
	}
	if len(result.Items) == 0 {
		return order.Order{}, false, errors.New("order has no items")
	}
	return result, true, nil
}

func (database *DB) observeQuery() {
	if database.queryObserver != nil {
		database.queryObserver()
	}
}

func scanOrderFound(row rowScanner) (order.Order, bool, error) {
	result, err := scanOrder(row)
	if errors.Is(err, sql.ErrNoRows) {
		return order.Order{}, false, nil
	}
	return result, err == nil, err
}

func scanOrder(row rowScanner) (order.Order, error) {
	var result order.Order
	var status, paymentStatus, createdAt, updatedAt string
	if err := row.Scan(&result.ID, &result.CustomerName, &status, &paymentStatus, &result.Currency, &result.TotalAmount, &result.Version, &createdAt, &updatedAt); err != nil {
		return order.Order{}, err
	}
	result.Status, result.PaymentStatus = order.Status(status), order.PaymentStatus(paymentStatus)
	var err error
	result.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return order.Order{}, fmt.Errorf("parse order created time: %w", err)
	}
	result.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return order.Order{}, fmt.Errorf("parse order updated time: %w", err)
	}
	if !order.ValidOrderID(result.ID) || result.CustomerName == "" || !result.Status.Valid() || !result.PaymentStatus.Valid() || result.Currency != "CNY" || result.TotalAmount < 1 || result.TotalAmount > 9999999999 || result.Version < 1 {
		return order.Order{}, errors.New("invalid order data")
	}
	return result, nil
}
