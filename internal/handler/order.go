package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/magicvr/allinme.core-api/internal/domain"
	"github.com/magicvr/allinme.core-api/internal/port"
	"github.com/magicvr/allinme.core-api/internal/response"
	orderservice "github.com/magicvr/allinme.core-api/internal/service/order"
)

const maxOrderRequestBody = 1 << 20

type createOrderRequest struct {
	OrderNo      string `json:"orderNo"`
	CustomerName string `json:"customerName"`
	AmountCents  int64  `json:"amountCents"`
	Currency     string `json:"currency"`
	Remark       string `json:"remark"`
}

type updateOrderRequest struct {
	Version      int64  `json:"version"`
	CustomerName string `json:"customerName"`
	AmountCents  int64  `json:"amountCents"`
	Currency     string `json:"currency"`
	Remark       string `json:"remark"`
}

type orderActionRequest struct {
	Version int64 `json:"version"`
}

type batchDeleteOrdersRequest struct {
	IDs []string `json:"ids"`
}

type orderListData struct {
	List  []domain.Order `json:"list"`
	Total int            `json:"total"`
}

func listOrders(service *orderservice.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filter, err := parseOrderListFilter(r)
		if err != nil {
			orderError(w, err)
			return
		}
		orders, total, err := service.List(r.Context(), filter)
		if err != nil {
			orderError(w, err)
			return
		}
		response.OK(w, orderListData{List: orders, Total: total})
	})
}

func getOrder(service *orderservice.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order, err := service.Get(r.Context(), r.PathValue("id"))
		if err != nil {
			orderError(w, err)
			return
		}
		response.OK(w, order)
	})
}

func createOrder(service *orderservice.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req createOrderRequest
		if err := decodeOrderJSON(w, r, &req); err != nil {
			return
		}
		order, err := service.Create(r.Context(), orderservice.CreateInput{
			OrderNo:      req.OrderNo,
			CustomerName: req.CustomerName,
			AmountCents:  req.AmountCents,
			Currency:     req.Currency,
			Remark:       req.Remark,
		})
		if err != nil {
			orderError(w, err)
			return
		}
		response.OK(w, order)
	})
}

func updateOrder(service *orderservice.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req updateOrderRequest
		if err := decodeOrderJSON(w, r, &req); err != nil {
			return
		}
		order, err := service.Update(r.Context(), r.PathValue("id"), orderservice.UpdateInput{
			Version:      req.Version,
			CustomerName: req.CustomerName,
			AmountCents:  req.AmountCents,
			Currency:     req.Currency,
			Remark:       req.Remark,
		})
		if err != nil {
			orderError(w, err)
			return
		}
		response.OK(w, order)
	})
}

func markOrderPaid(service *orderservice.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req orderActionRequest
		if err := decodeOrderJSON(w, r, &req); err != nil {
			return
		}
		order, err := service.MarkPaid(r.Context(), r.PathValue("id"), req.Version)
		if err != nil {
			orderError(w, err)
			return
		}
		response.OK(w, order)
	})
}

func cancelOrder(service *orderservice.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req orderActionRequest
		if err := decodeOrderJSON(w, r, &req); err != nil {
			return
		}
		order, err := service.Cancel(r.Context(), r.PathValue("id"), req.Version)
		if err != nil {
			orderError(w, err)
			return
		}
		response.OK(w, order)
	})
}

func batchDeleteOrders(service *orderservice.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req batchDeleteOrdersRequest
		if err := decodeOrderJSON(w, r, &req); err != nil {
			return
		}
		deleted, err := service.BatchDelete(r.Context(), req.IDs)
		if err != nil {
			orderError(w, err)
			return
		}
		response.OK(w, map[string]int{"deleted": deleted})
	})
}

func parseOrderListFilter(r *http.Request) (port.OrderListFilter, error) {
	query := r.URL.Query()
	page, err := queryInt(query.Get("page"), 1)
	if err != nil {
		return port.OrderListFilter{}, port.ErrInvalidArgument
	}
	pageSize, err := queryInt(query.Get("pageSize"), 20)
	if err != nil {
		return port.OrderListFilter{}, port.ErrInvalidArgument
	}
	return port.OrderListFilter{
		Status:   domain.OrderStatus(query.Get("status")),
		Query:    query.Get("q"),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func queryInt(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	return strconv.Atoi(raw)
}

func decodeOrderJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxOrderRequestBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		response.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		response.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return err
	}
	return nil
}

func orderError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, port.ErrInvalidArgument):
		response.Error(w, http.StatusBadRequest, "bad_request", "invalid order request")
	case errors.Is(err, port.ErrOrderNotFound):
		response.Error(w, http.StatusNotFound, "order_not_found", "order not found")
	case errors.Is(err, port.ErrOrderNoConflict):
		response.Error(w, http.StatusConflict, "order_no_conflict", "order number already exists")
	case errors.Is(err, port.ErrVersionConflict):
		response.Error(w, http.StatusConflict, "version_conflict", "order version conflict")
	case errors.Is(err, port.ErrInvalidState):
		response.Error(w, http.StatusConflict, "invalid_state", "operation is not allowed for current order state")
	default:
		response.Error(w, http.StatusInternalServerError, "internal", "order operation failed")
	}
}
