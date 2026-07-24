package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/magicvr/allinme.core-api/internal/domain"
	"github.com/magicvr/allinme.core-api/internal/port"
	"github.com/magicvr/allinme.core-api/internal/response"
	walletservice "github.com/magicvr/allinme.core-api/internal/service/wallet"
)

const maxWalletRequestBody = 1 << 20

type walletService interface {
	List(context.Context, port.WalletListFilter) ([]domain.Wallet, int, error)
	Get(context.Context, string) (domain.Wallet, error)
	Create(context.Context, walletservice.CreateInput) (domain.Wallet, error)
	Update(context.Context, string, walletservice.UpdateInput) (domain.Wallet, error)
	Freeze(context.Context, string, int64) (domain.Wallet, error)
	Unfreeze(context.Context, string, int64) (domain.Wallet, error)
	BatchFreeze(context.Context, []string) (int, error)
}

type createWalletRequest struct {
	AccountNo    string `json:"accountNo"`
	OwnerName    string `json:"ownerName"`
	BalanceCents int64  `json:"balanceCents"`
	Currency     string `json:"currency"`
}

type updateWalletRequest struct {
	Version   int64  `json:"version"`
	OwnerName string `json:"ownerName"`
}

type walletActionRequest struct {
	Version int64 `json:"version"`
}

type batchFreezeWalletsRequest struct {
	IDs []string `json:"ids"`
}

type walletListData struct {
	List  []domain.Wallet `json:"list"`
	Total int             `json:"total"`
}

func listWallets(service walletService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filter, err := parseWalletListFilter(r)
		if err != nil {
			walletError(w, err)
			return
		}
		wallets, total, err := service.List(r.Context(), filter)
		if err != nil {
			walletError(w, err)
			return
		}
		response.OK(w, walletListData{List: wallets, Total: total})
	})
}

func getWallet(service walletService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wallet, err := service.Get(r.Context(), r.PathValue("id"))
		if err != nil {
			walletError(w, err)
			return
		}
		response.OK(w, wallet)
	})
}

func createWallet(service walletService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req createWalletRequest
		if err := decodeWalletJSON(w, r, &req); err != nil {
			return
		}
		wallet, err := service.Create(r.Context(), walletservice.CreateInput{
			AccountNo:    req.AccountNo,
			OwnerName:    req.OwnerName,
			BalanceCents: req.BalanceCents,
			Currency:     req.Currency,
		})
		if err != nil {
			walletError(w, err)
			return
		}
		response.OK(w, wallet)
	})
}

func updateWallet(service walletService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req updateWalletRequest
		if err := decodeWalletJSON(w, r, &req); err != nil {
			return
		}
		wallet, err := service.Update(r.Context(), r.PathValue("id"), walletservice.UpdateInput{
			Version:   req.Version,
			OwnerName: req.OwnerName,
		})
		if err != nil {
			walletError(w, err)
			return
		}
		response.OK(w, wallet)
	})
}

func freezeWallet(service walletService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req walletActionRequest
		if err := decodeWalletJSON(w, r, &req); err != nil {
			return
		}
		wallet, err := service.Freeze(r.Context(), r.PathValue("id"), req.Version)
		if err != nil {
			walletError(w, err)
			return
		}
		response.OK(w, wallet)
	})
}

func unfreezeWallet(service walletService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req walletActionRequest
		if err := decodeWalletJSON(w, r, &req); err != nil {
			return
		}
		wallet, err := service.Unfreeze(r.Context(), r.PathValue("id"), req.Version)
		if err != nil {
			walletError(w, err)
			return
		}
		response.OK(w, wallet)
	})
}

func batchFreezeWallets(service walletService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req batchFreezeWalletsRequest
		if err := decodeWalletJSON(w, r, &req); err != nil {
			return
		}
		frozen, err := service.BatchFreeze(r.Context(), req.IDs)
		if err != nil {
			walletError(w, err)
			return
		}
		response.OK(w, map[string]int{"frozen": frozen})
	})
}

func parseWalletListFilter(r *http.Request) (port.WalletListFilter, error) {
	query := r.URL.Query()
	page, err := queryInt(query.Get("page"), 1)
	if err != nil {
		return port.WalletListFilter{}, port.ErrWalletInvalidArgument
	}
	pageSize, err := queryInt(query.Get("pageSize"), 20)
	if err != nil {
		return port.WalletListFilter{}, port.ErrWalletInvalidArgument
	}
	return port.WalletListFilter{
		Status:   domain.WalletStatus(query.Get("status")),
		Query:    query.Get("q"),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func decodeWalletJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxWalletRequestBody)
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

func walletError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, port.ErrWalletInvalidArgument):
		response.Error(w, http.StatusBadRequest, "bad_request", "invalid wallet request")
	case errors.Is(err, port.ErrWalletNotFound):
		response.Error(w, http.StatusNotFound, "wallet_not_found", "wallet not found")
	case errors.Is(err, port.ErrAccountNoConflict):
		response.Error(w, http.StatusConflict, "account_no_conflict", "account number already exists")
	case errors.Is(err, port.ErrWalletVersionConflict):
		response.Error(w, http.StatusConflict, "version_conflict", "wallet version conflict")
	case errors.Is(err, port.ErrWalletInvalidState):
		response.Error(w, http.StatusConflict, "invalid_state", "operation is not allowed for current wallet state")
	default:
		response.Error(w, http.StatusInternalServerError, "internal", "wallet operation failed")
	}
}
