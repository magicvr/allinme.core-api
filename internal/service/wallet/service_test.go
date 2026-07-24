package wallet_test

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/domain"
	"github.com/magicvr/allinme.core-api/internal/port"
	walletservice "github.com/magicvr/allinme.core-api/internal/service/wallet"
)

func TestServiceCreateDefaultsAndValidation(t *testing.T) {
	repository := newFakeWalletRepository()
	now := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	service := walletservice.NewWithDependencies(repository, func() time.Time { return now }, func() string { return "wal_test" })

	created, err := service.Create(context.Background(), walletservice.CreateInput{
		AccountNo: "  ACC-001  ", OwnerName: "  Alice  ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != "wal_test" || created.AccountNo != "ACC-001" || created.OwnerName != "Alice" {
		t.Fatalf("created wallet = %+v", created)
	}
	if created.BalanceCents != 0 || created.Currency != "CNY" || created.Status != domain.WalletStatusActive || created.Version != 1 {
		t.Fatalf("created defaults = %+v", created)
	}
	if !created.CreatedAt.Equal(now) || !created.UpdatedAt.Equal(now) {
		t.Fatalf("created timestamps = %+v", created)
	}

	currencyRepository := newFakeWalletRepository()
	currencyService := walletservice.NewWithDependencies(currencyRepository, func() time.Time { return now }, func() string { return "wal_currency" })
	customCurrency, err := currencyService.Create(context.Background(), walletservice.CreateInput{
		AccountNo: "ACC-USD", OwnerName: "USD Owner", BalanceCents: 500, Currency: " usd ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if customCurrency.Currency != "USD" || customCurrency.BalanceCents != 500 {
		t.Fatalf("custom currency wallet = %+v", customCurrency)
	}

	if _, err := service.Create(context.Background(), walletservice.CreateInput{
		AccountNo: "ACC-001", OwnerName: "Duplicate", Currency: "usd",
	}); !errors.Is(err, port.ErrAccountNoConflict) {
		t.Fatalf("duplicate account error = %v", err)
	}
	for name, input := range map[string]walletservice.CreateInput{
		"empty account":    {OwnerName: "Alice"},
		"empty owner":      {AccountNo: "ACC-002"},
		"negative balance": {AccountNo: "ACC-002", OwnerName: "Alice", BalanceCents: -1},
		"bad currency":     {AccountNo: "ACC-002", OwnerName: "Alice", Currency: "C1Y"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := service.Create(context.Background(), input); !errors.Is(err, port.ErrWalletInvalidArgument) {
				t.Fatalf("error = %v, want invalid argument", err)
			}
		})
	}
}

func TestServiceUpdatePreservesImmutableFieldsAndAllowsFrozenWallet(t *testing.T) {
	repository := newFakeWalletRepository()
	createdAt := time.Date(2026, time.July, 25, 9, 0, 0, 0, time.UTC)
	now := createdAt.Add(time.Hour)
	repository.wallets["active"] = domain.Wallet{
		ID: "active", AccountNo: "ACC-A", OwnerName: "Before", BalanceCents: 1200,
		Currency: "CNY", Status: domain.WalletStatusActive, Version: 1, CreatedAt: createdAt, UpdatedAt: createdAt,
	}
	repository.wallets["frozen"] = domain.Wallet{
		ID: "frozen", AccountNo: "ACC-F", OwnerName: "Frozen Before", BalanceCents: 3400,
		Currency: "USD", Status: domain.WalletStatusFrozen, Version: 4, CreatedAt: createdAt, UpdatedAt: createdAt,
	}
	service := walletservice.NewWithDependencies(repository, func() time.Time { return now }, func() string { return "unused" })

	updated, err := service.Update(context.Background(), "active", walletservice.UpdateInput{Version: 1, OwnerName: "  After  "})
	if err != nil {
		t.Fatal(err)
	}
	if updated.OwnerName != "After" || updated.Version != 2 || !updated.UpdatedAt.Equal(now) {
		t.Fatalf("updated wallet = %+v", updated)
	}
	if updated.AccountNo != "ACC-A" || updated.BalanceCents != 1200 || updated.Currency != "CNY" || updated.Status != domain.WalletStatusActive || !updated.CreatedAt.Equal(createdAt) {
		t.Fatalf("immutable fields changed = %+v", updated)
	}
	if _, err := service.Update(context.Background(), "active", walletservice.UpdateInput{Version: 1, OwnerName: "Stale"}); !errors.Is(err, port.ErrWalletVersionConflict) {
		t.Fatalf("stale update error = %v", err)
	}

	frozen, err := service.Update(context.Background(), "frozen", walletservice.UpdateInput{Version: 4, OwnerName: "Frozen After"})
	if err != nil {
		t.Fatal(err)
	}
	if frozen.OwnerName != "Frozen After" || frozen.Status != domain.WalletStatusFrozen || frozen.Version != 5 || frozen.BalanceCents != 3400 {
		t.Fatalf("updated frozen wallet = %+v", frozen)
	}
}

func TestServiceFreezeUnfreezeStateAndVersionRules(t *testing.T) {
	repository := newFakeWalletRepository()
	now := time.Date(2026, time.July, 25, 14, 0, 0, 0, time.UTC)
	repository.wallets["wallet"] = domain.Wallet{ID: "wallet", AccountNo: "ACC-001", OwnerName: "Alice", Currency: "CNY", Status: domain.WalletStatusActive, Version: 1}
	service := walletservice.NewWithDependencies(repository, func() time.Time { return now }, func() string { return "unused" })

	frozen, err := service.Freeze(context.Background(), "wallet", 1)
	if err != nil {
		t.Fatal(err)
	}
	if frozen.Status != domain.WalletStatusFrozen || frozen.Version != 2 || !frozen.UpdatedAt.Equal(now) {
		t.Fatalf("frozen wallet = %+v", frozen)
	}
	if _, err := service.Freeze(context.Background(), "wallet", 2); !errors.Is(err, port.ErrWalletInvalidState) {
		t.Fatalf("freeze frozen error = %v", err)
	}

	active, err := service.Unfreeze(context.Background(), "wallet", 2)
	if err != nil {
		t.Fatal(err)
	}
	if active.Status != domain.WalletStatusActive || active.Version != 3 {
		t.Fatalf("unfrozen wallet = %+v", active)
	}
	if _, err := service.Unfreeze(context.Background(), "wallet", 3); !errors.Is(err, port.ErrWalletInvalidState) {
		t.Fatalf("unfreeze active error = %v", err)
	}
	if _, err := service.Freeze(context.Background(), "wallet", 2); !errors.Is(err, port.ErrWalletVersionConflict) {
		t.Fatalf("stale freeze error = %v", err)
	}
}

func TestServiceBatchFreezeIsAtomicAndValidatesIDs(t *testing.T) {
	repository := newFakeWalletRepository()
	now := time.Date(2026, time.July, 25, 15, 0, 0, 0, time.UTC)
	repository.wallets["active-a"] = domain.Wallet{ID: "active-a", Status: domain.WalletStatusActive, Version: 1}
	repository.wallets["active-b"] = domain.Wallet{ID: "active-b", Status: domain.WalletStatusActive, Version: 1}
	repository.wallets["frozen"] = domain.Wallet{ID: "frozen", Status: domain.WalletStatusFrozen, Version: 2}
	service := walletservice.NewWithDependencies(repository, func() time.Time { return now }, func() string { return "unused" })

	if _, err := service.BatchFreeze(context.Background(), []string{"active-a", "frozen"}); !errors.Is(err, port.ErrWalletInvalidState) {
		t.Fatalf("mixed batch error = %v", err)
	}
	if repository.wallets["active-a"].Status != domain.WalletStatusActive {
		t.Fatal("active wallet changed despite batch rollback")
	}
	for name, ids := range map[string][]string{
		"empty":     {},
		"blank":     {" "},
		"duplicate": {"active-a", "active-a"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := service.BatchFreeze(context.Background(), ids); !errors.Is(err, port.ErrWalletInvalidArgument) {
				t.Fatalf("error = %v, want invalid argument", err)
			}
		})
	}
	tooMany := make([]string, 101)
	for i := range tooMany {
		tooMany[i] = "wallet"
	}
	if _, err := service.BatchFreeze(context.Background(), tooMany); !errors.Is(err, port.ErrWalletInvalidArgument) {
		t.Fatalf("too many IDs error = %v", err)
	}

	ids := []string{" active-a ", "active-b"}
	count, err := service.BatchFreeze(context.Background(), ids)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 || ids[0] != " active-a " {
		t.Fatalf("count=%d ids=%v", count, ids)
	}
	for _, id := range []string{"active-a", "active-b"} {
		wallet := repository.wallets[id]
		if wallet.Status != domain.WalletStatusFrozen || wallet.Version != 2 || !wallet.UpdatedAt.Equal(now) {
			t.Fatalf("batch frozen %s = %+v", id, wallet)
		}
	}
}

