package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	PasswordCost     = 11
	MinPasswordBytes = 12
	MaxPasswordBytes = 72
)

type Role string

const (
	RoleViewer   Role = "viewer"
	RoleOperator Role = "operator"
	RoleApprover Role = "approver"
	RoleAdmin    Role = "admin"
)

func (role Role) Valid() bool {
	switch role {
	case RoleViewer, RoleOperator, RoleApprover, RoleAdmin:
		return true
	default:
		return false
	}
}

func RoleAllowed(role Role, allowed ...Role) bool {
	for _, candidate := range allowed {
		if role == candidate {
			return true
		}
	}
	return false
}

type Principal struct {
	UserID         string
	Username       string
	Role           Role
	TokenID        string
	TokenExpiresAt string
}

func NormalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func ValidatePassword(password string) error {
	length := len([]byte(password))
	if length < MinPasswordBytes || length > MaxPasswordBytes {
		return fmt.Errorf("password must be %d to %d bytes", MinPasswordBytes, MaxPasswordBytes)
	}
	return nil
}

type Passwords struct {
	dummyHash []byte
}

func NewPasswords() (*Passwords, error) {
	dummyHash, err := bcrypt.GenerateFromPassword([]byte("dummy-password-value"), PasswordCost)
	if err != nil {
		return nil, fmt.Errorf("initialize password comparison: %w", err)
	}
	return &Passwords{dummyHash: dummyHash}, nil
}

func (passwords *Passwords) Hash(password string) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), PasswordCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

func (passwords *Passwords) Compare(hash, password string) (bool, error) {
	if err := ValidatePassword(password); err != nil {
		return false, err
	}
	if hash == "" {
		hash = string(passwords.dummyHash)
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err == nil {
		return true, nil
	}
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return false, nil
	}
	return false, fmt.Errorf("compare password: %w", err)
}

func NeedsPasswordUpgrade(hash string) (bool, error) {
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return false, fmt.Errorf("read password cost: %w", err)
	}
	return cost < PasswordCost, nil
}

type IDGenerator func() (string, error)

func RandomID() (string, error) {
	return RandomIDFrom(rand.Reader)
}

func RandomIDFrom(reader io.Reader) (string, error) {
	value := make([]byte, 16)
	if _, err := io.ReadFull(reader, value); err != nil {
		return "", fmt.Errorf("generate secure identifier: %w", err)
	}
	return hex.EncodeToString(value), nil
}
