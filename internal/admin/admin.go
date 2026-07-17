package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/magicvr/allinme.core-api/internal/applock"
	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/config"
	"github.com/magicvr/allinme.core-api/internal/files"
	"github.com/magicvr/allinme.core-api/internal/order"
	"github.com/magicvr/allinme.core-api/internal/store"
)

type Lookup func(string) (string, bool)

func Execute(ctx context.Context, lookup Lookup, arguments []string, output io.Writer, logger *slog.Logger) error {
	if len(arguments) > 0 && arguments[0] == "--" {
		arguments = arguments[1:]
	}
	if len(arguments) != 1 {
		return fmt.Errorf("usage: admin <migrate|seed|reset|bootstrap-admin|cleanup-attachments>")
	}
	switch arguments[0] {
	case "migrate":
		configuration, err := config.LoadBase(lookup)
		if err != nil {
			return err
		}
		return migrate(ctx, configuration, output)
	case "seed":
		base, err := config.LoadBase(lookup)
		if err != nil {
			return err
		}
		if !base.IsDevelopment() {
			return seed(ctx, base, output)
		}
		configuration, err := config.LoadDemoSeed(lookup)
		if err != nil {
			return err
		}
		return seedDemo(ctx, configuration, output)
	case "reset":
		base, err := config.LoadBase(lookup)
		if err != nil {
			return err
		}
		if !base.IsDevelopment() {
			return fmt.Errorf("reset is only available in development; stop the API before retrying")
		}
		configuration, err := config.LoadDemoSeed(lookup)
		if err != nil {
			return err
		}
		return resetDemo(ctx, configuration, output, logger)
	case "bootstrap-admin":
		configuration, err := config.LoadBootstrapAdmin(lookup)
		if err != nil {
			return err
		}
		return bootstrapAdmin(ctx, configuration, output)
	case "cleanup-attachments":
		configuration, err := config.LoadBase(lookup)
		if err != nil {
			return err
		}
		return cleanupAttachments(ctx, configuration, output)
	default:
		return fmt.Errorf("unknown subcommand %q", arguments[0])
	}
}