func TestServiceListAndGetValidateInputs(t *testing.T) {
	repository := newFakeWalletRepository()
	repository.wallets["a"] = domain.Wallet{ID: "a", AccountNo: "ACC-A", OwnerName: "Alice", Status: domain.WalletStatusActive}
	service := walletservice.NewWithDependencies(repository, time.Now, func() string { return "unused" })

	wallets, total, err := service.List(context.Background(), port.WalletListFilter{Status: domain.WalletStatusActive, Query: "Ali", Page: 1, PageSize: 20})
	if err != nil || total != 1 || len(wallets) != 1 || wallets[0].ID != "a" {
		t.Fatalf("list = %+v total=%d err=%v", wallets, total, err)
	}
	maxInt := int(^uint(0) >> 1)
	for name, filter := range map[string]port.WalletListFilter{
		"page":      {Page: 0, PageSize: 20},
		"page size": {Page: 1, PageSize: 101},
		"status":    {Status: "closed", Page: 1, PageSize: 20},
		"overflow":  {Page: maxInt, PageSize: 2},
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := service.List(context.Background(), filter); !errors.Is(err, port.ErrWalletInvalidArgument) {
				t.Fatalf("error = %v, want invalid argument", err)
			}
		})
	}
	if _, err := service.Get(context.Background(), " "); !errors.Is(err, port.ErrWalletInvalidArgument) {
		t.Fatalf("blank get error = %v", err)
	}
	if _, err := service.Get(context.Background(), "missing"); !errors.Is(err, port.ErrWalletNotFound) {
		t.Fatalf("missing get error = %v", err)
	}
}

