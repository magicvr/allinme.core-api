package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/httpapi"
	"github.com/magicvr/allinme.core-api/internal/order"
	"github.com/magicvr/allinme.core-api/internal/store"
)

type probeFunc func(context.Context) store.ReadinessStatus

func (probe probeFunc) Check(ctx context.Context) store.ReadinessStatus { return probe(ctx) }

type waitBuffer struct {
	mu      sync.Mutex
	buffer  bytes.Buffer
	written chan struct{}
	once    sync.Once
}

func newWaitBuffer() *waitBuffer {
	return &waitBuffer{written: make(chan struct{})}
}

func (buffer *waitBuffer) Write(value []byte) (int, error) {
	buffer.mu.Lock()
	written, err := buffer.buffer.Write(value)
	buffer.mu.Unlock()
	buffer.once.Do(func() { close(buffer.written) })
	return written, err
}

func (buffer *waitBuffer) String() string {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buffer.String()
}

func TestHealthDoesNotDependOnReadiness(t *testing.T) {
	handler := newHandler(probeFunc(func(context.Context) store.ReadinessStatus { return store.DatabaseUnavailable }))
	response := serve(handler, http.MethodGet, "/healthz", "")
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("health response = %d, %q", response.Code, response.Header().Get("Content-Type"))
	}
	var body struct {
		Status string `json:"status"`
	}
	decode(t, response, &body)
	if body.Status != "ok" {
		t.Fatalf("status = %q", body.Status)
	}
}

func TestReady(t *testing.T) {
	handler := newHandler(probeFunc(func(context.Context) store.ReadinessStatus { return store.Ready }))
	response := serve(handler, http.MethodGet, "/readyz", "client.request-1")
	if response.Code != http.StatusOK || response.Header().Get("X-Request-ID") != "client.request-1" {
		t.Fatalf("ready response = %d, request ID %q", response.Code, response.Header().Get("X-Request-ID"))
	}
}

func TestNotReadyUsesSafeEnvelope(t *testing.T) {
	handler := newHandler(probeFunc(func(context.Context) store.ReadinessStatus { return store.SchemaTooNew }))
	response := serve(handler, http.MethodGet, "/readyz", "request-7")
	if response.Code != http.StatusServiceUnavailable || strings.Contains(response.Body.String(), "schema") {
		t.Fatalf("not-ready response = %d %s", response.Code, response.Body.String())
	}
	assertError(t, response, "NOT_READY", "request-7")
}

func TestReadyMethodNotAllowed(t *testing.T) {
	response := serve(newHandler(nil), http.MethodPost, "/readyz", "method-1")
	if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("response = %d, Allow %q", response.Code, response.Header().Get("Allow"))
	}
	assertError(t, response, "METHOD_NOT_ALLOWED", "method-1")
}

func TestInvalidRequestIDIsReplaced(t *testing.T) {
	response := serve(newHandler(nil), http.MethodGet, "/healthz", "invalid request id")
	requestID := response.Header().Get("X-Request-ID")
	if !regexp.MustCompile(`^req_[0-9a-f]{32}$`).MatchString(requestID) {
		t.Fatalf("request ID = %q", requestID)
	}
}

func TestReadinessTimeout(t *testing.T) {
	probe := probeFunc(func(ctx context.Context) store.ReadinessStatus {
		<-ctx.Done()
		return store.DatabaseUnavailable
	})
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Readiness: probe, ReadinessTimeout: 10 * time.Millisecond})
	started := time.Now()
	response := serve(handler, http.MethodGet, "/readyz", "timeout-1")
	if response.Code != http.StatusServiceUnavailable || time.Since(started) > time.Second {
		t.Fatalf("timeout response = %d after %s", response.Code, time.Since(started))
	}
}

func TestPanicBeforeCommitReturnsInternalErrorAndLogsStatus(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	fallback := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("failed") })
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: logger, Fallback: fallback})
	response := serve(handler, http.MethodGet, "/panic", "panic-1")
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", response.Code)
	}
	assertError(t, response, "INTERNAL_ERROR", "panic-1")
	logText := logs.String()
	for _, expected := range []string{`"msg":"http panic recovered"`, `"request_id":"panic-1"`, `"method":"GET"`, `"path":"/panic"`, `"status":500`} {
		if !strings.Contains(logText, expected) {
			t.Errorf("logs missing %s: %s", expected, logText)
		}
	}
}

