package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/magicvr/allinme.core-api/internal/applock"
	"github.com/magicvr/allinme.core-api/internal/config"
	"github.com/magicvr/allinme.core-api/internal/httpapi"
	"github.com/magicvr/allinme.core-api/internal/store"
)

type API struct {
	server    *http.Server
	probe     *store.Probe
	lock      *applock.Lock
	logger    *slog.Logger
	closeOnce sync.Once
}

func NewAPI(configuration config.Config, logger *slog.Logger) (*API, error) {
	if logger == nil {
		logger = slog.Default()
	}
	probe := store.NewProbe(configuration.DatabasePath)
	lock, err := applock.Acquire(configuration.DatabasePath + ".api.lock")
	if err != nil {
		return nil, err
	}
	return &API{
		server: &http.Server{
			Addr:              configuration.Address,
			Handler:           httpapi.NewHandler(httpapi.Dependencies{Logger: logger, Readiness: probe, ReadinessTimeout: time.Second}),
			ReadHeaderTimeout: 5 * time.Second,
		},
		probe:  probe,
		lock:   lock,
		logger: logger,
	}, nil
}

func (application *API) Handler() http.Handler {
	return application.server.Handler
}

func (application *API) Run(ctx context.Context) error {
	shutdownComplete := make(chan struct{})
	go func() {
		defer close(shutdownComplete)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := application.server.Shutdown(shutdownCtx); err != nil {
			application.logger.Error("server shutdown failed", "error", err)
		}
	}()

	application.logger.Info("API listening", "address", application.server.Addr)
	err := application.server.ListenAndServe()
	if ctx.Err() != nil {
		<-shutdownComplete
	}
	application.Close()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	return nil
}

func (application *API) Close() {
	application.closeOnce.Do(func() {
		application.probe.Close()
		if err := application.lock.Close(); err != nil {
			application.logger.Error("release API process lock failed", "error", err)
		}
	})
}
