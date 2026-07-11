package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
)

func TestServiceLoginAuthenticateLogout(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	passwords, _ := auth.NewPasswords()
	hash, _ := passwords.Hash("123456789012")
	repository := &memoryRepository{users: map[string]auth.User{
		"viewer": {ID: "user-1", Username: "viewer", PasswordHash: hash, Role: auth.RoleViewer},
	}, sessions: map[string]auth.Session{}}
	tokens, _ := auth.NewTokens([]byte("12345678901234567890123456789012"), func() time.Time { return now })
	ids := []string{"token-1", "session-1"}
	service, _ := auth.NewService(repository, passwords, tokens, func() time.Time { return now }, func() (string, error) {
		value := ids[0]
		ids = ids[1:]
		return value, nil
	})

	login, err := service.Login(context.Background(), " VIEWER ", "123456789012")
	if err != nil {
		t.Fatal(err)
	}
	principal, err := service.Authenticate(context.Background(), login.AccessToken)
	if err != nil || principal.UserID != "user-1" {
		t.Fatalf("Authenticate() = %+v, %v", principal, err)
	}
	if err := service.Logout(context.Background(), principal); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Authenticate(context.Background(), login.AccessToken); !errors.Is(err, auth.ErrUnauthenticated) {
		t.Fatalf("Authenticate() after logout error = %v", err)
	}
}

func TestServiceRejectsUnknownAndRoleChange(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	passwords, _ := auth.NewPasswords()
	hash, _ := passwords.Hash("123456789012")
	repository := &memoryRepository{users: map[string]auth.User{
		"viewer": {ID: "user-1", Username: "viewer", PasswordHash: hash, Role: auth.RoleViewer},
	}, sessions: map[string]auth.Session{}}
	tokens, _ := auth.NewTokens([]byte("12345678901234567890123456789012"), func() time.Time { return now })
	sequence := 0
	service, _ := auth.NewService(repository, passwords, tokens, func() time.Time { return now }, func() (string, error) {
		sequence++
		if sequence == 1 {
			return "token-1", nil
		}
		return "session-1", nil
	})
	if _, err := service.Login(context.Background(), "missing", "123456789012"); !errors.Is(err, auth.ErrAuthenticationFailed) {
		t.Fatalf("unknown Login() error = %v", err)
	}
	if len(repository.sessions) != 0 {
		t.Fatal("unknown login created a session")
	}
	login, err := service.Login(context.Background(), "viewer", "123456789012")
	if err != nil {
		t.Fatal(err)
	}
	user := repository.users["viewer"]
	user.Role = auth.RoleAdmin
	repository.users["viewer"] = user
	if _, err := service.Authenticate(context.Background(), login.AccessToken); !errors.Is(err, auth.ErrUnauthenticated) {
		t.Fatalf("role-change Authenticate() error = %v", err)
	}
}

func TestServiceRejectsSessionMismatchDisabledUserAndExpiry(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	clock := now
	passwords, _ := auth.NewPasswords()
	hash, _ := passwords.Hash("123456789012")
	repository := &memoryRepository{users: map[string]auth.User{
		"viewer": {ID: "user-1", Username: "viewer", PasswordHash: hash, Role: auth.RoleViewer},
	}, sessions: map[string]auth.Session{}}
	tokens, _ := auth.NewTokens([]byte("12345678901234567890123456789012"), func() time.Time { return clock })
	sequence := 0
	service, _ := auth.NewService(repository, passwords, tokens, func() time.Time { return clock }, func() (string, error) {
		sequence++
		if sequence%2 == 1 {
			return "token-" + string(rune('0'+sequence)), nil
		}
		return "session-" + string(rune('0'+sequence)), nil
	})

	login, err := service.Login(context.Background(), "viewer", "123456789012")
	if err != nil {
		t.Fatal(err)
	}
	session := repository.sessions["token-1"]
	session.UserID = "user-2"
	repository.sessions["token-1"] = session
	if _, err := service.Authenticate(context.Background(), login.AccessToken); !errors.Is(err, auth.ErrUnauthenticated) {
		t.Fatalf("subject mismatch error = %v", err)
	}

	login, err = service.Login(context.Background(), "viewer", "123456789012")
	if err != nil {
		t.Fatal(err)
	}
	user := repository.users["viewer"]
	disabledAt := now
	user.DisabledAt = &disabledAt
	repository.users["viewer"] = user
	if _, err := service.Authenticate(context.Background(), login.AccessToken); !errors.Is(err, auth.ErrUnauthenticated) {
		t.Fatalf("disabled user error = %v", err)
	}

	user.DisabledAt = nil
	repository.users["viewer"] = user
	clock = now.Add(auth.TokenTTL + auth.ClockLeeway)
	if _, err := service.Authenticate(context.Background(), login.AccessToken); !errors.Is(err, auth.ErrUnauthenticated) {
		t.Fatalf("expired session error = %v", err)
	}
}

type memoryRepository struct {
	users    map[string]auth.User
	sessions map[string]auth.Session
}

func (repository *memoryRepository) UserByUsername(_ context.Context, username string) (auth.User, bool, error) {
	user, ok := repository.users[username]
	return user, ok, nil
}
func (repository *memoryRepository) UserByID(_ context.Context, id string) (auth.User, bool, error) {
	for _, user := range repository.users {
		if user.ID == id {
			return user, true, nil
		}
	}
	return auth.User{}, false, nil
}
func (repository *memoryRepository) CreateSession(_ context.Context, session auth.Session) error {
	if _, exists := repository.sessions[session.TokenID]; exists {
		return errors.New("duplicate")
	}
	repository.sessions[session.TokenID] = session
	return nil
}
func (repository *memoryRepository) SessionByTokenID(_ context.Context, tokenID string) (auth.Session, bool, error) {
	session, ok := repository.sessions[tokenID]
	return session, ok, nil
}
func (repository *memoryRepository) RevokeSession(_ context.Context, tokenID string, at time.Time) error {
	session := repository.sessions[tokenID]
	if session.RevokedAt == nil {
		session.RevokedAt = &at
	}
	repository.sessions[tokenID] = session
	return nil
}
