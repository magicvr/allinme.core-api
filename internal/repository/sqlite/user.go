package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/magicvr/allinme.core-api/internal/domain"
	"github.com/magicvr/allinme.core-api/internal/port"
)

// UserRepository is the SQLite implementation of port.UserRepository.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository wraps db.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// FindByUsername implements port.UserRepository.
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (domain.User, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, username, name, password_hash, roles_json FROM users WHERE username = ?
`, username)
	return scanUser(row)
}

// FindByID implements port.UserRepository.
func (r *UserRepository) FindByID(ctx context.Context, id string) (domain.User, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, username, name, password_hash, roles_json FROM users WHERE id = ?
`, id)
	return scanUser(row)
}

// Count implements port.UserRepository.
func (r *UserRepository) Count(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM users`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("user count: %w", err)
	}
	return n, nil
}

// Create implements port.UserRepository.
func (r *UserRepository) Create(ctx context.Context, user domain.User) error {
	rolesJSON, err := json.Marshal(user.Roles)
	if err != nil {
		return fmt.Errorf("user roles marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
INSERT INTO users (id, username, name, password_hash, roles_json)
VALUES (?, ?, ?, ?, ?)
`, user.ID, user.Username, user.Name, user.PasswordHash, string(rolesJSON))
	if err != nil {
		return fmt.Errorf("user create: %w", err)
	}
	return nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanUser(row scannable) (domain.User, error) {
	var u domain.User
	var rolesJSON string
	err := row.Scan(&u.ID, &u.Username, &u.Name, &u.PasswordHash, &rolesJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, port.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("user scan: %w", err)
	}
	if rolesJSON != "" {
		if err := json.Unmarshal([]byte(rolesJSON), &u.Roles); err != nil {
			return domain.User{}, fmt.Errorf("user roles unmarshal: %w", err)
		}
	}
	if u.Roles == nil {
		u.Roles = []string{}
	}
	return u, nil
}

var _ port.UserRepository = (*UserRepository)(nil)
