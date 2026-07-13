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

func (database *DB) ListRefunds(ctx context.Context, query order.RefundListQuery) (page order.RefundPage, resultErr error) {
	defer func() { resultErr = classifyOrderError(resultErr) }()
	transaction, err := database.sql.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return order.RefundPage{}, fmt.Errorf("begin refund list transaction: %w", err)
	}
	defer func() {
		if resultErr != nil {
			_ = transaction.Rollback()
		}
	}()
	where, arguments := refundListWhere(query)
	if err := transaction.QueryRowContext(ctx, `SELECT COUNT(*) FROM refunds `+where, arguments...).Scan(&page.Total); err != nil {
		return order.RefundPage{}, fmt.Errorf("count refunds: %w", err)
	}
	database.observeQuery()
	offset := (query.Page - 1) * query.PageSize
	pageArguments := append(append([]any{}, arguments...), query.PageSize, offset)
	rows, err := transaction.QueryContext(ctx, `
		SELECT id, order_id, amount, reason, status, version, requested_by, decided_by, created_at, updated_at, decided_at
		FROM refunds `+where+` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?
	`, pageArguments...)
	if err != nil {
		return order.RefundPage{}, fmt.Errorf("query refunds: %w", err)
	}
	database.observeQuery()
	defer rows.Close()
	stored := make([]storedRefundListRow, 0)
	actorIDs := make(map[string]bool)
	for rows.Next() {
		var value storedRefundListRow
		var status string
		if err := rows.Scan(&value.refund.ID, &value.refund.OrderID, &value.refund.Amount, &value.refund.Reason, &status, &value.refund.Version, &value.requestedByID, &value.decidedByID, &value.createdAt, &value.updatedAt, &value.decidedAt); err != nil {
			return order.RefundPage{}, fmt.Errorf("scan refund list row: %w", err)
		}
		value.refund.Currency = "CNY"
		value.refund.Status = order.RefundStatus(status)
		stored = append(stored, value)
		actorIDs[value.requestedByID] = true
		if value.decidedByID.Valid {
			actorIDs[value.decidedByID.String] = true
		}
	}
	if err := rows.Err(); err != nil {
		return order.RefundPage{}, fmt.Errorf("iterate refunds: %w", err)
	}
	if err := rows.Close(); err != nil {
		return order.RefundPage{}, fmt.Errorf("close refund rows: %w", err)
	}
	actors, err := loadRefundActorsTx(ctx, transaction, actorIDs)
	if err != nil {
		return order.RefundPage{}, err
	}
	if len(actorIDs) > 0 {
		database.observeQuery()
	}
	page.Items = make([]order.Refund, 0, len(stored))
	for _, value := range stored {
		requestedBy, ok := actors[value.requestedByID]
		if !ok {
			return order.RefundPage{}, errors.New("refund requester is missing")
		}
		value.refund.RequestedBy = requestedBy
		value.refund.CreatedAt, err = parseCanonicalRefundTime(value.createdAt)
		if err != nil {
			return order.RefundPage{}, err
		}
		value.refund.UpdatedAt, err = parseCanonicalRefundTime(value.updatedAt)
		if err != nil {
			return order.RefundPage{}, err
		}
		if value.decidedByID.Valid || value.decidedAt.Valid {
			if !value.decidedByID.Valid || !value.decidedAt.Valid {
				return order.RefundPage{}, errors.New("incomplete refund decision fields")
			}
			decidedBy, ok := actors[value.decidedByID.String]
			if !ok {
				return order.RefundPage{}, errors.New("refund decider is missing")
			}
			value.refund.DecidedBy = &decidedBy
			decidedAt, parseErr := parseCanonicalRefundTime(value.decidedAt.String)
			if parseErr != nil {
				return order.RefundPage{}, parseErr
			}
			value.refund.DecidedAt = &decidedAt
		}
		if err := order.ValidateRefund(value.refund); err != nil {
			return order.RefundPage{}, err
		}
		page.Items = append(page.Items, value.refund)
	}
	if err := transaction.Commit(); err != nil {
		return order.RefundPage{}, fmt.Errorf("commit refund list transaction: %w", err)
	}
	page.Page, page.PageSize = query.Page, query.PageSize
	return page, nil
}

type storedRefundListRow struct {
	refund        order.Refund
	requestedByID string
	decidedByID   sql.NullString
	createdAt     string
	updatedAt     string
	decidedAt     sql.NullString
}

