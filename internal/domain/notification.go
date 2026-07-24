package domain

import "time"

// NotificationStatus is the lifecycle state of a notification.
type NotificationStatus string

const (
	NotificationStatusDraft     NotificationStatus = "draft"
	NotificationStatusPublished NotificationStatus = "published"
	NotificationStatusArchived  NotificationStatus = "archived"
)

// NotificationChannel is the delivery channel enum (demo only; no real send).
type NotificationChannel string

const (
	NotificationChannelInbox NotificationChannel = "inbox"
	NotificationChannelEmail NotificationChannel = "email"
)

// Notification is an API-safe notification aggregate.
type Notification struct {
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	Body        string              `json:"body"`
	Channel     NotificationChannel `json:"channel"`
	Status      NotificationStatus  `json:"status"`
	Version     int64               `json:"version"`
	PublishedAt *time.Time          `json:"publishedAt"`
	CreatedAt   time.Time           `json:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt"`
}

// IsKnownNotificationStatus reports whether status is part of the notification lifecycle.
func IsKnownNotificationStatus(status NotificationStatus) bool {
	switch status {
	case NotificationStatusDraft, NotificationStatusPublished, NotificationStatusArchived:
		return true
	default:
		return false
	}
}

// IsKnownNotificationChannel reports whether channel is a supported enum value.
func IsKnownNotificationChannel(channel NotificationChannel) bool {
	switch channel {
	case NotificationChannelInbox, NotificationChannelEmail:
		return true
	default:
		return false
	}
}
