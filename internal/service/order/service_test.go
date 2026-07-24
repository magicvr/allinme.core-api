package order_test

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/domain"
	"github.com/magicvr/allinme.core-api/internal/port"
	orderservice "github.com/magicvr/allinme.core-api/internal/service/order"
)

func TestServiceEnforcesStateAndVersionRules(t *testing.T) {
	repository := newFakeRepository()
	now := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	service := orderservice.NewWithDependencies(repository, func() time.Time { return now }, func() string { return "ord_test" })

	created, err := service.Create(context.Background(), orderservice.CreateInput{
		OrderNo: "ORD-TEST", CustomerName: "Alice", AmountCents: 1234,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != domain.OrderStatusPending || created.Currency != "CNY" || created.Version != 1 {
		t.Fatalf("created order = %+v", created)
	}

	updated, err := service.Update(context.Background(), created.ID, orderservice.UpdateInput{
		Version: 1, CustomerName: "Alice Updated", AmountCents: 2345, Currency: "usd", Remark: "changed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 2 || updated.Currency != "USD" || updated.CustomerName != "Alice Updated" {
		t.Fatalf("updated order = %+v", updated)
	}
	if _, err := service.Update(context.Background(), created.ID, orderservice.UpdateInput{
		Version: 1, CustomerName: "stale", AmountCents: 1, Currency: "CNY",
	}); !errors.Is(err, port.ErrVersionConflict) {
		t.Fatalf("stale update error = %v, want version conflict", err)
	}

	paid, err := service.MarkPaid(context.Background(), created.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if paid.Status != domain.OrderStatusPaid || paid.Version != 3 {
		t.Fatalf("paid order = %+v", paid)
	}
	if _, err := service.Cancel(context.Background(), created.ID, 3); !errors.Is(err, port.ErrInvalidState) {
		t.Fatalf("cancel paid error = %v, want invalid state", err)
	}
}

func TestServiceBatchDeleteIsAtomic(t *testing.T) {
	repository := newFakeRepository()
	repository.orders["pending"] = domain.Order{ID: "pending", Status: domain.OrderStatusPending, Version: 1}
	repository.orders["paid"] = domain.Order{ID: "paid", Status: domain.OrderStatusPaid, Version: 1}
	service := orderservice.NewWithDependencies(repository, time.Now, func() string { return "unused" })

	if _, err := service.BatchDelete(context.Background(), []string{"pending", "paid"}); !errors.Is(err, port.ErrInvalidState) {
		t.Fatalf("batch delete error = %v, want invalid state", err)
	}
	if _, ok := repository.orders["pending"]; !ok {
		t.Fatal("pending order was deleted despite batch rollback")
	}
	if _, err := service.BatchDelete(context.Background(), []string{"pending", "pending"}); !errors.Is(err, port.ErrInvalidArgument) {
		t.Fatalf("duplicate IDs error = %v, want invalid argument", err)
	}
}

type fakeRepository struct {
	orders map[string]domain.Order
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{orders: make(map[string]domain.Order)}
}

func (r *fakeRepository) Create(_ context.Context, order domain.Order) error {
	for _, existing := range r.orders {
		if existing.OrderNo == order.OrderNo {
			return port.ErrOrderNoConflict
		}
	}
	r.orders[order.ID] = order
	return nil
}

func (r *fakeRepository) Get(_ context.Context, id string) (domain.Order, error) {
	order, ok := r.orders[id]
	if !ok {
		return domain.Order{}, port.ErrOrderNotFound
	}
	return order, nil
}

func (r *fakeRepository) List(_ context.Context, filter port.OrderListFilter) ([]domain.Order, int, error) {
	orders := make([]domain.Order, 0, len(r.orders))
	for _, order := range r.orders {
		if filter.Status == "" || order.Status == filter.Status {
			orders = append(orders, order)
		}
	}
	sort.Slice(orders, func(i, j int) bool { return orders[i].ID < orders[j].ID })
	return orders, len(orders), nil
}

func (r *fakeRepository) Update(_ context.Context, order domain.Order) error {
	stored, ok := r.orders[order.ID]
	if !ok || stored.Version != order.Version || stored.Status != domain.OrderStatusPending {
		return port.ErrVersionConflict
	}
	order.Version++
	r.orders[order.ID] = order
	return nil
}

func (r *fakeRepository) ChangeStatus(_ context.Context, id string, version int64, status domain.OrderStatus, _ time.Time) error {
	order, ok := r.orders[id]
	if !ok || order.Version != version || order.Status != domain.OrderStatusPending {
		return port.ErrVersionConflict
	}
	order.Status = status
	order.Version++
	r.orders[id] = order
	return nil
}

func (r *fakeRepository) BatchDelete(_ context.Context, ids []string) error {
	for _, id := range ids {
		order, ok := r.orders[id]
		if !ok {
			return port.ErrOrderNotFound
		}
		if order.Status != domain.OrderStatusPending && order.Status != domain.OrderStatusCancelled {
			return port.ErrInvalidState
		}
	}
	for _, id := range ids {
		delete(r.orders, id)
	}
	return nil
}

func (r *fakeRepository) Count(_ context.Context) (int, error) { return len(r.orders), nil }

var _ port.OrderRepository = (*fakeRepository)(nil)
