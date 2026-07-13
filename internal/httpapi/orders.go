package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/order"
)

type OrderService interface {
	List(context.Context, auth.Principal, order.ListQuery) (order.Page, error)
	Get(context.Context, auth.Principal, string) (order.Order, error)
	Create(context.Context, auth.Principal, string, order.CreateCommand) (order.Order, error)
	Edit(context.Context, auth.Principal, string, order.EditCommand) (order.Order, error)
	Transition(context.Context, auth.Principal, string, order.Action, order.TransitionCommand) (order.Order, error)
}

const orderWriteBodyLimit = 64 * 1024
const orderActionBodyLimit = 1 * 1024

var validIdempotencyKey = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

type orderDTO struct {
	ID                    string              `json:"id"`
	CustomerName          string              `json:"customerName"`
	Status                order.Status        `json:"status"`
	PaymentStatus         order.PaymentStatus `json:"paymentStatus"`
	Currency              string              `json:"currency"`
	TotalAmount           int64               `json:"totalAmount"`
	AvailableRefundAmount int64               `json:"availableRefundAmount"`
	Version               int64               `json:"version"`
	CreatedAt             string              `json:"createdAt"`
	UpdatedAt             string              `json:"updatedAt"`
	CanEdit               bool                `json:"canEdit"`
	CanAdvance            bool                `json:"canAdvance"`
	CanCancel             bool                `json:"canCancel"`
	CanRequestRefund      bool                `json:"canRequestRefund"`
	CanApproveRefund      bool                `json:"canApproveRefund"`
	Items                 []orderItemDTO      `json:"items,omitempty"`
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

func registerOrderRoutes(mux *http.ServeMux, authService AuthService, service OrderService, actionEnabled ...bool) {
	if authService == nil || service == nil {
		return
	}
	listHandler := RequireAuthentication(authService)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		principal, _ := PrincipalFromContext(request.Context())
		values, err := url.ParseQuery(request.URL.RawQuery)
		if err != nil {
			writeErrorDetails(response, request, http.StatusBadRequest, "INVALID_REQUEST", "invalid request", []errorDetail{{Field: "query", Message: "must be valid URL-encoded query parameters"}})
			return
		}
		query, err := parseOrderListQuery(values)
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
	createHandler := RequireAuthentication(authService)(RequireRoles(auth.RoleOperator, auth.RoleAdmin)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if err := requireJSONContentType(request); err != nil {
			handleOrderInputError(response, request, err)
			return
		}
		key := request.Header.Get("Idempotency-Key")
		if len(request.Header.Values("Idempotency-Key")) != 1 || !validIdempotencyKey.MatchString(key) {
			writeErrorDetails(response, request, http.StatusBadRequest, "INVALID_REQUEST", "invalid request", []errorDetail{{Field: "Idempotency-Key", Message: "must be a valid idempotency key"}})
			return
		}
		command, err := decodeCreateCommand(response, request)
		if err != nil {
			handleOrderInputError(response, request, err)
			return
		}
		principal, _ := PrincipalFromContext(request.Context())
		result, err := service.Create(request.Context(), principal, key, command)
		if handleOrderError(response, request, err) {
			return
		}
		writeJSON(response, http.StatusCreated, makeOrderDTO(principal, result, true))
	})))
	editHandler := RequireAuthentication(authService)(RequireRoles(auth.RoleOperator, auth.RoleAdmin)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if err := requireJSONContentType(request); err != nil {
			handleOrderInputError(response, request, err)
			return
		}
		id := request.PathValue("orderId")
		if !order.ValidOrderID(id) {
			writeError(response, request, http.StatusNotFound, "NOT_FOUND", "order not found")
			return
		}
		command, err := decodeEditCommand(response, request)
		if err != nil {
			handleOrderInputError(response, request, err)
			return
		}
		principal, _ := PrincipalFromContext(request.Context())
		result, err := service.Edit(request.Context(), principal, id, command)
		if handleOrderError(response, request, err) {
			return
		}
		writeJSON(response, http.StatusOK, makeOrderDTO(principal, result, true))
	})))
	collectionRoute := orderCollectionMetadata()
	detailRoute := orderDetailMetadata()
	mux.Handle(collectionRoute.pattern, orderRoute(collectionRoute, map[string]http.Handler{http.MethodGet: listHandler, http.MethodPost: createHandler}))
	mux.Handle(detailRoute.pattern, orderRoute(detailRoute, map[string]http.Handler{http.MethodGet: detailHandler, http.MethodPatch: editHandler}))
	if len(actionEnabled) == 0 || actionEnabled[0] {
		for action, target := range map[string]order.Action{"confirm": order.ActionConfirm, "fulfill": order.ActionFulfill, "ship": order.ActionShip, "complete": order.ActionComplete, "cancel": order.ActionCancel} {
			actionHandler := RequireAuthentication(authService)(RequireRoles(auth.RoleOperator, auth.RoleAdmin)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				if err := requireJSONContentType(request); err != nil {
					handleOrderInputError(response, request, err)
					return
				}
				id := request.PathValue("orderId")
				if !order.ValidOrderID(id) {
					writeError(response, request, http.StatusNotFound, "NOT_FOUND", "order not found")
					return
				}
				version, err := decodeActionVersion(response, request)
				if err != nil {
					handleOrderInputError(response, request, err)
					return
				}
				principal, _ := PrincipalFromContext(request.Context())
				result, err := service.Transition(request.Context(), principal, id, target, order.TransitionCommand{Version: version})
				if handleOrderError(response, request, err) {
					return
				}
				writeJSON(response, http.StatusOK, makeOrderDTO(principal, result, true))
			})))
			route := orderActionMetadata(action)
			mux.Handle(route.pattern, orderRoute(route, map[string]http.Handler{http.MethodPost: actionHandler}))
		}
	}
}

