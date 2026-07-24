package app

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/magicvr/allinme.core-api/internal/config"
	"github.com/magicvr/allinme.core-api/internal/handler"
	"github.com/magicvr/allinme.core-api/internal/port"
	"github.com/magicvr/allinme.core-api/internal/repository/sqlite"
	"github.com/magicvr/allinme.core-api/internal/service/meta"
)

// App is the wired application graph (composition result).
type App struct {
	Handler http.Handler
	Logger  *slog.Logger
	// Meta is exported for tests and future modules; handlers use it via deps.
	Meta *meta.Service

	cleanup func() error
}

// Close releases resources (e.g. DB connections).
func (a *App) Close() error {
	if a.cleanup != nil {
		return a.cleanup()
	}
	return nil
}

// New is the composition root: constructs concrete adapters and injects ports.
// Business packages must not New concrete repositories; only this package (and cmd) may.
func New(cfg *config.Config, logger *slog.Logger) (*App, error) {
	if cfg == nil {
		return nil, fmt.Errorf("app: config is nil")
	}
	if logger == nil {
		logger = slog.Default()
	}

	db, err := sqlite.Open(cfg.DB.SQLitePath)
	if err != nil {
		return nil, fmt.Errorf("app: open sqlite: %w", err)
	}

	var store port.MetaStore = sqlite.NewMetaStore(db)
	metaSvc := meta.New(store)

	mux := http.NewServeMux()
	handler.Register(mux, handler.Deps{
		Logger: logger,
		Meta:   metaSvc,
	})

	return &App{
		Handler: mux,
		Logger:  logger,
		Meta:    metaSvc,
		cleanup: func() error {
			return db.Close()
		},
	}, nil
}
