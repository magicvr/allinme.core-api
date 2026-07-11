package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
)

func (database *DB) UserByUsername(ctx context.Context, username string) (auth.User, bool, error) {
	return scanUser(database.sql.QueryRowContext(ctx, `SELECT id, username, password_hash, role, disabled_at FROM users WHERE username = ?`, username))
}

func (database *DB) UserByID(ctx context.Context, id string) (auth.User, bool, error) {
	return scanUser(database.sql.QueryRowContext(ctx, `SELECT id, username, password_hash, role, disabled_at FROM users WHERE id = ?`, id))
}

type rowScanner interface {
	Scan(...any) error
}

func scanUser(row rowScanner) (auth.User, bool, error) {
	var user auth.User
	var role string
	var disabledAt sql.NullString
	if err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &role, &disabledAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auth.User{}, false, nil
		}
		return auth.User{}, false, fmt.Errorf("scan user: %w", err)
	}
	user.Role = auth.Role(role)
	if disabledAt.Valid {
		parsed, err := time.Parse(time.RFC3339, disabledAt.String)
		if err != nil {
			return auth.User{}, false, fmt.Errorf("parse disabled time: %w", err)
		}
		user.DisabledAt = &parsed
	}
	return user, true, nil
}

func (database *DB) CreateSession(ctx context.Context, session auth.Session) error {
	_, err := database.sql.ExecContext(ctx, `INSERT INTO sessions(id, user_id, token_id, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		session.ID, session.UserID, session.TokenID, session.ExpiresAt.UTC().Format(time.RFC3339), session.CreatedAt.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (database *DB) SessionByTokenID(ctx context.Context, tokenID string) (auth.Session, bool, error) {
	var session auth.Session
	var expiresAt string
	var revokedAt sql.NullString
	err := database.sql.QueryRowContext(ctx, `SELECT id, user_id, token_id, expires_at, revoked_at FROM sessions WHERE token_id = ?`, tokenID).
		Scan(&session.ID, &session.UserID, &session.TokenID, &expiresAt, &revokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return auth.Session{}, false, nil
	}
	if err != nil {
		return auth.Session{}, false, fmt.Errorf("scan session: %w", err)
	}
	session.ExpiresAt, err = time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return auth.Session{}, false, fmt.Errorf("parse session expiry: %w", err)
	}
	if revokedAt.Valid {
		parsed, parseErr := time.Parse(time.RFC3339, revokedAt.String)
		if parseErr != nil {
			return auth.Session{}, false, fmt.Errorf("parse session revocation: %w", parseErr)
		}
		session.RevokedAt = &parsed
	}
	return session, true, nil
}

func (database *DB) RevokeSession(ctx context.Context, tokenID string, revokedAt time.Time) error {
	_, err := database.sql.ExecContext(ctx, `UPDATE sessions SET revoked_at = COALESCE(revoked_at, ?) WHERE token_id = ?`, revokedAt.UTC().Format(time.RFC3339), tokenID)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

type AuthSeedResult struct {
	Name        string
	FromVersion int
	ToVersion   int
}

func (database *DB) SeedAuthDemo(ctx context.Context, passwords *auth.Passwords, password string, now time.Time, newID auth.IDGenerator) (AuthSeedResult, error) {
	const name = "auth_demo"
	const version = 1
	result := AuthSeedResult{Name: name, ToVersion: version}
	err := database.WithTx(ctx, func(transaction *sql.Tx) error {
		var current int
		err := transaction.QueryRowContext(ctx, `SELECT version FROM seed_versions WHERE name = ?`, name).Scan(&current)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("read seed version %q: %w", name, err)
		}
		result.FromVersion = current
		if current > version {
			return fmt.Errorf("seed %q version %d is newer than supported version %d", name, current, version)
		}
		roles := []auth.Role{auth.RoleViewer, auth.RoleOperator, auth.RoleApprover, auth.RoleAdmin}
		if current == version {
			for _, role := range roles {
				user, found, err := queryUser(transaction, string(role))
				if err != nil || !found || user.Role != role || user.DisabledAt != nil {
					return fmt.Errorf("auth demo seed is inconsistent; reset is required")
				}
				matched, err := passwords.Compare(user.PasswordHash, password)
				if err != nil || !matched {
					return fmt.Errorf("auth demo seed is inconsistent; reset is required")
				}
			}
			return nil
		}
		for _, role := range roles {
			id, err := newID()
			if err != nil {
				return err
			}
			hash, err := passwords.Hash(password)
			if err != nil {
				return err
			}
			timestamp := now.UTC().Format(time.RFC3339)
			if _, err := transaction.ExecContext(ctx, `INSERT INTO users(id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, id, role, hash, role, timestamp, timestamp); err != nil {
				return fmt.Errorf("insert auth demo user: %w", err)
			}
		}
		_, err = transaction.ExecContext(ctx, `INSERT INTO seed_versions(name, version, applied_at) VALUES (?, ?, ?)`, name, version, now.UTC().Format(time.RFC3339))
		return err
	})
	if err != nil {
		return AuthSeedResult{}, err
	}
	return result, nil
}

func queryUser(transaction *sql.Tx, username string) (auth.User, bool, error) {
	return scanUser(transaction.QueryRow(`SELECT id, username, password_hash, role, disabled_at FROM users WHERE username = ?`, username))
}

func (database *DB) BootstrapAdmin(ctx context.Context, id, username, passwordHash string, now time.Time) error {
	result, err := database.sql.ExecContext(ctx, `
		INSERT INTO users(id, username, password_hash, role, created_at, updated_at)
		SELECT ?, ?, ?, 'admin', ?, ? WHERE NOT EXISTS (SELECT 1 FROM users)
	`, id, username, passwordHash, now.UTC().Format(time.RFC3339), now.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("bootstrap admin: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		return fmt.Errorf("bootstrap admin requires an empty users table")
	}
	return nil
}
