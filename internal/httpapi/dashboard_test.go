package httpapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/httpapi"
	"github.com/magicvr/allinme.core-api/internal/order"
)

func TestDashboardRoutesResponsesQueriesAndRoles(t *testing.T) {
	service := &fakeDashboardService{
		summary: order.DashboardSummary{OrderCount: 10, GrossAmount: 460000, CompletedRefundAmount: 120000, NetAmount: 340000, Currency: "CNY"},
		status:  order.DashboardOrderStatus{Items: []order.DashboardStatusItem{{Status: order.StatusDraft, Count: 1}, {Status: order.StatusConfirmed, Count: 0}}},
		trend:   order.DashboardTrend{Days: 7, StartDate: "2026-01-01", EndDate: "2026-01-07", Items: []order.DashboardTrendItem{{Date: "2026-01-01", OrderCount: 10, GrossAmount: 460000, NetAmount: 460000}, {Date: "2026-01-02", CompletedRefundAmount: 120000, NetAmount: -120000}}},
	}
	authService := &fakeAuthService{authenticatedRole: auth.RoleViewer}
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: authService, Dashboard: service})
	request := func(path string) *httptest.ResponseRecorder {
		value := httptest.NewRequest(http.MethodGet, path, nil)
		value.Header.Set("Authorization", "Bearer access-token")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, value)
		return response
	}
	if response := request("/api/v1/dashboard/summary"); response.Code != http.StatusOK || response.Body.String() != "{\"orderCount\":10,\"grossAmount\":460000,\"completedRefundAmount\":120000,\"netAmount\":340000,\"currency\":\"CNY\"}\n" {
		t.Fatalf("summary = %d %s", response.Code, response.Body.String())
	}
	if response := request("/api/v1/dashboard/order-status"); response.Code != http.StatusOK || response.Body.String() != "{\"items\":[{\"status\":\"DRAFT\",\"count\":1},{\"status\":\"CONFIRMED\",\"count\":0}]}\n" {
		t.Fatalf("order status = %d %s", response.Code, response.Body.String())
	}
	if response := request("/api/v1/dashboard/trend?days=7"); response.Code != http.StatusOK || service.days != 7 || !strings.Contains(response.Body.String(), `"netAmount":-120000`) {
		t.Fatalf("trend = %d %s days=%d", response.Code, response.Body.String(), service.days)
	}
	for _, role := range []auth.Role{auth.RoleViewer, auth.RoleOperator, auth.RoleApprover, auth.RoleAdmin} {
		authService.authenticatedRole = role
		if response := request("/api/v1/dashboard/summary"); response.Code != http.StatusOK {
			t.Fatalf("%s summary = %d %s", role, response.Code, response.Body.String())
		}
	}
	unauthenticated := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	unauthenticatedResponse := httptest.NewRecorder()
	handler.ServeHTTP(unauthenticatedResponse, unauthenticated)
	if unauthenticatedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated dashboard = %d %s", unauthenticatedResponse.Code, unauthenticatedResponse.Body.String())
	}
	for _, path := range []string{"/api/v1/dashboard/summary?days=7", "/api/v1/dashboard/order-status?x=1"} {
		if response := request(path); response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"INVALID_REQUEST"`) {
			t.Fatalf("query %s = %d %s", path, response.Code, response.Body.String())
		}
	}
	for _, test := range []struct {
		path  string
		field string
	}{
		{"/api/v1/dashboard/trend", "days"},
		{"/api/v1/dashboard/trend?days=", "days"},
		{"/api/v1/dashboard/trend?days=7&days=30", "days"},
		{"/api/v1/dashboard/trend?days=07", "days"},
		{"/api/v1/dashboard/trend?days=+7", "days"},
		{"/api/v1/dashboard/trend?days=%2B7", "days"},
		{"/api/v1/dashboard/trend?days=%2D7", "days"},
		{"/api/v1/dashboard/trend?days=8", "days"},
		{"/api/v1/dashboard/trend?days=7&first=1&second=2", "first"},
	} {
		response := request(test.path)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"field":"`+test.field+`"`) {
			t.Fatalf("invalid trend %s = %d %s", test.path, response.Code, response.Body.String())
		}
	}
}

