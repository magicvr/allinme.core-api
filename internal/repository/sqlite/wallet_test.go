package sqlite_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/domain"
	"github.com/magicvr/allinme.core-api/internal/port"
	"github.com/magicvr/allinme.core-api/internal/repository/sqlite"
)

func TestSeedWalletsRollsBackOnInsertFailure(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "wallet-seed-rollback.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository := sqlite.NewWalletRepository(db)

	if _, err := db.Exec(`
CREATE TRIGGER fail_wallet_seed
BEFORE INSERT ON wallets
WHEN NEW.account_no = 'WAL-1002'
BEGIN
	SELECT RAISE(ABORT, 'forced wallet seed failure');
END;`); err != nil {
		t.Fatal(err)
	}
	if err := sqlite.SeedWallets(ctx, repository); err == nil {
		t.Fatal("SeedWallets succeeded despite forced insert failure")
	}
	if count, err := repository.Count(ctx); err != nil || count != 0 {
		t.Fatalf("failed seed count = %d, err=%v; want 0", count, err)
	}
	if _, err := db.Exec(`DROP TRIGGER fail_wallet_seed`); err != nil {
		t.Fatal(err)
	}
	if err := sqlite.SeedWallets(ctx, repository); err != nil {
		t.Fatal(err)
	}
	if count, err := repository.Count(ctx); err != nil || count != 2 {
		t.Fatalf("retry seed count = %d, err=%v; want 2", count, err)
	}
}

