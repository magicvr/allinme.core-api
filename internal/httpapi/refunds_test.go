package httpapi_test

import (
	"bytes"
	"context"
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

func TestRefundListRouteQueryDTOAndAuthorization(t *testing.T) {
	pending := testRefund(order.RefundStatusPending)
	completed := testRefund(order.RefundStatusCompleted)
	service := &fakeRefundService{page: order.RefundPage{Items: []order.Refund{pending, completed}, Total: 2, Page: 2, PageSize: 5}}
	authService := &fakeAuthService{authenticatedRole: auth.RoleApprover}
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: authService, Refunds: service})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/refunds?status=PENDING&orderId=ord_00000000000000000000000000000008&page=2&pageSize=5", nil)
	request.Header.Set("Authorization", "Bearer access-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || service.query.Status != order.RefundStatusPending || service.query.OrderID != "ord_00000000000000000000000000000008" || service.query.Page != 2 || service.query.PageSize != 5 {
		t.Fatalf("list = %d %s query=%+v", response.Code, response.Body.String(), service.query)
	}
	if !strings.Contains(response.Body.String(), `"requestedBy":{"id":"user-operator","username":"operator"}`) || !strings.Contains(response.Body.String(), `"decidedBy":null`) || !strings.Contains(response.Body.String(), `"decidedAt":null`) || !strings.Contains(response.Body.String(), `"canApprove":true`) || !strings.Contains(response.Body.String(), `"canReject":true`) {
		t.Fatalf("list DTO = %s", response.Body.String())
	}
	for _, path := range []string{
		"/api/v1/refunds?sort=createdAt",
		"/api/v1/refunds?status=",
		"/api/v1/refunds?status=PAID",
		"/api/v1/refunds?orderId=bad",
		"/api/v1/refunds?page=01",
		"/api/v1/refunds?pageSize=101",
		"/api/v1/refunds?page=1&page=2",
		"/api/v1/refunds?page=9223372036854775807&pageSize=2",
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("Authorization", "Bearer access-token")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"INVALID_REQUEST"`) {
			t.Fatalf("%s = %d %s", path, response.Code, response.Body.String())
		}
	}
	authService.authenticatedRole = auth.RoleOperator
	request = httptest.NewRequest(http.MethodGet, "/api/v1/refunds", nil)
	request.Header.Set("Authorization", "Bearer access-token")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("operator list = %d %s", response.Code, response.Body.String())
	}
}

func TestRefundCreateRouteStrictInputAndErrorMapping(t *testing.T) {
	service := &fakeRefundService{result: order.RefundResult{Refund: testRefund(order.RefundStatusPending)}}
	authService := &fakeAuthService{authenticatedRole: auth.RoleOperator}
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: authService, Refunds: service})
	request := func(body []byte, contentType, key, orderID string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+orderID+"/refunds", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer access-token")
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		if key != "" {
			req.Header.Set("Idempotency-Key", key)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}
	valid := []byte(`{"amount":100,"reason":" customer request ","orderVersion":3}`)
	response := request(valid, "application/json", "refund-key", "ord_00000000000000000000000000000008")
	if response.Code != http.StatusCreated || service.createKey != "refund-key" || service.createCommand.Amount != 100 || service.createCommand.OrderVersion != 3 {
		t.Fatalf("create = %d %s key=%q command=%+v", response.Code, response.Body.String(), service.createKey, service.createCommand)
	}
	invalidUTF8 := append([]byte(`{"amount":100,"reason":"`), 0xff)
	invalidUTF8 = append(invalidUTF8, []byte(`","orderVersion":3}`)...)
	beforeCalls := service.createCalls
	if response := request(invalidUTF8, "application/json", "", "bad"); response.Code != http.StatusBadRequest || service.createCalls != beforeCalls {
		t.Fatalf("invalid UTF-8 = %d %s calls=%d", response.Code, response.Body.String(), service.createCalls)
	}
	if response := request(valid, "", "refund-key", "bad"); response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("content type before path = %d %s", response.Code, response.Body.String())
	}
	if response := request(valid, "application/json", "", "bad"); response.Code != http.StatusBadRequest {
		t.Fatalf("key before path = %d %s", response.Code, response.Body.String())
	}
	if response := request(valid, "application/json", "refund-key", "bad"); response.Code != http.StatusNotFound {
		t.Fatalf("path before JSON = %d %s", response.Code, response.Body.String())
	}
	for _, body := range [][]byte{
		[]byte(`{"amount":1.0,"reason":"request","orderVersion":3}`),
		[]byte(`{"amount":1,"reason":"request","orderVersion":3,"extra":true}`),
		[]byte(`{"Amount":1,"Reason":"request","OrderVersion":3}`),
		[]byte(`{"amount":1,"amount":2,"reason":"request","orderVersion":3}`),
		[]byte(`{"amount":1,"reason":null,"orderVersion":3}`),
		[]byte(`{"amount":1,"reason":"request"}`),
		[]byte(`{"amount":1,"reason":"request","orderVersion":3}{}`),
		bytes.Repeat([]byte(" "), 8*1024+1),
	} {
		if response := request(body, "application/json", "refund-key", "ord_00000000000000000000000000000008"); response.Code != http.StatusBadRequest {
			t.Fatalf("invalid body = %d %s", response.Code, response.Body.String())
		}
	}
	authService.authenticatedRole = auth.RoleViewer
	if response := request([]byte(`{`), "text/plain", "", "bad"); response.Code != http.StatusForbidden {
		t.Fatalf("role before content type = %d %s", response.Code, response.Body.String())
	}
	authService.authenticatedRole = auth.RoleOperator
	for _, test := range []struct {
		err    error
		status int
		code   string
	}{
		{&order.ValidationError{Details: []order.FieldError{{Field: "amount", Message: "must be between 1 and 9999999999"}}}, 422, "VALIDATION_FAILED"},
		{order.ErrNotFound, 404, "NOT_FOUND"},
		{order.ErrIdempotencyConflict, 409, "IDEMPOTENCY_CONFLICT"},
		{order.ErrUnavailable, 503, "SERVICE_UNAVAILABLE"},
	} {
		service.err = test.err
		response := request(valid, "application/json", "mapping-key", "ord_00000000000000000000000000000008")
		if response.Code != test.status || !strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
			t.Fatalf("%s = %d %s", test.code, response.Code, response.Body.String())
		}
		if test.status == http.StatusServiceUnavailable && response.Header().Get("Retry-After") != "1" {
			t.Fatalf("Retry-After = %q", response.Header().Get("Retry-After"))
		}
	}
}

