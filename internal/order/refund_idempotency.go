package order

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
)

const (
	RefundCreateMethod       = "POST"
	RefundCreateOperation    = "POST /api/v1/orders/{orderId}/refunds"
	RefundSnapshotVersionOne = int64(1)
)

type RefundIdempotencyScope struct {
	PrincipalUserID string
	Method          string
	Operation       string
	OrderID         string
	Key             string
	RequestDigest   [sha256.Size]byte
}

type RefundIdempotencyRecord struct {
	Scope           RefundIdempotencyScope
	RefundID        string
	SnapshotVersion int64
	SnapshotJSON    []byte
	SnapshotDigest  [sha256.Size]byte
	CreatedAt       string
}

type RefundCreatePersistence struct {
	OrderVersion int64
	Refund       Refund
	Record       RefundIdempotencyRecord
}

type RefundResult struct {
	Refund       Refund
	Capabilities RefundCapabilities
}

type RefundRepository interface {
	GetRefundIdempotency(context.Context, RefundIdempotencyScope) (RefundIdempotencyRecord, bool, error)
	CreateRefundIdempotent(context.Context, RefundCreatePersistence) (RefundIdempotencyRecord, bool, error)
}

type RefundDecisionPersistence struct {
	RefundID  string
	Version   int64
	Actor     RefundActor
	DecidedAt string
}

type RefundDecisionRepository interface {
	RefundRepository
	ApproveRefund(context.Context, RefundDecisionPersistence) (Refund, error)
	RejectRefund(context.Context, RefundDecisionPersistence) (Refund, error)
}

type RefundService struct {
	repository  RefundRepository
	clock       Clock
	newRefundID func() (string, error)
}

func NewRefundService(repository RefundRepository) (*RefundService, error) {
	return NewRefundServiceWithDependencies(repository, nil, nil)
}

func NewRefundServiceWithDependencies(repository RefundRepository, clock Clock, newRefundID func() (string, error)) (*RefundService, error) {
	if repository == nil {
		return nil, errors.New("refund repository is required")
	}
	if newRefundID == nil {
		newRefundID = NewRefundID
	}
	return &RefundService{repository: repository, clock: clock, newRefundID: newRefundID}, nil
}

type refundCreateDigest struct {
	Operation    string `json:"operation"`
	OrderID      string `json:"orderId"`
	Amount       int64  `json:"amount"`
	Reason       string `json:"reason"`
	OrderVersion int64  `json:"orderVersion"`
}

type refundSnapshotV1 struct {
	Refund refundSnapshotValueV1 `json:"refund"`
}

type refundSnapshotValueV1 struct {
	ID          string                 `json:"id"`
	OrderID     string                 `json:"orderId"`
	Amount      int64                  `json:"amount"`
	Currency    string                 `json:"currency"`
	Reason      string                 `json:"reason"`
	Status      RefundStatus           `json:"status"`
	Version     int64                  `json:"version"`
	RequestedBy refundSnapshotActorV1  `json:"requestedBy"`
	DecidedBy   *refundSnapshotActorV1 `json:"decidedBy"`
	CreatedAt   string                 `json:"createdAt"`
	UpdatedAt   string                 `json:"updatedAt"`
	DecidedAt   *string                `json:"decidedAt"`
	CanApprove  bool                   `json:"canApprove"`
	CanReject   bool                   `json:"canReject"`
}

type refundSnapshotActorV1 struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

