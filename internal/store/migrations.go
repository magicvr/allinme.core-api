package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type MigrationResult struct {
	FromVersion int
	ToVersion   int
}

func LatestSchemaVersion() int {
	migrations, err := loadMigrations()
	if err != nil {
		panic(err)
	}
	return len(migrations)
}

func (database *DB) Migrate(ctx context.Context) (MigrationResult, error) {
	migrations, err := loadMigrations()
	if err != nil {
		return MigrationResult{}, err
	}
	currentVersion, err := database.SchemaVersion(ctx)
	if err != nil {
		return MigrationResult{}, err
	}
	result := MigrationResult{FromVersion: currentVersion, ToVersion: currentVersion}
	if currentVersion > len(migrations) {
		return result, fmt.Errorf("database schema version %d is newer than supported version %d", currentVersion, len(migrations))
	}

	for _, migration := range migrations[currentVersion:] {
		if err := database.WithTx(ctx, func(transaction *sql.Tx) error {
			if _, err := transaction.ExecContext(ctx, migration.sql); err != nil {
				return fmt.Errorf("execute migration %d: %w", migration.version, err)
			}
			if _, err := transaction.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", migration.version)); err != nil {
				return fmt.Errorf("set schema version %d: %w", migration.version, err)
			}
			return nil
		}); err != nil {
			return result, err
		}
		result.ToVersion = migration.version
	}
	return result, nil
}

func (database *DB) SchemaVersion(ctx context.Context) (int, error) {
	var version int
	if err := database.sql.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	return version, nil
}

type migration struct {
	version int
	sql     string
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}
	var migrations []migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		prefix, _, found := strings.Cut(entry.Name(), "_")
		if !found {
			return nil, fmt.Errorf("invalid migration filename %q", entry.Name())
		}
		version, err := strconv.Atoi(prefix)
		if err != nil || version < 1 {
			return nil, fmt.Errorf("invalid migration version in %q", entry.Name())
		}
		contents, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", entry.Name(), err)
		}
		migrations = append(migrations, migration{version: version, sql: string(contents)})
	}
	sort.Slice(migrations, func(left, right int) bool { return migrations[left].version < migrations[right].version })
	for index, migration := range migrations {
		if migration.version != index+1 {
			return nil, fmt.Errorf("migrations must be unique and contiguous from version 1")
		}
	}
	return migrations, nil
}
