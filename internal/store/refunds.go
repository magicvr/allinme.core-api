package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/magicvr/allinme.core-api/internal/order"
)

func (database *DB) GetRefundIdempotency(ctx context.Context, scope order.RefundIdempotencyScope) (order.RefundIdempotencyRecord, bool, error) {
	record, found, err := getRefundIdempotencyRecord(database.sql.QueryRowContext(ctx, `
		SELECT principal_user_id, method, operation, order_id, idempotency_key, request_digest,
			refund_id, snapshot_version, snapshot_json, snapshot_digest, created_at
		FROM refund_idempotency_keys
		WHERE principal_user_id = ? AND method = ? AND operation = ? AND order_id = ? AND idempotency_key = ?
	`, scope.PrincipalUserID, scope.Method, scope.Operation, scope.OrderID, scope.Key))
	if err != nil {
		return order.RefundIdempotencyRecord{}, false, classifyOrderError(fmt.Errorf("read refund idempotency key: %w", err))
	}
	return record, found, nil
}

func (database *DB) CreateRefundIdempotent(ctx context.Context, persistence order.RefundCreatePersistence) (result order.RefundIdempotencyRecord, created bool, resultErr error) {
	defer func() { resultErr = classifyOrderError(resultErr) }()
	if err := order.ValidateRefundCreatePersistence(persistence); err != nil {
		return order.RefundIdempotencyRecord{}, false, err
	}
	transaction, err := database.sql.BeginTx(ctx, nil)
	if err != nil {
		return order.RefundIdempotencyRecord{}, false, fmt.Errorf("begin create refund transaction: %w", err)
	}
	defer func() {
		if resultErr != nil {
			_ = transaction.Rollback()
		}
	}()

	existing, found, err := getRefundIdempotencyRecord(transaction.QueryRowContext(ctx, `
		SELECT principal_user_id, method, operation, order_id, idempotency_key, request_digest,
			refund_id, snapshot_version, snapshot_json, snapshot_digest, created_at
		FROM refund_idempotency_keys
		WHERE principal_user_id = ? AND method = ? AND operation = ? AND order_id = ? AND idempotency_key = ?
	`, persistence.Record.Scope.PrincipalUserID, persistence.Record.Scope.Method, persistence.Record.Scope.Operation, persistence.Record.Scope.OrderID, persistence.Record.Scope.Key))
	if err != nil {
		return order.RefundIdempotencyRecord{}, false, err
	}
	if found {
		if err := transaction.Commit(); err != nil {
			return order.RefundIdempotencyRecord{}, false, fmt.Errorf("commit refund replay transaction: %w", err)
		}
		return existing, false, nil
	}

	version, found, err := getRawOrderVersion(ctx, transaction, persistence.Refund.OrderID)
	if err != nil {
		return order.RefundIdempotencyRecord{}, false, err
	}
	if !found {
		return order.RefundIdempotencyRecord{}, false, order.ErrNotFound
	}
	if version != persistence.OrderVersion {
		return order.RefundIdempotencyRecord{}, false, order.ErrVersionConflict
	}
	fence, err := transaction.ExecContext(ctx, `UPDATE orders SET version = version WHERE id = ? AND version = ?`, persistence.Refund.OrderID, persistence.OrderVersion)
	if err != nil {
		return order.RefundIdempotencyRecord{}, false, fmt.Errorf("acquire refund order writer fence: %w", err)
	}
	affected, err := fence.RowsAffected()
	if err != nil {
		return order.RefundIdempotencyRecord{}, false, fmt.Errorf("read refund order writer fence rows: %w", err)
	}
	if affected != 1 {
		return order.RefundIdempotencyRecord{}, false, classifyRawOrderVersionMiss(ctx, transaction, persistence.Refund.OrderID, persistence.OrderVersion)
	}
	current, found, err := getOrderTx(ctx, transaction, persistence.Refund.OrderID)
	if err != nil {
		return order.RefundIdempotencyRecord{}, false, err
	}
	if !found {
		return order.RefundIdempotencyRecord{}, false, errors.New("fenced refund order is missing")
	}
	if current.Version != persistence.OrderVersion {
		return order.RefundIdempotencyRecord{}, false, order.ErrVersionConflict
	}
	aggregate, err := loadRefundAggregateTx(ctx, transaction, current.ID)
	if err != nil {
		return order.RefundIdempotencyRecord{}, false, err
	}
	if err := validatePaymentStatusForRefundAggregate(current, aggregate); err != nil {
		return order.RefundIdempotencyRecord{}, false, err
	}
	if current.PaymentStatus != order.PaymentStatusPaid && current.PaymentStatus != order.PaymentStatusPartiallyRefunded {
		return order.RefundIdempotencyRecord{}, false, order.ErrStateConflict
	}
	available := current.TotalAmount - aggregate.pending - aggregate.completed
	if persistence.Refund.Amount > available {
		return order.RefundIdempotencyRecord{}, false, &order.ValidationError{Details: []order.FieldError{{Field: "amount", Message: "must not exceed availableRefundAmount"}}}
	}
	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO refunds(id, order_id, amount, reason, status, version, requested_by, decided_by, created_at, updated_at, decided_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NULL, ?, ?, NULL)
	`, persistence.Refund.ID, persistence.Refund.OrderID, persistence.Refund.Amount, persistence.Refund.Reason, persistence.Refund.Status, persistence.Refund.Version, persistence.Refund.RequestedBy.ID, order.FormatTime(persistence.Refund.CreatedAt), order.FormatTime(persistence.Refund.UpdatedAt)); err != nil {
		return order.RefundIdempotencyRecord{}, false, fmt.Errorf("insert refund: %w", err)
	}
	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO refund_idempotency_keys(
			principal_user_id, method, operation, order_id, idempotency_key, request_digest,
			refund_id, snapshot_version, snapshot_json, snapshot_digest, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, persistence.Record.Scope.PrincipalUserID, persistence.Record.Scope.Method, persistence.Record.Scope.Operation, persistence.Record.Scope.OrderID, persistence.Record.Scope.Key, persistence.Record.Scope.RequestDigest[:], persistence.Record.RefundID, persistence.Record.SnapshotVersion, string(persistence.Record.SnapshotJSON), persistence.Record.SnapshotDigest[:], persistence.Record.CreatedAt); err != nil {
		return order.RefundIdempotencyRecord{}, false, fmt.Errorf("insert refund idempotency key: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return order.RefundIdempotencyRecord{}, false, fmt.Errorf("commit create refund transaction: %w", err)
	}
	return persistence.Record, true, nil
}

type refundAggregate struct {
	rows      int
	pending   int64
	completed int64
}

func loadRefundAggregateTx(ctx context.Context, transaction *sql.Tx, orderID string) (refundAggregate, error) {
	rows, err := transaction.QueryContext(ctx, `
		SELECT refunds.id, refunds.order_id, refunds.amount, refunds.reason, refunds.status, refunds.version,
			requester.id, requester.username, decider.id, decider.username,
			refunds.created_at, refunds.updated_at, refunds.decided_at
		FROM refunds
		LEFT JOIN users requester ON requester.id = refunds.requested_by
		LEFT JOIN users decider ON decider.id = refunds.decided_by
		WHERE refunds.order_id = ?
		ORDER BY refunds.created_at, refunds.id
	`, orderID)
	if err != nil {
		return refundAggregate{}, fmt.Errorf("query order refunds: %w", err)
	}
	defer rows.Close()
	var aggregate refundAggregate
	for rows.Next() {
		var value order.Refund
		var status, createdAt, updatedAt string
		var requestedID, requestedUsername, decidedID, decidedUsername, decidedAt sql.NullString
		if err := rows.Scan(&value.ID, &value.OrderID, &value.Amount, &value.Reason, &status, &value.Version, &requestedID, &requestedUsername, &decidedID, &decidedUsername, &createdAt, &updatedAt, &decidedAt); err != nil {
			return refundAggregate{}, fmt.Errorf("scan order refund: %w", err)
		}
		if !requestedID.Valid || !requestedUsername.Valid || value.OrderID != orderID {
			return refundAggregate{}, errors.New("invalid refund actor or order relation")
		}
		value.Currency = "CNY"
		value.Status = order.RefundStatus(status)
		value.RequestedBy = order.RefundActor{ID: requestedID.String, Username: requestedUsername.String}
		value.CreatedAt, err = parseCanonicalRefundTime(createdAt)
		if err != nil {
			return refundAggregate{}, err
		}
		value.UpdatedAt, err = parseCanonicalRefundTime(updatedAt)
		if err != nil {
			return refundAggregate{}, err
		}
		if decidedID.Valid || decidedUsername.Valid || decidedAt.Valid {
			if !decidedID.Valid || !decidedUsername.Valid || !decidedAt.Valid {
				return refundAggregate{}, errors.New("incomplete refund decision fields")
			}
			actor := order.RefundActor{ID: decidedID.String, Username: decidedUsername.String}
			value.DecidedBy = &actor
			parsed, err := parseCanonicalRefundTime(decidedAt.String)
			if err != nil {
				return refundAggregate{}, err
			}
			value.DecidedAt = &parsed
		}
		if err := order.ValidateRefund(value); err != nil {
			return refundAggregate{}, err
		}
		aggregate.rows++
		switch value.Status {
		case order.RefundStatusPending:
			aggregate.pending, err = checkedRefundSum(aggregate.pending, value.Amount)
			if err != nil {
				return refundAggregate{}, fmt.Errorf("pending refund amount: %w", err)
			}
		case order.RefundStatusCompleted:
			aggregate.completed, err = checkedRefundSum(aggregate.completed, value.Amount)
			if err != nil {
				return refundAggregate{}, fmt.Errorf("completed refund amount: %w", err)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return refundAggregate{}, fmt.Errorf("iterate order refunds: %w", err)
	}
	if aggregate.pending > math.MaxInt64-aggregate.completed {
		return refundAggregate{}, errors.New("occupied refund amount overflow")
	}
	return aggregate, nil
}

func checkedRefundSum(current, amount int64) (int64, error) {
	if amount < 0 || current > math.MaxInt64-amount {
		return 0, errors.New("refund amount overflow")
	}
	return current + amount, nil
}

func validatePaymentStatusForRefundAggregate(value order.Order, aggregate refundAggregate) error {
	occupied := aggregate.pending + aggregate.completed
	if occupied > value.TotalAmount {
		return errors.New("occupied refund amount exceeds order total")
	}
	var expected order.PaymentStatus
	switch {
	case aggregate.completed == 0 && aggregate.rows == 0 && value.PaymentStatus == order.PaymentStatusUnpaid:
		expected = order.PaymentStatusUnpaid
	case aggregate.completed == 0:
		expected = order.PaymentStatusPaid
	case aggregate.completed < value.TotalAmount:
		expected = order.PaymentStatusPartiallyRefunded
	case aggregate.completed == value.TotalAmount:
		expected = order.PaymentStatusRefunded
	default:
		return errors.New("completed refund amount exceeds order total")
	}
	if value.PaymentStatus != expected {
		return errors.New("order payment status differs from refund aggregate")
	}
	return nil
}

func getRawOrderVersion(ctx context.Context, transaction *sql.Tx, orderID string) (int64, bool, error) {
	var version int64
	err := transaction.QueryRowContext(ctx, `SELECT version FROM orders WHERE id = ?`, orderID).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("read refund order version: %w", err)
	}
	return version, true, nil
}

func classifyRawOrderVersionMiss(ctx context.Context, transaction *sql.Tx, orderID string, version int64) error {
	current, found, err := getRawOrderVersion(ctx, transaction, orderID)
	if err != nil {
		return err
	}
	if !found {
		return order.ErrNotFound
	}
	if current != version {
		return order.ErrVersionConflict
	}
	return errors.New("refund order writer fence missed current order")
}

func parseCanonicalRefundTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil || order.FormatTime(parsed) != value {
		return time.Time{}, errors.New("invalid refund time")
	}
	return parsed, nil
}

func getRefundIdempotencyRecord(row rowScanner) (order.RefundIdempotencyRecord, bool, error) {
	var result order.RefundIdempotencyRecord
	var requestDigest, snapshotDigest []byte
	err := row.Scan(
		&result.Scope.PrincipalUserID, &result.Scope.Method, &result.Scope.Operation, &result.Scope.OrderID, &result.Scope.Key, &requestDigest,
		&result.RefundID, &result.SnapshotVersion, &result.SnapshotJSON, &snapshotDigest, &result.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return order.RefundIdempotencyRecord{}, false, nil
	}
	if err != nil {
		return order.RefundIdempotencyRecord{}, false, err
	}
	if len(requestDigest) != len(result.Scope.RequestDigest) || len(snapshotDigest) != len(result.SnapshotDigest) {
		return order.RefundIdempotencyRecord{}, false, errors.New("invalid refund idempotency digest")
	}
	copy(result.Scope.RequestDigest[:], requestDigest)
	copy(result.SnapshotDigest[:], snapshotDigest)
	return result, true, nil
}
