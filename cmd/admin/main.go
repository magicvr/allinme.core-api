package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/magicvr/allinme.core-api/internal/admin"
	"github.com/magicvr/allinme.core-api/internal/config"
)

func main() {
	configuration, err := config.Load(os.LookupEnv)
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	if err := admin.Run(context.Background(), configuration, os.Args[1:], os.Stdout, slog.Default()); err != nil {
		slog.Error("admin command failed", "error", err)
		os.Exit(1)
	}
}
