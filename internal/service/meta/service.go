package meta

import (
	"context"
	"fmt"

	"github.com/magicvr/allinme.core-api/internal/port"
)

// Service provides application operations over MetaStore.
// It depends only on the port interface (constructor injection).
type Service struct {
	store port.MetaStore
}

// New constructs a meta Service.
func New(store port.MetaStore) *Service {
	if store == nil {
		panic("meta.Service: store is nil")
	}
	return &Service{store: store}
}

// Ready reports whether the backing store is reachable.
func (s *Service) Ready(ctx context.Context) error {
	if err := s.store.Ping(ctx); err != nil {
		return fmt.Errorf("meta ready: %w", err)
	}
	return nil
}

// Get returns a meta value.
func (s *Service) Get(ctx context.Context, key string) (string, error) {
	return s.store.Get(ctx, key)
}

// Set stores a meta value.
func (s *Service) Set(ctx context.Context, key, value string) error {
	return s.store.Set(ctx, key, value)
}
