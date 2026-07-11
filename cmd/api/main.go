package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/magicvr/allinme.core-api/internal/app"
	"github.com/magicvr/allinme.core-api/internal/config"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	configuration, err := config.LoadAPI(os.LookupEnv)
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	application, err := app.NewAuthenticatedAPI(configuration, slog.Default())
	if err != nil {
		slog.Error("assemble API", "error", err)
		os.Exit(1)
	}
	if err := application.Run(ctx); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
