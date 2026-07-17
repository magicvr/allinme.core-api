package server

import (
	"log/slog"
	"net/http"

	"github.com/magicvr/allinme.core-api/internal/config"
)

// New builds an http.Server with timeouts from config.
func New(cfg *config.Config, handler http.Handler, logger *slog.Logger) *http.Server {
	return &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      handler,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
}
