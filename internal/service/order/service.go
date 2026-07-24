package order

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/magicvr/allinme.core-api/internal/domain"
	"github.com/magicvr/allinme.core-api/internal/port"
)

// Service implements the order use cases using only the OrderRepository port.
type Service struct {
	repository port.OrderRepository
	now        func() time.Time
	newID      func() string
}

// New constructs an Order service with production clock and ID generation.
func New(repository port.OrderRepository) *Service {
	return NewWithDependencies(repository, time.Now, newOrderID)
}

// NewWithDependencies constructs an Order service with testable time and ID sources.
func NewWithDependencies(repository port.OrderRepository, now func() time.Time, newID func() string) *Service {
	if repository == nil || now == nil || newID == nil {
		panic("order.Service: nil dependency")
	}
	return &Service{repository: repository, now: now, newID: newID}
}

// CreateInput contains the editable fields accepted on order creation.
type CreateInput struct {
	OrderNo      string
	CustomerName string
	AmountCents  int64
	Currency     string
	Remark       string
}

// UpdateInput contains the only fields mutable by PUT in this slice.
type UpdateInput struct {
	Version      int64
	CustomerName string
	AmountCents  int64
	Currency     string
	Remark       string
}

// List returns a paginated order list.
func (s *Service) List(ctx context.Context, filter port.OrderListFilter) ([]domain.Order, int, error) {
	if filter.Page < 1 || filter.PageSize < 1 || filter.PageSize > 100 {
		return nil, 0, port.ErrInvalidArgument
	}
	maxInt := int(^uint(0) >> 1)
	if filter.Page > 1 && filter.Page-1 > maxInt/filter.PageSize {
		return nil, 0, port.ErrInvalidArgument
	}
	if filter.Status != "" && !domain.IsKnownOrderStatus(filter.Status) {
		return nil, 0, port.ErrInvalidArgument
	}
	orders, total, err := s.repository.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("order list: %w", err)
	}
	return orders, total, nil
}

// Get returns one order.
func (s *Service) Get(ctx context.Context, id string) (domain.Order, error) {
	if strings.TrimSpace(id) == "" {
		return domain.Order{}, port.ErrInvalidArgument
	}
	order, err := s.repository.Get(ctx, id)
	if err != nil {
		return domain.Order{}, fmt.Errorf("order get: %w", err)
	}
	return order, nil
}

// Create creates a pending order with default currency and initial version.
func (s *Service) Create(ctx context.Context, input CreateInput) (domain.Order, error) {
	if err := validateWritable(input.CustomerName, input.AmountCents, input.Currency); err != nil {
		return domain.Order{}, err
	}
	if strings.TrimSpace(input.OrderNo) == "" {
		return domain.Order{}, port.ErrInvalidArgument
	}
	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if currency == "" {
		currency = "CNY"
	}
	now := s.now().UTC()
	order := domain.Order{
		ID:           s.newID(),
		OrderNo:      strings.TrimSpace(input.OrderNo),
		CustomerName: strings.TrimSpace(input.CustomerName),
		Status:       domain.OrderStatusPending,
		AmountCents:  input.AmountCents,
		Currency:     currency,
		Remark:       input.Remark,
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if order.ID == "" {
		return domain.Order{}, fmt.Errorf("order create: empty generated ID")
	}
	if err := s.repository.Create(ctx, order); err != nil {
		return domain.Order{}, fmt.Errorf("order create: %w", err)
	}
	return order, nil
}

// Update modifies the pending-only editable fields under optimistic locking.
func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (domain.Order, error) {
	if input.Version < 1 || strings.TrimSpace(id) == "" {
		return domain.Order{}, port.ErrInvalidArgument
	}
	if err := validateWritable(input.CustomerName, input.AmountCents, input.Currency); err != nil {
		return domain.Order{}, err
	}
	order, err := s.Get(ctx, id)
	if err != nil {
		return domain.Order{}, err
	}
	if order.Version != input.Version {
		return domain.Order{}, port.ErrVersionConflict
	}
	if order.Status != domain.OrderStatusPending {
		return domain.Order{}, port.ErrInvalidState
	}
	order.CustomerName = strings.TrimSpace(input.CustomerName)
	order.AmountCents = input.AmountCents
	order.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	order.Remark = input.Remark
	order.UpdatedAt = s.now().UTC()
	if err := s.repository.Update(ctx, order); err != nil {
		return domain.Order{}, fmt.Errorf("order update: %w", err)
	}
	order.Version++
	return order, nil
}

// MarkPaid changes a pending order to paid under optimistic locking.
func (s *Service) MarkPaid(ctx context.Context, id string, version int64) (domain.Order, error) {
	return s.changeStatus(ctx, id, version, domain.OrderStatusPaid)
}

// Cancel changes a pending order to cancelled under optimistic locking.
func (s *Service) Cancel(ctx context.Context, id string, version int64) (domain.Order, error) {
	return s.changeStatus(ctx, id, version, domain.OrderStatusCancelled)
}

func (s *Service) changeStatus(ctx context.Context, id string, version int64, target domain.OrderStatus) (domain.Order, error) {
	if strings.TrimSpace(id) == "" || version < 1 {
		return domain.Order{}, port.ErrInvalidArgument
	}
	order, err := s.Get(ctx, id)
	if err != nil {
		return domain.Order{}, err
	}
	if order.Version != version {
		return domain.Order{}, port.ErrVersionConflict
	}
	if order.Status != domain.OrderStatusPending {
		return domain.Order{}, port.ErrInvalidState
	}
	now := s.now().UTC()
	if err := s.repository.ChangeStatus(ctx, id, version, target, now); err != nil {
		return domain.Order{}, fmt.Errorf("order change status: %w", err)
	}
	order.Status = target
	order.Version++
	order.UpdatedAt = now
	return order, nil
}

// BatchDelete removes pending or cancelled orders atomically.
func (s *Service) BatchDelete(ctx context.Context, ids []string) (int, error) {
	if len(ids) == 0 || len(ids) > 100 {
		return 0, port.ErrInvalidArgument
	}
	seen := make(map[string]struct{}, len(ids))
	for i, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			return 0, port.ErrInvalidArgument
		}
		if _, exists := seen[id]; exists {
			return 0, port.ErrInvalidArgument
		}
		seen[id] = struct{}{}
		ids[i] = id
	}
	if err := s.repository.BatchDelete(ctx, ids); err != nil {
		return 0, fmt.Errorf("order batch delete: %w", err)
	}
	return len(ids), nil
}

func validateWritable(customerName string, amountCents int64, currency string) error {
	if strings.TrimSpace(customerName) == "" || amountCents < 0 {
		return port.ErrInvalidArgument
	}
	currency = strings.TrimSpace(currency)
	if currency != "" && len(currency) != 3 {
		return port.ErrInvalidArgument
	}
	return nil
}

func newOrderID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return ""
	}
	return "ord_" + hex.EncodeToString(bytes[:])
}
