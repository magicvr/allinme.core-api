package order

import (
	"context"
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
	ID            string
	CustomerName  string
	Status        Status
	PaymentStatus PaymentStatus
	Currency      string
	TotalAmount   int64
	Version       int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Items         []Item
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
}