func refundListWhere(query order.RefundListQuery) (string, []any) {
	conditions := []string{"1 = 1"}
	arguments := make([]any, 0, 2)
	if query.Status != "" {
		conditions = append(conditions, "status = ?")
		arguments = append(arguments, query.Status)
	}
	if query.OrderID != "" {
		conditions = append(conditions, "order_id = ?")
		arguments = append(arguments, query.OrderID)
	}
	return "WHERE " + strings.Join(conditions, " AND "), arguments
}

func loadRefundActorsTx(ctx context.Context, transaction *sql.Tx, ids map[string]bool) (map[string]order.RefundActor, error) {
	actors := make(map[string]order.RefundActor, len(ids))
	if len(ids) == 0 {
		return actors, nil
	}
	placeholders := make([]string, 0, len(ids))
	arguments := make([]any, 0, len(ids))
	for id := range ids {
		placeholders = append(placeholders, "?")
		arguments = append(arguments, id)
	}
	rows, err := transaction.QueryContext(ctx, `SELECT id, username FROM users WHERE id IN (`+strings.Join(placeholders, ",")+`)`, arguments...)
	if err != nil {
		return nil, fmt.Errorf("query refund actors: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var actor order.RefundActor
		if err := rows.Scan(&actor.ID, &actor.Username); err != nil {
			return nil, fmt.Errorf("scan refund actor: %w", err)
		}
		actors[actor.ID] = actor
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate refund actors: %w", err)
	}
	return actors, nil
}

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

func (database *DB) ApproveRefund(ctx context.Context, persistence order.RefundDecisionPersistence) (result order.Refund, resultErr error) {
	defer func() { resultErr = classifyOrderError(resultErr) }()
	transaction, err := database.sql.BeginTx(ctx, nil)
	if err != nil {
		return order.Refund{}, fmt.Errorf("begin approve refund transaction: %w", err)
	}
	defer func() {
		if resultErr != nil {
			_ = transaction.Rollback()
		}
	}()
	current, found, err := getRefundTx(ctx, transaction, persistence.RefundID)
	if err != nil {
		return order.Refund{}, err
	}
	if !found {
		return order.Refund{}, order.ErrNotFound
	}
	if current.Version != persistence.Version {
		return order.Refund{}, order.ErrVersionConflict
	}
	if current.Status != order.RefundStatusPending {
		return order.Refund{}, order.ErrStateConflict
	}
	if current.RequestedBy.ID == persistence.Actor.ID {
		return order.Refund{}, order.ErrForbidden
	}
	orderVersion, found, err := getRawOrderVersion(ctx, transaction, current.OrderID)
	if err != nil {
		return order.Refund{}, err
	}
	if !found {
		return order.Refund{}, errors.New("approved refund order is missing")
	}
	fence, err := transaction.ExecContext(ctx, `UPDATE orders SET version = version WHERE id = ? AND version = ?`, current.OrderID, orderVersion)
	if err != nil {
		return order.Refund{}, fmt.Errorf("acquire approve order writer fence: %w", err)
	}
	affected, err := fence.RowsAffected()
	if err != nil {
		return order.Refund{}, fmt.Errorf("read approve order writer fence rows: %w", err)
	}
	if affected != 1 {
		if _, stillFound, readErr := getRawOrderVersion(ctx, transaction, current.OrderID); readErr != nil {
			return order.Refund{}, readErr
		} else if !stillFound {
			return order.Refund{}, errors.New("approved refund order disappeared")
		}
		return order.Refund{}, order.Unavailable(errors.New("approve order writer fence lost adjacent write"))
	}
	value, found, err := getOrderTx(ctx, transaction, current.OrderID)
	if err != nil {
		return order.Refund{}, err
	}
	if !found {
		return order.Refund{}, errors.New("fenced approve order is missing")
	}
	aggregate, err := loadRefundAggregateTx(ctx, transaction, current.OrderID)
	if err != nil {
		return order.Refund{}, err
	}
	if err := validatePaymentStatusForRefundAggregate(value, aggregate); err != nil {
		return order.Refund{}, err
	}
	target, found := aggregate.refundByID(persistence.RefundID)
	if !found {
		return order.Refund{}, errors.New("approved refund disappeared from aggregate")
	}
	if target.Version != persistence.Version {
		return order.Refund{}, order.ErrVersionConflict
	}
	if target.Status != order.RefundStatusPending {
		return order.Refund{}, order.ErrStateConflict
	}
	if target.RequestedBy.ID == persistence.Actor.ID {
		return order.Refund{}, order.ErrForbidden
	}
	decidedAt, err := parseCanonicalRefundTime(persistence.DecidedAt)
	if err != nil {
		return order.Refund{}, err
	}
	decided, err := order.DecideRefund(target, order.RefundStatusCompleted, persistence.Actor, decidedAt)
	if err != nil {
		return order.Refund{}, err
	}
	completedAfter, err := checkedRefundSum(aggregate.completed, target.Amount)
	if err != nil {
		return order.Refund{}, err
	}
	paymentStatus, err := paymentStatusForCompleted(value.TotalAmount, completedAfter)
	if err != nil {
		return order.Refund{}, err
	}
	if value.Version == math.MaxInt64 {
		return order.Refund{}, errors.New("order version exhausted")
	}
	refundUpdate, err := transaction.ExecContext(ctx, `
		UPDATE refunds
		SET status = 'COMPLETED', version = version + 1, decided_by = ?, updated_at = ?, decided_at = ?
		WHERE id = ? AND version = ? AND status = 'PENDING'
	`, persistence.Actor.ID, persistence.DecidedAt, persistence.DecidedAt, persistence.RefundID, persistence.Version)
	if err != nil {
		return order.Refund{}, fmt.Errorf("complete refund: %w", err)
	}
	refundAffected, err := refundUpdate.RowsAffected()
	if err != nil {
		return order.Refund{}, fmt.Errorf("read completed refund rows: %w", err)
	}
	if refundAffected != 1 {
		return order.Refund{}, classifyRefundDecisionMiss(ctx, transaction, persistence)
	}
	orderUpdate, err := transaction.ExecContext(ctx, `
		UPDATE orders SET payment_status = ?, version = version + 1, updated_at = ?
		WHERE id = ? AND version = ?
	`, paymentStatus, persistence.DecidedAt, value.ID, value.Version)
	if err != nil {
		return order.Refund{}, fmt.Errorf("update approved refund order: %w", err)
	}
	orderAffected, err := orderUpdate.RowsAffected()
	if err != nil {
		return order.Refund{}, fmt.Errorf("read approved refund order rows: %w", err)
	}
	if orderAffected != 1 {
		return order.Refund{}, order.Unavailable(errors.New("approve order CAS lost adjacent write"))
	}
	result, found, err = getRefundTx(ctx, transaction, persistence.RefundID)
	if err != nil {
		return order.Refund{}, err
	}
	if !found || !sameRefund(result, decided) {
		return order.Refund{}, errors.New("completed refund differs from decision")
	}
	if err := transaction.Commit(); err != nil {
		return order.Refund{}, fmt.Errorf("commit approve refund transaction: %w", err)
	}
	return result, nil
}

func (database *DB) RejectRefund(ctx context.Context, persistence order.RefundDecisionPersistence) (result order.Refund, resultErr error) {
	defer func() { resultErr = classifyOrderError(resultErr) }()
	transaction, err := database.sql.BeginTx(ctx, nil)
	if err != nil {
		return order.Refund{}, fmt.Errorf("begin reject refund transaction: %w", err)
	}
	defer func() {
		if resultErr != nil {
			_ = transaction.Rollback()
		}
	}()
	current, found, err := getRefundTx(ctx, transaction, persistence.RefundID)
	if err != nil {
		return order.Refund{}, err
	}
	if !found {
		return order.Refund{}, order.ErrNotFound
	}
	if current.Version != persistence.Version {
		return order.Refund{}, order.ErrVersionConflict
	}
	if current.Status != order.RefundStatusPending {
		return order.Refund{}, order.ErrStateConflict
	}
	if current.RequestedBy.ID == persistence.Actor.ID {
		return order.Refund{}, order.ErrForbidden
	}
	decidedAt, err := parseCanonicalRefundTime(persistence.DecidedAt)
	if err != nil {
		return order.Refund{}, err
	}
	decided, err := order.DecideRefund(current, order.RefundStatusRejected, persistence.Actor, decidedAt)
	if err != nil {
		return order.Refund{}, err
	}
	update, err := transaction.ExecContext(ctx, `
		UPDATE refunds
		SET status = 'REJECTED', version = version + 1, decided_by = ?, updated_at = ?, decided_at = ?
		WHERE id = ? AND version = ? AND status = 'PENDING'
	`, persistence.Actor.ID, persistence.DecidedAt, persistence.DecidedAt, persistence.RefundID, persistence.Version)
	if err != nil {
		return order.Refund{}, fmt.Errorf("reject refund: %w", err)
	}
	affected, err := update.RowsAffected()
	if err != nil {
		return order.Refund{}, fmt.Errorf("read rejected refund rows: %w", err)
	}
	if affected != 1 {
		return order.Refund{}, classifyRefundDecisionMiss(ctx, transaction, persistence)
	}
	result, found, err = getRefundTx(ctx, transaction, persistence.RefundID)
	if err != nil {
		return order.Refund{}, err
	}
	if !found || !sameRefund(result, decided) {
		return order.Refund{}, errors.New("rejected refund differs from decision")
	}
	if err := transaction.Commit(); err != nil {
		return order.Refund{}, fmt.Errorf("commit reject refund transaction: %w", err)
	}
	return result, nil
}

func classifyRefundDecisionMiss(ctx context.Context, transaction *sql.Tx, persistence order.RefundDecisionPersistence) error {
	current, found, err := getRefundTx(ctx, transaction, persistence.RefundID)
	if err != nil {
		return err
	}
	if !found {
		return order.ErrNotFound
	}
	if current.Version != persistence.Version {
		return order.ErrVersionConflict
	}
	if current.Status != order.RefundStatusPending {
		return order.ErrStateConflict
	}
	if current.RequestedBy.ID == persistence.Actor.ID {
		return order.ErrForbidden
	}
	return errors.New("refund decision CAS missed current refund")
}

type refundAggregate struct {
	rows      int
	pending   int64
	completed int64
	refunds   []order.Refund
}

func (aggregate refundAggregate) refundByID(id string) (order.Refund, bool) {
	for _, value := range aggregate.refunds {
		if value.ID == id {
			return value, true
		}
	}
	return order.Refund{}, false
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
		value, err := scanRefund(rows)
		if err != nil {
			return refundAggregate{}, fmt.Errorf("scan order refund: %w", err)
		}
		if value.OrderID != orderID {
			return refundAggregate{}, errors.New("invalid refund order relation")
		}
		if err := order.ValidateRefund(value); err != nil {
			return refundAggregate{}, err
		}
		aggregate.rows++
		aggregate.refunds = append(aggregate.refunds, value)
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

func applyOrderRefundCapabilitiesTx(ctx context.Context, transaction *sql.Tx, values []order.Order) error {
	if len(values) == 0 {
		return nil
	}
	placeholders := make([]string, len(values))
	arguments := make([]any, len(values))
	for index, value := range values {
		placeholders[index] = "?"
		arguments[index] = value.ID
	}
	rows, err := transaction.QueryContext(ctx, `
		SELECT refunds.id, refunds.order_id, refunds.amount, refunds.reason, refunds.status, refunds.version,
			requester.id, requester.username, decider.id, decider.username,
			refunds.created_at, refunds.updated_at, refunds.decided_at
		FROM refunds
		LEFT JOIN users requester ON requester.id = refunds.requested_by
		LEFT JOIN users decider ON decider.id = refunds.decided_by
		WHERE refunds.order_id IN (`+strings.Join(placeholders, ",")+") ORDER BY refunds.order_id, refunds.created_at, refunds.id", arguments...)
	if err != nil {
		return fmt.Errorf("query listed order refunds: %w", err)
	}
	defer rows.Close()
	aggregates := make(map[string]refundAggregate, len(values))
	for _, value := range values {
		aggregates[value.ID] = refundAggregate{}
	}
	for rows.Next() {
		value, err := scanRefund(rows)
		if err != nil {
			return fmt.Errorf("scan listed order refund: %w", err)
		}
		aggregate, ok := aggregates[value.OrderID]
		if !ok {
			return errors.New("refund references unexpected listed order")
		}
		if err := order.ValidateRefund(value); err != nil {
			return err
		}
		aggregate.rows++
		aggregate.refunds = append(aggregate.refunds, value)
		switch value.Status {
		case order.RefundStatusPending:
			aggregate.pending, err = checkedRefundSum(aggregate.pending, value.Amount)
		case order.RefundStatusCompleted:
			aggregate.completed, err = checkedRefundSum(aggregate.completed, value.Amount)
		}
		if err != nil {
			return err
		}
		aggregates[value.OrderID] = aggregate
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate listed order refunds: %w", err)
	}
	for index := range values {
		aggregate := aggregates[values[index].ID]
		if err := validatePaymentStatusForRefundAggregate(values[index], aggregate); err != nil {
			return err
		}
		if values[index].PaymentStatus == order.PaymentStatusPaid || values[index].PaymentStatus == order.PaymentStatusPartiallyRefunded {
			values[index].AvailableRefundAmount = values[index].TotalAmount - aggregate.pending - aggregate.completed
		} else {
			values[index].AvailableRefundAmount = 0
		}
	}
	return nil
}

func getRefundTx(ctx context.Context, transaction *sql.Tx, refundID string) (order.Refund, bool, error) {
	value, err := scanRefund(transaction.QueryRowContext(ctx, `
		SELECT refunds.id, refunds.order_id, refunds.amount, refunds.reason, refunds.status, refunds.version,
			requester.id, requester.username, decider.id, decider.username,
			refunds.created_at, refunds.updated_at, refunds.decided_at
		FROM refunds
		LEFT JOIN users requester ON requester.id = refunds.requested_by
		LEFT JOIN users decider ON decider.id = refunds.decided_by
		WHERE refunds.id = ?
	`, refundID))
	if errors.Is(err, sql.ErrNoRows) {
		return order.Refund{}, false, nil
	}
	if err != nil {
		return order.Refund{}, false, err
	}
	if err := order.ValidateRefund(value); err != nil {
		return order.Refund{}, false, err
	}
	return value, true, nil
}

func scanRefund(row rowScanner) (order.Refund, error) {
	var value order.Refund
	var status, createdAt, updatedAt string
	var requestedID, requestedUsername, decidedID, decidedUsername, decidedAt sql.NullString
	if err := row.Scan(&value.ID, &value.OrderID, &value.Amount, &value.Reason, &status, &value.Version, &requestedID, &requestedUsername, &decidedID, &decidedUsername, &createdAt, &updatedAt, &decidedAt); err != nil {
		return order.Refund{}, err
	}
	if !requestedID.Valid || !requestedUsername.Valid {
		return order.Refund{}, errors.New("invalid refund requester")
	}
	value.Currency = "CNY"
	value.Status = order.RefundStatus(status)
	value.RequestedBy = order.RefundActor{ID: requestedID.String, Username: requestedUsername.String}
	var err error
	value.CreatedAt, err = parseCanonicalRefundTime(createdAt)
	if err != nil {
		return order.Refund{}, err
	}
	value.UpdatedAt, err = parseCanonicalRefundTime(updatedAt)
	if err != nil {
		return order.Refund{}, err
	}
	if decidedID.Valid || decidedUsername.Valid || decidedAt.Valid {
		if !decidedID.Valid || !decidedUsername.Valid || !decidedAt.Valid {
			return order.Refund{}, errors.New("incomplete refund decision fields")
		}
		actor := order.RefundActor{ID: decidedID.String, Username: decidedUsername.String}
		value.DecidedBy = &actor
		parsed, err := parseCanonicalRefundTime(decidedAt.String)
		if err != nil {
			return order.Refund{}, err
		}
		value.DecidedAt = &parsed
	}
	return value, nil
}

func sameRefund(left, right order.Refund) bool {
	if left.ID != right.ID || left.OrderID != right.OrderID || left.Amount != right.Amount || left.Currency != right.Currency || left.Reason != right.Reason || left.Status != right.Status || left.Version != right.Version || left.RequestedBy != right.RequestedBy || !left.CreatedAt.Equal(right.CreatedAt) || !left.UpdatedAt.Equal(right.UpdatedAt) {
		return false
	}
	if (left.DecidedBy == nil) != (right.DecidedBy == nil) || (left.DecidedAt == nil) != (right.DecidedAt == nil) {
		return false
	}
	if left.DecidedBy != nil && *left.DecidedBy != *right.DecidedBy {
		return false
	}
	return left.DecidedAt == nil || left.DecidedAt.Equal(*right.DecidedAt)
}

func checkedRefundSum(current, amount int64) (int64, error) {
	if amount < 0 || current > math.MaxInt64-amount {
		return 0, errors.New("refund amount overflow")
	}
	return current + amount, nil
}

func validatePaymentStatusForRefundAggregate(value order.Order, aggregate refundAggregate) error {
	occupied, err := checkedRefundSum(aggregate.pending, aggregate.completed)
	if err != nil {
		return err
	}
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

func paymentStatusForCompleted(totalAmount, completedAmount int64) (order.PaymentStatus, error) {
	switch {
	case completedAmount == 0:
		return order.PaymentStatusPaid, nil
	case completedAmount > 0 && completedAmount < totalAmount:
		return order.PaymentStatusPartiallyRefunded, nil
	case completedAmount == totalAmount:
		return order.PaymentStatusRefunded, nil
	default:
		return "", errors.New("completed refund amount is outside order total")
	}
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