func (service *RefundService) Create(ctx context.Context, principal auth.Principal, orderID, key string, command RefundRequestCommand) (RefundResult, error) {
	if !auth.RoleAllowed(principal.Role, auth.RoleOperator, auth.RoleAdmin) {
		return RefundResult{}, ErrForbidden
	}
	if !ValidOrderID(orderID) {
		return RefundResult{}, ErrNotFound
	}
	normalized, err := NormalizeRefundRequest(command)
	if err != nil {
		return RefundResult{}, err
	}
	digest, err := normalizedRefundDigest(orderID, normalized)
	if err != nil {
		return RefundResult{}, err
	}
	scope := RefundIdempotencyScope{PrincipalUserID: principal.UserID, Method: RefundCreateMethod, Operation: RefundCreateOperation, OrderID: orderID, Key: key, RequestDigest: digest}
	existing, found, err := service.repository.GetRefundIdempotency(ctx, scope)
	if err != nil {
		return RefundResult{}, fmt.Errorf("read refund idempotency key: %w", err)
	}
	if found {
		return decodeRefundSnapshot(existing, scope)
	}
	refundID, err := service.newRefundID()
	if err != nil {
		return RefundResult{}, fmt.Errorf("generate refund ID: %w", err)
	}
	now := UTCNow(service.clock)
	value, err := NewPendingRefund(refundID, orderID, "CNY", normalized, RefundActor{ID: principal.UserID, Username: principal.Username}, now)
	if err != nil {
		return RefundResult{}, err
	}
	snapshot, err := encodeRefundSnapshotV1(value)
	if err != nil {
		return RefundResult{}, fmt.Errorf("encode refund snapshot: %w", err)
	}
	record, _, err := service.repository.CreateRefundIdempotent(ctx, RefundCreatePersistence{
		OrderVersion: normalized.OrderVersion,
		Refund:       value,
		Record: RefundIdempotencyRecord{
			Scope: scope, RefundID: value.ID, SnapshotVersion: RefundSnapshotVersionOne,
			SnapshotJSON: snapshot, SnapshotDigest: sha256.Sum256(snapshot), CreatedAt: FormatTime(value.CreatedAt),
		},
	})
	if err != nil {
		return RefundResult{}, fmt.Errorf("create refund: %w", err)
	}
	return decodeRefundSnapshot(record, scope)
}

func (service *RefundService) Approve(ctx context.Context, principal auth.Principal, refundID string, version int64) (RefundResult, error) {
	return service.decide(ctx, principal, refundID, version, RefundStatusCompleted)
}

func (service *RefundService) Reject(ctx context.Context, principal auth.Principal, refundID string, version int64) (RefundResult, error) {
	return service.decide(ctx, principal, refundID, version, RefundStatusRejected)
}

func (service *RefundService) decide(ctx context.Context, principal auth.Principal, refundID string, version int64, target RefundStatus) (RefundResult, error) {
	if !auth.RoleAllowed(principal.Role, auth.RoleApprover, auth.RoleAdmin) {
		return RefundResult{}, ErrForbidden
	}
	if !ValidRefundID(refundID) {
		return RefundResult{}, ErrNotFound
	}
	if version <= 0 {
		return RefundResult{}, &ValidationError{Details: []FieldError{{Field: "version", Message: "must be greater than 0"}}}
	}
	repository, ok := service.repository.(RefundDecisionRepository)
	if !ok {
		return RefundResult{}, ErrInternal
	}
	persistence := RefundDecisionPersistence{RefundID: refundID, Version: version, Actor: RefundActor{ID: principal.UserID, Username: principal.Username}, DecidedAt: FormatTime(UTCNow(service.clock))}
	var value Refund
	var err error
	if target == RefundStatusCompleted {
		value, err = repository.ApproveRefund(ctx, persistence)
	} else {
		value, err = repository.RejectRefund(ctx, persistence)
	}
	if err != nil {
		return RefundResult{}, fmt.Errorf("%s refund: %w", target, err)
	}
	if err := ValidateRefund(value); err != nil || value.Status != target {
		return RefundResult{}, ErrInternal
	}
	return RefundResult{Refund: value, Capabilities: RefundCapabilitiesFor(principal, value)}, nil
}

func normalizedRefundDigest(orderID string, command RefundRequestCommand) ([sha256.Size]byte, error) {
	encoded, err := json.Marshal(refundCreateDigest{Operation: RefundCreateOperation, OrderID: orderID, Amount: command.Amount, Reason: command.Reason, OrderVersion: command.OrderVersion})
	if err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("encode normalized refund digest: %w", err)
	}
	return sha256.Sum256(encoded), nil
}