func TestPanicAfterCommitDoesNotRewriteResponse(t *testing.T) {
	var logs bytes.Buffer
	fallback := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusAccepted)
		_, _ = response.Write([]byte("partial"))
		panic("failed after commit")
	})
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: slog.New(slog.NewJSONHandler(&logs, nil)), Fallback: fallback})
	response := serve(handler, http.MethodGet, "/panic", "panic-2")
	if response.Code != http.StatusAccepted || response.Body.String() != "partial" {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
	if !strings.Contains(logs.String(), `"status":202`) {
		t.Fatalf("logs = %s", logs.String())
	}
}

func TestCanceledRequestAbortsConnectionAndLogsWithoutStatus(t *testing.T) {
	logs := newWaitBuffer()
	handler := httpapi.NewHandler(httpapi.Dependencies{
		Logger: slog.New(slog.NewJSONHandler(logs, nil)),
		Auth:   &fakeAuthService{},
		Orders: &fakeOrderService{err: context.Canceled},
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/v1/orders", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer access-token")
	response, err := server.Client().Do(request)
	if response != nil {
		response.Body.Close()
	}
	if err == nil {
		t.Fatal("canceled request error = nil")
	}
	select {
	case <-logs.written:
	case <-time.After(time.Second):
		t.Fatal("canceled request access log was not written")
	}
	logText := logs.String()
	if !strings.Contains(logText, `"outcome":"canceled"`) || strings.Contains(logText, `"status"`) || strings.Contains(logText, "sensitive") {
		t.Fatalf("logs = %s", logText)
	}
}

func TestAccessLogOutcomesCompletedUnavailableAndCommittedCancel(t *testing.T) {
	t.Run("completed", func(t *testing.T) {
		var logs bytes.Buffer
		handler := httpapi.NewHandler(httpapi.Dependencies{Logger: slog.New(slog.NewJSONHandler(&logs, nil))})
		response := serve(handler, http.MethodGet, "/healthz", "completed-1")
		if response.Code != http.StatusOK || !strings.Contains(logs.String(), `"outcome":"completed"`) || !strings.Contains(logs.String(), `"status":200`) {
			t.Fatalf("response=%d logs=%s", response.Code, logs.String())
		}
	})

	t.Run("unavailable", func(t *testing.T) {
		var logs bytes.Buffer
		handler := httpapi.NewHandler(httpapi.Dependencies{Logger: slog.New(slog.NewJSONHandler(&logs, nil)), Auth: &fakeAuthService{}, Orders: &fakeOrderService{err: order.ErrUnavailable}})
		request := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
		request.Header.Set("Authorization", "Bearer access-token")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusServiceUnavailable || !strings.Contains(logs.String(), `"outcome":"unavailable"`) || !strings.Contains(logs.String(), `"status":503`) || response.Header().Get("Retry-After") != "1" {
			t.Fatalf("response=%d headers=%v logs=%s", response.Code, response.Header(), logs.String())
		}
	})

	t.Run("committed cancel", func(t *testing.T) {
		var logs bytes.Buffer
		fallback := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			response.WriteHeader(http.StatusAccepted)
			panic(http.ErrAbortHandler)
		})
		handler := httpapi.NewHandler(httpapi.Dependencies{Logger: slog.New(slog.NewJSONHandler(&logs, nil)), Fallback: fallback})
		func() {
			defer func() {
				if recovered := recover(); recovered != http.ErrAbortHandler {
					t.Fatalf("recovered = %v", recovered)
				}
			}()
			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/committed-cancel", nil))
		}()
		if !strings.Contains(logs.String(), `"outcome":"completed"`) || !strings.Contains(logs.String(), `"status":202`) {
			t.Fatalf("logs = %s", logs.String())
		}
	})
}

func newHandler(probe httpapi.ReadinessProbe) http.Handler {
	return httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Readiness: probe, ReadinessTimeout: time.Second})
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func serve(handler http.Handler, method, path, requestID string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, nil)
	if requestID != "" {
		request.Header.Set("X-Request-ID", requestID)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertError(t *testing.T, response *httptest.ResponseRecorder, code, requestID string) {
	t.Helper()
	var body struct {
		Error struct {
			Code      string `json:"code"`
			RequestID string `json:"requestId"`
		} `json:"error"`
	}
	decode(t, response, &body)
	if body.Error.Code != code || body.Error.RequestID != requestID || response.Header().Get("X-Request-ID") != requestID {
		t.Fatalf("error = %+v, header request ID = %q", body.Error, response.Header().Get("X-Request-ID"))
	}
}

func decode(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
