package order_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/order"
)

var _ order.Repository = repositoryStub{}

func TestStatusesAndPaymentStatuses(t *testing.T) {
	for _, status := range []order.Status{
		order.StatusDraft,
		order.StatusConfirmed,
		order.StatusFulfilling,
		order.StatusShipped,
		order.StatusCompleted,
		order.StatusCancelled,
	} {
		if !status.Valid() {
			t.Errorf("status %q is invalid", status)
		}
	}
	if order.Status("UNKNOWN").Valid() {
		t.Fatal("unknown order status is valid")
	}
	for _, status := range []order.PaymentStatus{
		order.PaymentStatusUnpaid,
		order.PaymentStatusPaid,
		order.PaymentStatusPartiallyRefunded,
		order.PaymentStatusRefunded,
	} {
		if !status.Valid() {
			t.Errorf("payment status %q is invalid", status)
		}
	}
	if order.PaymentStatus("UNKNOWN").Valid() {
		t.Fatal("unknown payment status is valid")
	}
}

func TestReadPermissionAndCapabilities(t *testing.T) {
	roles := []auth.Role{auth.RoleViewer, auth.RoleOperator, auth.RoleApprover, auth.RoleAdmin}
	statuses := []order.Status{
		order.StatusDraft,
		order.StatusConfirmed,
		order.StatusFulfilling,
		order.StatusShipped,
		order.StatusCompleted,
		order.StatusCancelled,
	}
	for _, role := range roles {
		principal := auth.Principal{Role: role}
		if !order.CanRead(principal) {
			t.Errorf("role %q cannot read", role)
		}
		for _, status := range statuses {
			capabilities := order.CapabilitiesFor(principal, status)
			canWrite := role == auth.RoleOperator || role == auth.RoleAdmin
			if capabilities.CanEdit != (canWrite && status == order.StatusDraft) {
				t.Errorf("role %q status %q CanEdit = %v", role, status, capabilities.CanEdit)
			}
			if capabilities.CanAdvance != (canWrite && status.Advanceable()) {
				t.Errorf("role %q status %q CanAdvance = %v", role, status, capabilities.CanAdvance)
			}
			if capabilities.CanCancel != (canWrite && status.Cancellable()) {
				t.Errorf("role %q status %q CanCancel = %v", role, status, capabilities.CanCancel)
			}
			if capabilities.CanRequestRefund || capabilities.CanApproveRefund {
				t.Errorf("phase three refund capability enabled for role %q status %q", role, status)
			}
		}
	}
	if order.CanRead(auth.Principal{Role: auth.Role("unknown")}) {
		t.Fatal("unknown role can read")
	}
}

func TestOrderRefundCapabilityUsesRolePaymentStatusAndAvailableAmount(t *testing.T) {
	paid := order.Order{PaymentStatus: order.PaymentStatusPaid, AvailableRefundAmount: 100}
	partial := order.Order{PaymentStatus: order.PaymentStatusPartiallyRefunded, AvailableRefundAmount: 1}
	for _, role := range []auth.Role{auth.RoleOperator, auth.RoleAdmin} {
		if !order.CapabilitiesForOrder(auth.Principal{Role: role}, paid).CanRequestRefund || !order.CapabilitiesForOrder(auth.Principal{Role: role}, partial).CanRequestRefund {
			t.Errorf("role %s cannot request refund for available paid amount", role)
		}
	}
	for _, value := range []order.Order{
		{PaymentStatus: order.PaymentStatusUnpaid, AvailableRefundAmount: 100},
		{PaymentStatus: order.PaymentStatusRefunded, AvailableRefundAmount: 100},
		{PaymentStatus: order.PaymentStatusPaid, AvailableRefundAmount: 0},
	} {
		if order.CapabilitiesForOrder(auth.Principal{Role: auth.RoleOperator}, value).CanRequestRefund {
			t.Errorf("invalid capability enabled for %+v", value)
		}
	}
	if order.CapabilitiesForOrder(auth.Principal{Role: auth.RoleViewer}, paid).CanRequestRefund || order.CapabilitiesForOrder(auth.Principal{Role: auth.RoleApprover}, paid).CanRequestRefund {
		t.Fatal("non-requesting role can request refund")
	}
	if order.CapabilitiesForOrder(auth.Principal{Role: auth.RoleAdmin}, paid).CanApproveRefund {
		t.Fatal("Order DTO exposes approve capability")
	}
}

func TestIdentifiersUseFixedSecureFormat(t *testing.T) {
	input := bytes.Repeat([]byte{0xab}, 16)
	orderID, err := order.NewOrderIDFrom(bytes.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	itemID, err := order.NewItemIDFrom(bytes.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if orderID != "ord_abababababababababababababababab" || !order.ValidOrderID(orderID) {
		t.Fatalf("order ID = %q", orderID)
	}
	if itemID != "itm_abababababababababababababababab" || !order.ValidItemID(itemID) {
		t.Fatalf("item ID = %q", itemID)
	}
	for _, invalid := range []string{"", "ord_1", "ord_ABABABABABABABABABABABABABABABAB", "itm_abababababababababababababababab"} {
		if order.ValidOrderID(invalid) {
			t.Errorf("invalid order ID accepted: %q", invalid)
		}
	}
	readerError := errors.New("reader failed")
	if _, err := order.NewOrderIDFrom(failingReader{err: readerError}); !errors.Is(err, readerError) {
		t.Fatalf("NewOrderIDFrom() error = %v", err)
	}
}

func TestClockAndFormattingNormalizeUTC(t *testing.T) {
	local := time.Date(2026, 7, 12, 20, 30, 0, 0, time.FixedZone("CST", 8*60*60))
	got := order.UTCNow(func() time.Time { return local })
	if got.Location() != time.UTC || got.Format(time.RFC3339) != "2026-07-12T12:30:00Z" {
		t.Fatalf("UTCNow() = %v", got)
	}
	if formatted := order.FormatTime(local); formatted != "2026-07-12T12:30:00Z" {
		t.Fatalf("FormatTime() = %q", formatted)
	}
}

type failingReader struct {
	err error
}

func (reader failingReader) Read([]byte) (int, error) {
	return 0, reader.err
}

type repositoryStub struct{}

func (repositoryStub) ListOrders(context.Context, order.ListQuery) (order.Page, error) {
	return order.Page{}, nil
}

func (repositoryStub) GetOrder(context.Context, string) (order.Order, bool, error) {
	return order.Order{}, false, nil
}

func (repositoryStub) GetIdempotency(context.Context, order.IdempotencyScope) (order.IdempotencyRecord, bool, error) {
	return order.IdempotencyRecord{}, false, nil
}

func (repositoryStub) CreateOrderIdempotent(_ context.Context, persistence order.IdempotentCreatePersistence) (order.IdempotencyRecord, bool, error) {
	return persistence.Record, true, nil
}

func (repositoryStub) UpdateDraft(context.Context, order.UpdateDraftPersistence) (order.Order, error) {
	return order.Order{}, nil
}
func (repositoryStub) TransitionOrder(context.Context, order.TransitionPersistence) (order.Order, error) {
	return order.Order{}, nil
}
