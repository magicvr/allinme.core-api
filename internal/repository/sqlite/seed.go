package sqlite

import (
	"context"
	"fmt"

	"github.com/magicvr/allinme.core-api/internal/domain"
	"github.com/magicvr/allinme.core-api/internal/port"
)

// SeedUsers inserts demo users when the users table is empty.
// Password for all seeds: Demo@1234 (demo only).
func SeedUsers(ctx context.Context, users port.UserRepository, hasher port.PasswordHasher) error {
	n, err := users.Count(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	hash, err := hasher.Hash("Demo@1234")
	if err != nil {
		return err
	}
	seeds := []domain.User{
		{ID: "usr_admin", Username: "admin", Name: "Admin", PasswordHash: hash, Roles: []string{"admin"}},
		{ID: "usr_operator", Username: "operator", Name: "Operator", PasswordHash: hash, Roles: []string{"operator"}},
		{ID: "usr_viewer", Username: "viewer", Name: "Viewer", PasswordHash: hash, Roles: []string{"viewer"}},
	}
	for _, u := range seeds {
		if err := users.Create(ctx, u); err != nil {
			return fmt.Errorf("seed user %s: %w", u.Username, err)
		}
	}
	return nil
}
