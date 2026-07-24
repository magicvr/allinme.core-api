package meta_test

import (
	"context"
	"errors"
	"testing"

	"github.com/magicvr/allinme.core-api/internal/port"
	"github.com/magicvr/allinme.core-api/internal/repository/memory"
	"github.com/magicvr/allinme.core-api/internal/service/meta"
)

// TestService_UsesPortOnly_WithoutSQLite proves the service depends on the
// MetaStore interface and runs with a memory fake (GOAL-003 H3).
func TestService_UsesPortOnly_WithoutSQLite(t *testing.T) {
	ctx := context.Background()
	store := memory.NewMetaStore()
	svc := meta.New(store)

	if err := svc.Ready(ctx); err != nil {
		t.Fatalf("Ready: %v", err)
	}

	if err := svc.Set(ctx, "schema_ui_protocol", "2.4.1"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := svc.Get(ctx, "schema_ui_protocol")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "2.4.1" {
		t.Fatalf("Get = %q, want 2.4.1", got)
	}

	_, err = svc.Get(ctx, "missing")
	if !errors.Is(err, port.ErrNotFound) {
		t.Fatalf("Get missing err = %v, want ErrNotFound", err)
	}
}
