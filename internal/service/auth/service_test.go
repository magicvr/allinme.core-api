package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/domain"
	"github.com/magicvr/allinme.core-api/internal/port"
	"github.com/magicvr/allinme.core-api/internal/security"
	"github.com/magicvr/allinme.core-api/internal/service/auth"
)

type memUsers struct {
	byName map[string]domain.User
	byID   map[string]domain.User
}

func newMemUsers(users ...domain.User) *memUsers {
	m := &memUsers{byName: map[string]domain.User{}, byID: map[string]domain.User{}}
	for _, u := range users {
		m.byName[u.Username] = u
		m.byID[u.ID] = u
	}
	return m
}

func (m *memUsers) FindByUsername(_ context.Context, username string) (domain.User, error) {
	u, ok := m.byName[username]
	if !ok {
		return domain.User{}, port.ErrUserNotFound
	}
	return u, nil
}

func (m *memUsers) FindByID(_ context.Context, id string) (domain.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return domain.User{}, port.ErrUserNotFound
	}
	return u, nil
}

func (m *memUsers) Count(context.Context) (int, error) { return len(m.byID), nil }

func (m *memUsers) Create(context.Context, domain.User) error {
	return errors.New("not implemented")
}

func TestLoginAndMe(t *testing.T) {
	ctx := context.Background()
	hasher := security.NewBcryptHasher(bcryptMinCost())
	hash, err := hasher.Hash("Demo@1234")
	if err != nil {
		t.Fatal(err)
	}
	user := domain.User{
		ID: "usr_admin", Username: "admin", Name: "Admin",
		PasswordHash: hash, Roles: []string{"admin"},
	}
	svc := auth.New(newMemUsers(user), hasher, security.NewJWTService("test-secret", time.Hour))

	res, err := svc.Login(ctx, "admin", "Demo@1234")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if res.AccessToken == "" || res.ExpiresIn != 3600 {
		t.Fatalf("token/exp = %q / %d", res.AccessToken, res.ExpiresIn)
	}
	if res.User.Username != "admin" || !contains(res.User.Roles, "admin") {
		t.Fatalf("user = %+v", res.User)
	}

	_, err = svc.Login(ctx, "admin", "wrong")
	if !errors.Is(err, port.ErrInvalidCredentials) {
		t.Fatalf("bad password err = %v", err)
	}

	claims, err := svc.ParseToken(ctx, res.AccessToken)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	me, err := svc.Me(ctx, claims.UserID)
	if err != nil || me.ID != "usr_admin" {
		t.Fatalf("Me: %+v %v", me, err)
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

// bcryptMinCost keeps tests fast.
func bcryptMinCost() int { return 4 }
