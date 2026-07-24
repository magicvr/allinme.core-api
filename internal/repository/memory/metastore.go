package memory

import (
	"context"
	"sync"

	"github.com/magicvr/allinme.core-api/internal/port"
)

// MetaStore is an in-memory MetaStore for tests and doubles.
type MetaStore struct {
	mu   sync.RWMutex
	data map[string]string
}

// NewMetaStore constructs an empty memory MetaStore.
func NewMetaStore() *MetaStore {
	return &MetaStore{data: make(map[string]string)}
}

// Ping always succeeds.
func (s *MetaStore) Ping(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

// Get implements port.MetaStore.
func (s *MetaStore) Get(ctx context.Context, key string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	if !ok {
		return "", port.ErrNotFound
	}
	return v, nil
}

// Set implements port.MetaStore.
func (s *MetaStore) Set(ctx context.Context, key, value string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	return nil
}

// Ensure interface compliance.
var _ port.MetaStore = (*MetaStore)(nil)
