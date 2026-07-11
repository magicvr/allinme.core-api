package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/magicvr/allinme.core-api/internal/applock"
	"github.com/magicvr/allinme.core-api/internal/config"
	"github.com/magicvr/allinme.core-api/internal/store"
)

func Run(ctx context.Context, configuration config.Config, arguments []string, output io.Writer, logger *slog.Logger) error {
	if len(arguments) > 0 && arguments[0] == "--" {
		arguments = arguments[1:]
	}
	if len(arguments) != 1 {
		return fmt.Errorf("usage: admin <migrate|seed|reset>")
	}
	switch arguments[0] {
	case "migrate":
		return migrate(ctx, configuration, output)
	case "seed":
		return seed(ctx, configuration, output)
	case "reset":
		return reset(ctx, configuration, output, logger)
	default:
		return fmt.Errorf("unknown subcommand %q", arguments[0])
	}
}

func migrate(ctx context.Context, configuration config.Config, output io.Writer) error {
	database, err := store.Open(ctx, configuration.DatabasePath, store.OpenCreate)
	if err != nil {
		return err
	}
	defer database.Close()
	result, err := database.Migrate(ctx)
	if err != nil {
		return err
	}
	return writeSummary(output, "migrate", result)
}

func seed(ctx context.Context, configuration config.Config, output io.Writer) error {
	database, err := store.Open(ctx, configuration.DatabasePath, store.OpenExisting)
	if err != nil {
		return err
	}
	defer database.Close()
	result, err := database.Seed(ctx)
	if err != nil {
		return err
	}
	return writeSummary(output, "seed", result)
}

func reset(ctx context.Context, configuration config.Config, output io.Writer, logger *slog.Logger) error {
	if !configuration.IsDevelopment() {
		return fmt.Errorf("reset is only available in development; stop the API before retrying")
	}
	lock, err := applock.Acquire(configuration.DatabasePath + ".api.lock")
	if err != nil {
		return fmt.Errorf("reset requires the API process to be stopped: %w", err)
	}
	defer lock.Close()
	if err := validateResetTargets(configuration); err != nil {
		return err
	}
	for _, target := range []string{configuration.DatabasePath, configuration.WALPath, configuration.SHMPath} {
		if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", filepath.Base(target), err)
		}
	}
	if logger != nil {
		logger.Info("database files removed for reset")
	}
	database, err := store.Open(ctx, configuration.DatabasePath, store.OpenCreate)
	if err != nil {
		return err
	}
	defer database.Close()
	migration, err := database.Migrate(ctx)
	if err != nil {
		return err
	}
	seedResult, err := database.Seed(ctx)
	if err != nil {
		return err
	}
	return writeSummary(output, "reset", struct {
		Migration store.MigrationResult `json:"migration"`
		Seed      store.SeedResult      `json:"seed"`
	}{Migration: migration, Seed: seedResult})
}

func validateResetTargets(configuration config.Config) error {
	dataDir := filepath.Clean(configuration.DataDir)
	allowed := map[string]bool{
		"allinme.db":     true,
		"allinme.db-wal": true,
		"allinme.db-shm": true,
	}
	for _, target := range []string{configuration.DatabasePath, configuration.WALPath, configuration.SHMPath} {
		if filepath.Clean(filepath.Dir(target)) != dataDir || !allowed[filepath.Base(target)] {
			return fmt.Errorf("reset target is outside the configured data directory")
		}
	}
	for _, target := range []string{dataDir, configuration.DatabasePath, configuration.WALPath, configuration.SHMPath} {
		if err := rejectReparsePoint(target); err != nil {
			return err
		}
	}
	return nil
}

func writeSummary(output io.Writer, operation string, result any) error {
	return json.NewEncoder(output).Encode(struct {
		Operation string `json:"operation"`
		Result    any    `json:"result"`
	}{Operation: operation, Result: result})
}
