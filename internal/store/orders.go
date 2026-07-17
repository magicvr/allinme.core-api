package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/magicvr/allinme.core-api/internal/order"
)

func (database *DB) ListOrders(ctx context.Context, query order.ListQuery) (page order.Page, resultErr error) {
	defer func() { resultErr = classifyOrderError(resultErr) }()
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
	if err := transaction.QueryRowContext(ctx, "SELECT COUNT(*) FROM orders o "+where, arguments...).Scan(&page.Total); err != nil {
		return order.Page{}, fmt.Errorf("count orders: %w", err)
	}
	database.observeQuery()
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
	rows, err := transaction.QueryContext(ctx, `SELECT o.id, o.customer_name, o.status, o.payment_status, o.currency, o.total_amount, o.version, o.created_at, o.updated_at FROM orders o `+where+` ORDER BY `+sortColumn+` `+direction+`, o.id `+direction+` LIMIT ? OFFSET ?`, pageArguments...)
	if err != nil {
		return order.Page{}, fmt.Errorf("query orders: %w", err)
	}
	database.observeQuery()
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
	if err := rows.Close(); err != nil {
		return order.Page{}, fmt.Errorf("close order rows: %w", err)
	}
	if len(page.Items) > 0 {
		database.observeQuery()
	}
	if err := validatePageOrderAggregates(ctx, transaction, page.Items); err != nil {
		return order.Page{}, err
	}
	if len(page.Items) > 0 {
		database.observeQuery()
	}
	if err := applyOrderAttachmentCountsTx(ctx, transaction, page.Items); err != nil {
		return order.Page{}, err
	}
	if len(page.Items) > 0 {
		database.observeQuery()
	}
	if err := applyOrderRefundCapabilitiesTx(ctx, transaction, page.Items); err != nil {
		return order.Page{}, err
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
		keyword := "%" + escapeLike(asciiLower(query.Keyword)) + "%"
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
		arguments = append(arguments, orderCreatedFromBoundary(*query.CreatedFrom))
	}
	if query.CreatedTo != nil {
		conditions = append(conditions, "o.created_at <= ?")
		arguments = append(arguments, orderCreatedToBoundary(*query.CreatedTo))
	}
	return "WHERE " + strings.Join(conditions, " AND "), arguments
}

func orderCreatedFromBoundary(value time.Time) string {
	value = value.UTC()
	boundary := value.Truncate(time.Second)
	if !value.Equal(boundary) {
		boundary = boundary.Add(time.Second)
	}
	return boundary.Format(time.RFC3339)
}

func orderCreatedToBoundary(value time.Time) string {
	return value.UTC().Truncate(time.Second).Format(time.RFC3339)
}

func asciiLower(value string) string {
	buffer := []byte(value)
	for index, current := range buffer {
		if current >= 'A' && current <= 'Z' {
			buffer[index] = current + ('a' - 'A')
		}
	}
	return string(buffer)
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return strings.ReplaceAll(value, `_`, `\_`)
}

func (database *DB) GetOrder(ctx context.Context, id string) (result order.Order, found bool, resultErr error) {
	defer func() { resultErr = classifyOrderError(resultErr) }()
	transaction, err := database.sql.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return order.Order{}, false, fmt.Errorf("begin order detail transaction: %w", err)
	}
	defer transaction.Rollback()
	result, found, err = getOrderTx(ctx, transaction, id)
	if err != nil || !found {
		return order.Order{}, found, err
	}
	database.observeQuery()
	values := []order.Order{result}
	if err := applyOrderRefundCapabilitiesTx(ctx, transaction, values); err != nil {
		return order.Order{}, false, err
	}
	result = values[0]
	if err := transaction.Commit(); err != nil {
		return order.Order{}, false, fmt.Errorf("commit order detail transaction: %w", err)
	}
	return result, true, nil
}

