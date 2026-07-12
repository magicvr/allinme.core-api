package httpapi

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/order"
)

type OrderService interface {
	List(context.Context, auth.Principal, order.ListQuery) (order.Page, error)
	Get(context.Context, auth.Principal, string) (order.Order, error)
}

type orderDTO struct {
	ID               string              `json:"id"`
	CustomerName     string              `json:"customerName"`
	Status           order.Status        `json:"status"`
	PaymentStatus    order.PaymentStatus `json:"paymentStatus"`
	Currency         string              `json:"currency"`
	TotalAmount      int64               `json:"totalAmount"`
	Version          int64               `json:"version"`
	CreatedAt        string              `json:"createdAt"`
	UpdatedAt        string              `json:"updatedAt"`
	CanEdit          bool                `json:"canEdit"`
	CanAdvance       bool                `json:"canAdvance"`
	CanCancel        bool                `json:"canCancel"`
	CanRequestRefund bool                `json:"canRequestRefund"`
	CanApproveRefund bool                `json:"canApproveRefund"`
	Items            []orderItemDTO      `json:"items,omitempty"`
}
type orderItemDTO struct {
	ID        string `json:"id"`
	SKU       string `json:"sku"`
	Name      string `json:"name"`
	Quantity  int64  `json:"quantity"`
	UnitPrice int64  `json:"unitPrice"`
}
type orderPageDTO struct {
	Items    []orderDTO `json:"items"`
	Total    int64      `json:"total"`
	Page     int64      `json:"page"`
	PageSize int64      `json:"pageSize"`
}

func registerOrderRoutes(mux *http.ServeMux, authService AuthService, service OrderService) {
	if authService == nil || service == nil {
		return
	}
	listHandler := RequireAuthentication(authService)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		principal, _ := PrincipalFromContext(request.Context())
		query, err := parseOrderListQuery(request.URL.Query())
		if err != nil {
			var fieldError orderQueryError
			if errors.As(err, &fieldError) {
				writeErrorDetails(response, request, http.StatusBadRequest, "INVALID_REQUEST", "invalid request", []errorDetail{{Field: fieldError.field, Message: fieldError.message}})
				return
			}
			writeError(response, request, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
			return
		}
		page, err := service.List(request.Context(), principal, query)
		if handleOrderError(response, request, err) {
			return
		}
		items := make([]orderDTO, 0, len(page.Items))
		for _, item := range page.Items {
			items = append(items, makeOrderDTO(principal, item, false))
		}
		writeJSON(response, http.StatusOK, orderPageDTO{Items: items, Total: page.Total, Page: page.Page, PageSize: page.PageSize})
	}))
	detailHandler := RequireAuthentication(authService)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		id := request.PathValue("orderId")
		if !order.ValidOrderID(id) {
			writeError(response, request, http.StatusNotFound, "NOT_FOUND", "order not found")
			return
		}
		principal, _ := PrincipalFromContext(request.Context())
		result, err := service.Get(request.Context(), principal, id)
		if handleOrderError(response, request, err) {
			return
		}
		writeJSON(response, http.StatusOK, makeOrderDTO(principal, result, true))
	}))
	mux.Handle("/api/v1/orders", readOnlyOrderRoute(listHandler))
	mux.Handle("/api/v1/orders/{orderId}", readOnlyOrderRoute(detailHandler))
}

func readOnlyOrderRoute(get http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodGet:
			get.ServeHTTP(response, request)
		case http.MethodHead:
			response.Header().Set("Allow", http.MethodGet)
			writeError(response, request, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		default:
			http.NotFound(response, request)
		}
	})
}

func makeOrderDTO(principal auth.Principal, value order.Order, includeItems bool) orderDTO {
	capabilities := order.CapabilitiesFor(principal, value.Status)
	dto := orderDTO{ID: value.ID, CustomerName: value.CustomerName, Status: value.Status, PaymentStatus: value.PaymentStatus, Currency: value.Currency, TotalAmount: value.TotalAmount, Version: value.Version, CreatedAt: order.FormatTime(value.CreatedAt), UpdatedAt: order.FormatTime(value.UpdatedAt), CanEdit: capabilities.CanEdit, CanAdvance: capabilities.CanAdvance, CanCancel: capabilities.CanCancel, CanRequestRefund: capabilities.CanRequestRefund, CanApproveRefund: capabilities.CanApproveRefund}
	if includeItems {
		dto.Items = make([]orderItemDTO, 0, len(value.Items))
		for _, item := range value.Items {
			dto.Items = append(dto.Items, orderItemDTO{ID: item.ID, SKU: item.SKU, Name: item.Name, Quantity: item.Quantity, UnitPrice: item.UnitPrice})
		}
	}
	return dto
}