func Run(ctx context.Context, configuration config.Config, arguments []string, output io.Writer, logger *slog.Logger) error {
	if len(arguments) > 0 && arguments[0] == "--" {
		arguments = arguments[1:]
	}
	if len(arguments) != 1 {
		return fmt.Errorf("usage: admin <migrate|seed|reset|cleanup-attachments>")
	}
	switch arguments[0] {
	case "migrate":
		return migrate(ctx, configuration, output)
	case "seed":
		return seed(ctx, configuration, output)
	case "reset":
		return reset(ctx, configuration, output, logger)
	case "cleanup-attachments":
		return cleanupAttachments(ctx, configuration, output)
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

func seedDemo(ctx context.Context, configuration config.DemoSeedConfig, output io.Writer) error {
	database, err := store.Open(ctx, configuration.DatabasePath, store.OpenExisting)
	if err != nil {
		return err
	}
	defer database.Close()
	runtimeResult, err := database.Seed(ctx)
	if err != nil {
		return err
	}
	passwords, err := auth.NewPasswords()
	if err != nil {
		return err
	}
	authResult, err := database.SeedAuthDemo(ctx, passwords, configuration.DemoAccountPassword, time.Now(), auth.RandomID)
	if err != nil {
		return fmt.Errorf("auth demo seed failed after runtime seed committed: %w", err)
	}
	orderResult, err := database.SeedOrderDemo(ctx, time.Now())
	if err != nil {
		return fmt.Errorf("order demo seed failed after runtime and auth demo seeds committed: %w", err)
	}
	refundResult, err := database.SeedRefundDemo(ctx, time.Now())
	if err != nil {
		return fmt.Errorf("refund demo seed failed after runtime, auth demo, and order demo seeds committed: %w", err)
	}
	fileStore, err := files.NewLocal(configuration.DataDir)
	if err != nil {
		return fmt.Errorf("attachment demo seed failed after runtime, auth demo, order demo, and refund demo seeds committed: %w", err)
	}
	attachmentResult, err := database.SeedAttachmentDemo(ctx, fileStore, time.Now())
	if err != nil {
		return fmt.Errorf("attachment demo seed failed after runtime, auth demo, order demo, and refund demo seeds committed: %w", err)
	}
	return writeSummary(output, "seed", struct {
		Runtime    store.SeedResult           `json:"runtime"`
		Auth       store.AuthSeedResult       `json:"auth"`
		Order      store.OrderSeedResult      `json:"order"`
		Refund     store.RefundSeedResult     `json:"refund"`
		Attachment store.AttachmentSeedResult `json:"attachment"`
	}{Runtime: runtimeResult, Auth: authResult, Order: orderResult, Refund: refundResult, Attachment: attachmentResult})
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
	if err := removeResetTargets(configuration); err != nil {
		return err
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

func resetDemo(ctx context.Context, configuration config.DemoSeedConfig, output io.Writer, logger *slog.Logger) error {
	if !configuration.IsDevelopment() {
		return fmt.Errorf("reset is only available in development; stop the API before retrying")
	}
	lock, err := applock.Acquire(configuration.DatabasePath + ".api.lock")
	if err != nil {
		return fmt.Errorf("reset requires the API process to be stopped: %w", err)
	}
	defer lock.Close()
	if err := validateResetTargets(configuration.Config); err != nil {
		return err
	}
	if err := removeResetTargets(configuration.Config); err != nil {
		return err
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
	runtimeResult, err := database.Seed(ctx)
	if err != nil {
		return err
	}
	passwords, err := auth.NewPasswords()
	if err != nil {
		return err
	}
	authResult, err := database.SeedAuthDemo(ctx, passwords, configuration.DemoAccountPassword, time.Now(), auth.RandomID)
	if err != nil {
		return fmt.Errorf("auth demo seed failed after runtime seed committed: %w", err)
	}
	orderResult, err := database.SeedOrderDemo(ctx, time.Now())
	if err != nil {
		return fmt.Errorf("order demo seed failed after runtime and auth demo seeds committed: %w", err)
	}
	refundResult, err := database.SeedRefundDemo(ctx, time.Now())
	if err != nil {
		return fmt.Errorf("refund demo seed failed after runtime, auth demo, and order demo seeds committed: %w", err)
	}
	fileStore, err := files.NewLocal(configuration.DataDir)
	if err != nil {
		return fmt.Errorf("attachment demo seed failed after runtime, auth demo, order demo, and refund demo seeds committed: %w", err)
	}
	attachmentResult, err := database.SeedAttachmentDemo(ctx, fileStore, time.Now())
	if err != nil {
		return fmt.Errorf("attachment demo seed failed after runtime, auth demo, order demo, and refund demo seeds committed: %w", err)
	}
	return writeSummary(output, "reset", struct {
		Migration  store.MigrationResult      `json:"migration"`
		Runtime    store.SeedResult           `json:"runtime"`
		Auth       store.AuthSeedResult       `json:"auth"`
		Order      store.OrderSeedResult      `json:"order"`
		Refund     store.RefundSeedResult     `json:"refund"`
		Attachment store.AttachmentSeedResult `json:"attachment"`
	}{Migration: migration, Runtime: runtimeResult, Auth: authResult, Order: orderResult, Refund: refundResult, Attachment: attachmentResult})
}

func cleanupAttachments(ctx context.Context, configuration config.Config, output io.Writer) error {
	lock, err := applock.Acquire(configuration.DatabasePath + ".api.lock")
	if err != nil {
		return fmt.Errorf("cleanup-attachments requires the API process to be stopped: %w", err)
	}
	defer lock.Close()
	database, err := store.Open(ctx, configuration.DatabasePath, store.OpenExisting)
	if err != nil {
		return err
	}
	defer database.Close()
	fileStore, err := files.NewLocal(configuration.DataDir)
	if err != nil {
		return err
	}
	service, err := order.NewAttachmentService(database, fileStore)
	if err != nil {
		return err
	}
	result, err := service.CleanupAttachments(ctx, order.CleanupAttachmentsCommand{})
	if err != nil {
		return err
	}
	return writeSummary(output, "cleanup-attachments", result)
}

func bootstrapAdmin(ctx context.Context, configuration config.BootstrapAdminConfig, output io.Writer) error {
	passwords, err := auth.NewPasswords()
	if err != nil {
		return err
	}
	hash, err := passwords.Hash(configuration.Password)
	if err != nil {
		return err
	}
	id, err := auth.RandomID()
	if err != nil {
		return err
	}
	database, err := store.Open(ctx, configuration.DatabasePath, store.OpenExisting)
	if err != nil {
		return err
	}
	defer database.Close()
	version, err := database.SchemaVersion(ctx)
	if err != nil {
		return err
	}
	if version != store.LatestSchemaVersion() {
		return fmt.Errorf("database schema must be migrated before bootstrap-admin")
	}
	if err := database.BootstrapAdmin(ctx, id, configuration.Username, hash, time.Now()); err != nil {
		return err
	}
	return writeSummary(output, "bootstrap-admin", struct {
		Username string    `json:"username"`
		Role     auth.Role `json:"role"`
	}{Username: configuration.Username, Role: auth.RoleAdmin})
}

func validateResetTargets(configuration config.Config) error {
	dataDir := filepath.Clean(configuration.DataDir)
	attachmentRoot := filepath.Join(dataDir, "attachments")
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
	if filepath.Clean(attachmentRoot) != filepath.Join(dataDir, "attachments") || filepath.Clean(filepath.Dir(attachmentRoot)) != dataDir {
		return fmt.Errorf("reset attachment target is outside the configured data directory")
	}
	for _, target := range []string{dataDir, configuration.DatabasePath, configuration.WALPath, configuration.SHMPath, attachmentRoot} {
		if err := rejectReparsePoint(target); err != nil {
			return err
		}
	}
	return nil
}

func removeResetTargets(configuration config.Config) error {
	for _, target := range []string{configuration.DatabasePath, configuration.WALPath, configuration.SHMPath} {
		if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", filepath.Base(target), err)
		}
	}
	attachmentRoot := filepath.Join(filepath.Clean(configuration.DataDir), "attachments")
	if err := os.RemoveAll(attachmentRoot); err != nil {
		return fmt.Errorf("remove attachments: %w", err)
	}
	return nil
}

func writeSummary(output io.Writer, operation string, result any) error {
	return json.NewEncoder(output).Encode(struct {
		Operation string `json:"operation"`
		Result    any    `json:"result"`
	}{Operation: operation, Result: result})
}
