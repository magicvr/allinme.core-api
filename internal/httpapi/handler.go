package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"runtime/debug"
	"time"

	"github.com/magicvr/allinme.core-api/internal/store"
)

const requestIDHeader = "X-Request-ID"

var validRequestID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

type ReadinessProbe interface {
	Check(context.Context) store.ReadinessStatus
}

type Dependencies struct {
	Logger            *slog.Logger
	Readiness         ReadinessProbe
	ReadinessTimeout  time.Duration
	Auth              AuthService
	LoginLimiter      *LoginLimiter
	Orders            OrderService
	OrderActions      bool
	CORSAllowedOrigin string
	Fallback          http.Handler
}

type statusResponse struct {
	Status string `json:"status"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code      string        `json:"code"`
	Message   string        `json:"message"`
	RequestID string        `json:"requestId"`
	Details   []errorDetail `json:"details,omitempty"`
}

type errorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type requestIDKey struct{}

func NewHandler(dependencies Dependencies) http.Handler {
	logger := dependencies.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	readinessTimeout := dependencies.ReadinessTimeout
	if readinessTimeout <= 0 {
		readinessTimeout = time.Second
	}
	mux := http.NewServeMux()
	registerAuthRoutes(mux, dependencies.Auth, dependencies.LoginLimiter)
	registerOrderRoutes(mux, dependencies.Auth, dependencies.Orders, dependencies.OrderActions)
	mux.HandleFunc("GET /healthz", func(response http.ResponseWriter, _ *http.Request) {
		writeJSON(response, http.StatusOK, statusResponse{Status: "ok"})
	})
	mux.HandleFunc("/readyz", func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			response.Header().Set("Allow", http.MethodGet)
			writeError(response, request, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		if dependencies.Readiness == nil {
			writeError(response, request, http.StatusServiceUnavailable, "NOT_READY", "service is not ready")
			return
		}
		ctx, cancel := context.WithTimeout(request.Context(), readinessTimeout)
		defer cancel()
		if dependencies.Readiness.Check(ctx) != store.Ready {
			writeError(response, request, http.StatusServiceUnavailable, "NOT_READY", "service is not ready")
			return
		}
		writeJSON(response, http.StatusOK, statusResponse{Status: "ready"})
	})
	if dependencies.Fallback != nil {
		mux.Handle("/", dependencies.Fallback)
	}
	routes := activeRouteMetadata(dependencies)
	return requestIDMiddleware(accessLogMiddleware(logger, recoveryMiddleware(logger, corsMiddleware(dependencies.CORSAllowedOrigin, routes, mux))))
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestID := request.Header.Get(requestIDHeader)
		if !validRequestID.MatchString(requestID) {
			requestID = newRequestID()
		}
		response.Header().Set(requestIDHeader, requestID)
		ctx := context.WithValue(request.Context(), requestIDKey{}, requestID)
		next.ServeHTTP(response, request.WithContext(ctx))
	})
}

func accessLogMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		started := time.Now()
		tracked := &trackingResponseWriter{ResponseWriter: response}
		defer func() {
			recovered := recover()
			outcome := tracked.outcome
			if outcome == "" {
				if recovered == http.ErrAbortHandler && !tracked.committed {
					outcome = "canceled"
				} else {
					outcome = "completed"
				}
			}
			attributes := []any{"request_id", requestID(request), "method", request.Method, "path", request.URL.Path, "outcome", outcome, "duration", time.Since(started)}
			if tracked.committed {
				attributes = append(attributes, "status", tracked.status)
			}
			logger.InfoContext(request.Context(), "http request", attributes...)
			if recovered != nil {
				panic(recovered)
			}
		}()
		next.ServeHTTP(tracked, request)
	})
}

func recoveryMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		tracked, _ := response.(*trackingResponseWriter)
		defer func() {
			if recovered := recover(); recovered != nil {
				if recovered == http.ErrAbortHandler {
					panic(recovered)
				}
				logger.ErrorContext(request.Context(), "http panic recovered", "request_id", requestID(request), "panic", recovered, "stack", string(debug.Stack()))
				if tracked == nil || !tracked.committed {
					writeError(response, request, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
				}
			}
		}()
		next.ServeHTTP(response, request)
	})
}

type trackingResponseWriter struct {
	http.ResponseWriter
	status    int
	committed bool
	outcome   string
}

func markUnavailable(response http.ResponseWriter) {
	if tracked, ok := response.(*trackingResponseWriter); ok {
		tracked.outcome = "unavailable"
	}
}

func (writer *trackingResponseWriter) WriteHeader(status int) {
	if writer.committed {
		return
	}
	writer.status = status
	writer.committed = true
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *trackingResponseWriter) Write(body []byte) (int, error) {
	if !writer.committed {
		writer.WriteHeader(http.StatusOK)
	}
	return writer.ResponseWriter.Write(body)
}

func writeError(response http.ResponseWriter, request *http.Request, status int, code, message string) {
	writeJSON(response, status, errorEnvelope{Error: errorBody{Code: code, Message: message, RequestID: requestID(request)}})
}

func writeErrorDetails(response http.ResponseWriter, request *http.Request, status int, code, message string, details []errorDetail) {
	writeJSON(response, status, errorEnvelope{Error: errorBody{Code: code, Message: message, RequestID: requestID(request), Details: details}})
}

func writeJSON(response http.ResponseWriter, status int, body any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(body)
}

func requestID(request *http.Request) string {
	value, _ := request.Context().Value(requestIDKey{}).(string)
	return value
}

func newRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic("secure request ID generation failed")
	}
	return "req_" + hex.EncodeToString(bytes)
}