func handleOrderError(response http.ResponseWriter, request *http.Request, err error) bool {
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
	if errors.Is(err, order.ErrNotFound) {
		writeError(response, request, http.StatusNotFound, "NOT_FOUND", "order not found")
		return true
	}
	writeError(response, request, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	return true
}

func parseOrderListQuery(values url.Values) (order.ListQuery, error) {
	allowed := map[string]bool{"q": true, "status": true, "paymentStatus": true, "createdFrom": true, "createdTo": true, "page": true, "pageSize": true, "sort": true, "order": true}
	for key, entries := range values {
		if !allowed[key] {
			return order.ListQuery{}, errors.New("unknown query parameter")
		}
		if len(entries) != 1 {
			return order.ListQuery{}, errors.New("query parameters must not be repeated")
		}
	}
	query := order.ListQuery{Page: 1, PageSize: 20, Sort: "createdAt", Descending: true, Keyword: strings.TrimSpace(values.Get("q"))}
	if len([]byte(query.Keyword)) > 200 {
		return order.ListQuery{}, errors.New("q is too long")
	}
	if raw := values.Get("status"); raw != "" {
		query.Status = order.Status(raw)
		if !query.Status.Valid() {
			return order.ListQuery{}, errors.New("invalid status")
		}
	}
	if raw := values.Get("paymentStatus"); raw != "" {
		query.PaymentStatus = order.PaymentStatus(raw)
		if !query.PaymentStatus.Valid() {
			return order.ListQuery{}, errors.New("invalid paymentStatus")
		}
	}
	var err error
	if raw := values.Get("page"); raw != "" {
		query.Page, err = parsePositiveInt(raw)
		if err != nil {
			return order.ListQuery{}, errors.New("invalid page")
		}
	}
	if raw := values.Get("pageSize"); raw != "" {
		query.PageSize, err = parsePositiveInt(raw)
		if err != nil || query.PageSize > 100 {
			return order.ListQuery{}, errors.New("invalid pageSize")
		}
	}
	if query.Page-1 > math.MaxInt64/query.PageSize {
		return order.ListQuery{}, orderQueryError{field: "page", message: "page is too large"}
	}
	if raw := values.Get("sort"); raw != "" {
		if !map[string]bool{"createdAt": true, "updatedAt": true, "totalAmount": true, "customerName": true, "status": true}[raw] {
			return order.ListQuery{}, errors.New("invalid sort")
		}
		query.Sort = raw
	}
	if raw := values.Get("order"); raw != "" {
		if raw != "asc" && raw != "desc" {
			return order.ListQuery{}, errors.New("invalid order")
		}
		query.Descending = raw == "desc"
	}
	if raw := values.Get("createdFrom"); raw != "" {
		parsed, parseErr := time.Parse(time.RFC3339, raw)
		if parseErr != nil {
			return order.ListQuery{}, errors.New("invalid createdFrom")
		}
		parsed = parsed.UTC()
		query.CreatedFrom = &parsed
	}
	if raw := values.Get("createdTo"); raw != "" {
		parsed, parseErr := time.Parse(time.RFC3339, raw)
		if parseErr != nil {
			return order.ListQuery{}, errors.New("invalid createdTo")
		}
		parsed = parsed.UTC()
		query.CreatedTo = &parsed
	}
	if query.CreatedFrom != nil && query.CreatedTo != nil && query.CreatedFrom.After(*query.CreatedTo) {
		return order.ListQuery{}, errors.New("createdFrom must not be after createdTo")
	}
	return query, nil
}

type orderQueryError struct{ field, message string }

func (err orderQueryError) Error() string { return err.message }
func parsePositiveInt(value string) (int64, error) {
	if value == "" || strings.HasPrefix(value, "+") || (len(value) > 1 && value[0] == '0') {
		return 0, errors.New("invalid integer")
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 1 {
		return 0, errors.New("invalid integer")
	}
	return parsed, nil
}