func TestRefundDecisionRoutesStrictInputAndConflicts(t *testing.T) {
	service := &fakeRefundService{result: order.RefundResult{Refund: testRefund(order.RefundStatusCompleted)}}
	authService := &fakeAuthService{authenticatedRole: auth.RoleApprover}
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: authService, Refunds: service})
	request := func(action string, body []byte, contentType, refundID string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/refunds/"+refundID+"/"+action, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer access-token")
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}
	valid := []byte(`{"version":1}`)
	if response := request("approve", valid, "application/json", "rfd_00000000000000000000000000000001"); response.Code != http.StatusOK || service.approveVersion != 1 {
		t.Fatalf("approve = %d %s", response.Code, response.Body.String())
	}
	if response := request("reject", valid, "application/json", "rfd_00000000000000000000000000000001"); response.Code != http.StatusOK || service.rejectVersion != 1 {
		t.Fatalf("reject = %d %s", response.Code, response.Body.String())
	}
	invalidUTF8 := []byte{'{', '"', 'v', 'e', 'r', 's', 'i', 'o', 'n', '"', ':', '"', 0xff, '"', '}'}
	beforeCalls := service.approveCalls
	if response := request("approve", invalidUTF8, "application/json", "bad"); response.Code != http.StatusBadRequest || service.approveCalls != beforeCalls {
		t.Fatalf("approve invalid UTF-8 = %d %s", response.Code, response.Body.String())
	}
	if response := request("reject", invalidUTF8, "application/json", "bad"); response.Code != http.StatusBadRequest {
		t.Fatalf("reject invalid UTF-8 = %d %s", response.Code, response.Body.String())
	}
	if response := request("approve", valid, "", "bad"); response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("content type before ID = %d %s", response.Code, response.Body.String())
	}
	if response := request("approve", valid, "application/json", "bad"); response.Code != http.StatusNotFound {
		t.Fatalf("ID before JSON = %d %s", response.Code, response.Body.String())
	}
	for _, body := range [][]byte{
		[]byte(`{"version":1.0}`),
		[]byte(`{"version":"1"}`),
		[]byte(`{"version":1,"extra":true}`),
		[]byte(`{"Version":1}`),
		[]byte(`{"version":1,"version":2}`),
		[]byte(`{}`),
		[]byte(`{"version":1}{}`),
	} {
		if response := request("approve", body, "application/json", "rfd_00000000000000000000000000000001"); response.Code != http.StatusBadRequest {
			t.Fatalf("invalid version body = %d %s", response.Code, response.Body.String())
		}
	}
	for _, test := range []struct {
		err    error
		status int
		code   string
	}{
		{order.ErrVersionConflict, 409, "VERSION_CONFLICT"},
		{order.ErrStateConflict, 409, "STATE_CONFLICT"},
		{order.ErrForbidden, 403, "FORBIDDEN"},
		{order.ErrUnavailable, 503, "SERVICE_UNAVAILABLE"},
	} {
		service.err = test.err
		response := request("approve", valid, "application/json", "rfd_00000000000000000000000000000001")
		if response.Code != test.status || !strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
			t.Fatalf("%s = %d %s", test.code, response.Code, response.Body.String())
		}
	}
	service.err = nil
	authService.authenticatedRole = auth.RoleOperator
	if response := request("approve", []byte(`{`), "text/plain", "bad"); response.Code != http.StatusForbidden {
		t.Fatalf("role before body = %d %s", response.Code, response.Body.String())
	}
}

