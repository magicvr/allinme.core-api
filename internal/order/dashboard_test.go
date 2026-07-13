package order_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/order"
)

func TestDashboardServiceAuthorizationValidationAndInjectedClock(t *testing.T) {
	now := time.Date(2026, 1, 7, 12, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	repository := &dashboardRepository{
		summary: order.DashboardSummary{Currency: "CNY"},
		status:  order.DashboardOrderStatus{Items: []order.DashboardStatusItem{}},
		trend:   order.DashboardTrend{Days: 7},
	}
	service, err := order.NewDashboardService(repository, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Summary(context.Background(), auth.Principal{}); !errors.Is(err, order.ErrForbidden) {
		t.Fatalf("summary authorization error = %v", err)
	}
	principal := auth.Principal{UserID: "user-viewer", Role: auth.RoleViewer}
	if _, err := service.Summary(context.Background(), principal); err != nil {
		t.Fatal(err)
	}
	if _, err := service.OrderStatus(context.Background(), principal); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Trend(context.Background(), principal, 8); err == nil {
		t.Fatal("invalid trend days error = nil")
	}
	if _, err := service.Trend(context.Background(), principal, 7); err != nil {
		t.Fatal(err)
	}
	if repository.days != 7 || !repository.now.Equal(now.UTC()) {
		t.Fatalf("trend repository input = days %d now %s", repository.days, repository.now)
	}
}

type dashboardRepository struct {
	summary order.DashboardSummary
	status  order.DashboardOrderStatus
	trend   order.DashboardTrend
	days    int
	now     time.Time
}

func (repository *dashboardRepository) DashboardSummary(context.Context) (order.DashboardSummary, error) {
	return repository.summary, nil
}

func (repository *dashboardRepository) DashboardOrderStatus(context.Context) (order.DashboardOrderStatus, error) {
	return repository.status, nil
}

func (repository *dashboardRepository) DashboardTrend(_ context.Context, days int, now time.Time) (order.DashboardTrend, error) {
	repository.days, repository.now = days, now
	return repository.trend, nil
}
