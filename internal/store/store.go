package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type OpenMode string

const (
	OpenExisting OpenMode = "rw"
	OpenCreate   OpenMode = "rwc"
)

type DB struct {
	sql *sql.DB
}

func Open(ctx context.Context, path string, mode OpenMode) (*DB, error) {
	if mode != OpenExisting && mode != OpenCreate {
		return nil, fmt.Errorf("unsupported SQLite open mode %q", mode)
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve database path: %w", err)
	}
	if mode == OpenCreate {
		if err := os.MkdirAll(filepath.Dir(absolutePath), 0o750); err != nil {
			return nil, fmt.Errorf("create data directory: %w", err)
		}
	}

	dsn := sqliteDSN(absolutePath, mode)
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open SQLite: %w", err)
	}
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)
	if err := database.PingContext(ctx); err != nil {
		database.Close()
		return nil, fmt.Errorf("ping SQLite: %w", err)
	}
	if err := verifyPragmas(ctx, database); err != nil {
		database.Close()
		return nil, err
	}
	return &DB{sql: database}, nil
}

func sqliteDSN(path string, mode OpenMode) string {
	uriPath := filepath.ToSlash(path)
	if filepath.VolumeName(path) != "" {
		uriPath = "/" + uriPath
	}
	uri := url.URL{Scheme: "file", Path: uriPath}
	query := url.Values{}
	query.Set("mode", string(mode))
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "journal_mode(WAL)")
	uri.RawQuery = query.Encode()
	return uri.String()
}

func verifyPragmas(ctx context.Context, database *sql.DB) error {
	connection, err := database.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire SQLite connection: %w", err)
	}
	defer connection.Close()

	var foreignKeys int
	var busyTimeout int
	var journalMode string
	if err := connection.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		return fmt.Errorf("read foreign_keys pragma: %w", err)
	}
	if err := connection.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		return fmt.Errorf("read busy_timeout pragma: %w", err)
	}
	if err := connection.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode); err != nil {
		return fmt.Errorf("read journal_mode pragma: %w", err)
	}
	if foreignKeys != 1 || busyTimeout != 5000 || journalMode != "wal" {
		return fmt.Errorf("unexpected SQLite pragmas: foreign_keys=%d busy_timeout=%d journal_mode=%s", foreignKeys, busyTimeout, journalMode)
	}
	return nil
}

func (database *DB) Close() error {
	if database == nil || database.sql == nil {
		return nil
	}
	return database.sql.Close()
}

func (database *DB) SQL() *sql.DB {
	return database.sql
}

func (database *DB) WithTx(ctx context.Context, callback func(*sql.Tx) error) error {
	transaction, err := database.sql.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if err := callback(transaction); err != nil {
		if rollbackErr := transaction.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			return errors.Join(err, fmt.Errorf("rollback transaction: %w", rollbackErr))
		}
		return err
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
