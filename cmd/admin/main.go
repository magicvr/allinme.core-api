package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/magicvr/allinme.core-api/internal/admin"
)

func main() {
	if err := admin.Execute(context.Background(), os.LookupEnv, os.Args[1:], os.Stdout, slog.Default()); err != nil {
		slog.Error("admin command failed", "error", err)
		os.Exit(1)
	}
}
