package port

import (
	"context"
	"errors"
	"time"

	"github.com/magicvr/allinme.core-api/internal/domain"
)

var (
	// ErrNotificationNotFound is returned when a notification cannot be located.
	ErrNotificationNotFound = errors.New("notification: not found")
	// ErrNotificationVersionConflict is returned when a notification compare-and-swap write is stale.
	ErrNotificationVersionConflict = errors.New("notification: version conflict")
	// ErrNotificationInvalidState is returned when a notification operation is not permitted in the current state.
	ErrNotificationInvalidState = errors.New("notification: invalid state")
	// ErrNotificationInvalidArgument is returned when a notification use-case input violates its contract.
	ErrNotificationInvalidArgument = errors.New("notification: invalid argument")
)

// NotificationListFilter describes repository-side notification filtering and pagination.
type NotificationListFilter struct {
	Status   domain.NotificationStatus
	Channel  domain.NotificationChannel
	Query    string
	Page     int
	PageSize int
}

// NotificationRepository is the outbound persistence port for notifications.
type NotificationRepository interface {
	Create(ctx context.Context, notification domain.Notification) error
	Get(ctx context.Context, id string) (domain.Notification, error)
	List(ctx context.Context, filter NotificationListFilter) ([]domain.Notification, int, error)
	UpdateDraft(ctx context.Context, id string, version int64, title, body string, channel domain.NotificationChannel, updatedAt time.Time) error
	DeleteDraft(ctx context.Context, id string, version int64) error
	Publish(ctx context.Context, id string, version int64, publishedAt, updatedAt time.Time) error
	BatchArchive(ctx context.Context, ids []string, updatedAt time.Time) error
	Count(ctx context.Context) (int, error)
}
