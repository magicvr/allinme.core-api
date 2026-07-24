package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/magicvr/allinme.core-api/internal/domain"
	"github.com/magicvr/allinme.core-api/internal/port"
	walletservice "github.com/magicvr/allinme.core-api/internal/service/wallet"
)

func TestWalletInternalErrorDoesNotLeak(t *testing.T) {
	const sensitive = "sqlite: disk I/O error at C:\\secret\\wallets.db"
	handler := listWallets(failingWalletService{err: errors.New(sensitive)})
	req := httptest.NewRequest(http.MethodGet, "/v1/wallets?page=1&pageSize=20", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusInternalServerError, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"code":"internal"`) {
		t.Fatalf("missing internal error code: %s", rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), sensitive) || strings.Contains(rr.Body.String(), `C:\\secret`) {
		t.Fatalf("response leaked internal error: %s", rr.Body.String())
	}
}

type failingWalletService struct {
	err error
}

func (s failingWalletService) List(context.Context, port.WalletListFilter) ([]domain.Wallet, int, error) {
	return nil, 0, s.err
}

func (s failingWalletService) Get(context.Context, string) (domain.Wallet, error) {
	return domain.Wallet{}, s.err
}

func (s failingWalletService) Create(context.Context, walletservice.CreateInput) (domain.Wallet, error) {
	return domain.Wallet{}, s.err
}

func (s failingWalletService) Update(context.Context, string, walletservice.UpdateInput) (domain.Wallet, error) {
	return domain.Wallet{}, s.err
}

func (s failingWalletService) Freeze(context.Context, string, int64) (domain.Wallet, error) {
	return domain.Wallet{}, s.err
}

func (s failingWalletService) Unfreeze(context.Context, string, int64) (domain.Wallet, error) {
	return domain.Wallet{}, s.err
}

func (s failingWalletService) BatchFreeze(context.Context, []string) (int, error) {
	return 0, s.err
}

var _ walletService = failingWalletService{}
