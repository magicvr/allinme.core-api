package port

import (
	"context"
	"errors"
	"time"

	"github.com/magicvr/allinme.core-api/internal/domain"
)

// ErrInvalidCredentials is returned for bad username/password or invalid tokens.
var ErrInvalidCredentials = errors.New("auth: invalid credentials")

// ErrInvalidToken is returned when a bearer token cannot be validated.
var ErrInvalidToken = errors.New("auth: invalid token")

// PasswordHasher hashes and verifies passwords.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

// TokenClaims is the authenticated subject carried in a JWT.
type TokenClaims struct {
	UserID   string
	Username string
	Name     string
	Roles    []string
	Expires  time.Time
}

// ToUser builds a domain.User without password hash.
func (c TokenClaims) ToUser() domain.User {
	return domain.User{
		ID:       c.UserID,
		Username: c.Username,
		Name:     c.Name,
		Roles:    c.Roles,
	}
}

// TokenService issues and parses access tokens.
type TokenService interface {
	// Issue creates an access token for the user; returns token and TTL seconds.
	Issue(ctx context.Context, user domain.User) (token string, expiresIn int64, err error)
	// Parse validates token and returns claims.
	Parse(ctx context.Context, token string) (TokenClaims, error)
}
