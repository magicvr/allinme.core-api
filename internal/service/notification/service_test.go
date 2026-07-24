package notification_test

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/domain"
	"github.com/magicvr/allinme.core-api/internal/port"
	notificationservice "github.com/magicvr/allinme.core-api/internal/service/notification"
)

func TestServiceCreateDefaultsAndValidation(t *testing.T) {
	repository := newFakeNotificationRepository()
	now := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	service := notificationservice.NewWithDependencies(repository, func() time.Time { return now }, func() string { return "ntf_test" })

	created, err := service.Create(context.Background(), notificationservice.CreateInput{
		Title: "  Hello  ", Body: "  World  ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != "ntf_test" || created.Title != "Hello" || created.Body != "World" {
		t.Fatalf("created notification = %+v", created)
	}
	if created.Channel != domain.NotificationChannelInbox || created.Status != domain.NotificationStatusDraft || created.Version != 1 {
		t.Fatalf("created defaults = %+v", created)
	}
	if created.PublishedAt != nil {
		t.Fatalf("publishedAt should be nil on create, got %v", created.PublishedAt)
	}
	if !created.CreatedAt.Equal(now) || !created.UpdatedAt.Equal(now) {
		t.Fatalf("created timestamps = %+v", created)
	}

	emptyBody, err := service.Create(context.Background(), notificationservice.CreateInput{
		Title: "Empty body", Body: "   ", Channel: " email ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if emptyBody.Body != "" || emptyBody.Channel != domain.NotificationChannelEmail {
		t.Fatalf("empty body / email channel = %+v", emptyBody)
	}

	for name, input := range map[string]notificationservice.CreateInput{
		"empty title": {Body: "x"},
		"bad channel": {Title: "t", Channel: "sms"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := service.Create(context.Background(), input); !errors.Is(err, port.ErrNotificationInvalidArgument) {
				t.Fatalf("error = %v, want invalid argument", err)
			}
		})
	}
}

func TestServiceUpdateDraftOnlyAndOptionalChannel(t *testing.T) {
	repository := newFakeNotificationRepository()
	createdAt := time.Date(2026, time.July, 25, 9, 0, 0, 0, time.UTC)
	now := createdAt.Add(time.Hour)
	publishedAt := createdAt.Add(30 * time.Minute)
	repository.items["draft"] = domain.Notification{
		ID: "draft", Title: "Before", Body: "Body", Channel: domain.NotificationChannelInbox,
		Status: domain.NotificationStatusDraft, Version: 1, CreatedAt: createdAt, UpdatedAt: createdAt,
	}
	repository.items["published"] = domain.Notification{
		ID: "published", Title: "Pub", Body: "B", Channel: domain.NotificationChannelEmail,
		Status: domain.NotificationStatusPublished, Version: 2, PublishedAt: &publishedAt, CreatedAt: createdAt, UpdatedAt: createdAt,
	}
	service := notificationservice.NewWithDependencies(repository, func() time.Time { return now }, func() string { return "unused" })

	updated, err := service.Update(context.Background(), "draft", notificationservice.UpdateInput{
		Version: 1, Title: "  After  ", Body: "  New  ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "After" || updated.Body != "New" || updated.Channel != domain.NotificationChannelInbox {
		t.Fatalf("updated draft = %+v", updated)
	}
	if updated.Version != 2 || !updated.UpdatedAt.Equal(now) || updated.PublishedAt != nil || !updated.CreatedAt.Equal(createdAt) {
		t.Fatalf("updated meta = %+v", updated)
	}
	if _, err := service.Update(context.Background(), "draft", notificationservice.UpdateInput{
		Version: 1, Title: "Stale", Body: "x",
	}); !errors.Is(err, port.ErrNotificationVersionConflict) {
		t.Fatalf("stale update error = %v", err)
	}

	withChannel, err := service.Update(context.Background(), "draft", notificationservice.UpdateInput{
		Version: 2, Title: "After2", Body: "New2", Channel: "email",
	})
	if err != nil {
		t.Fatal(err)
	}
	if withChannel.Channel != domain.NotificationChannelEmail || withChannel.Version != 3 {
		t.Fatalf("channel update = %+v", withChannel)
	}

	if _, err := service.Update(context.Background(), "published", notificationservice.UpdateInput{
		Version: 2, Title: "No", Body: "No",
	}); !errors.Is(err, port.ErrNotificationInvalidState) {
		t.Fatalf("published update error = %v", err)
	}
}

func TestServicePublishStateAndPublishedAt(t *testing.T) {
	repository := newFakeNotificationRepository()
	now := time.Date(2026, time.July, 25, 14, 0, 0, 0, time.UTC)
	repository.items["n"] = domain.Notification{
		ID: "n", Title: "T", Body: "B", Channel: domain.NotificationChannelInbox,
		Status: domain.NotificationStatusDraft, Version: 1,
	}
	service := notificationservice.NewWithDependencies(repository, func() time.Time { return now }, func() string { return "unused" })

	published, err := service.Publish(context.Background(), "n", 1)
	if err != nil {
		t.Fatal(err)
	}
	if published.Status != domain.NotificationStatusPublished || published.Version != 2 {
		t.Fatalf("published = %+v", published)
	}
	if published.PublishedAt == nil || !published.PublishedAt.Equal(now) || !published.UpdatedAt.Equal(now) {
		t.Fatalf("published timestamps = %+v", published)
	}
	if _, err := service.Publish(context.Background(), "n", 2); !errors.Is(err, port.ErrNotificationInvalidState) {
		t.Fatalf("re-publish error = %v", err)
	}
	if _, err := service.Publish(context.Background(), "n", 1); !errors.Is(err, port.ErrNotificationVersionConflict) {
		t.Fatalf("stale publish error = %v", err)
	}
}

func TestServiceDeleteDraftOnly(t *testing.T) {
	repository := newFakeNotificationRepository()
	now := time.Date(2026, time.July, 25, 15, 0, 0, 0, time.UTC)
	publishedAt := now.Add(-time.Hour)
	repository.items["draft"] = domain.Notification{
		ID: "draft", Status: domain.NotificationStatusDraft, Version: 1, Title: "D",
	}
	repository.items["published"] = domain.Notification{
		ID: "published", Status: domain.NotificationStatusPublished, Version: 3, Title: "P", PublishedAt: &publishedAt,
	}
	service := notificationservice.NewWithDependencies(repository, func() time.Time { return now }, func() string { return "unused" })

	if err := service.Delete(context.Background(), "draft", 1); err != nil {
		t.Fatal(err)
	}
	if _, ok := repository.items["draft"]; ok {
		t.Fatal("draft should be deleted")
	}
	if err := service.Delete(context.Background(), "published", 3); !errors.Is(err, port.ErrNotificationInvalidState) {
		t.Fatalf("delete published error = %v", err)
	}
	if err := service.Delete(context.Background(), "missing", 1); !errors.Is(err, port.ErrNotificationNotFound) {
		t.Fatalf("delete missing error = %v", err)
	}
}

func TestServiceBatchArchiveIsAtomicAndValidatesIDs(t *testing.T) {
	repository := newFakeNotificationRepository()
	now := time.Date(2026, time.July, 25, 16, 0, 0, 0, time.UTC)
	pubAt := now.Add(-time.Hour)
	repository.items["pub-a"] = domain.Notification{ID: "pub-a", Status: domain.NotificationStatusPublished, Version: 1, PublishedAt: &pubAt}
	repository.items["pub-b"] = domain.Notification{ID: "pub-b", Status: domain.NotificationStatusPublished, Version: 2, PublishedAt: &pubAt}
	repository.items["draft"] = domain.Notification{ID: "draft", Status: domain.NotificationStatusDraft, Version: 1}
	service := notificationservice.NewWithDependencies(repository, func() time.Time { return now }, func() string { return "unused" })

	if _, err := service.BatchArchive(context.Background(), []string{"pub-a", "draft"}); !errors.Is(err, port.ErrNotificationInvalidState) {
		t.Fatalf("mixed batch error = %v", err)
	}
	if repository.items["pub-a"].Status != domain.NotificationStatusPublished {
		t.Fatal("published item changed despite batch rollback")
	}
	for name, ids := range map[string][]string{
		"empty":     {},
		"blank":     {" "},
		"duplicate": {"pub-a", "pub-a"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := service.BatchArchive(context.Background(), ids); !errors.Is(err, port.ErrNotificationInvalidArgument) {
				t.Fatalf("error = %v, want invalid argument", err)
			}
		})
	}
	tooMany := make([]string, 101)
	for i := range tooMany {
		tooMany[i] = "item"
	}
	if _, err := service.BatchArchive(context.Background(), tooMany); !errors.Is(err, port.ErrNotificationInvalidArgument) {
		t.Fatalf("too many IDs error = %v", err)
	}

	ids := []string{" pub-a ", "pub-b"}
	count, err := service.BatchArchive(context.Background(), ids)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 || ids[0] != " pub-a " {
		t.Fatalf("count=%d ids=%v", count, ids)
	}
	for _, id := range []string{"pub-a", "pub-b"} {
		item := repository.items[id]
		if item.Status != domain.NotificationStatusArchived || !item.UpdatedAt.Equal(now) {
			t.Fatalf("batch archived %s = %+v", id, item)
		}
		if item.PublishedAt == nil || !item.PublishedAt.Equal(pubAt) {
			t.Fatalf("publishedAt must be preserved for %s", id)
		}
		if id == "pub-a" && item.Version != 2 {
			t.Fatalf("pub-a version = %d", item.Version)
		}
		if id == "pub-b" && item.Version != 3 {
			t.Fatalf("pub-b version = %d", item.Version)
		}
	}
}

func TestServiceListAndGetValidateInputs(t *testing.T) {
	repository := newFakeNotificationRepository()
	repository.items["a"] = domain.Notification{
		ID: "a", Title: "Alice note", Channel: domain.NotificationChannelInbox, Status: domain.NotificationStatusDraft,
	}
	service := notificationservice.NewWithDependencies(repository, time.Now, func() string { return "unused" })

	items, total, err := service.List(context.Background(), port.NotificationListFilter{
		Status: domain.NotificationStatusDraft, Channel: domain.NotificationChannelInbox, Query: "Ali", Page: 1, PageSize: 20,
	})
	if err != nil || total != 1 || len(items) != 1 || items[0].ID != "a" {
		t.Fatalf("list = %+v total=%d err=%v", items, total, err)
	}
	maxInt := int(^uint(0) >> 1)
	for name, filter := range map[string]port.NotificationListFilter{
		"page":      {Page: 0, PageSize: 20},
		"page size": {Page: 1, PageSize: 101},
		"status":    {Status: "sent", Page: 1, PageSize: 20},
		"channel":   {Channel: "sms", Page: 1, PageSize: 20},
		"overflow":  {Page: maxInt, PageSize: 2},
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := service.List(context.Background(), filter); !errors.Is(err, port.ErrNotificationInvalidArgument) {
				t.Fatalf("error = %v, want invalid argument", err)
			}
		})
	}
	if _, err := service.Get(context.Background(), " "); !errors.Is(err, port.ErrNotificationInvalidArgument) {
		t.Fatalf("blank get error = %v", err)
	}
	if _, err := service.Get(context.Background(), "missing"); !errors.Is(err, port.ErrNotificationNotFound) {
		t.Fatalf("missing get error = %v", err)
	}
}