func TestWalletRepositoryListCASBatchAndSeed(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "wallets.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository := sqlite.NewWalletRepository(db)

	if err := sqlite.SeedWallets(ctx, repository); err != nil {
		t.Fatal(err)
	}
	if err := sqlite.SeedWallets(ctx, repository); err != nil {
		t.Fatal(err)
	}
	if count, err := repository.Count(ctx); err != nil || count != 2 {
		t.Fatalf("seed count = %d, err=%v; want 2", count, err)
	}
	seeded, total, err := repository.List(ctx, port.WalletListFilter{Page: 1, PageSize: 20})
	if err != nil || total != 2 || len(seeded) != 2 {
		t.Fatalf("seed list = %+v total=%d err=%v", seeded, total, err)
	}
	seededStates := make(map[domain.WalletStatus]bool)
	for _, wallet := range seeded {
		seededStates[wallet.Status] = true
	}
	for _, status := range []domain.WalletStatus{domain.WalletStatusActive, domain.WalletStatusFrozen} {
		if !seededStates[status] {
			t.Fatalf("missing seeded %s wallet", status)
		}
	}
	active, total, err := repository.List(ctx, port.WalletListFilter{Status: domain.WalletStatusActive, Page: 1, PageSize: 20})
	if err != nil || total != 1 || len(active) != 1 || active[0].ID != "wal_seed_active" {
		t.Fatalf("active list = %+v total=%d err=%v", active, total, err)
	}
	matched, total, err := repository.List(ctx, port.WalletListFilter{Query: "Frozen Owner", Page: 1, PageSize: 20})
	if err != nil || total != 1 || len(matched) != 1 || matched[0].ID != "wal_seed_frozen" {
		t.Fatalf("owner query = %+v total=%d err=%v", matched, total, err)
	}

	base := time.Date(2026, time.July, 25, 6, 0, 0, 0, time.UTC)
	for _, wallet := range []domain.Wallet{
		{ID: "wal_percent", AccountNo: "ACC-%", OwnerName: "Percent % Owner", BalanceCents: 100, Currency: "CNY", Status: domain.WalletStatusActive, Version: 1, CreatedAt: base, UpdatedAt: base},
		{ID: "wal_underscore", AccountNo: "ACC_UNDER", OwnerName: "Underscore Owner", BalanceCents: 200, Currency: "CNY", Status: domain.WalletStatusActive, Version: 1, CreatedAt: base.Add(time.Minute), UpdatedAt: base.Add(time.Minute)},
	} {
		if err := repository.Create(ctx, wallet); err != nil {
			t.Fatal(err)
		}
	}
	percent, total, err := repository.List(ctx, port.WalletListFilter{Query: "%", Page: 1, PageSize: 20})
	if err != nil || total != 1 || len(percent) != 1 || percent[0].ID != "wal_percent" {
		t.Fatalf("literal percent query = %+v total=%d err=%v", percent, total, err)
	}
	underscore, total, err := repository.List(ctx, port.WalletListFilter{Query: "_", Page: 1, PageSize: 20})
	if err != nil || total != 1 || len(underscore) != 1 || underscore[0].ID != "wal_underscore" {
		t.Fatalf("literal underscore query = %+v total=%d err=%v", underscore, total, err)
	}

	sameSecond := time.Date(2026, time.July, 25, 7, 0, 0, 0, time.UTC)
	for _, wallet := range []domain.Wallet{
		{ID: "wal_same_zero", AccountNo: "SAME-0", OwnerName: "SameSecond", Currency: "CNY", Status: domain.WalletStatusActive, Version: 1, CreatedAt: sameSecond, UpdatedAt: sameSecond},
		{ID: "wal_same_fraction", AccountNo: "SAME-100MS", OwnerName: "SameSecond", Currency: "CNY", Status: domain.WalletStatusActive, Version: 1, CreatedAt: sameSecond.Add(100 * time.Millisecond), UpdatedAt: sameSecond.Add(100 * time.Millisecond)},
	} {
		if err := repository.Create(ctx, wallet); err != nil {
			t.Fatal(err)
		}
	}
	same, total, err := repository.List(ctx, port.WalletListFilter{Query: "SameSecond", Page: 1, PageSize: 20})
	if err != nil || total != 2 || len(same) != 2 || same[0].ID != "wal_same_fraction" || same[1].ID != "wal_same_zero" {
		t.Fatalf("same-second order = %+v total=%d err=%v", same, total, err)
	}
	var storedTimestamp string
	if err := db.QueryRow(`SELECT created_at FROM wallets WHERE id = 'wal_same_zero'`).Scan(&storedTimestamp); err != nil {
		t.Fatal(err)
	}
	if storedTimestamp != "2026-07-25T07:00:00.000000000Z" {
		t.Fatalf("stored timestamp = %q", storedTimestamp)
	}

	pageOne, total, err := repository.List(ctx, port.WalletListFilter{Page: 1, PageSize: 1})
	if err != nil || total != 6 || len(pageOne) != 1 {
		t.Fatalf("page one = %+v total=%d err=%v", pageOne, total, err)
	}
	pageTwo, _, err := repository.List(ctx, port.WalletListFilter{Page: 2, PageSize: 1})
	if err != nil || len(pageTwo) != 1 || pageOne[0].ID == pageTwo[0].ID {
		t.Fatalf("page two = %+v err=%v", pageTwo, err)
	}
	maxInt := int(^uint(0) >> 1)
	if _, _, err := repository.List(ctx, port.WalletListFilter{Page: maxInt, PageSize: 2}); !errors.Is(err, port.ErrWalletInvalidArgument) {
		t.Fatalf("overflow list error = %v", err)
	}
	if _, _, err := repository.List(ctx, port.WalletListFilter{Status: "closed", Page: 1, PageSize: 20}); !errors.Is(err, port.ErrWalletInvalidArgument) {
		t.Fatalf("invalid status error = %v", err)
	}

	duplicate := active[0]
	duplicate.ID = "wal_duplicate"
	if err := repository.Create(ctx, duplicate); !errors.Is(err, port.ErrAccountNoConflict) {
		t.Fatalf("duplicate account error = %v", err)
	}
	if _, err := repository.Get(ctx, "missing"); !errors.Is(err, port.ErrWalletNotFound) {
		t.Fatalf("missing wallet error = %v", err)
	}

	original := active[0]
	updatedAt := base.Add(2 * time.Hour)
	if err := repository.UpdateOwner(ctx, original.ID, 1, "Changed Owner", updatedAt); err != nil {
		t.Fatal(err)
	}
	updated, err := repository.Get(ctx, original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.OwnerName != "Changed Owner" || updated.Version != 2 || !updated.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("updated wallet = %+v", updated)
	}
	if updated.AccountNo != original.AccountNo || updated.BalanceCents != original.BalanceCents || updated.Currency != original.Currency || updated.Status != original.Status || !updated.CreatedAt.Equal(original.CreatedAt) {
		t.Fatalf("immutable fields changed = %+v", updated)
	}
	if err := repository.UpdateOwner(ctx, original.ID, 1, "Stale", updatedAt); !errors.Is(err, port.ErrWalletVersionConflict) {
		t.Fatalf("stale owner update error = %v", err)
	}
	if err := repository.UpdateOwner(ctx, "missing", 1, "Missing", updatedAt); !errors.Is(err, port.ErrWalletNotFound) {
		t.Fatalf("missing owner update error = %v", err)
	}
	if err := repository.UpdateOwner(ctx, "wal_seed_frozen", 1, "Frozen Changed", updatedAt); err != nil {
		t.Fatal(err)
	}
	frozenUpdated, err := repository.Get(ctx, "wal_seed_frozen")
	if err != nil {
		t.Fatal(err)
	}
	if frozenUpdated.OwnerName != "Frozen Changed" || frozenUpdated.Status != domain.WalletStatusFrozen || frozenUpdated.Version != 2 || frozenUpdated.BalanceCents != 8800 || frozenUpdated.Currency != "USD" {
		t.Fatalf("updated frozen wallet = %+v", frozenUpdated)
	}

	if err := repository.ChangeStatus(ctx, original.ID, 2, domain.WalletStatusActive, domain.WalletStatusFrozen, updatedAt.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := repository.ChangeStatus(ctx, original.ID, 2, domain.WalletStatusActive, domain.WalletStatusFrozen, updatedAt.Add(2*time.Hour)); !errors.Is(err, port.ErrWalletVersionConflict) {
		t.Fatalf("stale status error = %v", err)
	}
	if err := repository.ChangeStatus(ctx, original.ID, 3, domain.WalletStatusActive, domain.WalletStatusFrozen, updatedAt.Add(2*time.Hour)); !errors.Is(err, port.ErrWalletInvalidState) {
		t.Fatalf("invalid state error = %v", err)
	}
	if err := repository.ChangeStatus(ctx, original.ID, 3, domain.WalletStatusFrozen, domain.WalletStatusActive, updatedAt.Add(3*time.Hour)); err != nil {
		t.Fatal(err)
	}

	batchCandidate := domain.Wallet{ID: "wal_batch", AccountNo: "BATCH-1", OwnerName: "Batch", Currency: "CNY", Status: domain.WalletStatusActive, Version: 1, CreatedAt: base, UpdatedAt: base}
	if err := repository.Create(ctx, batchCandidate); err != nil {
		t.Fatal(err)
	}
	if err := repository.BatchFreeze(ctx, []string{original.ID, "wal_seed_frozen"}, updatedAt.Add(4*time.Hour)); !errors.Is(err, port.ErrWalletInvalidState) {
		t.Fatalf("mixed batch error = %v", err)
	}
	originalAfterRollback, err := repository.Get(ctx, original.ID)
	if err != nil || originalAfterRollback.Status != domain.WalletStatusActive {
		t.Fatalf("wallet changed despite rollback = %+v err=%v", originalAfterRollback, err)
	}
	if err := repository.BatchFreeze(ctx, []string{original.ID, "missing"}, updatedAt.Add(4*time.Hour)); !errors.Is(err, port.ErrWalletNotFound) {
		t.Fatalf("missing batch error = %v", err)
	}
	if err := repository.BatchFreeze(ctx, []string{original.ID, batchCandidate.ID}, updatedAt.Add(5*time.Hour)); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{original.ID, batchCandidate.ID} {
		wallet, err := repository.Get(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if wallet.Status != domain.WalletStatusFrozen || !wallet.UpdatedAt.Equal(updatedAt.Add(5*time.Hour)) {
			t.Fatalf("batch frozen %s = %+v", id, wallet)
		}
	}
}
