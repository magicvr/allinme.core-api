package store_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/order"
	"github.com/magicvr/allinme.core-api/internal/store"
)

func TestApproveRefundUpdatesRefundAndOrderAtomically(t *testing.T) {
	database := openRefundDemoDatabase(t)
	service := refundServiceFor(t, database, "unused")
	result, err := service.Approve(context.Background(), refundApprover(), "rfd_00000000000000000000000000000001", 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.Refund.Status != order.RefundStatusCompleted || result.Refund.Version != 2 || result.Refund.DecidedBy == nil || result.Refund.DecidedBy.Username != "approver" || result.Refund.DecidedAt == nil || result.Refund.DecidedAt.Format(time.RFC3339) != "2026-01-03T00:00:00Z" {
		t.Fatalf("approved refund = %+v", result)
	}
	var paymentStatus, updatedAt string
	var version int64
	if err := database.SQL().QueryRow(`SELECT payment_status, version, updated_at FROM orders WHERE id = 'ord_00000000000000000000000000000008'`).Scan(&paymentStatus, &version, &updatedAt); err != nil {
		t.Fatal(err)
	}
	if paymentStatus != "PARTIALLY_REFUNDED" || version != 2 || updatedAt != "2026-01-03T00:00:00Z" {
		t.Fatalf("approved order = %s version %d updated %s", paymentStatus, version, updatedAt)
	}
	var pending, completed int64
	if err := database.SQL().QueryRow(`SELECT COALESCE(SUM(CASE WHEN status = 'PENDING' THEN amount ELSE 0 END), 0), COALESCE(SUM(CASE WHEN status = 'COMPLETED' THEN amount ELSE 0 END), 0) FROM refunds WHERE order_id = 'ord_00000000000000000000000000000008'`).Scan(&pending, &completed); err != nil {
		t.Fatal(err)
	}
	if pending != 5000 || completed != 10000 {
		t.Fatalf("approved aggregate = pending %d completed %d", pending, completed)
	}
}

func TestRejectRefundDoesNotReadOrModifyOrder(t *testing.T) {
	database := openRefundDemoDatabase(t)
	if _, err := database.SQL().Exec(`UPDATE orders SET payment_status = 'REFUNDED', version = 9, updated_at = '2026-02-01T00:00:00Z' WHERE id = 'ord_00000000000000000000000000000008'`); err != nil {
		t.Fatal(err)
	}
	service := refundServiceFor(t, database, "unused")
	result, err := service.Reject(context.Background(), refundApprover(), "rfd_00000000000000000000000000000001", 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.Refund.Status != order.RefundStatusRejected || result.Refund.Version != 2 || result.Refund.DecidedBy == nil || result.Refund.DecidedBy.Username != "approver" {
		t.Fatalf("rejected refund = %+v", result)
	}
	var paymentStatus, updatedAt string
	var version int64
	if err := database.SQL().QueryRow(`SELECT payment_status, version, updated_at FROM orders WHERE id = 'ord_00000000000000000000000000000008'`).Scan(&paymentStatus, &version, &updatedAt); err != nil {
		t.Fatal(err)
	}
	if paymentStatus != "REFUNDED" || version != 9 || updatedAt != "2026-02-01T00:00:00Z" {
		t.Fatalf("reject changed order = %s version %d updated %s", paymentStatus, version, updatedAt)
	}
}

func TestRefundDecisionClassificationAndFailureStability(t *testing.T) {
	tests := []struct {
		name      string
		approve   bool
		refundID  string
		version   int64
		principal auth.Principal
		mutate    func(*testing.T, *store.DB)
		want      error
	}{
		{name: "missing", approve: true, refundID: "rfd_eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", version: 1, principal: refundApprover(), want: order.ErrNotFound},
		{name: "version before state", approve: true, refundID: "rfd_00000000000000000000000000000003", version: 1, principal: refundApprover(), want: order.ErrVersionConflict},
		{name: "state after version", approve: true, refundID: "rfd_00000000000000000000000000000003", version: 2, principal: refundApprover(), want: order.ErrStateConflict},
		{name: "self approval", approve: true, refundID: "rfd_00000000000000000000000000000002", version: 1, principal: refundAdmin(), want: order.ErrForbidden},
		{name: "self reject", refundID: "rfd_00000000000000000000000000000002", version: 1, principal: refundAdmin(), want: order.ErrForbidden},
		{name: "approve corrupt aggregate", approve: true, refundID: "rfd_00000000000000000000000000000001", version: 1, principal: refundApprover(), mutate: func(t *testing.T, database *store.DB) {
			if _, err := database.SQL().Exec(`UPDATE orders SET payment_status = 'REFUNDED' WHERE id = 'ord_00000000000000000000000000000008'`); err != nil {
				t.Fatal(err)
			}
		}, want: order.ErrInternal},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := openRefundDemoDatabase(t)
			if test.mutate != nil {
				test.mutate(t, database)
			}
			beforeRefund := readRefundState(t, database, test.refundID)
			beforeOrder := readOrderState(t, database, "ord_00000000000000000000000000000008")
			service := refundServiceFor(t, database, "unused")
			var err error
			if test.approve {
				_, err = service.Approve(context.Background(), test.principal, test.refundID, test.version)
			} else {
				_, err = service.Reject(context.Background(), test.principal, test.refundID, test.version)
			}
			if !errors.Is(err, test.want) {
				t.Fatalf("decision error = %v, want %v", err, test.want)
			}
			if after := readRefundState(t, database, test.refundID); after != beforeRefund {
				t.Fatalf("failed decision changed refund: %v -> %v", beforeRefund, after)
			}
			if after := readOrderState(t, database, "ord_00000000000000000000000000000008"); after != beforeOrder {
				t.Fatalf("failed decision changed order: %v -> %v", beforeOrder, after)
			}
		})
	}
}

