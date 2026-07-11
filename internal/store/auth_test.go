package store_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/store"
)

func TestAuthDemoSeedCreatesAndValidatesFixedUsers(t *testing.T) {
	database := openMigrated(t)
	passwords, err := auth.NewPasswords()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	sequence := 0
	newID := func() (string, error) { sequence++; return "user-" + string(rune('0'+sequence)), nil }
	if _, err := database.SeedAuthDemo(context.Background(), passwords, "123456789012", now, newID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SeedAuthDemo(context.Background(), passwords, "123456789012", now, newID); err != nil {
		t.Fatalf("repeat seed error = %v", err)
	}
	var count int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil || count != 4 {
		t.Fatalf("user count = %d, error = %v", count, err)
	}
	if _, err := database.SQL().Exec(`UPDATE users SET role = 'admin' WHERE username = 'viewer'`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SeedAuthDemo(context.Background(), passwords, "123456789012", now, newID); err == nil {
		t.Fatal("tampered repeat seed error = nil")
	}
}

func TestAuthDemoSeedRejectsEveryInconsistentStateWithoutRepair(t *testing.T) {
	mutations := []struct {
		name   string
		mutate string
		check  string
	}{
		{name: "deleted user", mutate: `DELETE FROM users WHERE username = 'viewer'`, check: `SELECT COUNT(*) FROM users WHERE username = 'viewer'`},
		{name: "changed role", mutate: `UPDATE users SET role = 'admin' WHERE username = 'viewer'`, check: `SELECT COUNT(*) FROM users WHERE username = 'viewer' AND role = 'viewer'`},
		{name: "disabled user", mutate: `UPDATE users SET disabled_at = '2026-07-12T12:00:00Z' WHERE username = 'viewer'`, check: `SELECT COUNT(*) FROM users WHERE username = 'viewer' AND disabled_at IS NULL`},
		{name: "changed hash", mutate: `UPDATE users SET password_hash = 'invalid' WHERE username = 'viewer'`, check: `SELECT COUNT(*) FROM users WHERE username = 'viewer' AND password_hash != 'invalid'`},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			database := openMigrated(t)
			passwords, err := auth.NewPasswords()
			if err != nil {
				t.Fatal(err)
			}
			now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
			sequence := 0
			newID := func() (string, error) {
				sequence++
				return "user-" + string(rune('0'+sequence)), nil
			}
			if _, err := database.SeedAuthDemo(context.Background(), passwords, "123456789012", now, newID); err != nil {
				t.Fatal(err)
			}
			if _, err := database.SQL().Exec(mutation.mutate); err != nil {
				t.Fatal(err)
			}
			if _, err := database.SeedAuthDemo(context.Background(), passwords, "123456789012", now, newID); err == nil {
				t.Fatal("inconsistent repeat seed error = nil")
			}
			var repaired int
			if err := database.SQL().QueryRow(mutation.check).Scan(&repaired); err != nil {
				t.Fatal(err)
			}
			if repaired != 0 {
				t.Fatal("repeat seed repaired inconsistent user")
			}
			var version int
			if err := database.SQL().QueryRow(`SELECT version FROM seed_versions WHERE name = 'auth_demo'`).Scan(&version); err != nil || version != 1 {
				t.Fatalf("auth seed version = %d, error = %v", version, err)
			}
		})
	}
}

func TestAuthDemoSeedConflictRollsBackGroup(t *testing.T) {
	database := openMigrated(t)
	passwords, err := auth.NewPasswords()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	if _, err := database.SQL().Exec(`
		INSERT INTO users(id, username, password_hash, role, created_at, updated_at)
		VALUES ('existing', 'viewer', 'existing-hash', 'viewer', ?, ?)
	`, now.Format(time.RFC3339), now.Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	sequence := 0
	_, err = database.SeedAuthDemo(context.Background(), passwords, "123456789012", now, func() (string, error) {
		sequence++
		return "generated-" + string(rune('0'+sequence)), nil
	})
	if err == nil {
		t.Fatal("conflicting seed error = nil")
	}
	var users int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM users`).Scan(&users); err != nil || users != 1 {
		t.Fatalf("users = %d, error = %v", users, err)
	}
	var versions int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM seed_versions WHERE name = 'auth_demo'`).Scan(&versions); err != nil || versions != 0 {
		t.Fatalf("auth seed versions = %d, error = %v", versions, err)
	}
}

