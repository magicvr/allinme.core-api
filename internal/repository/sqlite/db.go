package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (GOAL-003 I-003)
)

// Open opens (or creates) a SQLite database at path and runs migrations.
func Open(path string) (*sql.DB, error) {
	if path == "" {
		return nil, fmt.Errorf("sqlite: empty path")
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("sqlite: mkdir %s: %w", dir, err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open: %w", err)
	}
	// SQLite and modernc: single writer; limit open conns for demo simplicity.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite: ping: %w", err)
	}
	if err := migrate(context.Background(), db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS meta (
	key   TEXT PRIMARY KEY NOT NULL,
	value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
	id            TEXT PRIMARY KEY NOT NULL,
	username      TEXT NOT NULL UNIQUE,
	name          TEXT NOT NULL,
	password_hash TEXT NOT NULL,
	roles_json    TEXT NOT NULL DEFAULT '[]'
);

CREATE TABLE IF NOT EXISTS orders (
	id            TEXT PRIMARY KEY NOT NULL,
	order_no      TEXT NOT NULL UNIQUE,
	customer_name TEXT NOT NULL,
	status        TEXT NOT NULL,
	amount_cents  INTEGER NOT NULL,
	currency      TEXT NOT NULL,
	remark        TEXT NOT NULL DEFAULT '',
	version       INTEGER NOT NULL,
	created_at    TEXT NOT NULL,
	updated_at    TEXT NOT NULL
);
`
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("sqlite: migrate: %w", err)
	}
	return nil
}
