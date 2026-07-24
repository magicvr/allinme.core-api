package port

import (
	"context"
	"errors"

	"github.com/magicvr/allinme.core-api/internal/domain"
)

// ErrUserNotFound is returned when a user cannot be located.
var ErrUserNotFound = errors.New("user: not found")

// UserRepository loads users for authentication and identity.
type UserRepository interface {
	// FindByUsername returns the user or ErrUserNotFound.
	FindByUsername(ctx context.Context, username string) (domain.User, error)
	// FindByID returns the user or ErrUserNotFound.
	FindByID(ctx context.Context, id string) (domain.User, error)
	// Count returns the number of users (for seed decisions).
	Count(ctx context.Context) (int, error)
	// Create inserts a user (seed / admin bootstrap).
	Create(ctx context.Context, user domain.User) error
}
