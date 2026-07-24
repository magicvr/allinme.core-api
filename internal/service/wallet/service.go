package wallet

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/magicvr/allinme.core-api/internal/domain"
	"github.com/magicvr/allinme.core-api/internal/port"
)

// Service implements wallet use cases using only the WalletRepository port.
type Service struct {
	repository port.WalletRepository
	now        func() time.Time
	newID      func() string
}

// New constructs a Wallet service with production clock and ID generation.
func New(repository port.WalletRepository) *Service {
	return NewWithDependencies(repository, time.Now, newWalletID)
}

// NewWithDependencies constructs a Wallet service with testable time and ID sources.
func NewWithDependencies(repository port.WalletRepository, now func() time.Time, newID func() string) *Service {
	if repository == nil || now == nil || newID == nil {
		panic("wallet.Service: nil dependency")
	}
	return &Service{repository: repository, now: now, newID: newID}
}

// CreateInput contains the fields accepted on wallet creation.
type CreateInput struct {
	AccountNo    string
	OwnerName    string
	BalanceCents int64
	Currency     string
}

// UpdateInput contains the owner metadata mutable by PUT in this slice.
type UpdateInput struct {
	Version   int64
	OwnerName string
}

// List returns a paginated wallet list.
func (s *Service) List(ctx context.Context, filter port.WalletListFilter) ([]domain.Wallet, int, error) {
	if filter.Page < 1 || filter.PageSize < 1 || filter.PageSize > 100 {
		return nil, 0, port.ErrWalletInvalidArgument
	}
	maxInt := int(^uint(0) >> 1)
	if filter.Page > 1 && filter.Page-1 > maxInt/filter.PageSize {
		return nil, 0, port.ErrWalletInvalidArgument
	}
	if filter.Status != "" && !domain.IsKnownWalletStatus(filter.Status) {
		return nil, 0, port.ErrWalletInvalidArgument
	}
	wallets, total, err := s.repository.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("wallet list: %w", err)
	}
	return wallets, total, nil
}

// Get returns one wallet.
func (s *Service) Get(ctx context.Context, id string) (domain.Wallet, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.Wallet{}, port.ErrWalletInvalidArgument
	}
	wallet, err := s.repository.Get(ctx, id)
	if err != nil {
		return domain.Wallet{}, fmt.Errorf("wallet get: %w", err)
	}
	return wallet, nil
}

// Create creates an active wallet with immutable account and balance fields.
func (s *Service) Create(ctx context.Context, input CreateInput) (domain.Wallet, error) {
	accountNo := strings.TrimSpace(input.AccountNo)
	ownerName := strings.TrimSpace(input.OwnerName)
	if accountNo == "" || ownerName == "" || input.BalanceCents < 0 {
		return domain.Wallet{}, port.ErrWalletInvalidArgument
	}
	currency, err := normalizeCurrency(input.Currency)
	if err != nil {
		return domain.Wallet{}, err
	}
	now := s.now().UTC()
	wallet := domain.Wallet{
		ID:           s.newID(),
		AccountNo:    accountNo,
		OwnerName:    ownerName,
		BalanceCents: input.BalanceCents,
		Currency:     currency,
		Status:       domain.WalletStatusActive,
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if wallet.ID == "" {
		return domain.Wallet{}, fmt.Errorf("wallet create: empty generated ID")
	}
	if err := s.repository.Create(ctx, wallet); err != nil {
		return domain.Wallet{}, fmt.Errorf("wallet create: %w", err)
	}
	return wallet, nil
}

// Update changes only the wallet owner name under optimistic locking.
func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (domain.Wallet, error) {
	id = strings.TrimSpace(id)
	ownerName := strings.TrimSpace(input.OwnerName)
	if id == "" || input.Version < 1 || ownerName == "" {
		return domain.Wallet{}, port.ErrWalletInvalidArgument
	}
	wallet, err := s.Get(ctx, id)
	if err != nil {
		return domain.Wallet{}, err
	}
	if wallet.Version != input.Version {
		return domain.Wallet{}, port.ErrWalletVersionConflict
	}
	now := s.now().UTC()
	if err := s.repository.UpdateOwner(ctx, id, input.Version, ownerName, now); err != nil {
		return domain.Wallet{}, fmt.Errorf("wallet update: %w", err)
	}
	wallet.OwnerName = ownerName
	wallet.Version++
	wallet.UpdatedAt = now
	return wallet, nil
}

// Freeze changes an active wallet to frozen under optimistic locking.
func (s *Service) Freeze(ctx context.Context, id string, version int64) (domain.Wallet, error) {
	return s.changeStatus(ctx, id, version, domain.WalletStatusActive, domain.WalletStatusFrozen)
}

// Unfreeze changes a frozen wallet to active under optimistic locking.
func (s *Service) Unfreeze(ctx context.Context, id string, version int64) (domain.Wallet, error) {
	return s.changeStatus(ctx, id, version, domain.WalletStatusFrozen, domain.WalletStatusActive)
}

func (s *Service) changeStatus(ctx context.Context, id string, version int64, from, to domain.WalletStatus) (domain.Wallet, error) {
	id = strings.TrimSpace(id)
	if id == "" || version < 1 {
		return domain.Wallet{}, port.ErrWalletInvalidArgument
	}
	wallet, err := s.Get(ctx, id)
	if err != nil {
		return domain.Wallet{}, err
	}
	if wallet.Version != version {
		return domain.Wallet{}, port.ErrWalletVersionConflict
	}
	if wallet.Status != from {
		return domain.Wallet{}, port.ErrWalletInvalidState
	}
	now := s.now().UTC()
	if err := s.repository.ChangeStatus(ctx, id, version, from, to, now); err != nil {
		return domain.Wallet{}, fmt.Errorf("wallet change status: %w", err)
	}
	wallet.Status = to
	wallet.Version++
	wallet.UpdatedAt = now
	return wallet, nil
}

// BatchFreeze freezes active wallets atomically.
func (s *Service) BatchFreeze(ctx context.Context, ids []string) (int, error) {
	if len(ids) == 0 || len(ids) > 100 {
		return 0, port.ErrWalletInvalidArgument
	}
	normalized := make([]string, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for i, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			return 0, port.ErrWalletInvalidArgument
		}
		if _, exists := seen[id]; exists {
			return 0, port.ErrWalletInvalidArgument
		}
		seen[id] = struct{}{}
		normalized[i] = id
	}
	if err := s.repository.BatchFreeze(ctx, normalized, s.now().UTC()); err != nil {
		return 0, fmt.Errorf("wallet batch freeze: %w", err)
	}
	return len(normalized), nil
}

func normalizeCurrency(value string) (string, error) {
	currency := strings.ToUpper(strings.TrimSpace(value))
	if currency == "" {
		return "CNY", nil
	}
	if len(currency) != 3 {
		return "", port.ErrWalletInvalidArgument
	}
	for i := range len(currency) {
		if currency[i] < 'A' || currency[i] > 'Z' {
			return "", port.ErrWalletInvalidArgument
		}
	}
	return currency, nil
}

func newWalletID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return ""
	}
	return "wal_" + hex.EncodeToString(bytes[:])
}
