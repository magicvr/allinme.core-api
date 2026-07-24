package menu_test

import (
	"context"
	"testing"

	"github.com/magicvr/allinme.core-api/internal/domain"
	"github.com/magicvr/allinme.core-api/internal/service/menu"
)

func TestForUser_FiltersByRole(t *testing.T) {
	svc := menu.New()
	ctx := context.Background()

	admin := domain.User{Roles: []string{"admin"}}
	op := domain.User{Roles: []string{"operator"}}
	viewer := domain.User{Roles: []string{"viewer"}}

	if n := len(svc.ForUser(ctx, admin)); n < 5 {
		t.Fatalf("admin menu len = %d, want >= 5", n)
	}
	if hasID(svc.ForUser(ctx, op), "users") {
		t.Fatal("operator should not see users menu")
	}
	if !hasID(svc.ForUser(ctx, op), "orders") {
		t.Fatal("operator should see orders")
	}
	if hasID(svc.ForUser(ctx, viewer), "users") {
		t.Fatal("viewer should not see users")
	}
	if !hasID(svc.ForUser(ctx, viewer), "dashboard") {
		t.Fatal("viewer should see dashboard")
	}
}

func hasID(items []menu.PublicItem, id string) bool {
	for _, it := range items {
		if it.ID == id {
			return true
		}
	}
	return false
}
