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
	"github.com/magicvr/allinme.core-api/internal/files"
	"github.com/magicvr/allinme.core-api/internal/httpapi"
	"github.com/magicvr/allinme.core-api/internal/order"
	"github.com/magicvr/allinme.core-api/internal/store"
)

type API struct {
	server          *http.Server
	serve           func() error
	shutdown        func(context.Context) error
	closeServer     func() error
	shutdownTimeout time.Duration
	probe           *store.Probe
	database        *store.DB
	lock            *applock.Lock
	logger          *slog.Logger
	stopOnce        sync.Once
	closeOnce       sync.Once
}

func NewAPI(configuration config.Config, logger *slog.Logger) (*API, error) {
	return newAPI(configuration, nil, "", AuthDependencies{}, logger)
}

func NewAuthenticatedAPI(configuration config.APIConfig, logger *slog.Logger) (*API, error) {
	return NewAuthenticatedAPIWithDependencies(configuration, AuthDependencies{}, logger)
}

type AuthDependencies struct {
	Clock                   auth.Clock
	NewID                   auth.IDGenerator
	LimiterClock            httpapi.LimiterClock
	DisableOrderActions     bool
	RefundClock             order.Clock
	NewRefundID             func() (string, error)
	DashboardClock          order.Clock
	AttachmentClock         order.Clock
	NewAttachmentID         func() (string, error)
	AttachmentFileStore     order.AttachmentFileStore
	DisableRefundRoutes     bool
	DisableDashboardRoutes  bool
	DisableAttachmentRoutes bool
}

func NewAuthenticatedAPIWithDependencies(configuration config.APIConfig, dependencies AuthDependencies, logger *slog.Logger) (*API, error) {
	return newAPI(configuration.Config, configuration.JWTSigningKey, configuration.CORSAllowedOrigin, dependencies, logger)
}

func newAPI(configuration config.Config, signingKey []byte, corsAllowedOrigin string, authDependencies AuthDependencies, logger *slog.Logger) (*API, error) {
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
		orderService, orderServiceErr := order.NewService(database)
		if orderServiceErr != nil {
			database.Close()
			lock.Close()
			return nil, orderServiceErr
		}
		dependencies.Orders = orderService
		dependencies.OrderActions = !authDependencies.DisableOrderActions
		refundService, refundServiceErr := order.NewRefundServiceWithDependencies(database, authDependencies.RefundClock, authDependencies.NewRefundID)
		if refundServiceErr != nil {
			database.Close()
			lock.Close()
			return nil, refundServiceErr
		}
		dependencies.Refunds = refundService
		dependencies.DisableRefundRoutes = authDependencies.DisableRefundRoutes
		dependencies.DisableDashboardRoutes = authDependencies.DisableDashboardRoutes
		attachmentFileStore := authDependencies.AttachmentFileStore
		if attachmentFileStore == nil {
			attachmentFileStore, err = files.NewLocal(configuration.DataDir)
			if err != nil {
				database.Close()
				lock.Close()
				return nil, err
			}
		}
		attachmentService, attachmentServiceErr := order.NewAttachmentServiceWithDependencies(database, attachmentFileStore, authDependencies.AttachmentClock, authDependencies.NewAttachmentID)
		if attachmentServiceErr != nil {
			database.Close()
			lock.Close()
			return nil, attachmentServiceErr
		}
		dependencies.Attachments = attachmentService
		dependencies.DisableAttachmentRoutes = authDependencies.DisableAttachmentRoutes
		dashboardService, dashboardServiceErr := order.NewDashboardService(database, authDependencies.DashboardClock)
		if dashboardServiceErr != nil {
			database.Close()
			lock.Close()
			return nil, dashboardServiceErr
		}
		dependencies.Dashboard = dashboardService
		dependencies.CORSAllowedOrigin = corsAllowedOrigin
	}
	return &API{
		server: &http.Server{
			Addr:              configuration.Address,
			Handler:           httpapi.NewHandler(dependencies),
			ReadHeaderTimeout: 5 * time.Second,
		},
		probe: probe, database: database, lock: lock, logger: logger, shutdownTimeout: 10 * time.Second,
	}, nil
}

func (application *API) Handler() http.Handler {
	return application.server.Handler
}

func (application *API) Run(ctx context.Context) error {
	serveComplete := make(chan struct{})
	shutdownComplete := make(chan struct{})
	go func() {
		defer close(shutdownComplete)
		select {
		case <-ctx.Done():
		case <-serveComplete:
			return
		}
		application.stopServer()
	}()

	application.logger.Info("API listening", "address", application.server.Addr)
	serve := application.server.ListenAndServe
	if application.serve != nil {
		serve = application.serve
	}
	err := serve()
	close(serveComplete)
	<-shutdownComplete
	application.closeResources()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	return nil
}

func (application *API) shutdownServer(ctx context.Context) error {
	if application.shutdown != nil {
		return application.shutdown(ctx)
	}
	return application.server.Shutdown(ctx)
}

func (application *API) Close() {
	application.stopServer()
	application.closeResources()
}

func (application *API) stopServer() {
	application.stopOnce.Do(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), application.shutdownTimeout)
		defer cancel()
		if err := application.shutdownServer(shutdownCtx); err != nil {
			application.logger.Error("server shutdown failed", "error", err)
			closeServer := application.server.Close
			if application.closeServer != nil {
				closeServer = application.closeServer
			}
			if closeErr := closeServer(); closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
				application.logger.Error("force close server failed", "error", closeErr)
			}
		}
	})
}

func (application *API) closeResources() {
	application.closeOnce.Do(func() {
		if application.probe != nil {
			application.probe.Close()
		}
		if application.database != nil {
			if err := application.database.Close(); err != nil {
				application.logger.Error("close database failed", "error", err)
			}
		}
		if application.lock != nil {
			if err := application.lock.Close(); err != nil {
				application.logger.Error("release API process lock failed", "error", err)
			}
		}
	})
}
