package security

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/magicvr/allinme.core-api/internal/port"
)

// BcryptHasher implements port.PasswordHasher.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher constructs a hasher with the given cost (or default).
func NewBcryptHasher(cost int) *BcryptHasher {
	if cost < bcrypt.MinCost {
		cost = bcrypt.DefaultCost
	}
	return &BcryptHasher{cost: cost}
}

// Hash implements port.PasswordHasher.
func (h *BcryptHasher) Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash: %w", err)
	}
	return string(b), nil
}

// Compare implements port.PasswordHasher.
func (h *BcryptHasher) Compare(hash, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return port.ErrInvalidCredentials
	}
	return nil
}

var _ port.PasswordHasher = (*BcryptHasher)(nil)
