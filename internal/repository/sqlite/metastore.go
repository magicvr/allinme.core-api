package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/magicvr/allinme.core-api/internal/port"
)

// MetaStore is the SQLite implementation of port.MetaStore.
type MetaStore struct {
	db *sql.DB
}

// NewMetaStore wraps an open *sql.DB.
func NewMetaStore(db *sql.DB) *MetaStore {
	return &MetaStore{db: db}
}

// Ping implements port.MetaStore.
func (s *MetaStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Get implements port.MetaStore.
func (s *MetaStore) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", port.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("sqlite meta get: %w", err)
	}
	return value, nil
}

// Set implements port.MetaStore.
func (s *MetaStore) Set(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO meta(key, value) VALUES(?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value
`, key, value)
	if err != nil {
		return fmt.Errorf("sqlite meta set: %w", err)
	}
	return nil
}

var _ port.MetaStore = (*MetaStore)(nil)