func TestDashboardMethodsCORSDisableAndErrorMapping(t *testing.T) {
	service := &fakeDashboardService{summary: order.DashboardSummary{Currency: "CNY"}}
	refunds := &fakeRefundService{page: order.RefundPage{Items: []order.Refund{}, Page: 1, PageSize: 20}}
	authService := &fakeAuthService{authenticatedRole: auth.RoleAdmin}
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: authService, Refunds: refunds, Dashboard: service, DisableRefundRoutes: true, CORSAllowedOrigin: testAllowedOrigin})
	for _, path := range []string{"/api/v1/dashboard/summary", "/api/v1/dashboard/order-status", "/api/v1/dashboard/trend"} {
		request := httptest.NewRequest(http.MethodHead, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != http.MethodGet {
			t.Fatalf("HEAD %s = %d Allow=%q", path, response.Code, response.Header().Get("Allow"))
		}
		preflight := corsRequest(http.MethodOptions, path, testAllowedOrigin)
		preflight.Header.Set("Access-Control-Request-Method", http.MethodGet)
		preflightResponse := httptest.NewRecorder()
		handler.ServeHTTP(preflightResponse, preflight)
		if preflightResponse.Code != http.StatusNoContent {
			t.Fatalf("preflight %s = %d %s", path, preflightResponse.Code, preflightResponse.Body.String())
		}
	}
	missingMethod := corsRequest(http.MethodOptions, "/api/v1/dashboard/summary", testAllowedOrigin)
	missingMethodResponse := httptest.NewRecorder()
	handler.ServeHTTP(missingMethodResponse, missingMethod)
	if missingMethodResponse.Code != http.StatusForbidden || !strings.Contains(missingMethodResponse.Body.String(), `"code":"CORS_PREFLIGHT_DENIED"`) {
		t.Fatalf("missing requested method = %d %s", missingMethodResponse.Code, missingMethodResponse.Body.String())
	}
	withoutCORS := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: authService, Dashboard: service})
	ordinaryOptions := httptest.NewRequest(http.MethodOptions, "/api/v1/dashboard/summary", nil)
	ordinaryOptionsResponse := httptest.NewRecorder()
	withoutCORS.ServeHTTP(ordinaryOptionsResponse, ordinaryOptions)
	if ordinaryOptionsResponse.Code != http.StatusMethodNotAllowed || ordinaryOptionsResponse.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("ordinary OPTIONS = %d Allow=%q", ordinaryOptionsResponse.Code, ordinaryOptionsResponse.Header().Get("Allow"))
	}
	service.err = order.ErrUnavailable
	request := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	request.Header.Set("Authorization", "Bearer access-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || response.Header().Get("Retry-After") != "1" || !strings.Contains(response.Body.String(), `"code":"SERVICE_UNAVAILABLE"`) {
		t.Fatalf("unavailable = %d %v %s", response.Code, response.Header(), response.Body.String())
	}
	service.err = order.ErrInternal
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), `"code":"INTERNAL_ERROR"`) {
		t.Fatalf("internal = %d %s", response.Code, response.Body.String())
	}
	disabled := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: authService, Refunds: refunds, Dashboard: service, DisableDashboardRoutes: true})
	for _, path := range []string{"/api/v1/dashboard/summary", "/api/v1/dashboard/order-status", "/api/v1/dashboard/trend?days=7"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		disabled.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("disabled %s = %d", path, response.Code)
		}
	}
	refundRequest := httptest.NewRequest(http.MethodGet, "/api/v1/refunds", nil)
	refundRequest.Header.Set("Authorization", "Bearer access-token")
	refundResponse := httptest.NewRecorder()
	disabled.ServeHTTP(refundResponse, refundRequest)
	if refundResponse.Code != http.StatusOK {
		t.Fatalf("refund list with dashboard disabled = %d %s", refundResponse.Code, refundResponse.Body.String())
	}
}

type fakeDashboardService struct {
	summary order.DashboardSummary
	status  order.DashboardOrderStatus
	trend   order.DashboardTrend
	days    int
	err     error
}

func (service *fakeDashboardService) Summary(context.Context, auth.Principal) (order.DashboardSummary, error) {
	return service.summary, service.err
}

func (service *fakeDashboardService) OrderStatus(context.Context, auth.Principal) (order.DashboardOrderStatus, error) {
	return service.status, service.err
}

func (service *fakeDashboardService) Trend(_ context.Context, _ auth.Principal, days int) (order.DashboardTrend, error) {
	service.days = days
	return service.trend, service.err
}
