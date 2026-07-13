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

func (database *DB) DashboardSummary(ctx context.Context) (result order.DashboardSummary, resultErr error) {
	defer func() { resultErr = classifyOrderError(resultErr) }()
	err := database.withDashboardFacts(ctx, func(facts dashboardFacts) error {
		result.Currency = "CNY"
		for _, value := range facts.orders {
			var err error
			result.OrderCount, err = checkedDashboardAdd(result.OrderCount, 1)
			if err != nil {
				return err
			}
			if dashboardGrossEligible(value.PaymentStatus) {
				result.GrossAmount, err = checkedDashboardAdd(result.GrossAmount, value.TotalAmount)
				if err != nil {
					return err
				}
			}
		}
		for _, value := range facts.refunds {
			if value.Status == order.RefundStatusCompleted {
				var err error
				result.CompletedRefundAmount, err = checkedDashboardAdd(result.CompletedRefundAmount, value.Amount)
				if err != nil {
					return err
				}
			}
		}
		var err error
		result.NetAmount, err = checkedDashboardSubtract(result.GrossAmount, result.CompletedRefundAmount)
		if err != nil {
			return err
		}
		if result.NetAmount < 0 {
			return errors.New("dashboard summary net amount is negative")
		}
		return nil
	})
	return result, err
}

func (database *DB) DashboardOrderStatus(ctx context.Context) (result order.DashboardOrderStatus, resultErr error) {
	defer func() { resultErr = classifyOrderError(resultErr) }()
	err := database.withDashboardFacts(ctx, func(facts dashboardFacts) error {
		counts := map[order.Status]int64{}
		for _, value := range facts.orders {
			count, err := checkedDashboardAdd(counts[value.Status], 1)
			if err != nil {
				return err
			}
			counts[value.Status] = count
		}
		result.Items = make([]order.DashboardStatusItem, 0, 6)
		for _, status := range []order.Status{order.StatusDraft, order.StatusConfirmed, order.StatusFulfilling, order.StatusShipped, order.StatusCompleted, order.StatusCancelled} {
			result.Items = append(result.Items, order.DashboardStatusItem{Status: status, Count: counts[status]})
		}
		return nil
	})
	return result, err
}

