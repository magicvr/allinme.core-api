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
	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/config"
	"github.com/magicvr/allinme.core-api/internal/httpapi"
	"github.com/magicvr/allinme.core-api/internal/store"
)

type API struct {
	server    *http.Server
	probe     *store.Probe
	database  *store.DB
	lock      *applock.Lock
	logger    *slog.Logger
	closeOnce sync.Once
}

func NewAPI(configuration config.Config, logger *slog.Logger) (*API, error) {
	return newAPI(configuration, nil, AuthDependencies{}, logger)
}

func NewAuthenticatedAPI(configuration config.APIConfig, logger *slog.Logger) (*API, error) {
	return NewAuthenticatedAPIWithDependencies(configuration, AuthDependencies{}, logger)
}

type AuthDependencies struct {
	Clock        auth.Clock
	NewID        auth.IDGenerator
	LimiterClock httpapi.LimiterClock
}

func NewAuthenticatedAPIWithDependencies(configuration config.APIConfig, dependencies AuthDependencies, logger *slog.Logger) (*API, error) {
	return newAPI(configuration.Config, configuration.JWTSigningKey, dependencies, logger)
}

func newAPI(configuration config.Config, signingKey []byte, authDependencies AuthDependencies, logger *slog.Logger) (*API, error) {
	if logger == nil {
		logger = slog.Default()
	}
	probe := store.NewProbe(configuration.DatabasePath)
	lock, err := applock.Acquire(configuration.DatabasePath + ".api.lock")
	if err != nil {
		return nil, err
	}
	dependencies := httpapi.Dependencies{Logger: logger, Readiness: probe, ReadinessTimeout: time.Second}
	var database *store.DB
	if len(signingKey) > 0 {
		clock := authDependencies.Clock
		if clock == nil {
			clock = time.Now
		}
		newID := authDependencies.NewID
		if newID == nil {
			newID = auth.RandomID
		}
		database, err = store.Open(context.Background(), configuration.DatabasePath, store.OpenExisting)
		if err != nil {
			lock.Close()
			return nil, err
		}
		passwords, passwordErr := auth.NewPasswords()
		if passwordErr != nil {
			database.Close()
			lock.Close()
			return nil, passwordErr
		}
		tokens, tokenErr := auth.NewTokens(signingKey, clock)
		if tokenErr != nil {
			database.Close()
			lock.Close()
			return nil, tokenErr
		}
		service, serviceErr := auth.NewService(database, passwords, tokens, clock, newID)
		if serviceErr != nil {
			database.Close()
			lock.Close()
			return nil, serviceErr
		}
		dependencies.Auth = service
		dependencies.LoginLimiter = httpapi.NewLoginLimiter(authDependencies.LimiterClock)
	}
	return &API{
		server: &http.Server{
			Addr:              configuration.Address,
			Handler:           httpapi.NewHandler(dependencies),
			ReadHeaderTimeout: 5 * time.Second,
		},
		probe: probe, database: database, lock: lock, logger: logger,
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
		if application.database != nil {
			if err := application.database.Close(); err != nil {
				application.logger.Error("close database failed", "error", err)
			}
		}
		if err := application.lock.Close(); err != nil {
			application.logger.Error("release API process lock failed", "error", err)
		}
	})
}
