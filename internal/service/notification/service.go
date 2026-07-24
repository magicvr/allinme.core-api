package notification

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/magicvr/allinme.core-api/internal/domain"
	"github.com/magicvr/allinme.core-api/internal/port"
)

// Service implements notification use cases using only the NotificationRepository port.
type Service struct {
	repository port.NotificationRepository
	now        func() time.Time
	newID      func() string
}

// New constructs a Notification service with production clock and ID generation.
func New(repository port.NotificationRepository) *Service {
	return NewWithDependencies(repository, time.Now, newNotificationID)
}

// NewWithDependencies constructs a Notification service with testable time and ID sources.
func NewWithDependencies(repository port.NotificationRepository, now func() time.Time, newID func() string) *Service {
	if repository == nil || now == nil || newID == nil {
		panic("notification.Service: nil dependency")
	}
	return &Service{repository: repository, now: now, newID: newID}
}

// CreateInput contains the fields accepted on notification creation.
type CreateInput struct {
	Title   string
	Body    string
	Channel string
}

// UpdateInput contains the draft content fields mutable by PUT in this slice.
type UpdateInput struct {
	Version int64
	Title   string
	Body    string
	Channel string // empty means keep existing channel
}

// List returns a paginated notification list.
func (s *Service) List(ctx context.Context, filter port.NotificationListFilter) ([]domain.Notification, int, error) {
	if filter.Page < 1 || filter.PageSize < 1 || filter.PageSize > 100 {
		return nil, 0, port.ErrNotificationInvalidArgument
	}
	maxInt := int(^uint(0) >> 1)
	if filter.Page > 1 && filter.Page-1 > maxInt/filter.PageSize {
		return nil, 0, port.ErrNotificationInvalidArgument
	}
	if filter.Status != "" && !domain.IsKnownNotificationStatus(filter.Status) {
		return nil, 0, port.ErrNotificationInvalidArgument
	}
	if filter.Channel != "" && !domain.IsKnownNotificationChannel(filter.Channel) {
		return nil, 0, port.ErrNotificationInvalidArgument
	}
	items, total, err := s.repository.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("notification list: %w", err)
	}
	return items, total, nil
}

// Get returns one notification.
func (s *Service) Get(ctx context.Context, id string) (domain.Notification, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.Notification{}, port.ErrNotificationInvalidArgument
	}
	item, err := s.repository.Get(ctx, id)
	if err != nil {
		return domain.Notification{}, fmt.Errorf("notification get: %w", err)
	}
	return item, nil
}

// Create creates a draft notification.
func (s *Service) Create(ctx context.Context, input CreateInput) (domain.Notification, error) {
	title := strings.TrimSpace(input.Title)
	// Body is required as a string field but may be empty after trim (D-003).
	body := strings.TrimSpace(input.Body)
	if title == "" {
		return domain.Notification{}, port.ErrNotificationInvalidArgument
	}
	channel, err := normalizeChannel(input.Channel, domain.NotificationChannelInbox)
	if err != nil {
		return domain.Notification{}, err
	}
	now := s.now().UTC()
	item := domain.Notification{
		ID:          s.newID(),
		Title:       title,
		Body:        body,
		Channel:     channel,
		Status:      domain.NotificationStatusDraft,
		Version:     1,
		PublishedAt: nil,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if item.ID == "" {
		return domain.Notification{}, fmt.Errorf("notification create: empty generated ID")
	}
	if err := s.repository.Create(ctx, item); err != nil {
		return domain.Notification{}, fmt.Errorf("notification create: %w", err)
	}
	return item, nil
}

// Update changes draft title/body/channel under optimistic locking.
func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (domain.Notification, error) {
	id = strings.TrimSpace(id)
	title := strings.TrimSpace(input.Title)
	body := strings.TrimSpace(input.Body)
	if id == "" || input.Version < 1 || title == "" {
		return domain.Notification{}, port.ErrNotificationInvalidArgument
	}
	item, err := s.Get(ctx, id)
	if err != nil {
		return domain.Notification{}, err
	}
	if item.Version != input.Version {
		return domain.Notification{}, port.ErrNotificationVersionConflict
	}
	if item.Status != domain.NotificationStatusDraft {
		return domain.Notification{}, port.ErrNotificationInvalidState
	}
	channel := item.Channel
	if strings.TrimSpace(input.Channel) != "" {
		channel, err = normalizeChannel(input.Channel, item.Channel)
		if err != nil {
			return domain.Notification{}, err
		}
	}
	now := s.now().UTC()
	if err := s.repository.UpdateDraft(ctx, id, input.Version, title, body, channel, now); err != nil {
		return domain.Notification{}, fmt.Errorf("notification update: %w", err)
	}
	item.Title = title
	item.Body = body
	item.Channel = channel
	item.Version++
	item.UpdatedAt = now
	return item, nil
}

// Delete removes a draft notification under optimistic locking.
func (s *Service) Delete(ctx context.Context, id string, version int64) error {
	id = strings.TrimSpace(id)
	if id == "" || version < 1 {
		return port.ErrNotificationInvalidArgument
	}
	item, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if item.Version != version {
		return port.ErrNotificationVersionConflict
	}
	if item.Status != domain.NotificationStatusDraft {
		return port.ErrNotificationInvalidState
	}
	if err := s.repository.DeleteDraft(ctx, id, version); err != nil {
		return fmt.Errorf("notification delete: %w", err)
	}
	return nil
}

// Publish changes a draft notification to published under optimistic locking.
func (s *Service) Publish(ctx context.Context, id string, version int64) (domain.Notification, error) {
	id = strings.TrimSpace(id)
	if id == "" || version < 1 {
		return domain.Notification{}, port.ErrNotificationInvalidArgument
	}
	item, err := s.Get(ctx, id)
	if err != nil {
		return domain.Notification{}, err
	}
	if item.Version != version {
		return domain.Notification{}, port.ErrNotificationVersionConflict
	}
	if item.Status != domain.NotificationStatusDraft {
		return domain.Notification{}, port.ErrNotificationInvalidState
	}
	now := s.now().UTC()
	if err := s.repository.Publish(ctx, id, version, now, now); err != nil {
		return domain.Notification{}, fmt.Errorf("notification publish: %w", err)
	}
	publishedAt := now
	item.Status = domain.NotificationStatusPublished
	item.PublishedAt = &publishedAt
	item.Version++
	item.UpdatedAt = now
	return item, nil
}

// BatchArchive archives published notifications atomically.
func (s *Service) BatchArchive(ctx context.Context, ids []string) (int, error) {
	if len(ids) == 0 || len(ids) > 100 {
		return 0, port.ErrNotificationInvalidArgument
	}
	normalized := make([]string, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for i, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			return 0, port.ErrNotificationInvalidArgument
		}
		if _, exists := seen[id]; exists {
			return 0, port.ErrNotificationInvalidArgument
		}
		seen[id] = struct{}{}
		normalized[i] = id
	}
	if err := s.repository.BatchArchive(ctx, normalized, s.now().UTC()); err != nil {
		return 0, fmt.Errorf("notification batch archive: %w", err)
	}
	return len(normalized), nil
}

func normalizeChannel(value string, defaultChannel domain.NotificationChannel) (domain.NotificationChannel, error) {
	channel := domain.NotificationChannel(strings.TrimSpace(value))
	if channel == "" {
		return defaultChannel, nil
	}
	if !domain.IsKnownNotificationChannel(channel) {
		return "", port.ErrNotificationInvalidArgument
	}
	return channel, nil
}

func newNotificationID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return ""
	}
	return "ntf_" + hex.EncodeToString(bytes[:])
}
