package store_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/order"
	"github.com/magicvr/allinme.core-api/internal/store"
)

func TestEditAndRefundCreateAcrossTwoDatabasesPreserveTotalAndOccupancy(t *testing.T) {
	first, second := openSharedDraftRefundDatabases(t, 0)
	editService := orderServiceForRefundEdit(t, first, func() (string, error) { return "itm_dddddddddddddddddddddddddddddddd", nil })
	createService := refundServiceFor(t, second, "rfd_dddddddddddddddddddddddddddddddd")
	editCommand := order.EditCommand{CustomerName: "Edited", Currency: "CNY", Version: 1, Items: []order.ItemCommand{{SKU: "EDIT", Name: "Edited", Quantity: 1, UnitPrice: 40}}}
	createCommand := order.RefundRequestCommand{Amount: 50, Reason: "concurrent create", OrderVersion: 1}
	errorsFound := runTwoOperations(t,
		func(ctx context.Context) error {
			_, err := editService.Edit(ctx, auth.Principal{Role: auth.RoleAdmin}, draftRefundOrderID, editCommand)
			return err
		},
		func(ctx context.Context) error {
			_, err := createService.Create(ctx, refundOperator(), draftRefundOrderID, "edit-create", createCommand)
			return err
		},
	)
	if errors.Is(errorsFound[0], order.ErrUnavailable) {
		_, errorsFound[0] = editService.Edit(context.Background(), auth.Principal{Role: auth.RoleAdmin}, draftRefundOrderID, editCommand)
	}
	if errors.Is(errorsFound[1], order.ErrUnavailable) {
		_, errorsFound[1] = createService.Create(context.Background(), refundOperator(), draftRefundOrderID, "edit-create", createCommand)
	}
	successes := 0
	for _, err := range errorsFound {
		if err == nil {
			successes++
		} else if !errors.Is(err, order.ErrVersionConflict) {
			if _, ok := order.ValidationDetails(err); !ok {
				t.Fatalf("edit/create error = %v", err)
			}
		}
	}
	if successes != 1 {
		t.Fatalf("edit/create results = %v", errorsFound)
	}
	state := readDraftRefundAggregate(t, first)
	if state.totalAmount == 40 {
		if state.version != 2 || state.pending != 0 || state.refunds != 0 {
			t.Fatalf("edit-first state = %+v", state)
		}
	} else if state.totalAmount == 100 {
		if state.version != 1 || state.pending != 50 || state.refunds != 1 {
			t.Fatalf("refund-first state = %+v", state)
		}
	} else {
		t.Fatalf("unexpected edit/create state = %+v", state)
	}
}

func TestEditAndApproveAcrossTwoDatabasesReobserveNewTotal(t *testing.T) {
	first, second := openSharedDraftRefundDatabases(t, 20)
	editService := orderServiceForRefundEdit(t, first, func() (string, error) { return "itm_dddddddddddddddddddddddddddddddd", nil })
	approveService := refundServiceFor(t, second, "unused")
	editCommand := order.EditCommand{CustomerName: "Edited", Currency: "CNY", Version: 1, Items: []order.ItemCommand{{SKU: "EDIT", Name: "Edited", Quantity: 1, UnitPrice: 80}}}
	errorsFound := runTwoOperations(t,
		func(ctx context.Context) error {
			_, err := editService.Edit(ctx, auth.Principal{Role: auth.RoleAdmin}, draftRefundOrderID, editCommand)
			return err
		},
		func(ctx context.Context) error {
			_, err := approveService.Approve(ctx, refundApprover(), draftRefundID, 1)
			return err
		},
	)
	if errors.Is(errorsFound[0], order.ErrUnavailable) {
		_, errorsFound[0] = editService.Edit(context.Background(), auth.Principal{Role: auth.RoleAdmin}, draftRefundOrderID, editCommand)
	}
	if errors.Is(errorsFound[1], order.ErrUnavailable) {
		_, errorsFound[1] = approveService.Approve(context.Background(), refundApprover(), draftRefundID, 1)
	}
	if errorsFound[1] != nil {
		t.Fatalf("approve did not converge: %v", errorsFound)
	}
	if errorsFound[0] != nil && !errors.Is(errorsFound[0], order.ErrVersionConflict) {
		t.Fatalf("edit/approve error = %v", errorsFound[0])
	}
	state := readDraftRefundAggregate(t, first)
	if state.completed != 20 || state.pending != 0 || state.paymentStatus != "PARTIALLY_REFUNDED" {
		t.Fatalf("edit/approve aggregate = %+v", state)
	}
	if errorsFound[0] == nil {
		if state.totalAmount != 80 || state.version != 3 {
			t.Fatalf("edit-first approve state = %+v", state)
		}
	} else if state.totalAmount != 100 || state.version != 2 {
		t.Fatalf("approve-first edit state = %+v", state)
	}
}