func TestRefundRouteMethodsAndDisableSwitch(t *testing.T) {
	authService := &fakeAuthService{authenticatedRole: auth.RoleAdmin}
	service := &fakeRefundService{}
	for _, test := range []struct {
		method string
		path   string
		allow  string
	}{
		{http.MethodHead, "/api/v1/refunds", http.MethodGet},
		{http.MethodOptions, "/api/v1/refunds", http.MethodGet},
		{http.MethodGet, "/api/v1/orders/ord_00000000000000000000000000000008/refunds", http.MethodPost},
		{http.MethodHead, "/api/v1/refunds/rfd_00000000000000000000000000000001/approve", http.MethodPost},
	} {
		handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: authService, Refunds: service})
		request := httptest.NewRequest(test.method, test.path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != test.allow {
			t.Fatalf("%s %s = %d Allow=%q", test.method, test.path, response.Code, response.Header().Get("Allow"))
		}
	}
	disabled := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: authService, Refunds: service, DisableRefundRoutes: true})
	for _, path := range []string{"/api/v1/refunds", "/api/v1/orders/ord_00000000000000000000000000000008/refunds", "/api/v1/refunds/rfd_00000000000000000000000000000001/approve", "/api/v1/refunds/rfd_00000000000000000000000000000001/reject"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		disabled.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("disabled %s = %d", path, response.Code)
		}
	}
}

func TestRefundCORSMetadataMatchesEnabledRoutes(t *testing.T) {
	authService := &fakeAuthService{authenticatedRole: auth.RoleAdmin}
	service := &fakeRefundService{}
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: authService, Refunds: service, CORSAllowedOrigin: testAllowedOrigin})
	for _, test := range []struct {
		path   string
		method string
	}{
		{"/api/v1/refunds", http.MethodGet},
		{"/api/v1/orders/ord_00000000000000000000000000000008/refunds", http.MethodPost},
		{"/api/v1/refunds/rfd_00000000000000000000000000000001/approve", http.MethodPost},
		{"/api/v1/refunds/rfd_00000000000000000000000000000001/reject", http.MethodPost},
	} {
		request := corsRequest(http.MethodOptions, test.path, testAllowedOrigin)
		request.Header.Set("Access-Control-Request-Method", test.method)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent || response.Header().Get("Access-Control-Allow-Origin") != testAllowedOrigin {
			t.Fatalf("preflight %s = %d %v %s", test.path, response.Code, response.Header(), response.Body.String())
		}
	}
	request := corsRequest(http.MethodOptions, "/api/v1/refunds", testAllowedOrigin)
	request.Header.Set("Access-Control-Request-Method", http.MethodHead)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), `"code":"CORS_PREFLIGHT_DENIED"`) {
		t.Fatalf("HEAD preflight = %d %s", response.Code, response.Body.String())
	}
	disabled := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: authService, Refunds: service, DisableRefundRoutes: true, CORSAllowedOrigin: testAllowedOrigin})
	request = corsRequest(http.MethodOptions, "/api/v1/refunds", testAllowedOrigin)
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	response = httptest.NewRecorder()
	disabled.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("disabled preflight = %d %s", response.Code, response.Body.String())
	}
}

type fakeRefundService struct {
	page           order.RefundPage
	result         order.RefundResult
	err            error
	query          order.RefundListQuery
	createKey      string
	createOrderID  string
	createCommand  order.RefundRequestCommand
	createCalls    int
	approveVersion int64
	approveCalls   int
	rejectVersion  int64
	rejectCalls    int
}

func (service *fakeRefundService) List(_ context.Context, _ auth.Principal, query order.RefundListQuery) (order.RefundPage, error) {
	service.query = query
	return service.page, service.err
}

func (service *fakeRefundService) Create(_ context.Context, _ auth.Principal, orderID, key string, command order.RefundRequestCommand) (order.RefundResult, error) {
	service.createCalls++
	service.createOrderID, service.createKey, service.createCommand = orderID, key, command
	return service.result, service.err
}

func (service *fakeRefundService) Approve(_ context.Context, _ auth.Principal, _ string, version int64) (order.RefundResult, error) {
	service.approveCalls++
	service.approveVersion = version
	return service.result, service.err
}

func (service *fakeRefundService) Reject(_ context.Context, _ auth.Principal, _ string, version int64) (order.RefundResult, error) {
	service.rejectCalls++
	service.rejectVersion = version
	return service.result, service.err
}

func testRefund(status order.RefundStatus) order.Refund {
	createdAt := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	value := order.Refund{
		ID: "rfd_00000000000000000000000000000001", OrderID: "ord_00000000000000000000000000000008",
		Amount: 100, Currency: "CNY", Reason: "customer request", Status: status,
		Version: 1, RequestedBy: order.RefundActor{ID: "user-operator", Username: "operator"}, CreatedAt: createdAt, UpdatedAt: createdAt,
	}
	if status != order.RefundStatusPending {
		decidedAt := createdAt.Add(time.Hour)
		value.Version = 2
		value.DecidedBy = &order.RefundActor{ID: "user-approver", Username: "approver"}
		value.DecidedAt = &decidedAt
		value.UpdatedAt = decidedAt
	}
	return value
}

var _ = errors.Is