func orderRoute(route routeMetadata, handlers map[string]http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		handler, ok := handlers[request.Method]
		if !ok || !route.methods[request.Method] {
			methodNotAllowed(response, request, route.allow)
			return
		}
		handler.ServeHTTP(response, request)
	})
}

func decodeActionVersion(response http.ResponseWriter, request *http.Request) (int64, error) {
	var input struct {
		Version json.RawMessage `json:"version"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, orderActionBodyLimit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return 0, errors.New("invalid action body")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return 0, errors.New("invalid action body")
	}
	version, err := parseRawInteger(input.Version)
	if err != nil {
		return 0, errors.New("invalid action version")
	}
	return version, nil
}

type writeItemInput struct {
	SKU       string          `json:"sku"`
	Name      string          `json:"name"`
	Quantity  json.RawMessage `json:"quantity"`
	UnitPrice json.RawMessage `json:"unitPrice"`
}

type createOrderInput struct {
	CustomerName string           `json:"customerName"`
	Currency     string           `json:"currency"`
	Items        []writeItemInput `json:"items"`
}

func decodeCreateCommand(response http.ResponseWriter, request *http.Request) (order.CreateCommand, error) {
	var input createOrderInput
	if err := decodeOrderJSON(response, request, &input); err != nil {
		return order.CreateCommand{}, err
	}
	items, err := parseWriteItems(input.Items)
	if err != nil {
		return order.CreateCommand{}, err
	}
	return order.CreateCommand{CustomerName: input.CustomerName, Currency: input.Currency, Items: items}, nil
}

func decodeEditCommand(response http.ResponseWriter, request *http.Request) (order.EditCommand, error) {
	var input struct {
		createOrderInput
		Version json.RawMessage `json:"version"`
	}
	if err := decodeOrderJSON(response, request, &input); err != nil {
		return order.EditCommand{}, err
	}
	items, err := parseWriteItems(input.Items)
	if err != nil {
		return order.EditCommand{}, err
	}
	version, err := parseRawInteger(input.Version)
	if err != nil {
		return order.EditCommand{}, errors.New("invalid integer field")
	}
	return order.EditCommand{CustomerName: input.CustomerName, Currency: input.Currency, Items: items, Version: version}, nil
}

func decodeOrderJSON(response http.ResponseWriter, request *http.Request, destination any) error {
	if err := requireJSONContentType(request); err != nil {
		return err
	}
	decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, orderWriteBodyLimit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return errors.New("invalid JSON body")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("invalid JSON body")
	}
	return nil
}

func requireJSONContentType(request *http.Request) error {
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return errUnsupportedMedia
	}
	return nil
}

func parseWriteItems(inputs []writeItemInput) ([]order.ItemCommand, error) {
	items := make([]order.ItemCommand, 0, len(inputs))
	for _, input := range inputs {
		quantity, err := parseRawInteger(input.Quantity)
		if err != nil {
			return nil, errors.New("invalid integer field")
		}
		unitPrice, err := parseRawInteger(input.UnitPrice)
		if err != nil {
			return nil, errors.New("invalid integer field")
		}
		items = append(items, order.ItemCommand{SKU: input.SKU, Name: input.Name, Quantity: quantity, UnitPrice: unitPrice})
	}
	return items, nil
}

func parseRawInteger(raw json.RawMessage) (int64, error) {
	if len(raw) == 0 {
		return 0, errors.New("missing integer")
	}
	return order.ParseIntegerLexeme(string(raw))
}

var errUnsupportedMedia = errors.New("unsupported media type")

func handleOrderInputError(response http.ResponseWriter, request *http.Request, err error) {
	if errors.Is(err, errUnsupportedMedia) {
		writeError(response, request, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "content type must be application/json")
		return
	}
	writeError(response, request, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
}

func makeOrderDTO(principal auth.Principal, value order.Order, includeItems bool) orderDTO {
	capabilities := order.CapabilitiesForOrder(principal, value)
	dto := orderDTO{ID: value.ID, CustomerName: value.CustomerName, Status: value.Status, PaymentStatus: value.PaymentStatus, Currency: value.Currency, TotalAmount: value.TotalAmount, AvailableRefundAmount: value.AvailableRefundAmount, Version: value.Version, CreatedAt: order.FormatTime(value.CreatedAt), UpdatedAt: order.FormatTime(value.UpdatedAt), CanEdit: capabilities.CanEdit, CanAdvance: capabilities.CanAdvance, CanCancel: capabilities.CanCancel, CanRequestRefund: capabilities.CanRequestRefund, CanApproveRefund: capabilities.CanApproveRefund}
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
	if details, ok := order.ValidationDetails(err); ok {
		mapped := make([]errorDetail, 0, len(details))
		for _, detail := range details {
			mapped = append(mapped, errorDetail{Field: detail.Field, Message: detail.Message})
		}
		writeErrorDetails(response, request, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "validation failed", mapped)
		return true
	}
	if errors.Is(err, order.ErrVersionConflict) {
		writeError(response, request, http.StatusConflict, "VERSION_CONFLICT", "order version conflict")
		return true
	}
	if errors.Is(err, order.ErrStateConflict) {
		writeError(response, request, http.StatusConflict, "STATE_CONFLICT", "order state conflict")
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
