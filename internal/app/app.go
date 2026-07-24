package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/magicvr/allinme.core-api/internal/config"
	"github.com/magicvr/allinme.core-api/internal/handler"
	"github.com/magicvr/allinme.core-api/internal/port"
	"github.com/magicvr/allinme.core-api/internal/repository/sqlite"
	"github.com/magicvr/allinme.core-api/internal/security"
	"github.com/magicvr/allinme.core-api/internal/service/auth"
	"github.com/magicvr/allinme.core-api/internal/service/menu"
	"github.com/magicvr/allinme.core-api/internal/service/meta"
	orderservice "github.com/magicvr/allinme.core-api/internal/service/order"
)

// App is the wired application graph (composition result).
type App struct {
	Handler http.Handler
	Logger  *slog.Logger
	Meta    *meta.Service
	Auth    *auth.Service
	Menu    *menu.Service
	Order   *orderservice.Service

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

	var metaStore port.MetaStore = sqlite.NewMetaStore(db)
	metaSvc := meta.New(metaStore)

	hasher := security.NewBcryptHasher(0)
	tokens := security.NewJWTService(cfg.Auth.JWTSecret, cfg.Auth.JWTTTL)
	var users port.UserRepository = sqlite.NewUserRepository(db)
	if err := sqlite.SeedUsers(context.Background(), users, hasher); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("app: seed users: %w", err)
	}
	authSvc := auth.New(users, hasher, tokens)
	menuSvc := menu.New()
	orderRepository := sqlite.NewOrderRepository(db)
	if err := sqlite.SeedOrders(context.Background(), orderRepository); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("app: seed orders: %w", err)
	}
	var orders port.OrderRepository = orderRepository
	orderSvc := orderservice.New(orders)

	mux := http.NewServeMux()
	handler.Register(mux, handler.Deps{
		Logger: logger,
		Meta:   metaSvc,
		Auth:   authSvc,
		Menu:   menuSvc,
		Order:  orderSvc,
	})

	return &App{
		Handler: mux,
		Logger:  logger,
		Meta:    metaSvc,
		Auth:    authSvc,
		Menu:    menuSvc,
		Order:   orderSvc,
		cleanup: func() error {
			return db.Close()
		},
	}, nil
}
