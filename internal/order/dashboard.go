package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
)

type DashboardSummary struct {
	OrderCount            int64
	GrossAmount           int64
	CompletedRefundAmount int64
	NetAmount             int64
	Currency              string
}

type DashboardStatusItem struct {
	Status Status
	Count  int64
}

type DashboardOrderStatus struct {
	Items []DashboardStatusItem
}

type DashboardTrendItem struct {
	Date                  string
	OrderCount            int64
	GrossAmount           int64
	CompletedRefundAmount int64
	NetAmount             int64
}

type DashboardTrend struct {
	Days      int
	StartDate string
	EndDate   string
	Items     []DashboardTrendItem
}

type DashboardRepository interface {
	DashboardSummary(context.Context) (DashboardSummary, error)
	DashboardOrderStatus(context.Context) (DashboardOrderStatus, error)
	DashboardTrend(context.Context, int, time.Time) (DashboardTrend, error)
}

type DashboardService struct {
	repository DashboardRepository
	clock      Clock
}

func NewDashboardService(repository DashboardRepository, clock Clock) (*DashboardService, error) {
	if repository == nil {
		return nil, errors.New("dashboard repository is required")
	}
	return &DashboardService{repository: repository, clock: clock}, nil
}

func (service *DashboardService) Summary(ctx context.Context, principal auth.Principal) (DashboardSummary, error) {
	if !CanRead(principal) {
		return DashboardSummary{}, ErrForbidden
	}
	result, err := service.repository.DashboardSummary(ctx)
	if err != nil {
		return DashboardSummary{}, fmt.Errorf("dashboard summary: %w", err)
	}
	return result, nil
}

func (service *DashboardService) OrderStatus(ctx context.Context, principal auth.Principal) (DashboardOrderStatus, error) {
	if !CanRead(principal) {
		return DashboardOrderStatus{}, ErrForbidden
	}
	result, err := service.repository.DashboardOrderStatus(ctx)
	if err != nil {
		return DashboardOrderStatus{}, fmt.Errorf("dashboard order status: %w", err)
	}
	return result, nil
}

func (service *DashboardService) Trend(ctx context.Context, principal auth.Principal, days int) (DashboardTrend, error) {
	if !CanRead(principal) {
		return DashboardTrend{}, ErrForbidden
	}
	if days != 7 && days != 30 {
		return DashboardTrend{}, &ValidationError{Details: []FieldError{{Field: "days", Message: "must be 7 or 30"}}}
	}
	result, err := service.repository.DashboardTrend(ctx, days, UTCNow(service.clock))
	if err != nil {
		return DashboardTrend{}, fmt.Errorf("dashboard trend: %w", err)
	}
	return result, nil
}
