package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/url"
	"unicode/utf8"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/order"
)

const refundBodyLimit = 8 * 1024

type RefundService interface {
	List(context.Context, auth.Principal, order.RefundListQuery) (order.RefundPage, error)
	Create(context.Context, auth.Principal, string, string, order.RefundRequestCommand) (order.RefundResult, error)
	Approve(context.Context, auth.Principal, string, int64) (order.RefundResult, error)
	Reject(context.Context, auth.Principal, string, int64) (order.RefundResult, error)
}

type refundPageDTO struct {
	Items    []refundDTO `json:"items"`
	Total    int64       `json:"total"`
	Page     int64       `json:"page"`
	PageSize int64       `json:"pageSize"`
}

type refundDTO struct {
	ID          string             `json:"id"`
	OrderID     string             `json:"orderId"`
	Amount      int64              `json:"amount"`
	Currency    string             `json:"currency"`
	Reason      string             `json:"reason"`
	Status      order.RefundStatus `json:"status"`
	Version     int64              `json:"version"`
	RequestedBy refundActorDTO     `json:"requestedBy"`
	DecidedBy   *refundActorDTO    `json:"decidedBy"`
	CreatedAt   string             `json:"createdAt"`
	UpdatedAt   string             `json:"updatedAt"`
	DecidedAt   *string            `json:"decidedAt"`
	CanApprove  bool               `json:"canApprove"`
	CanReject   bool               `json:"canReject"`
}

type refundActorDTO struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

func registerRefundRoutes(mux *http.ServeMux, authService AuthService, service RefundService, disabled bool) {
	if disabled || authService == nil || service == nil {
		return
	}
	listHandler := RequireAuthentication(authService)(RequireRoles(auth.RoleApprover, auth.RoleAdmin)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		values, err := url.ParseQuery(request.URL.RawQuery)
		if err != nil {
			writeErrorDetails(response, request, http.StatusBadRequest, "INVALID_REQUEST", "invalid request", []errorDetail{{Field: "query", Message: "must be valid URL-encoded query parameters"}})
			return
		}
		query, err := parseRefundListQuery(values)
		if err != nil {
			if field, ok := err.(refundQueryError); ok {
				writeErrorDetails(response, request, http.StatusBadRequest, "INVALID_REQUEST", "invalid request", []errorDetail{{Field: field.field, Message: field.message}})
				return
			}
			writeError(response, request, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
			return
		}
		principal, _ := PrincipalFromContext(request.Context())
		page, err := service.List(request.Context(), principal, query)
		if handleRefundError(response, request, err) {
			return
		}
		items := make([]refundDTO, 0, len(page.Items))
		for _, value := range page.Items {
			items = append(items, makeRefundDTO(principal, value, order.RefundCapabilitiesFor(principal, value)))
		}
		writeJSON(response, http.StatusOK, refundPageDTO{Items: items, Total: page.Total, Page: page.Page, PageSize: page.PageSize})
	})))
	mux.Handle(refundCollectionMetadata().pattern, orderRoute(refundCollectionMetadata(), map[string]http.Handler{http.MethodGet: listHandler}))

	createHandler := RequireAuthentication(authService)(RequireRoles(auth.RoleOperator, auth.RoleAdmin)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if err := requireJSONContentType(request); err != nil {
			handleRefundInputError(response, request, err)
			return
		}
		body, err := readRefundBody(response, request)
		if err != nil {
			handleRefundInputError(response, request, err)
			return
		}
		key := request.Header.Get("Idempotency-Key")
		if len(request.Header.Values("Idempotency-Key")) != 1 || !validIdempotencyKey.MatchString(key) {
			writeErrorDetails(response, request, http.StatusBadRequest, "INVALID_REQUEST", "invalid request", []errorDetail{{Field: "Idempotency-Key", Message: "must be a valid idempotency key"}})
			return
		}
		orderID := request.PathValue("orderId")
		if !order.ValidOrderID(orderID) {
			writeError(response, request, http.StatusNotFound, "NOT_FOUND", "order not found")
			return
		}
		command, err := decodeRefundRequestBody(body)
		if err != nil {
			handleRefundInputError(response, request, err)
			return
		}
		principal, _ := PrincipalFromContext(request.Context())
		result, err := service.Create(request.Context(), principal, orderID, key, command)
		if handleRefundError(response, request, err) {
			return
		}
		writeJSON(response, http.StatusCreated, makeRefundDTO(principal, result.Refund, result.Capabilities))
	})))
	mux.Handle(refundCreateMetadata().pattern, orderRoute(refundCreateMetadata(), map[string]http.Handler{http.MethodPost: createHandler}))

	for _, decision := range []struct {
		action  string
		handler func(context.Context, auth.Principal, string, int64) (order.RefundResult, error)
	}{
		{action: "approve", handler: service.Approve},
		{action: "reject", handler: service.Reject},
	} {
		decision := decision
		handler := RequireAuthentication(authService)(RequireRoles(auth.RoleApprover, auth.RoleAdmin)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if err := requireJSONContentType(request); err != nil {
				handleRefundInputError(response, request, err)
				return
			}
			body, err := readRefundBody(response, request)
			if err != nil {
				handleRefundInputError(response, request, err)
				return
			}
			refundID := request.PathValue("refundId")
			if !order.ValidRefundID(refundID) {
				writeError(response, request, http.StatusNotFound, "NOT_FOUND", "refund not found")
				return
			}
			version, err := decodeRefundVersionBody(body)
			if err != nil {
				handleRefundInputError(response, request, err)
				return
			}
			principal, _ := PrincipalFromContext(request.Context())
			result, err := decision.handler(request.Context(), principal, refundID, version)
			if handleRefundError(response, request, err) {
				return
			}
			writeJSON(response, http.StatusOK, makeRefundDTO(principal, result.Refund, result.Capabilities))
		})))
		route := refundDecisionMetadata(decision.action)
		mux.Handle(route.pattern, orderRoute(route, map[string]http.Handler{http.MethodPost: handler}))
	}
}

