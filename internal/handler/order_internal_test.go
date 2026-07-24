package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/magicvr/allinme.core-api/internal/domain"
	"github.com/magicvr/allinme.core-api/internal/port"
	orderservice "github.com/magicvr/allinme.core-api/internal/service/order"
)

func TestOrderInternalErrorDoesNotLeak(t *testing.T) {
	const sensitive = "sqlite: disk I/O error at C:\\secret\\orders.db"
	handler := listOrders(failingOrderService{err: errors.New(sensitive)})
	req := httptest.NewRequest(http.MethodGet, "/v1/orders?page=1&pageSize=20", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusInternalServerError, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"code":"internal"`) {
		t.Fatalf("missing internal error code: %s", rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), sensitive) || strings.Contains(rr.Body.String(), `C:\\secret`) {
		t.Fatalf("response leaked internal error: %s", rr.Body.String())
	}
}

type failingOrderService struct {
	err error
}

func (s failingOrderService) List(context.Context, port.OrderListFilter) ([]domain.Order, int, error) {
	return nil, 0, s.err
}

func (s failingOrderService) Get(context.Context, string) (domain.Order, error) {
	return domain.Order{}, s.err
}

func (s failingOrderService) Create(context.Context, orderservice.CreateInput) (domain.Order, error) {
	return domain.Order{}, s.err
}

func (s failingOrderService) Update(context.Context, string, orderservice.UpdateInput) (domain.Order, error) {
	return domain.Order{}, s.err
}

func (s failingOrderService) MarkPaid(context.Context, string, int64) (domain.Order, error) {
	return domain.Order{}, s.err
}

func (s failingOrderService) Cancel(context.Context, string, int64) (domain.Order, error) {
	return domain.Order{}, s.err
}

func (s failingOrderService) BatchDelete(context.Context, []string) (int, error) {
	return 0, s.err
}

var _ orderService = failingOrderService{}