func encodeRefundSnapshotV1(value Refund) ([]byte, error) {
	requestedBy := refundSnapshotActorV1{ID: value.RequestedBy.ID, Username: value.RequestedBy.Username}
	return json.Marshal(refundSnapshotV1{Refund: refundSnapshotValueV1{
		ID: value.ID, OrderID: value.OrderID, Amount: value.Amount, Currency: value.Currency, Reason: value.Reason,
		Status: value.Status, Version: value.Version, RequestedBy: requestedBy,
		CreatedAt: FormatTime(value.CreatedAt), UpdatedAt: FormatTime(value.UpdatedAt), CanApprove: false, CanReject: false,
	}})
}

func decodeRefundSnapshot(record RefundIdempotencyRecord, expected RefundIdempotencyScope) (RefundResult, error) {
	if record.Scope.PrincipalUserID != expected.PrincipalUserID || record.Scope.Method != RefundCreateMethod || record.Scope.Operation != RefundCreateOperation || record.Scope.OrderID != expected.OrderID || record.Scope.Key != expected.Key || record.RefundID == "" || record.SnapshotVersion != RefundSnapshotVersionOne {
		return RefundResult{}, ErrInternal
	}
	if !bytes.Equal(record.Scope.RequestDigest[:], expected.RequestDigest[:]) {
		return RefundResult{}, ErrIdempotencyConflict
	}
	snapshotDigest := sha256.Sum256(record.SnapshotJSON)
	if !bytes.Equal(record.SnapshotDigest[:], snapshotDigest[:]) {
		return RefundResult{}, ErrInternal
	}
	decoder := json.NewDecoder(bytes.NewReader(record.SnapshotJSON))
	decoder.DisallowUnknownFields()
	var snapshot refundSnapshotV1
	if err := decoder.Decode(&snapshot); err != nil {
		return RefundResult{}, ErrInternal
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return RefundResult{}, ErrInternal
	}
	value := snapshot.Refund
	createdAt, createdErr := time.Parse(time.RFC3339, value.CreatedAt)
	updatedAt, updatedErr := time.Parse(time.RFC3339, value.UpdatedAt)
	if createdErr != nil || updatedErr != nil || value.DecidedBy != nil || value.DecidedAt != nil || value.CanApprove || value.CanReject || value.ID != record.RefundID || value.OrderID != record.Scope.OrderID || record.CreatedAt != value.CreatedAt {
		return RefundResult{}, ErrInternal
	}
	result := Refund{
		ID: value.ID, OrderID: value.OrderID, Amount: value.Amount, Currency: value.Currency, Reason: value.Reason,
		Status: value.Status, Version: value.Version,
		RequestedBy: RefundActor{ID: value.RequestedBy.ID, Username: value.RequestedBy.Username},
		CreatedAt:   createdAt, UpdatedAt: updatedAt,
	}
	if err := ValidateRefund(result); err != nil {
		return RefundResult{}, ErrInternal
	}
	return RefundResult{Refund: result}, nil
}

func ValidateRefundCreatePersistence(value RefundCreatePersistence) error {
	if value.OrderVersion <= 0 || value.Record.Scope.PrincipalUserID != value.Refund.RequestedBy.ID || value.Record.Scope.Method != RefundCreateMethod || value.Record.Scope.Operation != RefundCreateOperation || value.Record.Scope.OrderID != value.Refund.OrderID || value.Record.RefundID != value.Refund.ID {
		return ErrInternal
	}
	digest, err := normalizedRefundDigest(value.Refund.OrderID, RefundRequestCommand{Amount: value.Refund.Amount, Reason: value.Refund.Reason, OrderVersion: value.OrderVersion})
	if err != nil || !bytes.Equal(digest[:], value.Record.Scope.RequestDigest[:]) {
		return ErrInternal
	}
	decoded, err := decodeRefundSnapshot(value.Record, value.Record.Scope)
	if err != nil || decoded.Refund != value.Refund || decoded.Capabilities.CanApprove || decoded.Capabilities.CanReject {
		return ErrInternal
	}
	return nil
}
