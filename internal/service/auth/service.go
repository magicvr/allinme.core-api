package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/magicvr/allinme.core-api/internal/domain"
	"github.com/magicvr/allinme.core-api/internal/port"
)

// Service provides login and identity operations.
type Service struct {
	users  port.UserRepository
	hasher port.PasswordHasher
	tokens port.TokenService
}

// New constructs an auth Service (constructor injection).
func New(users port.UserRepository, hasher port.PasswordHasher, tokens port.TokenService) *Service {
	if users == nil || hasher == nil || tokens == nil {
		panic("auth.Service: nil dependency")
	}
	return &Service{users: users, hasher: hasher, tokens: tokens}
}

// LoginResult is returned on successful authentication.
type LoginResult struct {
	AccessToken string
	ExpiresIn   int64
	User        domain.PublicUser
}

// Login authenticates username/password and issues a JWT.
func (s *Service) Login(ctx context.Context, username, password string) (LoginResult, error) {
	user, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, port.ErrUserNotFound) {
			return LoginResult{}, port.ErrInvalidCredentials
		}
		return LoginResult{}, fmt.Errorf("auth login find: %w", err)
	}
	if err := s.hasher.Compare(user.PasswordHash, password); err != nil {
		return LoginResult{}, port.ErrInvalidCredentials
	}
	token, expiresIn, err := s.tokens.Issue(ctx, user)
	if err != nil {
		return LoginResult{}, fmt.Errorf("auth login issue: %w", err)
	}
	return LoginResult{
		AccessToken: token,
		ExpiresIn:   expiresIn,
		User:        user.ToPublic(),
	}, nil
}

// Me loads the current user by id (from token subject).
func (s *Service) Me(ctx context.Context, userID string) (domain.PublicUser, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, port.ErrUserNotFound) {
			return domain.PublicUser{}, port.ErrInvalidToken
		}
		return domain.PublicUser{}, fmt.Errorf("auth me: %w", err)
	}
	return user.ToPublic(), nil
}

// ParseToken validates a bearer token.
func (s *Service) ParseToken(ctx context.Context, token string) (port.TokenClaims, error) {
	return s.tokens.Parse(ctx, token)
}
