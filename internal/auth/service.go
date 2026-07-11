package auth

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrAuthenticationFailed = errors.New("authentication failed")
	ErrUnauthenticated      = errors.New("unauthenticated")
)

type User struct {
	ID           string
	Username     string
	PasswordHash string
	Role         Role
	DisabledAt   *time.Time
}

type Session struct {
	ID        string
	UserID    string
	TokenID   string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

type Repository interface {
	UserByUsername(context.Context, string) (User, bool, error)
	UserByID(context.Context, string) (User, bool, error)
	CreateSession(context.Context, Session) error
	SessionByTokenID(context.Context, string) (Session, bool, error)
	RevokeSession(context.Context, string, time.Time) error
}

type LoginResult struct {
	AccessToken string
	ExpiresAt   time.Time
	User        User
}

type Service struct {
	repository Repository
	passwords  *Passwords
	tokens     *Tokens
	clock      Clock
	newID      IDGenerator
}

func NewService(repository Repository, passwords *Passwords, tokens *Tokens, clock Clock, newID IDGenerator) (*Service, error) {
	if repository == nil || passwords == nil || tokens == nil {
		return nil, fmt.Errorf("authentication dependencies are required")
	}
	if clock == nil {
		clock = time.Now
	}
	if newID == nil {
		newID = RandomID
	}
	return &Service{repository: repository, passwords: passwords, tokens: tokens, clock: clock, newID: newID}, nil
}

func (service *Service) Login(ctx context.Context, username, password string) (LoginResult, error) {
	if err := ValidatePassword(password); err != nil {
		return LoginResult{}, ErrAuthenticationFailed
	}
	user, found, err := service.repository.UserByUsername(ctx, NormalizeUsername(username))
	if err != nil {
		return LoginResult{}, fmt.Errorf("lookup login identity: %w", err)
	}
	hash := ""
	if found {
		hash = user.PasswordHash
	}
	matched, err := service.passwords.Compare(hash, password)
	if err != nil {
		return LoginResult{}, fmt.Errorf("verify login password: %w", err)
	}
	if !found || !matched || user.DisabledAt != nil {
		return LoginResult{}, ErrAuthenticationFailed
	}
	tokenID, err := service.newID()
	if err != nil {
		return LoginResult{}, err
	}
	sessionID, err := service.newID()
	if err != nil {
		return LoginResult{}, err
	}
	encoded, expiresAt, err := service.tokens.Issue(user.ID, user.Role, tokenID)
	if err != nil {
		return LoginResult{}, err
	}
	if err := service.repository.CreateSession(ctx, Session{ID: sessionID, UserID: user.ID, TokenID: tokenID, ExpiresAt: expiresAt, CreatedAt: service.clock().UTC()}); err != nil {
		return LoginResult{}, fmt.Errorf("create login session: %w", err)
	}
	return LoginResult{AccessToken: encoded, ExpiresAt: expiresAt, User: user}, nil
}

func (service *Service) Authenticate(ctx context.Context, encoded string) (Principal, error) {
	claims, err := service.tokens.Parse(encoded)
	if err != nil {
		return Principal{}, ErrUnauthenticated
	}
	session, found, err := service.repository.SessionByTokenID(ctx, claims.ID)
	if err != nil {
		return Principal{}, fmt.Errorf("lookup authentication session: %w", err)
	}
	if !found || session.RevokedAt != nil || session.UserID != claims.Subject || !service.clock().Before(session.ExpiresAt.Add(ClockLeeway)) {
		return Principal{}, ErrUnauthenticated
	}
	user, found, err := service.repository.UserByID(ctx, claims.Subject)
	if err != nil {
		return Principal{}, fmt.Errorf("lookup authenticated user: %w", err)
	}
	if !found || user.DisabledAt != nil || user.Role != claims.Role {
		return Principal{}, ErrUnauthenticated
	}
	return Principal{UserID: user.ID, Username: user.Username, Role: user.Role, TokenID: claims.ID, TokenExpiresAt: session.ExpiresAt.UTC().Format(time.RFC3339)}, nil
}

func (service *Service) Logout(ctx context.Context, principal Principal) error {
	if principal.TokenID == "" {
		return ErrUnauthenticated
	}
	if err := service.repository.RevokeSession(ctx, principal.TokenID, service.clock().UTC()); err != nil {
		return fmt.Errorf("revoke authentication session: %w", err)
	}
	return nil
}
