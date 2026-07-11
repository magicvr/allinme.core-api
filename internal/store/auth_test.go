package store_test

import (
	"context"
	"path/filepath"
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
