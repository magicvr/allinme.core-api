package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/httpapi"
)

const testAllowedOrigin = "https://ui.example.com"

func TestCORSActualRequestsAndShortCircuitOrder(t *testing.T) {
	authService := &fakeAuthService{authenticatedRole: auth.RoleViewer}
	service := &fakeOrderService{result: testOrder()}
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: authService, Orders: service, OrderActions: true, CORSAllowedOrigin: testAllowedOrigin})

	allowed := corsRequest(http.MethodGet, "/api/v1/orders", testAllowedOrigin)
	allowedResponse := httptest.NewRecorder()
	handler.ServeHTTP(allowedResponse, allowed)
	if allowedResponse.Code != http.StatusUnauthorized || allowedResponse.Header().Get("Access-Control-Allow-Origin") != testAllowedOrigin || allowedResponse.Header().Get("Access-Control-Expose-Headers") != "X-Request-ID" || !headerContains(allowedResponse.Header(), "Vary", "Origin") {
		t.Fatalf("allowed actual = %d headers=%v body=%s", allowedResponse.Code, allowedResponse.Header(), allowedResponse.Body.String())
	}

	denied := corsRequest(http.MethodGet, "/api/v1/orders", "https://evil.example.com")
	deniedResponse := httptest.NewRecorder()
	handler.ServeHTTP(deniedResponse, denied)
	if deniedResponse.Code != http.StatusForbidden || !strings.Contains(deniedResponse.Body.String(), `"code":"CORS_ORIGIN_DENIED"`) || deniedResponse.Header().Get("Access-Control-Allow-Origin") != "" || authService.authenticateCount != 0 {
		t.Fatalf("denied actual = %d headers=%v authCount=%d body=%s", deniedResponse.Code, deniedResponse.Header(), authService.authenticateCount, deniedResponse.Body.String())
	}

	unknown := corsRequest(http.MethodGet, "/api/v1/unknown", "https://evil.example.com")
	unknownResponse := httptest.NewRecorder()
	handler.ServeHTTP(unknownResponse, unknown)
	if unknownResponse.Code != http.StatusNotFound || unknownResponse.Header().Get("Access-Control-Allow-Origin") != "" || strings.Contains(unknownResponse.Body.String(), "CORS_") {
		t.Fatalf("unknown path = %d headers=%v body=%s", unknownResponse.Code, unknownResponse.Header(), unknownResponse.Body.String())
	}

	wrongMethod := corsRequest(http.MethodGet, "/api/v1/auth/login", "https://evil.example.com")
	wrongMethodResponse := httptest.NewRecorder()
	handler.ServeHTTP(wrongMethodResponse, wrongMethod)
	if wrongMethodResponse.Code != http.StatusMethodNotAllowed || strings.Contains(wrongMethodResponse.Body.String(), "CORS_") {
		t.Fatalf("wrong method = %d %s", wrongMethodResponse.Code, wrongMethodResponse.Body.String())
	}

	viewer := corsRequest(http.MethodPost, "/api/v1/orders", testAllowedOrigin)
	viewer.Header.Set("Authorization", "Bearer access-token")
	viewer.Header.Set("Content-Type", "text/plain")
	viewerResponse := httptest.NewRecorder()
	handler.ServeHTTP(viewerResponse, viewer)
	if viewerResponse.Code != http.StatusForbidden || !strings.Contains(viewerResponse.Body.String(), `"code":"FORBIDDEN"`) {
		t.Fatalf("viewer short circuit = %d %s", viewerResponse.Code, viewerResponse.Body.String())
	}
}

