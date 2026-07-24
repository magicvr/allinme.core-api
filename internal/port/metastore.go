package port

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a meta key is missing.
var ErrNotFound = errors.New("meta: not found")

// MetaStore is the outbound port for key-value metadata storage.
// Used for readiness probes and to prove swappable persistence (GOAL-003).
type MetaStore interface {
	// Ping verifies the store is reachable.
	Ping(ctx context.Context) error
	// Get returns the value for key, or ErrNotFound.
	Get(ctx context.Context, key string) (string, error)
	// Set upserts key to value.
	Set(ctx context.Context, key, value string) error
}
