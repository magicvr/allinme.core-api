package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const (
	runtimeSeedName    = "runtime"
	runtimeSeedVersion = 1
)

type SeedResult struct {
	Name        string
	FromVersion int
	ToVersion   int
}

func (database *DB) Seed(ctx context.Context) (SeedResult, error) {
	result := SeedResult{Name: runtimeSeedName, ToVersion: runtimeSeedVersion}
	err := database.WithTx(ctx, func(transaction *sql.Tx) error {
		var currentVersion int
		err := transaction.QueryRowContext(ctx, "SELECT version FROM seed_versions WHERE name = ?", runtimeSeedName).Scan(&currentVersion)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("read seed version %q: %w", runtimeSeedName, err)
		}
		result.FromVersion = currentVersion
		if currentVersion > runtimeSeedVersion {
			return fmt.Errorf("seed %q version %d is newer than supported version %d", runtimeSeedName, currentVersion, runtimeSeedVersion)
		}
		if currentVersion == runtimeSeedVersion {
			return nil
		}
		_, err = transaction.ExecContext(ctx, `
			INSERT INTO seed_versions(name, version, applied_at) VALUES (?, ?, ?)
			ON CONFLICT(name) DO UPDATE SET version = excluded.version, applied_at = excluded.applied_at
		`, runtimeSeedName, runtimeSeedVersion, time.Now().UTC().Format(time.RFC3339))
		if err != nil {
			return fmt.Errorf("apply seed %q: %w", runtimeSeedName, err)
		}
		return nil
	})
	if err != nil {
		return SeedResult{}, err
	}
	return result, nil
}