func TestCORSPreflightMatrixUsesKnownRoutes(t *testing.T) {
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: &fakeAuthService{}, Orders: &fakeOrderService{}, OrderActions: true, CORSAllowedOrigin: testAllowedOrigin})
	tests := []struct {
		name, path, origin, method, headers string
		status                              int
		code                                string
	}{
		{name: "orders post", path: "/api/v1/orders", origin: testAllowedOrigin, method: http.MethodPost, headers: "authorization, content-type, idempotency-key, x-request-id", status: 204},
		{name: "detail patch", path: "/api/v1/orders/ord_1", origin: testAllowedOrigin, method: http.MethodPatch, headers: "Authorization, Content-Type", status: 204},
		{name: "action post", path: "/api/v1/orders/ord_1/confirm", origin: testAllowedOrigin, method: http.MethodPost, status: 204},
		{name: "auth get", path: "/api/v1/auth/me", origin: testAllowedOrigin, method: http.MethodGet, headers: "Authorization", status: 204},
		{name: "origin mismatch", path: "/api/v1/orders", origin: "https://evil.example.com", method: http.MethodGet, status: 403, code: "CORS_PREFLIGHT_DENIED"},
		{name: "invalid origin", path: "/api/v1/orders", origin: "null", method: http.MethodGet, status: 403, code: "CORS_PREFLIGHT_DENIED"},
		{name: "method mismatch", path: "/api/v1/auth/me", origin: testAllowedOrigin, method: http.MethodPost, status: 403, code: "CORS_PREFLIGHT_DENIED"},
		{name: "header mismatch", path: "/api/v1/orders", origin: testAllowedOrigin, method: http.MethodGet, headers: "Authorization, X-Secret", status: 403, code: "CORS_PREFLIGHT_DENIED"},
		{name: "unknown path", path: "/api/v1/unknown", origin: testAllowedOrigin, method: http.MethodGet, status: 404},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := corsRequest(http.MethodOptions, test.path, test.origin)
			request.Header.Set("Access-Control-Request-Method", test.method)
			if test.headers != "" {
				request.Header.Set("Access-Control-Request-Headers", test.headers)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d headers=%v body=%s", response.Code, response.Header(), response.Body.String())
			}
			if test.code != "" && !strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("body = %s", response.Body.String())
			}
			if test.status != http.StatusNotFound {
				for _, vary := range []string{"Origin", "Access-Control-Request-Method", "Access-Control-Request-Headers"} {
					if !headerContains(response.Header(), "Vary", vary) {
						t.Fatalf("Vary missing %q: %v", vary, response.Header().Values("Vary"))
					}
				}
			}
			if test.status == http.StatusNoContent {
				if response.Header().Get("Access-Control-Allow-Origin") != testAllowedOrigin || response.Header().Get("Access-Control-Allow-Methods") != "GET, POST, PATCH, OPTIONS" || response.Header().Get("Access-Control-Allow-Headers") != "Authorization, Content-Type, Idempotency-Key, X-Request-ID" || response.Header().Get("Access-Control-Max-Age") != "600" || response.Header().Get("Access-Control-Expose-Headers") != "X-Request-ID" {
					t.Fatalf("preflight headers = %v", response.Header())
				}
			} else if response.Header().Get("Access-Control-Allow-Origin") != "" {
				t.Fatalf("denied allow origin = %q", response.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

func TestCORSDisabledPreservesM3AAPI(t *testing.T) {
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: &fakeAuthService{}, Orders: &fakeOrderService{result: testOrder()}, OrderActions: true})
	preflight := corsRequest(http.MethodOptions, "/api/v1/orders", testAllowedOrigin)
	preflight.Header.Set("Access-Control-Request-Method", http.MethodGet)
	preflightResponse := httptest.NewRecorder()
	handler.ServeHTTP(preflightResponse, preflight)
	if preflightResponse.Code != http.StatusMethodNotAllowed || preflightResponse.Header().Get("Allow") != "GET, POST" || preflightResponse.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("disabled preflight = %d %v", preflightResponse.Code, preflightResponse.Header())
	}
	request := corsRequest(http.MethodGet, "/api/v1/orders", testAllowedOrigin)
	request.Header.Set("Authorization", "Bearer access-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("disabled actual = %d %v %s", response.Code, response.Header(), response.Body.String())
	}
}

func corsRequest(method, path, origin string) *http.Request {
	request := httptest.NewRequest(method, path, nil)
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	return request
}

func headerContains(header http.Header, name, expected string) bool {
	for _, line := range header.Values(name) {
		for _, value := range strings.Split(line, ",") {
			if strings.EqualFold(strings.TrimSpace(value), expected) {
				return true
			}
		}
	}
	return false
}