func TestRefundCreateAndApproveAcrossTwoDatabasesSerializeOccupancy(t *testing.T) {
	first, second := openSharedDraftRefundDatabases(t, 20)
	createService := refundServiceFor(t, first, "rfd_dddddddddddddddddddddddddddddddd")
	approveService := refundServiceFor(t, second, "unused")
	createCommand := order.RefundRequestCommand{Amount: 30, Reason: "concurrent pending", OrderVersion: 1}
	errorsFound := runTwoOperations(t,
		func(ctx context.Context) error {
			_, err := createService.Create(ctx, refundOperator(), draftRefundOrderID, "create-approve", createCommand)
			return err
		},
		func(ctx context.Context) error {
			_, err := approveService.Approve(ctx, refundApprover(), draftRefundID, 1)
			return err
		},
	)
	if errors.Is(errorsFound[0], order.ErrUnavailable) {
		_, errorsFound[0] = createService.Create(context.Background(), refundOperator(), draftRefundOrderID, "create-approve", createCommand)
	}
	if errors.Is(errorsFound[1], order.ErrUnavailable) {
		_, errorsFound[1] = approveService.Approve(context.Background(), refundApprover(), draftRefundID, 1)
	}
	if errorsFound[1] != nil {
		t.Fatalf("approve did not converge: %v", errorsFound)
	}
	if errorsFound[0] != nil && !errors.Is(errorsFound[0], order.ErrVersionConflict) {
		t.Fatalf("create/approve error = %v", errorsFound[0])
	}
	state := readDraftRefundAggregate(t, first)
	if state.completed != 20 || state.paymentStatus != "PARTIALLY_REFUNDED" || state.version != 2 {
		t.Fatalf("create/approve aggregate = %+v", state)
	}
	if errorsFound[0] == nil {
		if state.pending != 30 || state.refunds != 2 {
			t.Fatalf("create-first approve state = %+v", state)
		}
	} else if state.pending != 0 || state.refunds != 1 {
		t.Fatalf("approve-first create state = %+v", state)
	}
}

const (
	draftRefundOrderID = "ord_cccccccccccccccccccccccccccccccc"
	draftRefundID      = "rfd_cccccccccccccccccccccccccccccccc"
)

func openSharedDraftRefundDatabases(t *testing.T, pendingAmount int64) (*store.DB, *store.DB) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "draft-refund-race.db")
	first, err := store.Open(context.Background(), path, store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { first.Close() })
	if _, err := first.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	insertRefundSeedActors(t, first)
	if _, err := first.SQL().Exec(`
		INSERT INTO orders(id, customer_name, status, payment_status, currency, total_amount, version, created_at, updated_at)
		VALUES (?, 'Concurrent Draft', 'DRAFT', 'PAID', 'CNY', 100, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO order_items(id, order_id, position, sku, name, quantity, unit_price)
		VALUES ('itm_cccccccccccccccccccccccccccccccc', ?, 0, 'ORIGINAL', 'Original', 1, 100)
	`, draftRefundOrderID, draftRefundOrderID); err != nil {
		t.Fatal(err)
	}
	if pendingAmount > 0 {
		operatorID := refundSeedActorID(t, first, "operator")
		if _, err := first.SQL().Exec(`
			INSERT INTO refunds(id, order_id, amount, reason, status, version, requested_by, created_at, updated_at)
			VALUES (?, ?, ?, 'existing pending', 'PENDING', 1, ?, '2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z')
		`, draftRefundID, draftRefundOrderID, pendingAmount, operatorID); err != nil {
			t.Fatal(err)
		}
	}
	second, err := store.Open(context.Background(), path, store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { second.Close() })
	if _, err := second.SQL().Exec(`PRAGMA busy_timeout = 10`); err != nil {
		t.Fatal(err)
	}
	return first, second
}

type draftRefundAggregateState struct {
	totalAmount   int64
	version       int64
	paymentStatus string
	pending       int64
	completed     int64
	refunds       int
}

func readDraftRefundAggregate(t *testing.T, database *store.DB) draftRefundAggregateState {
	t.Helper()
	var state draftRefundAggregateState
	if err := database.SQL().QueryRow(`SELECT total_amount, version, payment_status FROM orders WHERE id = ?`, draftRefundOrderID).Scan(&state.totalAmount, &state.version, &state.paymentStatus); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRow(`
		SELECT COUNT(*),
			COALESCE(SUM(CASE WHEN status = 'PENDING' THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'COMPLETED' THEN amount ELSE 0 END), 0)
		FROM refunds WHERE order_id = ?
	`, draftRefundOrderID).Scan(&state.refunds, &state.pending, &state.completed); err != nil {
		t.Fatal(err)
	}
	return state
}

func runTwoOperations(t *testing.T, first, second func(context.Context) error) []error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	start := make(chan struct{})
	foundErrors := make([]error, 2)
	operations := []func(context.Context) error{first, second}
	var wait sync.WaitGroup
	for index := range operations {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			foundErrors[index] = operations[index](ctx)
		}(index)
	}
	close(start)
	done := make(chan struct{})
	go func() { wait.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatalf("cross-invariant deadline: %v", ctx.Err())
	}
	return foundErrors
}