func refundQueryErrorFrom(field, message string) error {
	return refundQueryError{field: field, message: message}
}

type refundQueryError struct {
	field   string
	message string
}

func (err refundQueryError) Error() string { return err.message }

func parseRefundListQuery(values url.Values) (order.RefundListQuery, error) {
	allowed := map[string]bool{"status": true, "orderId": true, "page": true, "pageSize": true}
	for key, entries := range values {
		if !allowed[key] {
			return order.RefundListQuery{}, refundQueryErrorFrom(key, "unknown query parameter")
		}
		if len(entries) != 1 {
			return order.RefundListQuery{}, refundQueryErrorFrom(key, "query parameters must not be repeated")
		}
	}
	query := order.RefundListQuery{Page: 1, PageSize: 20}
	if entries, ok := values["status"]; ok {
		if len(entries) != 1 || entries[0] == "" {
			return order.RefundListQuery{}, refundQueryErrorFrom("status", "invalid status")
		}
		query.Status = order.RefundStatus(entries[0])
		if !query.Status.Valid() {
			return order.RefundListQuery{}, refundQueryErrorFrom("status", "invalid status")
		}
	}
	if entries, ok := values["orderId"]; ok {
		if len(entries) != 1 || !order.ValidOrderID(entries[0]) {
			return order.RefundListQuery{}, refundQueryErrorFrom("orderId", "invalid orderId")
		}
		query.OrderID = entries[0]
	}
	var err error
	if entries, ok := values["page"]; ok {
		query.Page, err = parsePositiveInt(entries[0])
		if err != nil {
			return order.RefundListQuery{}, refundQueryErrorFrom("page", "invalid page")
		}
	}
	if entries, ok := values["pageSize"]; ok {
		query.PageSize, err = parsePositiveInt(entries[0])
		if err != nil || query.PageSize > 100 {
			return order.RefundListQuery{}, refundQueryErrorFrom("pageSize", "invalid pageSize")
		}
	}
	if query.Page-1 > math.MaxInt64/query.PageSize {
		return order.RefundListQuery{}, refundQueryErrorFrom("page", "page is too large")
	}
	return query, nil
}

