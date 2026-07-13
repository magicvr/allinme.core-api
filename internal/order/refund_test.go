package order_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/order"
)

func TestRefundStatusesAndIdentifiers(t *testing.T) {
	for _, status := range []order.RefundStatus{order.RefundStatusPending, order.RefundStatusRejected, order.RefundStatusCompleted} {
		if !status.Valid() {
			t.Errorf("status %q is invalid", status)
		}
	}
	if order.RefundStatus("APPROVED").Valid() {
		t.Fatal("APPROVED is a persisted refund status")
	}
	id, err := order.NewRefundIDFrom(bytes.NewReader(bytes.Repeat([]byte{0xcd}, 16)))
	if err != nil {
		t.Fatal(err)
	}
	if id != "rfd_cdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcd" || !order.ValidRefundID(id) {
		t.Fatalf("refund ID = %q", id)
	}
	for _, invalid := range []string{"", "rfd_1", "rfd_CDCDCDCDCDCDCDCDCDCDCDCDCDCDCDCD", "ord_cdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcd"} {
		if order.ValidRefundID(invalid) {
			t.Errorf("invalid refund ID accepted: %q", invalid)
		}
	}
}

func TestNormalizeRefundRequestBoundaries(t *testing.T) {
	command, err := order.NormalizeRefundRequest(order.RefundRequestCommand{Amount: 100, Reason: "  line one\nline two  ", OrderVersion: 3})
	if err != nil {
		t.Fatal(err)
	}
	if command.Amount != 100 || command.Reason != "line one\nline two" || command.OrderVersion != 3 {
		t.Fatalf("normalized command = %+v", command)
	}
	invalidUTF8 := string([]byte{0xff})
	for _, reason := range []string{" ", strings.Repeat("a", 501), "contains\x00nul", invalidUTF8} {
		if _, err := order.NormalizeRefundReason(reason); err == nil {
			t.Errorf("NormalizeRefundReason(%q) error = nil", reason)
		}
	}
	_, err = order.NormalizeRefundRequest(order.RefundRequestCommand{Amount: 0, Reason: " ", OrderVersion: 0})
	details, ok := order.ValidationDetails(err)
	if !ok || len(details) != 3 || details[0].Field != "amount" || details[1].Field != "reason" || details[2].Field != "orderVersion" {
		t.Fatalf("validation details = %+v, %v", details, ok)
	}
}

func TestValidateRefundAuditFields(t *testing.T) {
	created := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	decided := created.Add(time.Hour)
	requested := order.RefundActor{ID: "user-operator", Username: "operator"}
	approver := order.RefundActor{ID: "user-approver", Username: "approver"}
	pending := order.Refund{
		ID: "rfd_00000000000000000000000000000001", OrderID: "ord_00000000000000000000000000000001",
		Amount: 100, Currency: "CNY", Reason: "customer request", Status: order.RefundStatusPending, Version: 1,
		RequestedBy: requested, CreatedAt: created, UpdatedAt: created,
	}
	if err := order.ValidateRefund(pending); err != nil {
		t.Fatal(err)
	}
	completed := pending
	completed.Status = order.RefundStatusCompleted
	completed.Version = 2
	completed.DecidedBy = &approver
	completed.DecidedAt = &decided
	completed.UpdatedAt = decided
	if err := order.ValidateRefund(completed); err != nil {
		t.Fatal(err)
	}
	invalid := []order.Refund{
		func() order.Refund { value := pending; value.ID = "bad"; return value }(),
		func() order.Refund { value := pending; value.Reason = " outer "; return value }(),
		func() order.Refund { value := pending; value.Version = 2; return value }(),
		func() order.Refund { value := completed; value.DecidedAt = nil; return value }(),
		func() order.Refund { value := completed; value.UpdatedAt = created; return value }(),
		func() order.Refund { value := completed; value.RequestedBy.Username = "Operator"; return value }(),
		func() order.Refund {
			value := pending
			value.CreatedAt = created.In(time.FixedZone("UTC+0", 0))
			value.UpdatedAt = value.CreatedAt
			return value
		}(),
	}
	for index, value := range invalid {
		if err := order.ValidateRefund(value); !errors.Is(err, order.ErrInternal) {
			t.Errorf("invalid refund %d error = %v", index, err)
		}
	}
}

