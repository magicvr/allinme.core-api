package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
)

type Status string

const (
	StatusDraft      Status = "DRAFT"
	StatusConfirmed  Status = "CONFIRMED"
	StatusFulfilling Status = "FULFILLING"
	StatusShipped    Status = "SHIPPED"
	StatusCompleted  Status = "COMPLETED"
	StatusCancelled  Status = "CANCELLED"
)

func (status Status) Valid() bool {
	switch status {
	case StatusDraft, StatusConfirmed, StatusFulfilling, StatusShipped, StatusCompleted, StatusCancelled:
		return true
	default:
		return false
	}
}

type PaymentStatus string

const (
	PaymentStatusUnpaid            PaymentStatus = "UNPAID"
	PaymentStatusPaid              PaymentStatus = "PAID"
	PaymentStatusPartiallyRefunded PaymentStatus = "PARTIALLY_REFUNDED"
	PaymentStatusRefunded          PaymentStatus = "REFUNDED"
)

func (status PaymentStatus) Valid() bool {
	switch status {
	case PaymentStatusUnpaid, PaymentStatusPaid, PaymentStatusPartiallyRefunded, PaymentStatusRefunded:
		return true
	default:
		return false
	}
}

type Item struct {
	ID        string
	SKU       string
	Name      string
	Quantity  int64
	UnitPrice int64
}

type Order struct {
	ID                    string
	CustomerName          string
	Status                Status
	PaymentStatus         PaymentStatus
	Currency              string
	TotalAmount           int64
	Version               int64
	CreatedAt             time.Time
	UpdatedAt             time.Time
	AvailableRefundAmount int64
	AttachmentCount       int64
	Items                 []Item
	Attachments           []Attachment
}

type Capabilities struct {
	CanEdit          bool
	CanAdvance       bool
	CanCancel        bool
	CanRequestRefund bool
	CanApproveRefund bool
}

func CanRead(principal auth.Principal) bool {
	return auth.RoleAllowed(principal.Role, auth.RoleViewer, auth.RoleOperator, auth.RoleApprover, auth.RoleAdmin)
}

func CapabilitiesFor(principal auth.Principal, status Status) Capabilities {
	canWrite := auth.RoleAllowed(principal.Role, auth.RoleOperator, auth.RoleAdmin)
	return Capabilities{
		CanEdit:    canWrite && status == StatusDraft,
		CanAdvance: canWrite && status.Advanceable(),
		CanCancel:  canWrite && status.Cancellable(),
	}
}

func CapabilitiesForOrder(principal auth.Principal, value Order) Capabilities {
	capabilities := CapabilitiesFor(principal, value.Status)
	capabilities.CanRequestRefund = CanRequestRefund(principal, value.PaymentStatus, value.AvailableRefundAmount)
	capabilities.CanApproveRefund = false
	return capabilities
}

func (status Status) Advanceable() bool {
	switch status {
	case StatusDraft, StatusConfirmed, StatusFulfilling, StatusShipped:
		return true
	default:
		return false
	}
}

func (status Status) Cancellable() bool {
	switch status {
	case StatusDraft, StatusConfirmed, StatusFulfilling:
		return true
	default:
		return false
	}
}

type ListQuery struct {
	Keyword       string
	Status        Status
	PaymentStatus PaymentStatus
	CreatedFrom   *time.Time
	CreatedTo     *time.Time
	Page          int64
	PageSize      int64
	Sort          string
	Descending    bool
}

type Page struct {
	Items    []Order
	Total    int64
	Page     int64
	PageSize int64
}

type Repository interface {
	ListOrders(context.Context, ListQuery) (Page, error)
	GetOrder(context.Context, string) (Order, bool, error)
	GetIdempotency(context.Context, IdempotencyScope) (IdempotencyRecord, bool, error)
	PrepareAttachmentsForOrder(context.Context, string, []string, time.Time) ([]Attachment, error)
	CreateOrderIdempotent(context.Context, IdempotentCreatePersistence) (IdempotencyRecord, bool, error)
	UpdateDraft(context.Context, UpdateDraftPersistence) (Order, error)
	TransitionOrder(context.Context, TransitionPersistence) (Order, error)
}

var (
	ErrForbidden           = errors.New("order access forbidden")
	ErrNotFound            = errors.New("order not found")
	ErrVersionConflict     = errors.New("order version conflict")
	ErrStateConflict       = errors.New("order state conflict")
	ErrIdempotencyConflict = errors.New("order idempotency conflict")
	ErrInternal            = errors.New("order internal error")
	ErrUnavailable         = errors.New("order store unavailable")
)

type classifiedError struct {
	kind  error
	cause error
}

func (err classifiedError) Error() string   { return err.kind.Error() }
func (err classifiedError) Unwrap() []error { return []error{err.kind, err.cause} }

func Internal(cause error) error {
	return classifiedError{kind: ErrInternal, cause: cause}
}

func Unavailable(cause error) error {
	return classifiedError{kind: ErrUnavailable, cause: cause}
}

type Service struct {
	repository Repository
	clock      Clock
	newOrderID func() (string, error)
	newItemID  func() (string, error)
}

func NewService(repository Repository) (*Service, error) {
	return NewServiceWithDependencies(repository, nil, nil, nil)
}

func NewServiceWithDependencies(repository Repository, clock Clock, newOrderID, newItemID func() (string, error)) (*Service, error) {
	if repository == nil {
		return nil, errors.New("order repository is required")
	}
	if newOrderID == nil {
		newOrderID = NewOrderID
	}
	if newItemID == nil {
		newItemID = NewItemID
	}
	return &Service{repository: repository, clock: clock, newOrderID: newOrderID, newItemID: newItemID}, nil
}

func (service *Service) List(ctx context.Context, principal auth.Principal, query ListQuery) (Page, error) {
	if !CanRead(principal) {
		return Page{}, ErrForbidden
	}
	page, err := service.repository.ListOrders(ctx, query)
	if err != nil {
		return Page{}, fmt.Errorf("list orders: %w", err)
	}
	return page, nil
}

func (service *Service) Get(ctx context.Context, principal auth.Principal, id string) (Order, error) {
	if !CanRead(principal) {
		return Order{}, ErrForbidden
	}
	result, found, err := service.repository.GetOrder(ctx, id)
	if err != nil {
		return Order{}, fmt.Errorf("get order: %w", err)
	}
	if !found {
		return Order{}, ErrNotFound
	}
	return result, nil
}