func readRefundBody(response http.ResponseWriter, request *http.Request) ([]byte, error) {
	body, err := io.ReadAll(http.MaxBytesReader(response, request.Body, refundBodyLimit))
	if err != nil || !utf8.Valid(body) {
		return nil, errors.New("invalid refund body")
	}
	return body, nil
}

func decodeRefundRequestBody(body []byte) (order.RefundRequestCommand, error) {
	var input struct {
		Amount       json.RawMessage `json:"amount"`
		Reason       string          `json:"reason"`
		OrderVersion json.RawMessage `json:"orderVersion"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return order.RefundRequestCommand{}, errors.New("invalid refund body")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return order.RefundRequestCommand{}, errors.New("invalid refund body")
	}
	amount, err := parseRawInteger(input.Amount)
	if err != nil {
		return order.RefundRequestCommand{}, errors.New("invalid integer field")
	}
	orderVersion, err := parseRawInteger(input.OrderVersion)
	if err != nil {
		return order.RefundRequestCommand{}, errors.New("invalid integer field")
	}
	return order.RefundRequestCommand{Amount: amount, Reason: input.Reason, OrderVersion: orderVersion}, nil
}

func decodeRefundVersionBody(body []byte) (int64, error) {
	var input struct {
		Version json.RawMessage `json:"version"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return 0, errors.New("invalid refund body")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return 0, errors.New("invalid refund body")
	}
	version, err := parseRawInteger(input.Version)
	if err != nil {
		return 0, errors.New("invalid integer field")
	}
	return version, nil
}

func handleRefundInputError(response http.ResponseWriter, request *http.Request, err error) {
	if errors.Is(err, errUnsupportedMedia) {
		writeError(response, request, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "content type must be application/json")
		return
	}
	writeError(response, request, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
}

func handleRefundError(response http.ResponseWriter, request *http.Request, err error) bool {
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
		writeError(response, request, http.StatusNotFound, "NOT_FOUND", "refund not found")
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
	if errors.Is(err, order.ErrVersionConflict) {
		writeError(response, request, http.StatusConflict, "VERSION_CONFLICT", "refund version conflict")
		return true
	}
	if errors.Is(err, order.ErrStateConflict) {
		writeError(response, request, http.StatusConflict, "STATE_CONFLICT", "refund state conflict")
		return true
	}
	if errors.Is(err, order.ErrIdempotencyConflict) {
		writeError(response, request, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "idempotency key conflict")
		return true
	}
	if errors.Is(err, order.ErrUnavailable) {
		markUnavailable(response)
		response.Header().Set("Retry-After", "1")
		writeError(response, request, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "service unavailable")
		return true
	}
	writeError(response, request, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	return true
}

func makeRefundDTO(principal auth.Principal, value order.Refund, capabilities order.RefundCapabilities) refundDTO {
	dto := refundDTO{
		ID: value.ID, OrderID: value.OrderID, Amount: value.Amount, Currency: value.Currency, Reason: value.Reason,
		Status: value.Status, Version: value.Version, RequestedBy: refundActorDTO{ID: value.RequestedBy.ID, Username: value.RequestedBy.Username},
		CreatedAt: order.FormatTime(value.CreatedAt), UpdatedAt: order.FormatTime(value.UpdatedAt), CanApprove: capabilities.CanApprove, CanReject: capabilities.CanReject,
	}
	if value.DecidedBy != nil {
		actor := refundActorDTO{ID: value.DecidedBy.ID, Username: value.DecidedBy.Username}
		dto.DecidedBy = &actor
	}
	if value.DecidedAt != nil {
		decidedAt := order.FormatTime(*value.DecidedAt)
		dto.DecidedAt = &decidedAt
	}
	return dto
}
