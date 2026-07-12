package order_test

import (
	"context"
	"errors"
	"testing"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/order"
)

func TestServiceReadAuthorizationAndErrors(t *testing.T) {
	repository := &serviceRepository{page: order.Page{Total: 1}, found: true, result: order.Order{ID: "ord_00000000000000000000000000000001"}}
	service, err := order.NewService(repository)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.List(context.Background(), auth.Principal{Role: auth.Role("unknown")}, order.ListQuery{}); !errors.Is(err, order.ErrForbidden) {
		t.Fatalf("List() error = %v", err)
	}
	if _, err := service.Get(context.Background(), auth.Principal{Role: auth.RoleViewer}, "missing"); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	repository.found = false
	if _, err := service.Get(context.Background(), auth.Principal{Role: auth.RoleViewer}, "missing"); !errors.Is(err, order.ErrNotFound) {
		t.Fatalf("Get() missing error = %v", err)
	}
	repository.err = context.Canceled
	if _, err := service.List(context.Background(), auth.Principal{Role: auth.RoleViewer}, order.ListQuery{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("List() cancel error = %v", err)
	}
}

type serviceRepository struct {
	page   order.Page
	result order.Order
	found  bool
	err    error
}

func (repository *serviceRepository) ListOrders(context.Context, order.ListQuery) (order.Page, error) {
	return repository.page, repository.err
}
func (repository *serviceRepository) GetOrder(context.Context, string) (order.Order, bool, error) {
	return repository.result, repository.found, repository.err
}
