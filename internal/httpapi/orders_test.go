package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/httpapi"
	"github.com/magicvr/allinme.core-api/internal/order"
)

func TestOrderRoutesQueryDetailAndErrors(t *testing.T) {
	service := &fakeOrderService{page: order.Page{Items: []order.Order{testOrder()}, Total: 1, Page: 2, PageSize: 5}, result: testOrder()}
	authService := &fakeAuthService{}
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: authService, Orders: service})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/orders?q=+demo+&page=2&pageSize=5&sort=updatedAt&order=asc&status=DRAFT&paymentStatus=UNPAID&createdFrom=2026-01-01T08:00:00%2B08:00&createdTo=2026-01-02T00:00:00Z", nil)
	request.Header.Set("Authorization", "Bearer access-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("list = %d %s", response.Code, response.Body.String())
	}
	if service.query.Keyword != "demo" || service.query.Page != 2 || service.query.PageSize != 5 || service.query.Sort != "updatedAt" || service.query.Descending || service.query.CreatedFrom.Format(time.RFC3339) != "2026-01-01T00:00:00Z" {
		t.Fatalf("query = %+v", service.query)
	}
	var page map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	items := page["items"].([]any)
	item := items[0].(map[string]any)
	if _, ok := item["items"]; ok {
		t.Fatal("list item exposes items")
	}
	if item["canEdit"] != false || item["canRequestRefund"] != false {
		t.Fatalf("viewer capabilities = %+v", item)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/orders/ord_00000000000000000000000000000001", nil)
	authService.authenticatedRole = auth.RoleOperator
	request.Header.Set("Authorization", "Bearer access-token")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !contains(response.Body.String(), `"canEdit":true`) || !contains(response.Body.String(), `"items":[`) {
		t.Fatalf("detail = %d %s", response.Code, response.Body.String())
	}

	for _, path := range []string{"/api/v1/orders?page=1&page=2", "/api/v1/orders?sort=invalid", "/api/v1/orders?createdFrom=2026-01-02T00:00:00Z&createdTo=2026-01-01T00:00:00Z"} {
		request = httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("Authorization", "Bearer access-token")
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s = %d %s", path, response.Code, response.Body.String())
		}
	}
	request = httptest.NewRequest(http.MethodGet, "/api/v1/orders?page=9223372036854775807&pageSize=1", nil)
	request.Header.Set("Authorization", "Bearer access-token")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("maximum page = %d %s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodGet, "/api/v1/orders?page=9223372036854775807&pageSize=2", nil)
	request.Header.Set("Authorization", "Bearer access-token")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || !contains(response.Body.String(), `"details":[{"field":"page","message":"page is too large"}]`) {
		t.Fatalf("overflow page = %d %s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodGet, "/api/v1/orders/not-an-order", nil)
	request.Header.Set("Authorization", "Bearer access-token")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("invalid id = %d %s", response.Code, response.Body.String())
	}
	for _, method := range []string{http.MethodPost, http.MethodPatch} {
		request = httptest.NewRequest(method, "/api/v1/orders", nil)
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("disabled %s = %d %s", method, response.Code, response.Body.String())
		}
	}
}

type fakeOrderService struct {
	page   order.Page
	result order.Order
	query  order.ListQuery
	err    error
}

func (service *fakeOrderService) List(_ context.Context, _ auth.Principal, query order.ListQuery) (order.Page, error) {
	service.query = query
	return service.page, service.err
}
func (service *fakeOrderService) Get(context.Context, auth.Principal, string) (order.Order, error) {
	return service.result, service.err
}

func testOrder() order.Order {
	timestamp := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return order.Order{ID: "ord_00000000000000000000000000000001", CustomerName: "Demo", Status: order.StatusDraft, PaymentStatus: order.PaymentStatusUnpaid, Currency: "CNY", TotalAmount: 100, Version: 1, CreatedAt: timestamp, UpdatedAt: timestamp, Items: []order.Item{{ID: "itm_00000000000000000000000000000001", SKU: "SKU", Name: "Item", Quantity: 1, UnitPrice: 100}}}
}
func contains(value, fragment string) bool {
	for index := 0; index+len(fragment) <= len(value); index++ {
		if value[index:index+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