type fakeNotificationRepository struct {
	items map[string]domain.Notification
}

func newFakeNotificationRepository() *fakeNotificationRepository {
	return &fakeNotificationRepository{items: make(map[string]domain.Notification)}
}

func (r *fakeNotificationRepository) Create(_ context.Context, notification domain.Notification) error {
	r.items[notification.ID] = notification
	return nil
}

func (r *fakeNotificationRepository) Get(_ context.Context, id string) (domain.Notification, error) {
	item, ok := r.items[id]
	if !ok {
		return domain.Notification{}, port.ErrNotificationNotFound
	}
	return item, nil
}

func (r *fakeNotificationRepository) List(_ context.Context, filter port.NotificationListFilter) ([]domain.Notification, int, error) {
	items := make([]domain.Notification, 0, len(r.items))
	for _, item := range r.items {
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		if filter.Channel != "" && item.Channel != filter.Channel {
			continue
		}
		if filter.Query != "" && !strings.Contains(item.Title, filter.Query) {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, len(items), nil
}

func (r *fakeNotificationRepository) UpdateDraft(_ context.Context, id string, version int64, title, body string, channel domain.NotificationChannel, updatedAt time.Time) error {
	item, ok := r.items[id]
	if !ok {
		return port.ErrNotificationNotFound
	}
	if item.Version != version {
		return port.ErrNotificationVersionConflict
	}
	if item.Status != domain.NotificationStatusDraft {
		return port.ErrNotificationInvalidState
	}
	item.Title = title
	item.Body = body
	item.Channel = channel
	item.Version++
	item.UpdatedAt = updatedAt
	r.items[id] = item
	return nil
}

func (r *fakeNotificationRepository) DeleteDraft(_ context.Context, id string, version int64) error {
	item, ok := r.items[id]
	if !ok {
		return port.ErrNotificationNotFound
	}
	if item.Version != version {
		return port.ErrNotificationVersionConflict
	}
	if item.Status != domain.NotificationStatusDraft {
		return port.ErrNotificationInvalidState
	}
	delete(r.items, id)
	return nil
}

func (r *fakeNotificationRepository) Publish(_ context.Context, id string, version int64, publishedAt, updatedAt time.Time) error {
	item, ok := r.items[id]
	if !ok {
		return port.ErrNotificationNotFound
	}
	if item.Version != version {
		return port.ErrNotificationVersionConflict
	}
	if item.Status != domain.NotificationStatusDraft {
		return port.ErrNotificationInvalidState
	}
	item.Status = domain.NotificationStatusPublished
	pub := publishedAt
	item.PublishedAt = &pub
	item.Version++
	item.UpdatedAt = updatedAt
	r.items[id] = item
	return nil
}

func (r *fakeNotificationRepository) BatchArchive(_ context.Context, ids []string, updatedAt time.Time) error {
	for _, id := range ids {
		item, ok := r.items[id]
		if !ok {
			return port.ErrNotificationNotFound
		}
		if item.Status != domain.NotificationStatusPublished {
			return port.ErrNotificationInvalidState
		}
	}
	for _, id := range ids {
		item := r.items[id]
		item.Status = domain.NotificationStatusArchived
		item.Version++
		item.UpdatedAt = updatedAt
		r.items[id] = item
	}
	return nil
}

func (r *fakeNotificationRepository) Count(_ context.Context) (int, error) {
	return len(r.items), nil
}

var _ port.NotificationRepository = (*fakeNotificationRepository)(nil)
