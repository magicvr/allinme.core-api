package order

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/magicvr/allinme.core-api/internal/auth"
)

const MaxRefundReasonBytes = 500

type RefundStatus string

const (
	RefundStatusPending   RefundStatus = "PENDING"
	RefundStatusRejected  RefundStatus = "REJECTED"
	RefundStatusCompleted RefundStatus = "COMPLETED"
)

func (status RefundStatus) Valid() bool {
	switch status {
	case RefundStatusPending, RefundStatusRejected, RefundStatusCompleted:
		return true
	default:
		return false
	}
}

type RefundActor struct {
	ID       string
	Username string
}

type Refund struct {
	ID          string
	OrderID     string
	Amount      int64
	Currency    string
	Reason      string
	Status      RefundStatus
	Version     int64
	RequestedBy RefundActor
	DecidedBy   *RefundActor
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DecidedAt   *time.Time
}

type RefundCapabilities struct {
	CanApprove bool
	CanReject  bool
}

type RefundRequestCommand struct {
	Amount       int64
	Reason       string
	OrderVersion int64
}

func NormalizeRefundRequest(command RefundRequestCommand) (RefundRequestCommand, error) {
	details := make([]FieldError, 0)
	if command.Amount < 1 || command.Amount > MaxAmount {
		details = append(details, FieldError{Field: "amount", Message: "must be between 1 and 9999999999"})
	}
	reason, reasonErr := NormalizeRefundReason(command.Reason)
	if reasonErr != nil {
		if reasonDetails, ok := ValidationDetails(reasonErr); ok {
			details = append(details, reasonDetails...)
		}
	}
	if command.OrderVersion <= 0 {
		details = append(details, FieldError{Field: "orderVersion", Message: "must be greater than 0"})
	}
	if len(details) > 0 {
		return RefundRequestCommand{}, &ValidationError{Details: details}
	}
	return RefundRequestCommand{Amount: command.Amount, Reason: reason, OrderVersion: command.OrderVersion}, nil
}

func NormalizeRefundReason(reason string) (string, error) {
	if !utf8.ValidString(reason) {
		return "", refundReasonValidation("must be between 1 and 500 UTF-8 bytes")
	}
	if strings.ContainsRune(reason, '\x00') {
		return "", refundReasonValidation("must not contain NUL")
	}
	normalized := strings.TrimSpace(reason)
	if length := len([]byte(normalized)); length < 1 || length > MaxRefundReasonBytes {
		return "", refundReasonValidation("must be between 1 and 500 UTF-8 bytes")
	}
	return normalized, nil
}

func refundReasonValidation(message string) error {
	return &ValidationError{Details: []FieldError{{Field: "reason", Message: message}}}
}

func RefundCapabilitiesFor(principal auth.Principal, value Refund) RefundCapabilities {
	allowed := auth.RoleAllowed(principal.Role, auth.RoleApprover, auth.RoleAdmin)
	canDecide := allowed && principal.UserID != "" && value.Status == RefundStatusPending && principal.UserID != value.RequestedBy.ID
	return RefundCapabilities{CanApprove: canDecide, CanReject: canDecide}
}

func CanRequestRefund(principal auth.Principal, paymentStatus PaymentStatus, available int64) bool {
	return auth.RoleAllowed(principal.Role, auth.RoleOperator, auth.RoleAdmin) && (paymentStatus == PaymentStatusPaid || paymentStatus == PaymentStatusPartiallyRefunded) && available > 0
}

func NewPendingRefund(id, orderID, currency string, command RefundRequestCommand, requestedBy RefundActor, now time.Time) (Refund, error) {
	normalized, err := NormalizeRefundRequest(command)
	if err != nil {
		return Refund{}, err
	}
	createdAt := now.UTC().Truncate(time.Second)
	value := Refund{
		ID:          id,
		OrderID:     orderID,
		Amount:      normalized.Amount,
		Currency:    currency,
		Reason:      normalized.Reason,
		Status:      RefundStatusPending,
		Version:     1,
		RequestedBy: requestedBy,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}
	if err := ValidateRefund(value); err != nil {
		return Refund{}, err
	}
	return value, nil
}

func DecideRefund(value Refund, target RefundStatus, decidedBy RefundActor, now time.Time) (Refund, error) {
	if err := ValidateRefund(value); err != nil {
		return Refund{}, err
	}
	if value.Status != RefundStatusPending {
		return Refund{}, ErrStateConflict
	}
	if target != RefundStatusRejected && target != RefundStatusCompleted {
		return Refund{}, ErrStateConflict
	}
	if decidedBy.ID == value.RequestedBy.ID {
		return Refund{}, ErrForbidden
	}
	decidedAt := now.UTC().Truncate(time.Second)
	result := value
	result.Status = target
	result.Version = 2
	result.DecidedBy = &decidedBy
	result.DecidedAt = &decidedAt
	result.UpdatedAt = decidedAt
	if err := ValidateRefund(result); err != nil {
		return Refund{}, err
	}
	return result, nil
}

func ValidateRefund(value Refund) error {
	if !ValidRefundID(value.ID) {
		return invalidRefund("invalid refund ID")
	}
	if !ValidOrderID(value.OrderID) {
		return invalidRefund("invalid refund order ID")
	}
	if value.Amount < 1 || value.Amount > MaxAmount || value.Currency != "CNY" {
		return invalidRefund("invalid refund amount or currency")
	}
	normalizedReason, err := NormalizeRefundReason(value.Reason)
	if err != nil || normalizedReason != value.Reason {
		return invalidRefund("invalid refund reason")
	}
	if !value.Status.Valid() || !validRefundActor(value.RequestedBy) || !validRefundTime(value.CreatedAt) || !validRefundTime(value.UpdatedAt) {
		return invalidRefund("invalid refund facts")
	}
	switch value.Status {
	case RefundStatusPending:
		if value.Version != 1 || value.DecidedBy != nil || value.DecidedAt != nil || !value.UpdatedAt.Equal(value.CreatedAt) {
			return invalidRefund("invalid pending refund audit fields")
		}
	case RefundStatusRejected, RefundStatusCompleted:
		if value.Version != 2 || value.DecidedBy == nil || !validRefundActor(*value.DecidedBy) || value.DecidedAt == nil || !validRefundTime(*value.DecidedAt) || !value.UpdatedAt.Equal(*value.DecidedAt) || value.DecidedAt.Before(value.CreatedAt) {
			return invalidRefund("invalid decided refund audit fields")
		}
	default:
		return invalidRefund("invalid refund status")
	}
	return nil
}

func invalidRefund(message string) error {
	return Internal(errors.New(message))
}

func validRefundActor(actor RefundActor) bool {
	return actor.ID != "" && utf8.ValidString(actor.Username) && actor.Username != "" && !strings.ContainsRune(actor.Username, '\x00') && auth.NormalizeUsername(actor.Username) == actor.Username
}

func validRefundTime(value time.Time) bool {
	return !value.IsZero() && value.Location() == time.UTC && value.Nanosecond() == 0
}
