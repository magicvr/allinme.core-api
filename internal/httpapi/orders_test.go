package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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
	for _, test := range []struct {
		method, path, allow string
		status              int
	}{{http.MethodPost, "/api/v1/orders", "", http.StatusUnauthorized}, {http.MethodPatch, "/api/v1/orders", "GET, POST", http.StatusMethodNotAllowed}, {http.MethodPost, "/api/v1/orders/ord_00000000000000000000000000000001", "GET, PATCH", http.StatusMethodNotAllowed}} {
		request = httptest.NewRequest(test.method, test.path, nil)
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != test.status || response.Header().Get("Allow") != test.allow {
			t.Fatalf("%s %s = %d %s", test.method, test.path, response.Code, response.Body.String())
		}
	}
}

func TestOrderWriteRoutesStrictInputAndErrorMapping(t *testing.T) {
	service := &fakeOrderService{result: testOrder()}
	authService := &fakeAuthService{authenticatedRole: auth.RoleOperator}
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: authService, Orders: service})
	request := func(method, path, body, key string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer access-token")
		req.Header.Set("Content-Type", "application/json")
		if key != "" {
			req.Header.Set("Idempotency-Key", key)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}
	valid := `{"customerName":" Alice ","currency":"CNY","items":[{"sku":" SKU ","name":" Item ","quantity":2,"unitPrice":100}]}`
	response := request(http.MethodPost, "/api/v1/orders", valid, "create-1")
	if response.Code != http.StatusCreated || service.createKey != "create-1" || service.createCommand.Items[0].Quantity != 2 {
		t.Fatalf("create = %d %s key=%q command=%+v", response.Code, response.Body.String(), service.createKey, service.createCommand)
	}
	if response := request(http.MethodPost, "/api/v1/orders", valid, ""); response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"field":"Idempotency-Key"`) {
		t.Fatalf("missing key = %d %s", response.Code, response.Body.String())
	}
	missingTypeAndKey := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(valid))
	missingTypeAndKey.Header.Set("Authorization", "Bearer access-token")
	missingTypeAndKeyResponse := httptest.NewRecorder()
	handler.ServeHTTP(missingTypeAndKeyResponse, missingTypeAndKey)
	if missingTypeAndKeyResponse.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("missing content type and key = %d %s", missingTypeAndKeyResponse.Code, missingTypeAndKeyResponse.Body.String())
	}
	if response := request(http.MethodPost, "/api/v1/orders", strings.Replace(valid, `2`, `2.0`, 1), "create-2"); response.Code != http.StatusBadRequest {
		t.Fatalf("decimal quantity = %d %s", response.Code, response.Body.String())
	}
	unsupported := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(valid))
	unsupported.Header.Set("Authorization", "Bearer access-token")
	unsupported.Header.Set("Idempotency-Key", "create-3")
	unsupportedResponse := httptest.NewRecorder()
	handler.ServeHTTP(unsupportedResponse, unsupported)
	if unsupportedResponse.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("unsupported = %d %s", unsupportedResponse.Code, unsupportedResponse.Body.String())
	}
	invalidIDType := httptest.NewRequest(http.MethodPatch, "/api/v1/orders/not-an-order", strings.NewReader(`{`))
	invalidIDType.Header.Set("Authorization", "Bearer access-token")
	invalidIDType.Header.Set("Content-Type", "text/plain")
	invalidIDTypeResponse := httptest.NewRecorder()
	handler.ServeHTTP(invalidIDTypeResponse, invalidIDType)
	if invalidIDTypeResponse.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("invalid edit ID and content type = %d %s", invalidIDTypeResponse.Code, invalidIDTypeResponse.Body.String())
	}
	service.err = nil
	for _, size := range []struct {
		name   string
		bytes  int
		status int
	}{{"at limit", 64 * 1024, http.StatusCreated}, {"over limit", 64*1024 + 1, http.StatusBadRequest}} {
		body := valid + strings.Repeat(" ", size.bytes-len(valid))
		if response := request(http.MethodPost, "/api/v1/orders", body, "body-"+strings.ReplaceAll(size.name, " ", "-")); response.Code != size.status {
			t.Fatalf("body %s = %d %s", size.name, response.Code, response.Body.String())
		}
	}
	validEdit := `{"customerName":"Alice","currency":"CNY","items":[{"sku":"SKU","name":"Item","quantity":1,"unitPrice":100}],"version":1}`
	for _, size := range []struct {
		name   string
		bytes  int
		status int
	}{{"at limit", 64 * 1024, http.StatusOK}, {"over limit", 64*1024 + 1, http.StatusBadRequest}} {
		body := validEdit + strings.Repeat(" ", size.bytes-len(validEdit))
		if response := request(http.MethodPatch, "/api/v1/orders/ord_00000000000000000000000000000001", body, ""); response.Code != size.status {
			t.Fatalf("edit body %s = %d %s", size.name, response.Code, response.Body.String())
		}
	}
	authService.authenticatedRole = auth.RoleViewer
	if response := request(http.MethodPost, "/api/v1/orders", `{`, "bad"); response.Code != http.StatusForbidden {
		t.Fatalf("viewer = %d %s", response.Code, response.Body.String())
	}
	authService.authenticatedRole = auth.RoleOperator
	service.err = &order.ValidationError{Details: []order.FieldError{{Field: "items[0].quantity", Message: "invalid"}}}
	if response := request(http.MethodPost, "/api/v1/orders", valid, "create-4"); response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), `"code":"VALIDATION_FAILED"`) {
		t.Fatalf("validation = %d %s", response.Code, response.Body.String())
	}
	for _, test := range []struct {
		err    error
		status int
		code   string
	}{{order.ErrIdempotencyConflict, 409, "IDEMPOTENCY_CONFLICT"}, {order.ErrVersionConflict, 409, "VERSION_CONFLICT"}, {order.ErrStateConflict, 409, "STATE_CONFLICT"}, {order.ErrUnavailable, 503, "SERVICE_UNAVAILABLE"}} {
		service.err = test.err
		method, path, body, key := http.MethodPost, "/api/v1/orders", valid, "mapping-key"
		if errors.Is(test.err, order.ErrVersionConflict) || errors.Is(test.err, order.ErrStateConflict) {
			method, path, body, key = http.MethodPatch, "/api/v1/orders/ord_00000000000000000000000000000001", `{"customerName":"A","currency":"CNY","items":[{"sku":"S","name":"N","quantity":1,"unitPrice":1}],"version":1}`, "ignored-invalid key"
		}
		response := request(method, path, body, key)
		if response.Code != test.status || !strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
			t.Fatalf("%s = %d %s", test.code, response.Code, response.Body.String())
		}
		if test.status == 503 && response.Header().Get("Retry-After") != "1" {
			t.Fatalf("Retry-After = %q", response.Header().Get("Retry-After"))
		}
	}
}

func TestOrderActionRoutesStrictInputAuthorizationAndConflicts(t *testing.T) {
	service := &fakeOrderService{result: testOrder()}
	authService := &fakeAuthService{authenticatedRole: auth.RoleOperator}
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: authService, Orders: service, OrderActions: true})
	request := func(method, path, contentType, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer access-token")
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}
	for path, action := range map[string]order.Action{"confirm": order.ActionConfirm, "fulfill": order.ActionFulfill, "ship": order.ActionShip, "complete": order.ActionComplete, "cancel": order.ActionCancel} {
		response := request(http.MethodPost, "/api/v1/orders/ord_00000000000000000000000000000001/"+path, "application/json; charset=utf-8", `{"version":7}`)
		if response.Code != http.StatusOK || service.action != action || service.transitionCommand.Version != 7 {
			t.Fatalf("%s = %d %s action=%q command=%+v", path, response.Code, response.Body.String(), service.action, service.transitionCommand)
		}
	}
	invalid := []struct {
		name, contentType, body string
		status                  int
	}{
		{"missing content type", "", `{"version":1}`, http.StatusUnsupportedMediaType},
		{"unknown field", "application/json", `{"version":1,"status":"CONFIRMED"}`, http.StatusBadRequest},
		{"missing version", "application/json", `{}`, http.StatusBadRequest},
		{"string version", "application/json", `{"version":"1"}`, http.StatusBadRequest},
		{"decimal version", "application/json", `{"version":1.0}`, http.StatusBadRequest},
		{"extra value", "application/json", `{"version":1}{}`, http.StatusBadRequest},
		{"body too large", "application/json", strings.Repeat(" ", 1025) + `{"version":1}`, http.StatusBadRequest},
	}
	for _, test := range invalid {
		if response := request(http.MethodPost, "/api/v1/orders/ord_00000000000000000000000000000001/confirm", test.contentType, test.body); response.Code != test.status {
			t.Fatalf("%s = %d %s", test.name, response.Code, response.Body.String())
		}
	}
	service.err = &order.ValidationError{Details: []order.FieldError{{Field: "version", Message: "must be greater than 0"}}}
	if response := request(http.MethodPost, "/api/v1/orders/ord_00000000000000000000000000000001/confirm", "application/json", `{"version":0}`); response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("zero version = %d %s", response.Code, response.Body.String())
	}
	for _, test := range []struct {
		err    error
		status int
		code   string
	}{{order.ErrNotFound, 404, "NOT_FOUND"}, {order.ErrVersionConflict, 409, "VERSION_CONFLICT"}, {order.ErrStateConflict, 409, "STATE_CONFLICT"}, {order.ErrUnavailable, 503, "SERVICE_UNAVAILABLE"}} {
		service.err = test.err
		response := request(http.MethodPost, "/api/v1/orders/ord_00000000000000000000000000000001/cancel", "application/json", `{"version":1}`)
		if response.Code != test.status || !strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
			t.Fatalf("%s = %d %s", test.code, response.Code, response.Body.String())
		}
		if test.status == http.StatusServiceUnavailable && response.Header().Get("Retry-After") != "1" {
			t.Fatalf("action Retry-After = %q", response.Header().Get("Retry-After"))
		}
	}
	service.err = nil
	authService.authenticatedRole = auth.RoleViewer
	if response := request(http.MethodPost, "/api/v1/orders/ord_00000000000000000000000000000001/confirm", "text/plain", `{`); response.Code != http.StatusForbidden {
		t.Fatalf("viewer action = %d %s", response.Code, response.Body.String())
	}
	authService.authenticatedRole = auth.RoleOperator
	if response := request(http.MethodPost, "/api/v1/orders/not-an-order/confirm", "application/json", `{`); response.Code != http.StatusNotFound {
		t.Fatalf("invalid ID = %d %s", response.Code, response.Body.String())
	}
	if response := request(http.MethodPost, "/api/v1/orders/not-an-order/confirm", "text/plain", `{`); response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("invalid action ID and content type = %d %s", response.Code, response.Body.String())
	}
	if response := request(http.MethodGet, "/api/v1/orders/ord_00000000000000000000000000000001/confirm", "application/json", `{"version":1}`); response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("action GET = %d Allow=%q %s", response.Code, response.Header().Get("Allow"), response.Body.String())
	}
}

type fakeOrderService struct {
	page              order.Page
	result            order.Order
	query             order.ListQuery
	err               error
	createKey         string
	createCommand     order.CreateCommand
	editCommand       order.EditCommand
	action            order.Action
	transitionCommand order.TransitionCommand
}

func (service *fakeOrderService) List(_ context.Context, _ auth.Principal, query order.ListQuery) (order.Page, error) {
	service.query = query
	return service.page, service.err
}
func (service *fakeOrderService) Get(context.Context, auth.Principal, string) (order.Order, error) {
	return service.result, service.err
}

func (service *fakeOrderService) Create(_ context.Context, _ auth.Principal, key string, command order.CreateCommand) (order.Order, error) {
	service.createKey, service.createCommand = key, command
	return service.result, service.err
}
func (service *fakeOrderService) Edit(_ context.Context, _ auth.Principal, _ string, command order.EditCommand) (order.Order, error) {
	service.editCommand = command
	return service.result, service.err
}
func (service *fakeOrderService) Transition(_ context.Context, _ auth.Principal, _ string, action order.Action, command order.TransitionCommand) (order.Order, error) {
	service.action, service.transitionCommand = action, command
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
