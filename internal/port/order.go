package port

import (
	"context"
	"errors"
	"time"

	"github.com/magicvr/allinme.core-api/internal/domain"
)

var (
	// ErrOrderNotFound is returned when an order cannot be located.
	ErrOrderNotFound = errors.New("order: not found")
	// ErrOrderNoConflict is returned when an order number is already used.
	ErrOrderNoConflict = errors.New("order: order number conflict")
	// ErrVersionConflict is returned when a compare-and-swap write is stale.
	ErrVersionConflict = errors.New("order: version conflict")
	// ErrInvalidState is returned when an operation is not permitted in the current state.
	ErrInvalidState = errors.New("order: invalid state")
	// ErrInvalidArgument is returned when a use-case input violates its contract.
	ErrInvalidArgument = errors.New("order: invalid argument")
)

// OrderListFilter describes repository-side list filtering and pagination.
type OrderListFilter struct {
	Status   domain.OrderStatus
	Query    string
	Page     int
	PageSize int
}

// OrderRepository is the outbound persistence port for orders.
type OrderRepository interface {
	Create(ctx context.Context, order domain.Order) error
	Get(ctx context.Context, id string) (domain.Order, error)
	List(ctx context.Context, filter OrderListFilter) ([]domain.Order, int, error)
	Update(ctx context.Context, order domain.Order) error
	ChangeStatus(ctx context.Context, id string, version int64, status domain.OrderStatus, updatedAt time.Time) error
	BatchDelete(ctx context.Context, ids []string) error
	Count(ctx context.Context) (int, error)
}