func TestRefundCreationAndDecisionUseFrozenAuditTransitions(t *testing.T) {
	local := time.Date(2026, 1, 2, 8, 0, 0, 500, time.FixedZone("CST", 8*60*60))
	requested := order.RefundActor{ID: "user-operator", Username: "operator"}
	pending, err := order.NewPendingRefund(
		"rfd_00000000000000000000000000000001",
		"ord_00000000000000000000000000000001",
		"CNY",
		order.RefundRequestCommand{Amount: 100, Reason: " request ", OrderVersion: 3},
		requested,
		local,
	)
	if err != nil {
		t.Fatal(err)
	}
	if pending.Status != order.RefundStatusPending || pending.Version != 1 || pending.Reason != "request" || pending.CreatedAt.Format(time.RFC3339) != "2026-01-02T00:00:00Z" || !pending.UpdatedAt.Equal(pending.CreatedAt) || pending.DecidedBy != nil || pending.DecidedAt != nil {
		t.Fatalf("pending refund = %+v", pending)
	}
	original := pending
	approver := order.RefundActor{ID: "user-approver", Username: "approver"}
	completed, err := order.DecideRefund(pending, order.RefundStatusCompleted, approver, local.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != order.RefundStatusCompleted || completed.Version != 2 || completed.DecidedAt == nil || completed.UpdatedAt.Format(time.RFC3339) != "2026-01-02T02:00:00Z" || completed.DecidedBy == nil || completed.DecidedBy.ID != approver.ID {
		t.Fatalf("completed refund = %+v", completed)
	}
	if pending != original {
		t.Fatalf("decision mutated pending input: before %+v after %+v", original, pending)
	}
	if _, err := order.DecideRefund(pending, order.RefundStatusCompleted, requested, local.Add(time.Hour)); !errors.Is(err, order.ErrForbidden) || pending != original {
		t.Fatalf("self decision error = %v, pending = %+v", err, pending)
	}
	if _, err := order.DecideRefund(completed, order.RefundStatusRejected, approver, local.Add(3*time.Hour)); !errors.Is(err, order.ErrStateConflict) || pending != original {
		t.Fatalf("terminal decision error = %v", err)
	}
}

func TestRefundCapabilitiesRequireDifferentApprover(t *testing.T) {
	value := order.Refund{Status: order.RefundStatusPending, RequestedBy: order.RefundActor{ID: "requester", Username: "operator"}}
	for _, test := range []struct {
		principal auth.Principal
		want      bool
	}{
		{principal: auth.Principal{UserID: "approver", Role: auth.RoleApprover}, want: true},
		{principal: auth.Principal{UserID: "admin", Role: auth.RoleAdmin}, want: true},
		{principal: auth.Principal{UserID: "requester", Role: auth.RoleAdmin}},
		{principal: auth.Principal{UserID: "operator", Role: auth.RoleOperator}},
		{principal: auth.Principal{Role: auth.RoleApprover}},
	} {
		capabilities := order.RefundCapabilitiesFor(test.principal, value)
		if capabilities.CanApprove != test.want || capabilities.CanReject != test.want {
			t.Errorf("principal %+v capabilities = %+v, want %v", test.principal, capabilities, test.want)
		}
	}
	value.Status = order.RefundStatusCompleted
	if capabilities := order.RefundCapabilitiesFor(auth.Principal{UserID: "approver", Role: auth.RoleApprover}, value); capabilities.CanApprove || capabilities.CanReject {
		t.Fatalf("completed capabilities = %+v", capabilities)
	}
}
