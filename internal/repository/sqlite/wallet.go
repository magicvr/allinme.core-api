package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/magicvr/allinme.core-api/internal/domain"
	"github.com/magicvr/allinme.core-api/internal/port"
)

// WalletRepository is the SQLite implementation of port.WalletRepository.
type WalletRepository struct {
	db *sql.DB
}

// NewWalletRepository wraps db.
func NewWalletRepository(db *sql.DB) *WalletRepository {
	return &WalletRepository{db: db}
}

const walletTimestampLayout = "2006-01-02T15:04:05.000000000Z"

func walletTimestamp(value time.Time) string {
	return value.UTC().Format(walletTimestampLayout)
}

// Create implements port.WalletRepository.
func (r *WalletRepository) Create(ctx context.Context, wallet domain.Wallet) error {
	_, err := r.db.ExecContext(ctx, `
	INSERT INTO wallets (id, account_no, owner_name, balance_cents, currency, status, version, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, wallet.ID, wallet.AccountNo, wallet.OwnerName, wallet.BalanceCents, wallet.Currency, wallet.Status, wallet.Version, walletTimestamp(wallet.CreatedAt), walletTimestamp(wallet.UpdatedAt))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: wallets.account_no") {
			return port.ErrAccountNoConflict
		}
		return fmt.Errorf("wallet create: %w", err)
	}
	return nil
}

// Get implements port.WalletRepository.
func (r *WalletRepository) Get(ctx context.Context, id string) (domain.Wallet, error) {
	row := r.db.QueryRowContext(ctx, walletSelect+` WHERE id = ?`, id)
	return scanWallet(row)
}

// List implements port.WalletRepository.
func (r *WalletRepository) List(ctx context.Context, filter port.WalletListFilter) ([]domain.Wallet, int, error) {
	maxInt := int(^uint(0) >> 1)
	if filter.Page < 1 || filter.PageSize < 1 || filter.PageSize > 100 || (filter.Page > 1 && filter.Page-1 > maxInt/filter.PageSize) {
		return nil, 0, port.ErrWalletInvalidArgument
	}
	if filter.Status != "" && !domain.IsKnownWalletStatus(filter.Status) {
		return nil, 0, port.ErrWalletInvalidArgument
	}
	where, args := walletFilter(filter)
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM wallets`+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("wallet list count: %w", err)
	}
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := r.db.QueryContext(ctx, walletSelect+where+` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("wallet list query: %w", err)
	}
	defer rows.Close()

	wallets := make([]domain.Wallet, 0)
	for rows.Next() {
		wallet, err := scanWallet(rows)
		if err != nil {
			return nil, 0, err
		}
		wallets = append(wallets, wallet)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("wallet list rows: %w", err)
	}
	return wallets, total, nil
}

// UpdateOwner implements an owner-only wallet compare-and-swap update.
func (r *WalletRepository) UpdateOwner(ctx context.Context, id string, version int64, ownerName string, updatedAt time.Time) error {
	result, err := r.db.ExecContext(ctx, `
	UPDATE wallets
	SET owner_name = ?, version = version + 1, updated_at = ?
	WHERE id = ? AND version = ?
	`, ownerName, walletTimestamp(updatedAt), id, version)
	if err != nil {
		return fmt.Errorf("wallet update owner: %w", err)
	}
	return r.classifyWalletWrite(ctx, result, id, version, "")
}

// ChangeStatus implements a wallet state compare-and-swap transition.
func (r *WalletRepository) ChangeStatus(ctx context.Context, id string, version int64, from, to domain.WalletStatus, updatedAt time.Time) error {
	result, err := r.db.ExecContext(ctx, `
	UPDATE wallets
	SET status = ?, version = version + 1, updated_at = ?
	WHERE id = ? AND version = ? AND status = ?
	`, to, walletTimestamp(updatedAt), id, version, from)
	if err != nil {
		return fmt.Errorf("wallet change status: %w", err)
	}
	return r.classifyWalletWrite(ctx, result, id, version, from)
}

// BatchFreeze validates all targets then freezes them in one transaction.
func (r *WalletRepository) BatchFreeze(ctx context.Context, ids []string, updatedAt time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("wallet batch freeze begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, id := range ids {
		var status domain.WalletStatus
		err := tx.QueryRowContext(ctx, `SELECT status FROM wallets WHERE id = ?`, id).Scan(&status)
		if errors.Is(err, sql.ErrNoRows) {
			return port.ErrWalletNotFound
		}
		if err != nil {
			return fmt.Errorf("wallet batch freeze check: %w", err)
		}
		if status != domain.WalletStatusActive {
			return port.ErrWalletInvalidState
		}
	}

	for _, id := range ids {
		result, err := tx.ExecContext(ctx, `
		UPDATE wallets
		SET status = ?, version = version + 1, updated_at = ?
		WHERE id = ? AND status = ?
		`, domain.WalletStatusFrozen, walletTimestamp(updatedAt), id, domain.WalletStatusActive)
		if err != nil {
			return fmt.Errorf("wallet batch freeze update: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("wallet batch freeze rows affected: %w", err)
		}
		if affected != 1 {
			return port.ErrWalletInvalidState
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("wallet batch freeze commit: %w", err)
	}
	return nil
}

// Count implements port.WalletRepository.
func (r *WalletRepository) Count(ctx context.Context) (int, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM wallets`).Scan(&count); err != nil {
		return 0, fmt.Errorf("wallet count: %w", err)
	}
	return count, nil
}