func (database *DB) DashboardTrend(ctx context.Context, days int, now time.Time) (result order.DashboardTrend, resultErr error) {
	defer func() { resultErr = classifyOrderError(resultErr) }()
	if days != 7 && days != 30 {
		return order.DashboardTrend{}, order.Internal(errors.New("invalid dashboard trend days"))
	}
	end := dashboardUTCDate(now)
	start := end.AddDate(0, 0, -(days - 1))
	result = order.DashboardTrend{Days: days, StartDate: endDateString(start), EndDate: endDateString(end), Items: make([]order.DashboardTrendItem, days)}
	indexes := make(map[string]int, days)
	for index := 0; index < days; index++ {
		date := endDateString(start.AddDate(0, 0, index))
		indexes[date] = index
		result.Items[index].Date = date
	}
	err := database.withDashboardFacts(ctx, func(facts dashboardFacts) error {
		for _, value := range facts.orders {
			index, ok := indexes[endDateString(value.CreatedAt)]
			if !ok {
				continue
			}
			item := result.Items[index]
			var err error
			item.OrderCount, err = checkedDashboardAdd(item.OrderCount, 1)
			if err != nil {
				return err
			}
			if dashboardGrossEligible(value.PaymentStatus) {
				item.GrossAmount, err = checkedDashboardAdd(item.GrossAmount, value.TotalAmount)
				if err != nil {
					return err
				}
			}
			result.Items[index] = item
		}
		for _, value := range facts.refunds {
			if value.Status != order.RefundStatusCompleted || value.DecidedAt == nil {
				continue
			}
			index, ok := indexes[endDateString(*value.DecidedAt)]
			if !ok {
				continue
			}
			item := result.Items[index]
			var err error
			item.CompletedRefundAmount, err = checkedDashboardAdd(item.CompletedRefundAmount, value.Amount)
			if err != nil {
				return err
			}
			result.Items[index] = item
		}
		for index := range result.Items {
			var err error
			result.Items[index].NetAmount, err = checkedDashboardSubtract(result.Items[index].GrossAmount, result.Items[index].CompletedRefundAmount)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return result, err
}

type dashboardFacts struct {
	orders  []order.Order
	refunds []order.Refund
}

func (database *DB) withDashboardFacts(ctx context.Context, callback func(dashboardFacts) error) (resultErr error) {
	transaction, err := database.sql.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("begin dashboard transaction: %w", err)
	}
	defer func() {
		if resultErr != nil {
			_ = transaction.Rollback()
		}
	}()
	facts, err := database.loadDashboardFactsTx(ctx, transaction)
	if err != nil {
		return err
	}
	if err := callback(facts); err != nil {
		return err
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit dashboard transaction: %w", err)
	}
	return nil
}

func (database *DB) loadDashboardFactsTx(ctx context.Context, transaction *sql.Tx) (dashboardFacts, error) {
	rows, err := transaction.QueryContext(ctx, `SELECT id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at FROM orders ORDER BY id`)
	if err != nil {
		return dashboardFacts{}, fmt.Errorf("query dashboard orders: %w", err)
	}
	database.observeQuery()
	defer rows.Close()
	var facts dashboardFacts
	orderIDs := make(map[string]bool)
	for rows.Next() {
		value, err := scanOrder(rows)
		if err != nil {
			return dashboardFacts{}, fmt.Errorf("scan dashboard order: %w", err)
		}
		facts.orders = append(facts.orders, value)
		orderIDs[value.ID] = true
	}
	if err := rows.Err(); err != nil {
		return dashboardFacts{}, fmt.Errorf("iterate dashboard orders: %w", err)
	}
	if err := rows.Close(); err != nil {
		return dashboardFacts{}, fmt.Errorf("close dashboard order rows: %w", err)
	}
	if len(facts.orders) > 0 {
		database.observeQuery()
	}
	if err := validatePageOrderAggregates(ctx, transaction, facts.orders); err != nil {
		return dashboardFacts{}, err
	}
	refundRows, err := transaction.QueryContext(ctx, `
		SELECT refunds.id, refunds.order_id, refunds.amount, refunds.reason, refunds.status, refunds.version,
			requester.id, requester.username, decider.id, decider.username,
			refunds.created_at, refunds.updated_at, refunds.decided_at
		FROM refunds
		LEFT JOIN users requester ON requester.id = refunds.requested_by
		LEFT JOIN users decider ON decider.id = refunds.decided_by
		ORDER BY refunds.order_id, refunds.created_at, refunds.id
	`)
	if err != nil {
		return dashboardFacts{}, fmt.Errorf("query dashboard refunds: %w", err)
	}
	database.observeQuery()
	defer refundRows.Close()
	aggregates := make(map[string]refundAggregate, len(facts.orders))
	for _, value := range facts.orders {
		aggregates[value.ID] = refundAggregate{}
	}
	for refundRows.Next() {
		value, err := scanRefund(refundRows)
		if err != nil {
			return dashboardFacts{}, fmt.Errorf("scan dashboard refund: %w", err)
		}
		if !orderIDs[value.OrderID] {
			return dashboardFacts{}, errors.New("dashboard refund references missing order")
		}
		if err := order.ValidateRefund(value); err != nil {
			return dashboardFacts{}, err
		}
		facts.refunds = append(facts.refunds, value)
		aggregate := aggregates[value.OrderID]
		aggregate.rows++
		var sumErr error
		switch value.Status {
		case order.RefundStatusPending:
			aggregate.pending, sumErr = checkedRefundSum(aggregate.pending, value.Amount)
		case order.RefundStatusCompleted:
			aggregate.completed, sumErr = checkedRefundSum(aggregate.completed, value.Amount)
		}
		if sumErr != nil {
			return dashboardFacts{}, sumErr
		}
		aggregates[value.OrderID] = aggregate
	}
	if err := refundRows.Err(); err != nil {
		return dashboardFacts{}, fmt.Errorf("iterate dashboard refunds: %w", err)
	}
	for _, value := range facts.orders {
		if err := validatePaymentStatusForRefundAggregate(value, aggregates[value.ID]); err != nil {
			return dashboardFacts{}, err
		}
	}
	return facts, nil
}

func dashboardGrossEligible(status order.PaymentStatus) bool {
	return status == order.PaymentStatusPaid || status == order.PaymentStatusPartiallyRefunded || status == order.PaymentStatusRefunded
}

func checkedDashboardAdd(current, amount int64) (int64, error) {
	if amount > 0 && current > math.MaxInt64-amount {
		return 0, errors.New("dashboard amount overflow")
	}
	if amount < 0 && current < math.MinInt64-amount {
		return 0, errors.New("dashboard amount underflow")
	}
	return current + amount, nil
}

func checkedDashboardSubtract(left, right int64) (int64, error) {
	if right > 0 && left < math.MinInt64+right {
		return 0, errors.New("dashboard subtraction underflow")
	}
	if right < 0 && left > math.MaxInt64+right {
		return 0, errors.New("dashboard subtraction overflow")
	}
	return left - right, nil
}

func dashboardUTCDate(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func endDateString(value time.Time) string {
	return value.UTC().Format("2006-01-02")
}