type fakeWalletRepository struct {
	wallets map[string]domain.Wallet
}

func newFakeWalletRepository() *fakeWalletRepository {
	return &fakeWalletRepository{wallets: make(map[string]domain.Wallet)}
}

func (r *fakeWalletRepository) Create(_ context.Context, wallet domain.Wallet) error {
	for _, existing := range r.wallets {
		if existing.AccountNo == wallet.AccountNo {
			return port.ErrAccountNoConflict
		}
	}
	r.wallets[wallet.ID] = wallet
	return nil
}

func (r *fakeWalletRepository) Get(_ context.Context, id string) (domain.Wallet, error) {
	wallet, ok := r.wallets[id]
	if !ok {
		return domain.Wallet{}, port.ErrWalletNotFound
	}
	return wallet, nil
}

func (r *fakeWalletRepository) List(_ context.Context, filter port.WalletListFilter) ([]domain.Wallet, int, error) {
	wallets := make([]domain.Wallet, 0, len(r.wallets))
	for _, wallet := range r.wallets {
		if filter.Status != "" && wallet.Status != filter.Status {
			continue
		}
		if filter.Query != "" && !strings.Contains(wallet.AccountNo, filter.Query) && !strings.Contains(wallet.OwnerName, filter.Query) {
			continue
		}
		wallets = append(wallets, wallet)
	}
	sort.Slice(wallets, func(i, j int) bool { return wallets[i].ID < wallets[j].ID })
	return wallets, len(wallets), nil
}

func (r *fakeWalletRepository) UpdateOwner(_ context.Context, id string, version int64, ownerName string, updatedAt time.Time) error {
	wallet, ok := r.wallets[id]
	if !ok {
		return port.ErrWalletNotFound
	}
	if wallet.Version != version {
		return port.ErrWalletVersionConflict
	}
	wallet.OwnerName = ownerName
	wallet.Version++
	wallet.UpdatedAt = updatedAt
	r.wallets[id] = wallet
	return nil
}

func (r *fakeWalletRepository) ChangeStatus(_ context.Context, id string, version int64, from, to domain.WalletStatus, updatedAt time.Time) error {
	wallet, ok := r.wallets[id]
	if !ok {
		return port.ErrWalletNotFound
	}
	if wallet.Version != version {
		return port.ErrWalletVersionConflict
	}
	if wallet.Status != from {
		return port.ErrWalletInvalidState
	}
	wallet.Status = to
	wallet.Version++
	wallet.UpdatedAt = updatedAt
	r.wallets[id] = wallet
	return nil
}

func (r *fakeWalletRepository) BatchFreeze(_ context.Context, ids []string, updatedAt time.Time) error {
	for _, id := range ids {
		wallet, ok := r.wallets[id]
		if !ok {
			return port.ErrWalletNotFound
		}
		if wallet.Status != domain.WalletStatusActive {
			return port.ErrWalletInvalidState
		}
	}
	for _, id := range ids {
		wallet := r.wallets[id]
		wallet.Status = domain.WalletStatusFrozen
		wallet.Version++
		wallet.UpdatedAt = updatedAt
		r.wallets[id] = wallet
	}
	return nil
}

func (r *fakeWalletRepository) Count(_ context.Context) (int, error) {
	return len(r.wallets), nil
}

var _ port.WalletRepository = (*fakeWalletRepository)(nil)
