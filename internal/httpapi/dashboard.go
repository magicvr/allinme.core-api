package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/order"
)

type DashboardService interface {
	Summary(context.Context, auth.Principal) (order.DashboardSummary, error)
	OrderStatus(context.Context, auth.Principal) (order.DashboardOrderStatus, error)
	Trend(context.Context, auth.Principal, int) (order.DashboardTrend, error)
}

type dashboardSummaryDTO struct {
	OrderCount            int64  `json:"orderCount"`
	GrossAmount           int64  `json:"grossAmount"`
	CompletedRefundAmount int64  `json:"completedRefundAmount"`
	NetAmount             int64  `json:"netAmount"`
	Currency              string `json:"currency"`
}

type dashboardStatusDTO struct {
	Items []dashboardStatusItemDTO `json:"items"`
}

type dashboardStatusItemDTO struct {
	Status order.Status `json:"status"`
	Count  int64        `json:"count"`
}

type dashboardTrendDTO struct {
	Days      int                     `json:"days"`
	StartDate string                  `json:"startDate"`
	EndDate   string                  `json:"endDate"`
	Items     []dashboardTrendItemDTO `json:"items"`
}

type dashboardTrendItemDTO struct {
	Date                  string `json:"date"`
	OrderCount            int64  `json:"orderCount"`
	GrossAmount           int64  `json:"grossAmount"`
	CompletedRefundAmount int64  `json:"completedRefundAmount"`
	NetAmount             int64  `json:"netAmount"`
}

func registerDashboardRoutes(mux *http.ServeMux, authService AuthService, service DashboardService, disabled bool) {
	if disabled || authService == nil || service == nil {
		return
	}
	summaryHandler := RequireAuthentication(authService)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if err := requireEmptyDashboardQuery(request); err != nil {
			handleDashboardInputError(response, request, err)
			return
		}
		principal, _ := PrincipalFromContext(request.Context())
		result, err := service.Summary(request.Context(), principal)
		if handleDashboardError(response, request, err) {
			return
		}
		writeJSON(response, http.StatusOK, dashboardSummaryDTO{OrderCount: result.OrderCount, GrossAmount: result.GrossAmount, CompletedRefundAmount: result.CompletedRefundAmount, NetAmount: result.NetAmount, Currency: result.Currency})
	}))
	summaryRoute := dashboardSummaryMetadata()
	mux.Handle(summaryRoute.pattern, orderRoute(summaryRoute, map[string]http.Handler{http.MethodGet: summaryHandler}))

	statusHandler := RequireAuthentication(authService)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if err := requireEmptyDashboardQuery(request); err != nil {
			handleDashboardInputError(response, request, err)
			return
		}
		principal, _ := PrincipalFromContext(request.Context())
		result, err := service.OrderStatus(request.Context(), principal)
		if handleDashboardError(response, request, err) {
			return
		}
		items := make([]dashboardStatusItemDTO, 0, len(result.Items))
		for _, item := range result.Items {
			items = append(items, dashboardStatusItemDTO{Status: item.Status, Count: item.Count})
		}
		writeJSON(response, http.StatusOK, dashboardStatusDTO{Items: items})
	}))
	statusRoute := dashboardOrderStatusMetadata()
	mux.Handle(statusRoute.pattern, orderRoute(statusRoute, map[string]http.Handler{http.MethodGet: statusHandler}))

	trendHandler := RequireAuthentication(authService)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		days, err := parseDashboardTrendQuery(request)
		if err != nil {
			handleDashboardInputError(response, request, err)
			return
		}
		principal, _ := PrincipalFromContext(request.Context())
		result, err := service.Trend(request.Context(), principal, days)
		if handleDashboardError(response, request, err) {
			return
		}
		items := make([]dashboardTrendItemDTO, 0, len(result.Items))
		for _, item := range result.Items {
			items = append(items, dashboardTrendItemDTO{Date: item.Date, OrderCount: item.OrderCount, GrossAmount: item.GrossAmount, CompletedRefundAmount: item.CompletedRefundAmount, NetAmount: item.NetAmount})
		}
		writeJSON(response, http.StatusOK, dashboardTrendDTO{Days: result.Days, StartDate: result.StartDate, EndDate: result.EndDate, Items: items})
	}))
	trendRoute := dashboardTrendMetadata()
	mux.Handle(trendRoute.pattern, orderRoute(trendRoute, map[string]http.Handler{http.MethodGet: trendHandler}))
}

func requireEmptyDashboardQuery(request *http.Request) error {
	values, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		return errors.New("invalid dashboard query")
	}
	if len(values) > 0 {
		key := firstQueryKey(request.URL.RawQuery)
		if key == "" {
			key = "query"
		}
		return dashboardQueryError{field: key, message: "query parameter is not allowed"}
	}
	return nil
}

func parseDashboardTrendQuery(request *http.Request) (int, error) {
	values, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		return 0, errors.New("invalid dashboard query")
	}
	for key := range values {
		if key != "days" {
			return 0, dashboardQueryError{field: firstUnknownQueryKey(request.URL.RawQuery, map[string]bool{"days": true}), message: "unknown query parameter"}
		}
	}
	entries, ok := values["days"]
	if !ok || len(entries) != 1 || entries[0] == "" {
		return 0, dashboardQueryError{field: "days", message: "must be 7 or 30"}
	}
	days, err := strconv.Atoi(entries[0])
	if err != nil || (days != 7 && days != 30) || (len(entries[0]) > 1 && entries[0][0] == '0') {
		return 0, dashboardQueryError{field: "days", message: "must be 7 or 30"}
	}
	return days, nil
}

type dashboardQueryError struct {
	field   string
	message string
}

func (err dashboardQueryError) Error() string { return err.message }

func firstQueryKey(raw string) string {
	return firstUnknownQueryKey(raw, nil)
}

func firstUnknownQueryKey(raw string, allowed map[string]bool) string {
	for _, part := range strings.Split(raw, "&") {
		if part == "" {
			continue
		}
		key := part
		if index := strings.IndexByte(part, '='); index >= 0 {
			key = part[:index]
		}
		if decoded, err := url.QueryUnescape(key); err == nil {
			if allowed == nil || !allowed[decoded] {
				return decoded
			}
			continue
		}
		if allowed == nil || !allowed[key] {
			return key
		}
	}
	return ""
}

func handleDashboardInputError(response http.ResponseWriter, request *http.Request, err error) {
	if field, ok := err.(dashboardQueryError); ok {
		writeErrorDetails(response, request, http.StatusBadRequest, "INVALID_REQUEST", "invalid request", []errorDetail{{Field: field.field, Message: field.message}})
		return
	}
	writeError(response, request, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
}

func handleDashboardError(response http.ResponseWriter, request *http.Request, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		panic(http.ErrAbortHandler)
	}
	if errors.Is(err, order.ErrForbidden) {
		writeError(response, request, http.StatusForbidden, "FORBIDDEN", "forbidden")
		return true
	}
	if errors.Is(err, order.ErrUnavailable) {
		markUnavailable(response)
		response.Header().Set("Retry-After", "1")
		writeError(response, request, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "service unavailable")
		return true
	}
	if details, ok := order.ValidationDetails(err); ok {
		mapped := make([]errorDetail, 0, len(details))
		for _, detail := range details {
			mapped = append(mapped, errorDetail{Field: detail.Field, Message: detail.Message})
		}
		writeErrorDetails(response, request, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "validation failed", mapped)
		return true
	}
	writeError(response, request, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	return true
}