func TestApproveRefundRollsBackWhenOrderUpdateFailsOrMisses(t *testing.T) {
	for _, test := range []struct {
		name       string
		triggerSQL string
		want       error
	}{
		{name: "failure", triggerSQL: `CREATE TRIGGER fail_approve_order BEFORE UPDATE OF payment_status ON orders BEGIN SELECT RAISE(ABORT, 'forced order failure'); END`, want: order.ErrInternal},
		{name: "zero rows", triggerSQL: `CREATE TRIGGER ignore_approve_order BEFORE UPDATE OF payment_status ON orders BEGIN SELECT RAISE(IGNORE); END`, want: order.ErrUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			database := openRefundDemoDatabase(t)
			if _, err := database.SQL().Exec(test.triggerSQL); err != nil {
				t.Fatal(err)
			}
			beforeRefund := readRefundState(t, database, "rfd_00000000000000000000000000000001")
			beforeOrder := readOrderState(t, database, "ord_00000000000000000000000000000008")
			service := refundServiceFor(t, database, "unused")
			if _, err := service.Approve(context.Background(), refundApprover(), "rfd_00000000000000000000000000000001", 1); !errors.Is(err, test.want) {
				t.Fatalf("approve error = %v, want %v", err, test.want)
			}
			if after := readRefundState(t, database, "rfd_00000000000000000000000000000001"); after != beforeRefund {
				t.Fatalf("rollback refund = %v -> %v", beforeRefund, after)
			}
			if after := readOrderState(t, database, "ord_00000000000000000000000000000008"); after != beforeOrder {
				t.Fatalf("rollback order = %v -> %v", beforeOrder, after)
			}
		})
	}
}

func TestApproveRejectAcrossTwoDatabasesChangesTerminalStateOnce(t *testing.T) {
	first, second := openSharedRefundDemoDatabases(t)
	services := []*order.RefundService{refundServiceFor(t, first, "unused"), refundServiceFor(t, second, "unused")}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	start := make(chan struct{})
	errorsFound := make([]error, 2)
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		<-start
		_, errorsFound[0] = services[0].Approve(ctx, refundApprover(), "rfd_00000000000000000000000000000001", 1)
	}()
	go func() {
		defer wait.Done()
		<-start
		_, errorsFound[1] = services[1].Reject(ctx, refundAdmin(), "rfd_00000000000000000000000000000001", 1)
	}()
	close(start)
	done := make(chan struct{})
	go func() { wait.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatalf("approve/reject deadline: %v", ctx.Err())
	}
	for index, err := range errorsFound {
		if errors.Is(err, order.ErrUnavailable) {
			if index == 0 {
				_, errorsFound[index] = services[index].Approve(context.Background(), refundApprover(), "rfd_00000000000000000000000000000001", 1)
			} else {
				_, errorsFound[index] = services[index].Reject(context.Background(), refundAdmin(), "rfd_00000000000000000000000000000001", 1)
			}
		}
	}
	successes, versions := 0, 0
	for _, err := range errorsFound {
		if err == nil {
			successes++
		} else if errors.Is(err, order.ErrVersionConflict) {
			versions++
		} else {
			t.Fatalf("decision race error = %v", err)
		}
	}
	if successes != 1 || versions != 1 {
		t.Fatalf("decision race results = %v", errorsFound)
	}
	state := readRefundState(t, first, "rfd_00000000000000000000000000000001")
	if state.version != 2 || (state.status != "COMPLETED" && state.status != "REJECTED") {
		t.Fatalf("terminal refund state = %+v", state)
	}
	orderState := readOrderState(t, first, "ord_00000000000000000000000000000008")
	if state.status == "COMPLETED" {
		if orderState.paymentStatus != "PARTIALLY_REFUNDED" || orderState.version != 2 {
			t.Fatalf("completed race order = %+v", orderState)
		}
	} else if orderState.paymentStatus != "PAID" || orderState.version != 1 {
		t.Fatalf("rejected race order = %+v", orderState)
	}
}

type storedRefundState struct {
	status    string
	version   int64
	decidedBy string
	updatedAt string
	decidedAt string
}

func readRefundState(t *testing.T, database *store.DB, refundID string) storedRefundState {
	t.Helper()
	var state storedRefundState
	var decidedBy, decidedAt sql.NullString
	err := database.SQL().QueryRow(`SELECT status, version, decided_by, updated_at, decided_at FROM refunds WHERE id = ?`, refundID).Scan(&state.status, &state.version, &decidedBy, &state.updatedAt, &decidedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return state
	}
	if err != nil {
		t.Fatal(err)
	}
	if decidedBy.Valid {
		state.decidedBy = decidedBy.String
	}
	if decidedAt.Valid {
		state.decidedAt = decidedAt.String
	}
	return state
}

type storedOrderState struct {
	paymentStatus string
	version       int64
	updatedAt     string
}

func readOrderState(t *testing.T, database *store.DB, orderID string) storedOrderState {
	t.Helper()
	var state storedOrderState
	if err := database.SQL().QueryRow(`SELECT payment_status, version, updated_at FROM orders WHERE id = ?`, orderID).Scan(&state.paymentStatus, &state.version, &state.updatedAt); err != nil {
		t.Fatal(err)
	}
	return state
}

func refundApprover() auth.Principal {
	return auth.Principal{UserID: "user-approver", Username: "approver", Role: auth.RoleApprover}
}

func refundAdmin() auth.Principal {
	return auth.Principal{UserID: "user-admin", Username: "admin", Role: auth.RoleAdmin}
}