func (database *DB) PrepareAttachmentsForOrder(ctx context.Context, owner string, ids []string, now time.Time) (result []order.Attachment, resultErr error) {
	defer func() { resultErr = classifyOrderError(resultErr) }()
	if err := order.ValidateAttachmentIDs(ids); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []order.Attachment{}, nil
	}
	transaction, err := database.sql.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin attachment preparation transaction: %w", err)
	}
	defer transaction.Rollback()
	result, err = loadAttachmentsByIDTx(ctx, transaction, ids)
	if err != nil {
		return nil, err
	}
	mapped, err := mappedOrderAttachmentPositionsTx(ctx, transaction, ids)
	if err != nil {
		return nil, err
	}
	for index, attachment := range result {
		if mapped[index] || attachment.CreatedBy != owner || attachment.Status != order.AttachmentStatusUploaded || attachment.ExpiresAt == nil || now.Before(attachment.CreatedAt) || !now.Before(*attachment.ExpiresAt) {
			return nil, unavailableOrderAttachment(index)
		}
	}
	if err := transaction.Commit(); err != nil {
		return nil, fmt.Errorf("commit attachment preparation transaction: %w", err)
	}
	return result, nil
}

func (database *DB) CreateOrderIdempotent(ctx context.Context, persistence order.IdempotentCreatePersistence) (result order.IdempotencyRecord, created bool, resultErr error) {
	defer func() { resultErr = classifyOrderError(resultErr) }()
	transaction, err := database.sql.BeginTx(ctx, nil)
	if err != nil {
		return order.IdempotencyRecord{}, false, fmt.Errorf("begin create order transaction: %w", err)
	}
	defer func() {
		if resultErr != nil {
			_ = transaction.Rollback()
		}
	}()
	insert, err := transaction.ExecContext(ctx, `INSERT OR IGNORE INTO idempotency_keys(principal_user_id, method, route, idempotency_key, request_digest, order_id, snapshot_version, snapshot_json, snapshot_digest, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, persistence.Record.Scope.PrincipalUserID, persistence.Record.Scope.Method, persistence.Record.Scope.Route, persistence.Record.Scope.Key, persistence.Record.Scope.RequestDigest[:], persistence.Record.OrderID, persistence.Record.SnapshotVersion, string(persistence.Record.SnapshotJSON), persistence.Record.SnapshotDigest[:], persistence.Record.CreatedAt)
	if err != nil {
		return order.IdempotencyRecord{}, false, fmt.Errorf("reserve idempotency key: %w", err)
	}
	affected, err := insert.RowsAffected()
	if err != nil {
		return order.IdempotencyRecord{}, false, fmt.Errorf("read idempotency reservation rows: %w", err)
	}
	if affected == 0 {
		result, err = getIdempotencyRecordTx(ctx, transaction, persistence.Record.Scope)
		if err != nil {
			return order.IdempotencyRecord{}, false, err
		}
		if err := transaction.Commit(); err != nil {
			return order.IdempotencyRecord{}, false, fmt.Errorf("commit idempotency replay transaction: %w", err)
		}
		return result, false, nil
	}
	create := persistence.Create
	if _, err := transaction.ExecContext(ctx, `INSERT INTO orders(id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at) VALUES (?, ?, 'DRAFT', 'UNPAID', ?, ?, 1, ?, ?)`, create.ID, create.CustomerName, create.Currency, create.TotalAmount, create.CreatedAt, create.CreatedAt); err != nil {
		return order.IdempotencyRecord{}, false, fmt.Errorf("insert order: %w", err)
	}
	if err := insertOrderItems(ctx, transaction, create.ID, create.Items); err != nil {
		return order.IdempotencyRecord{}, false, err
	}
	if err := bindOrderAttachments(ctx, transaction, persistence.Record.Scope.PrincipalUserID, create.ID, create.AttachmentIDs, create.CreatedAt); err != nil {
		return order.IdempotencyRecord{}, false, err
	}
	if err := transaction.Commit(); err != nil {
		return order.IdempotencyRecord{}, false, fmt.Errorf("commit create order transaction: %w", err)
	}
	return persistence.Record, true, nil
}

func (database *DB) GetIdempotency(ctx context.Context, scope order.IdempotencyScope) (order.IdempotencyRecord, bool, error) {
	var result order.IdempotencyRecord
	var digest, snapshotDigest []byte
	err := database.sql.QueryRowContext(ctx, `SELECT principal_user_id, method, route, idempotency_key, request_digest, order_id, snapshot_version, snapshot_json, snapshot_digest, created_at FROM idempotency_keys WHERE principal_user_id = ? AND method = ? AND route = ? AND idempotency_key = ?`, scope.PrincipalUserID, scope.Method, scope.Route, scope.Key).Scan(&result.Scope.PrincipalUserID, &result.Scope.Method, &result.Scope.Route, &result.Scope.Key, &digest, &result.OrderID, &result.SnapshotVersion, &result.SnapshotJSON, &snapshotDigest, &result.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return order.IdempotencyRecord{}, false, nil
	}
	if err != nil {
		return order.IdempotencyRecord{}, false, classifyOrderError(fmt.Errorf("read idempotency key: %w", err))
	}
	if len(digest) != len(result.Scope.RequestDigest) || len(snapshotDigest) != len(result.SnapshotDigest) {
		return order.IdempotencyRecord{}, false, order.Internal(errors.New("invalid idempotency digest"))
	}
	copy(result.Scope.RequestDigest[:], digest)
	copy(result.SnapshotDigest[:], snapshotDigest)
	return result, true, nil
}

func getIdempotencyRecordTx(ctx context.Context, transaction *sql.Tx, scope order.IdempotencyScope) (order.IdempotencyRecord, error) {
	var result order.IdempotencyRecord
	var digest, snapshotDigest []byte
	if err := transaction.QueryRowContext(ctx, `SELECT principal_user_id, method, route, idempotency_key, request_digest, order_id, snapshot_version, snapshot_json, snapshot_digest, created_at FROM idempotency_keys WHERE principal_user_id = ? AND method = ? AND route = ? AND idempotency_key = ?`, scope.PrincipalUserID, scope.Method, scope.Route, scope.Key).Scan(&result.Scope.PrincipalUserID, &result.Scope.Method, &result.Scope.Route, &result.Scope.Key, &digest, &result.OrderID, &result.SnapshotVersion, &result.SnapshotJSON, &snapshotDigest, &result.CreatedAt); err != nil {
		return order.IdempotencyRecord{}, fmt.Errorf("read idempotency replay: %w", err)
	}
	if len(digest) != len(result.Scope.RequestDigest) || len(snapshotDigest) != len(result.SnapshotDigest) {
		return order.IdempotencyRecord{}, errors.New("invalid idempotency digest")
	}
	copy(result.Scope.RequestDigest[:], digest)
	copy(result.SnapshotDigest[:], snapshotDigest)
	return result, nil
}

func (database *DB) UpdateDraft(ctx context.Context, persistence order.UpdateDraftPersistence) (result order.Order, resultErr error) {
	defer func() { resultErr = classifyOrderError(resultErr) }()
	transaction, err := database.sql.BeginTx(ctx, nil)
	if err != nil {
		return order.Order{}, fmt.Errorf("begin edit order transaction: %w", err)
	}
	defer func() {
		if resultErr != nil {
			_ = transaction.Rollback()
		}
	}()
	currentVersion, currentStatus, found, err := getRawOrderVersionStatus(ctx, transaction, persistence.ID)
	if err != nil {
		return order.Order{}, err
	}
	if !found {
		return order.Order{}, order.ErrNotFound
	}
	if currentVersion != persistence.Version {
		return order.Order{}, order.ErrVersionConflict
	}
	if currentStatus != order.StatusDraft {
		return order.Order{}, order.ErrStateConflict
	}
	fence, err := transaction.ExecContext(ctx, `UPDATE orders SET version = version WHERE id = ? AND version = ? AND status = 'DRAFT'`, persistence.ID, persistence.Version)
	if err != nil {
		return order.Order{}, fmt.Errorf("acquire edit order writer fence: %w", err)
	}
	fenceAffected, err := fence.RowsAffected()
	if err != nil {
		return order.Order{}, fmt.Errorf("read edit order writer fence rows: %w", err)
	}
	if fenceAffected != 1 {
		return order.Order{}, classifyUpdateMiss(ctx, transaction, persistence.ID, persistence.Version)
	}
	current, found, err := getOrderTx(ctx, transaction, persistence.ID)
	if err != nil {
		return order.Order{}, err
	}
	if !found {
		return order.Order{}, errors.New("fenced edit order is missing")
	}
	if current.Version != persistence.Version {
		return order.Order{}, order.ErrVersionConflict
	}
	if current.Status != order.StatusDraft {
		return order.Order{}, order.ErrStateConflict
	}
	aggregate, err := loadRefundAggregateTx(ctx, transaction, persistence.ID)
	if err != nil {
		return order.Order{}, err
	}
	if err := validatePaymentStatusForRefundAggregate(current, aggregate); err != nil {
		return order.Order{}, err
	}
	occupied, err := checkedRefundSum(aggregate.pending, aggregate.completed)
	if err != nil {
		return order.Order{}, err
	}
	if persistence.TotalAmount < occupied {
		return order.Order{}, &order.ValidationError{Details: []order.FieldError{{Field: "items", Message: "calculated total must not be less than occupied refund amount"}}}
	}
	paymentStatus := current.PaymentStatus
	if aggregate.rows == 0 && current.PaymentStatus == order.PaymentStatusUnpaid {
		paymentStatus = order.PaymentStatusUnpaid
	} else {
		paymentStatus, err = paymentStatusForCompleted(persistence.TotalAmount, aggregate.completed)
		if err != nil {
			return order.Order{}, err
		}
	}
	if current.Version == math.MaxInt64 {
		return order.Order{}, errors.New("order version exhausted")
	}
	update, err := transaction.ExecContext(ctx, `UPDATE orders SET customer_name = ?, currency = ?, total_amount = ?, payment_status = ?, version = version + 1, updated_at = ? WHERE id = ? AND status = 'DRAFT' AND version = ?`, persistence.CustomerName, persistence.Currency, persistence.TotalAmount, paymentStatus, persistence.UpdatedAt, persistence.ID, persistence.Version)
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
	updatedValues := []order.Order{result}
	if err := applyOrderRefundCapabilitiesTx(ctx, transaction, updatedValues); err != nil {
		return order.Order{}, err
	}
	result = updatedValues[0]
	if err := transaction.Commit(); err != nil {
		return order.Order{}, fmt.Errorf("commit edit order transaction: %w", err)
	}
	return result, nil
}

func getRawOrderVersionStatus(ctx context.Context, transaction *sql.Tx, orderID string) (int64, order.Status, bool, error) {
	var version int64
	var status order.Status
	err := transaction.QueryRowContext(ctx, `SELECT version, status FROM orders WHERE id = ?`, orderID).Scan(&version, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", false, nil
	}
	if err != nil {
		return 0, "", false, fmt.Errorf("read order version and status: %w", err)
	}
	return version, status, true, nil
}

func (database *DB) TransitionOrder(ctx context.Context, persistence order.TransitionPersistence) (result order.Order, resultErr error) {
	defer func() { resultErr = classifyOrderError(resultErr) }()
	transaction, err := database.sql.BeginTx(ctx, nil)
	if err != nil {
		return order.Order{}, fmt.Errorf("begin transition transaction: %w", err)
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
	if !containsStatus(persistence.AllowedSources, current.Status) {
		return order.Order{}, order.ErrStateConflict
	}
	if current.Version == math.MaxInt64 {
		return order.Order{}, errors.New("order version exhausted")
	}
	placeholders := make([]string, len(persistence.AllowedSources))
	args := []any{persistence.Target, persistence.UpdatedAt, persistence.ID, persistence.Version}
	for index, source := range persistence.AllowedSources {
		placeholders[index] = "?"
		args = append(args, source)
	}
	query := `UPDATE orders SET status = ?, version = version + 1, updated_at = ? WHERE id = ? AND version = ? AND status IN (` + strings.Join(placeholders, ",") + `)`
	update, err := transaction.ExecContext(ctx, query, args...)
	if err != nil {
		return order.Order{}, fmt.Errorf("transition order: %w", err)
	}
	affected, err := update.RowsAffected()
	if err != nil {
		return order.Order{}, fmt.Errorf("read transitioned order rows: %w", err)
	}
	if affected != 1 {
		return order.Order{}, classifyTransitionMiss(ctx, transaction, persistence)
	}
	result, found, err = getOrderTx(ctx, transaction, persistence.ID)
	if err != nil {
		return order.Order{}, err
	}
	if !found {
		return order.Order{}, errors.New("transitioned order is missing")
	}
	transitionedValues := []order.Order{result}
	if err := applyOrderRefundCapabilitiesTx(ctx, transaction, transitionedValues); err != nil {
		return order.Order{}, err
	}
	result = transitionedValues[0]
	if err := transaction.Commit(); err != nil {
		return order.Order{}, fmt.Errorf("commit transition transaction: %w", err)
	}
	return result, nil
}

func containsStatus(values []order.Status, value order.Status) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func classifyTransitionMiss(ctx context.Context, transaction *sql.Tx, persistence order.TransitionPersistence) error {
	current, found, err := getOrderTx(ctx, transaction, persistence.ID)
	if err != nil {
		return err
	}
	if !found {
		return order.ErrNotFound
	}
	if current.Version != persistence.Version {
		return order.ErrVersionConflict
	}
	return order.ErrStateConflict
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
	rows, err := transaction.QueryContext(ctx, `SELECT id, position, sku, name, quantity, unit_price FROM order_items WHERE order_id = ? ORDER BY position`, id)
	if err != nil {
		return order.Order{}, false, fmt.Errorf("query order items: %w", err)
	}
	defer rows.Close()
	validator := newOrderAggregateValidator(result.TotalAmount)
	for rows.Next() {
		var item order.Item
		var position int64
		if err := rows.Scan(&item.ID, &position, &item.SKU, &item.Name, &item.Quantity, &item.UnitPrice); err != nil {
			return order.Order{}, false, fmt.Errorf("scan order item: %w", err)
		}
		if err := validator.add(position, item); err != nil {
			return order.Order{}, false, err
		}
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return order.Order{}, false, fmt.Errorf("iterate order items: %w", err)
	}
	if err := validator.finish(); err != nil {
		return order.Order{}, false, err
	}
	if err := loadOrderAttachmentsTx(ctx, transaction, &result); err != nil {
		return order.Order{}, false, err
	}
	return result, true, nil
}

func applyOrderAttachmentCountsTx(ctx context.Context, transaction *sql.Tx, orders []order.Order) error {
	if len(orders) == 0 {
		return nil
	}
	byID := make(map[string]*order.Order, len(orders))
	arguments := make([]any, 0, len(orders))
	placeholders := make([]string, 0, len(orders))
	for index := range orders {
		orders[index].AttachmentCount = 0
		byID[orders[index].ID] = &orders[index]
		arguments = append(arguments, orders[index].ID)
		placeholders = append(placeholders, "?")
	}
	rows, err := transaction.QueryContext(ctx, `SELECT order_id, COUNT(*) FROM order_attachments WHERE order_id IN (`+strings.Join(placeholders, ",")+`) GROUP BY order_id`, arguments...)
	if err != nil {
		return fmt.Errorf("query listed order attachment counts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var orderID string
		var count int64
		if err := rows.Scan(&orderID, &count); err != nil {
			return fmt.Errorf("scan listed order attachment count: %w", err)
		}
		value := byID[orderID]
		if value == nil || count < 1 || count > order.MaxOrderAttachments {
			return errors.New("invalid listed order attachment count")
		}
		value.AttachmentCount = count
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate listed order attachment counts: %w", err)
	}
	return nil
}

func loadOrderAttachmentsTx(ctx context.Context, transaction *sql.Tx, value *order.Order) error {
	rows, err := transaction.QueryContext(ctx, `SELECT a.id, oa.position, a.file_name, a.storage_key, a.content_type, a.size_bytes, a.sha256, a.status, a.created_by, a.expires_at, a.created_at, a.updated_at FROM order_attachments oa JOIN attachments a ON a.id = oa.attachment_id WHERE oa.order_id = ? ORDER BY oa.position`, value.ID)
	if err != nil {
		return fmt.Errorf("query order attachments: %w", err)
	}
	defer rows.Close()
	value.Attachments = []order.Attachment{}
	var position int64
	for rows.Next() {
		attachment, storedPosition, err := scanOrderAttachment(rows)
		if err != nil {
			return err
		}
		if storedPosition != position || attachment.Status != order.AttachmentStatusBound || attachment.ExpiresAt != nil {
			return errors.New("invalid bound order attachment")
		}
		value.Attachments = append(value.Attachments, attachment)
		position++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate order attachments: %w", err)
	}
	if len(value.Attachments) > order.MaxOrderAttachments {
		return errors.New("too many order attachments")
	}
	value.AttachmentCount = int64(len(value.Attachments))
	return nil
}

func loadAttachmentsByIDTx(ctx context.Context, transaction *sql.Tx, ids []string) ([]order.Attachment, error) {
	arguments := make([]any, 0, len(ids))
	placeholders := make([]string, 0, len(ids))
	positions := make(map[string]int, len(ids))
	for index, id := range ids {
		arguments = append(arguments, id)
		placeholders = append(placeholders, "?")
		positions[id] = index
	}
	rows, err := transaction.QueryContext(ctx, `SELECT id, file_name, storage_key, content_type, size_bytes, sha256, status, created_by, expires_at, created_at, updated_at FROM attachments WHERE id IN (`+strings.Join(placeholders, ",")+`)`, arguments...)
	if err != nil {
		return nil, fmt.Errorf("query attachments for order: %w", err)
	}
	defer rows.Close()
	result := make([]order.Attachment, len(ids))
	found := make([]bool, len(ids))
	for rows.Next() {
		attachment, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		position, ok := positions[attachment.ID]
		if !ok || found[position] {
			return nil, errors.New("unexpected prepared attachment")
		}
		result[position], found[position] = attachment, true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate attachments for order: %w", err)
	}
	for index, ok := range found {
		if !ok {
			return nil, unavailableOrderAttachment(index)
		}
	}
	return result, nil
}

func mappedOrderAttachmentPositionsTx(ctx context.Context, transaction *sql.Tx, ids []string) ([]bool, error) {
	arguments := make([]any, 0, len(ids))
	placeholders := make([]string, 0, len(ids))
	positions := make(map[string]int, len(ids))
	for index, id := range ids {
		arguments = append(arguments, id)
		placeholders = append(placeholders, "?")
		positions[id] = index
	}
	rows, err := transaction.QueryContext(ctx, `SELECT attachment_id FROM order_attachments WHERE attachment_id IN (`+strings.Join(placeholders, ",")+`)`, arguments...)
	if err != nil {
		return nil, fmt.Errorf("query existing order attachment mappings: %w", err)
	}
	defer rows.Close()
	result := make([]bool, len(ids))
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan existing order attachment mapping: %w", err)
		}
		position, ok := positions[id]
		if !ok || result[position] {
			return nil, errors.New("unexpected order attachment mapping")
		}
		result[position] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate existing order attachment mappings: %w", err)
	}
	return result, nil
}

func scanOrderAttachment(row rowScanner) (order.Attachment, int64, error) {
	var attachment order.Attachment
	var position int64
	var status, expiresAt, createdAt, updatedAt sql.NullString
	var digest []byte
	if err := row.Scan(&attachment.ID, &position, &attachment.FileName, &attachment.StorageKey, &attachment.ContentType, &attachment.SizeBytes, &digest, &status, &attachment.CreatedBy, &expiresAt, &createdAt, &updatedAt); err != nil {
		return order.Attachment{}, 0, fmt.Errorf("scan order attachment: %w", err)
	}
	if err := populateStoredAttachment(&attachment, digest, status, expiresAt, createdAt, updatedAt); err != nil {
		return order.Attachment{}, 0, err
	}
	return attachment, position, nil
}

func scanAttachment(row rowScanner) (order.Attachment, error) {
	var attachment order.Attachment
	var status, expiresAt, createdAt, updatedAt sql.NullString
	var digest []byte
	if err := row.Scan(&attachment.ID, &attachment.FileName, &attachment.StorageKey, &attachment.ContentType, &attachment.SizeBytes, &digest, &status, &attachment.CreatedBy, &expiresAt, &createdAt, &updatedAt); err != nil {
		return order.Attachment{}, fmt.Errorf("scan attachment: %w", err)
	}
	if err := populateStoredAttachment(&attachment, digest, status, expiresAt, createdAt, updatedAt); err != nil {
		return order.Attachment{}, err
	}
	return attachment, nil
}

func populateStoredAttachment(attachment *order.Attachment, digest []byte, status, expiresAt, createdAt, updatedAt sql.NullString) error {
	if !status.Valid || !createdAt.Valid || !updatedAt.Valid || len(digest) != len(attachment.SHA256) {
		return errors.New("invalid stored attachment data")
	}
	attachment.Status = order.AttachmentStatus(status.String)
	var err error
	attachment.CreatedAt, err = time.Parse(time.RFC3339, createdAt.String)
	if err != nil || order.FormatTime(attachment.CreatedAt) != createdAt.String {
		return errors.New("invalid stored attachment created time")
	}
	attachment.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt.String)
	if err != nil || order.FormatTime(attachment.UpdatedAt) != updatedAt.String {
		return errors.New("invalid stored attachment updated time")
	}
	if expiresAt.Valid {
		value, parseErr := time.Parse(time.RFC3339, expiresAt.String)
		if parseErr != nil || order.FormatTime(value) != expiresAt.String {
			return errors.New("invalid stored attachment expiry")
		}
		attachment.ExpiresAt = &value
	}
	copy(attachment.SHA256[:], digest)
	return order.ValidateAttachment(*attachment)
}

func bindOrderAttachments(ctx context.Context, transaction *sql.Tx, owner, orderID string, ids []string, createdAt string) error {
	if err := order.ValidateAttachmentIDs(ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	now, err := time.Parse(time.RFC3339, createdAt)
	if err != nil || order.FormatTime(now) != createdAt {
		return errors.New("invalid order attachment bind time")
	}
	attachments, err := loadAttachmentsByIDTx(ctx, transaction, ids)
	if err != nil {
		return err
	}
	mapped, err := mappedOrderAttachmentPositionsTx(ctx, transaction, ids)
	if err != nil {
		return err
	}
	for index, attachment := range attachments {
		if mapped[index] || attachment.CreatedBy != owner || attachment.Status != order.AttachmentStatusUploaded || attachment.ExpiresAt == nil || now.Before(attachment.CreatedAt) || !now.Before(*attachment.ExpiresAt) {
			return unavailableOrderAttachment(index)
		}
		if _, err := transaction.ExecContext(ctx, `INSERT INTO order_attachments(attachment_id, order_id, position, bound_at) VALUES (?, ?, ?, ?)`, attachment.ID, orderID, index, createdAt); err != nil {
			return fmt.Errorf("insert order attachment: %w", err)
		}
		updated, err := transaction.ExecContext(ctx, `UPDATE attachments SET status = 'BOUND', expires_at = NULL, updated_at = ? WHERE id = ? AND created_by = ? AND status = 'UPLOADED' AND expires_at > ?`, createdAt, attachment.ID, owner, createdAt)
		if err != nil {
			return fmt.Errorf("bind order attachment: %w", err)
		}
		affected, err := updated.RowsAffected()
		if err != nil {
			return fmt.Errorf("read bound attachment rows: %w", err)
		}
		if affected != 1 {
			return unavailableOrderAttachment(index)
		}
	}
	return nil
}

func unavailableOrderAttachment(index int) error {
	return &order.ValidationError{Details: []order.FieldError{{Field: fmt.Sprintf("attachmentIds[%d]", index), Message: "attachment is unavailable"}}}
}

func validatePageOrderAggregates(ctx context.Context, transaction *sql.Tx, orders []order.Order) error {
	if len(orders) == 0 {
		return nil
	}
	validators := make(map[string]*orderAggregateValidator, len(orders))
	arguments := make([]any, 0, len(orders))
	placeholders := make([]string, 0, len(orders))
	for _, value := range orders {
		validators[value.ID] = newOrderAggregateValidator(value.TotalAmount)
		arguments = append(arguments, value.ID)
		placeholders = append(placeholders, "?")
	}
	rows, err := transaction.QueryContext(ctx, `SELECT order_id, id, position, sku, name, quantity, unit_price FROM order_items WHERE order_id IN (`+strings.Join(placeholders, ",")+`) ORDER BY order_id, position`, arguments...)
	if err != nil {
		return fmt.Errorf("query listed order items: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var orderID string
		var item order.Item
		var position int64
		if err := rows.Scan(&orderID, &item.ID, &position, &item.SKU, &item.Name, &item.Quantity, &item.UnitPrice); err != nil {
			return fmt.Errorf("scan listed order item: %w", err)
		}
		validator := validators[orderID]
		if validator == nil {
			return errors.New("order item references unexpected listed order")
		}
		if err := validator.add(position, item); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate listed order items: %w", err)
	}
	for _, validator := range validators {
		if err := validator.finish(); err != nil {
			return err
		}
	}
	return nil
}

type orderAggregateValidator struct {
	expectedTotal int64
	total         int64
	nextPosition  int64
	itemCount     int
}

func newOrderAggregateValidator(expectedTotal int64) *orderAggregateValidator {
	return &orderAggregateValidator{expectedTotal: expectedTotal}
}

func (validator *orderAggregateValidator) add(position int64, item order.Item) error {
	if validator.itemCount >= order.MaxItems || position != validator.nextPosition || !order.ValidItemID(item.ID) || !validStoredString(item.SKU, order.MaxSKUBytes) || !validStoredString(item.Name, order.MaxItemNameBytes) || item.Quantity < 1 || item.Quantity > order.MaxQuantity || item.UnitPrice < 1 || item.UnitPrice > order.MaxAmount {
		return errors.New("invalid order item data")
	}
	if item.Quantity > math.MaxInt64/item.UnitPrice {
		return errors.New("order item amount overflow")
	}
	lineTotal := item.Quantity * item.UnitPrice
	if validator.total > math.MaxInt64-lineTotal {
		return errors.New("order total amount overflow")
	}
	validator.total += lineTotal
	validator.nextPosition++
	validator.itemCount++
	return nil
}

func (validator *orderAggregateValidator) finish() error {
	if validator.itemCount < 1 || validator.itemCount > order.MaxItems || validator.total != validator.expectedTotal {
		return errors.New("invalid order aggregate data")
	}
	return nil
}

func validStoredString(value string, maximum int) bool {
	return utf8.ValidString(value) && len(value) >= 1 && len(value) <= maximum
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
	if order.FormatTime(result.CreatedAt) != createdAt || order.FormatTime(result.UpdatedAt) != updatedAt || !order.ValidOrderID(result.ID) || !validStoredString(result.CustomerName, order.MaxCustomerNameBytes) || !result.Status.Valid() || !result.PaymentStatus.Valid() || result.Currency != "CNY" || result.TotalAmount < 1 || result.TotalAmount > order.MaxAmount || result.Version < 1 {
		return order.Order{}, errors.New("invalid order data")
	}
	return result, nil
}