func TestBootstrapAdminAcrossIndependentConnections(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "allinme.db")
	first, err := store.Open(ctx, path, store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.Migrate(ctx); err != nil {
		first.Close()
		t.Fatal(err)
	}
	second, err := store.Open(ctx, path, store.OpenExisting)
	if err != nil {
		first.Close()
		t.Fatal(err)
	}
	defer first.Close()
	defer second.Close()

	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	databases := []*store.DB{first, second}
	errorsByCall := make([]error, 2)
	var wait sync.WaitGroup
	for index := range databases {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			errorsByCall[index] = databases[index].BootstrapAdmin(ctx, "id-"+string(rune('0'+index)), "admin"+string(rune('0'+index)), "hash", now)
		}(index)
	}
	wait.Wait()
	successes := 0
	for _, err := range errorsByCall {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("bootstrap successes = %d, errors = %v", successes, errorsByCall)
	}
	var count int
	if err := first.SQL().QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("users = %d, error = %v", count, err)
	}
}

func TestSessionRoundTripAndIdempotentRevocation(t *testing.T) {
	database := openMigrated(t)
	passwords, _ := auth.NewPasswords()
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	sequence := 0
	_, err := database.SeedAuthDemo(context.Background(), passwords, "123456789012", now, func() (string, error) {
		sequence++
		return string(rune('a' + sequence)), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	user, found, err := database.UserByUsername(context.Background(), "viewer")
	if err != nil || !found {
		t.Fatalf("user = %+v, %v, %v", user, found, err)
	}
	session := auth.Session{ID: "session-1", UserID: user.ID, TokenID: "token-1", ExpiresAt: now.Add(time.Minute)}
	if err := database.CreateSession(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	if err := database.RevokeSession(context.Background(), session.TokenID, now); err != nil {
		t.Fatal(err)
	}
	if err := database.RevokeSession(context.Background(), session.TokenID, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	loaded, found, err := database.SessionByTokenID(context.Background(), session.TokenID)
	if err != nil || !found || loaded.RevokedAt == nil || !loaded.RevokedAt.Equal(now) {
		t.Fatalf("session = %+v, %v, %v", loaded, found, err)
	}
}

func TestAuthServiceClassifiesClosedStoreAndCanceledContext(t *testing.T) {
	for _, test := range []struct {
		name    string
		prepare func(*testing.T, *store.DB) context.Context
	}{
		{name: "closed store", prepare: func(t *testing.T, database *store.DB) context.Context {
			if err := database.Close(); err != nil {
				t.Fatal(err)
			}
			return context.Background()
		}},
		{name: "canceled context", prepare: func(_ *testing.T, _ *store.DB) context.Context {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "sensitive-database-name.db")
			database, err := store.Open(context.Background(), path, store.OpenCreate)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := database.Migrate(context.Background()); err != nil {
				database.Close()
				t.Fatal(err)
			}
			t.Cleanup(func() { database.Close() })
			passwords, _ := auth.NewPasswords()
			tokens, _ := auth.NewTokens([]byte("12345678901234567890123456789012"), time.Now)
			service, _ := auth.NewService(database, passwords, tokens, time.Now, auth.RandomID)
			_, err = service.Login(test.prepare(t, database), "viewer", "123456789012")
			if !errors.Is(err, auth.ErrInternal) || strings.Contains(err.Error(), "sensitive-database-name") || strings.Contains(err.Error(), "SQL") {
				t.Fatalf("Login() error = %v", err)
			}
		})
	}
}
