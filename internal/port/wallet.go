package port

import (
	"context"
	"errors"
	"time"

	"github.com/magicvr/allinme.core-api/internal/domain"
)

var (
	// ErrWalletNotFound is returned when a wallet cannot be located.
	ErrWalletNotFound = errors.New("wallet: not found")
	// ErrAccountNoConflict is returned when an account number is already used.
	ErrAccountNoConflict = errors.New("wallet: account number conflict")
	// ErrWalletVersionConflict is returned when a wallet compare-and-swap write is stale.
	ErrWalletVersionConflict = errors.New("wallet: version conflict")
	// ErrWalletInvalidState is returned when a wallet operation is not permitted in the current state.
	ErrWalletInvalidState = errors.New("wallet: invalid state")
	// ErrWalletInvalidArgument is returned when a wallet use-case input violates its contract.
	ErrWalletInvalidArgument = errors.New("wallet: invalid argument")
)

// WalletListFilter describes repository-side wallet filtering and pagination.
type WalletListFilter struct {
	Status   domain.WalletStatus
	Query    string
	Page     int
	PageSize int
}

// WalletRepository is the outbound persistence port for wallets.
type WalletRepository interface {
	Create(ctx context.Context, wallet domain.Wallet) error
	Get(ctx context.Context, id string) (domain.Wallet, error)
	List(ctx context.Context, filter WalletListFilter) ([]domain.Wallet, int, error)
	UpdateOwner(ctx context.Context, id string, version int64, ownerName string, updatedAt time.Time) error
	ChangeStatus(ctx context.Context, id string, version int64, from, to domain.WalletStatus, updatedAt time.Time) error
	BatchFreeze(ctx context.Context, ids []string, updatedAt time.Time) error
	Count(ctx context.Context) (int, error)
}