func (r *WalletRepository) classifyWalletWrite(ctx context.Context, result sql.Result, id string, version int64, expectedStatus domain.WalletStatus) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("wallet rows affected: %w", err)
	}
	if affected == 1 {
		return nil
	}

	var currentVersion int64
	var currentStatus domain.WalletStatus
	err = r.db.QueryRowContext(ctx, `SELECT version, status FROM wallets WHERE id = ?`, id).Scan(&currentVersion, &currentStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return port.ErrWalletNotFound
	}
	if err != nil {
		return fmt.Errorf("wallet classify write: %w", err)
	}
	if currentVersion != version {
		return port.ErrWalletVersionConflict
	}
	if expectedStatus != "" && currentStatus != expectedStatus {
		return port.ErrWalletInvalidState
	}
	return port.ErrWalletVersionConflict
}

const walletSelect = `
	SELECT id, account_no, owner_name, balance_cents, currency, status, version, created_at, updated_at FROM wallets`

func walletFilter(filter port.WalletListFilter) (string, []any) {
	clauses := make([]string, 0, 2)
	args := make([]any, 0, 3)
	if filter.Status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Query != "" {
		clauses = append(clauses, `(account_no LIKE ? ESCAPE '\' OR owner_name LIKE ? ESCAPE '\')`)
		query := "%" + escapeWalletLike(filter.Query) + "%"
		args = append(args, query, query)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func escapeWalletLike(value string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
}

type walletScannable interface {
	Scan(dest ...any) error
}

func scanWallet(row walletScannable) (domain.Wallet, error) {
	var wallet domain.Wallet
	var createdAt, updatedAt string
	err := row.Scan(&wallet.ID, &wallet.AccountNo, &wallet.OwnerName, &wallet.BalanceCents, &wallet.Currency, &wallet.Status, &wallet.Version, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Wallet{}, port.ErrWalletNotFound
	}
	if err != nil {
		return domain.Wallet{}, fmt.Errorf("wallet scan: %w", err)
	}
	wallet.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return domain.Wallet{}, fmt.Errorf("wallet parse created at: %w", err)
	}
	wallet.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return domain.Wallet{}, fmt.Errorf("wallet parse updated at: %w", err)
	}
	return wallet, nil
}

var _ port.WalletRepository = (*WalletRepository)(nil)
