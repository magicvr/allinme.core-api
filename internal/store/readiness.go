package store

import (
	"context"
	"errors"
	"os"
	"sync"
)

type ReadinessStatus string

const (
	DatabaseMissing     ReadinessStatus = "database_missing"
	DatabaseUnavailable ReadinessStatus = "database_unavailable"
	SchemaUninitialized ReadinessStatus = "schema_uninitialized"
	SchemaOutdated      ReadinessStatus = "schema_outdated"
	SchemaTooNew        ReadinessStatus = "schema_too_new"
	Ready               ReadinessStatus = "ready"
)

func ClassifySchemaVersion(version, latest int) ReadinessStatus {
	switch {
	case version == 0:
		return SchemaUninitialized
	case version < latest:
		return SchemaOutdated
	case version > latest:
		return SchemaTooNew
	default:
		return Ready
	}
}

type Probe struct {
	path   string
	mutex  sync.RWMutex
	closed bool
}

func NewProbe(path string) *Probe {
	return &Probe{path: path}
}

func (probe *Probe) Check(ctx context.Context) ReadinessStatus {
	probe.mutex.RLock()
	defer probe.mutex.RUnlock()
	if probe.closed || ctx.Err() != nil {
		return DatabaseUnavailable
	}
	if _, err := os.Stat(probe.path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DatabaseMissing
		}
		return DatabaseUnavailable
	}
	database, err := Open(ctx, probe.path, OpenExisting)
	if err != nil {
		return DatabaseUnavailable
	}
	defer database.Close()
	version, err := database.SchemaVersion(ctx)
	if err != nil {
		return DatabaseUnavailable
	}
	return ClassifySchemaVersion(version, LatestSchemaVersion())
}

func (probe *Probe) Close() {
	probe.mutex.Lock()
	probe.closed = true
	probe.mutex.Unlock()
}
