package admin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/admin"
	"github.com/magicvr/allinme.core-api/internal/config"
	"github.com/magicvr/allinme.core-api/internal/files"
	"github.com/magicvr/allinme.core-api/internal/order"
	"github.com/magicvr/allinme.core-api/internal/store"
)

func TestMigrateSeedAndReset(t *testing.T) {
	configuration := developmentConfig(t)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	var output bytes.Buffer
	for _, command := range []string{"migrate", "migrate", "seed", "seed"} {
		output.Reset()
		if err := admin.Run(context.Background(), configuration, []string{command}, &output, logger); err != nil {
			t.Fatalf("%s error = %v", command, err)
		}
		if output.Len() == 0 {
			t.Fatalf("%s produced no summary", command)
		}
	}

	unrelated := filepath.Join(configuration.DataDir, "keep.txt")
	if err := os.WriteFile(unrelated, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := admin.Run(context.Background(), configuration, []string{"reset"}, &output, logger); err != nil {
		t.Fatalf("reset error = %v", err)
	}
	if _, err := os.Stat(unrelated); err != nil {
		t.Fatalf("unrelated file removed: %v", err)
	}
	database, err := store.Open(context.Background(), configuration.DatabasePath, store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var version int
	if err := database.SQL().QueryRow("SELECT version FROM seed_versions WHERE name = 'runtime'").Scan(&version); err != nil || version != 1 {
		t.Fatalf("runtime seed version = %d, error = %v", version, err)
	}
	database.Close()
	values := map[string]string{"DATA_DIR": configuration.DataDir, "DEMO_ACCOUNT_PASSWORD": "123456789012"}
	output.Reset()
	if err := admin.Execute(context.Background(), mapLookup(values), []string{"reset"}, &output, logger); err != nil {
		t.Fatalf("demo reset error = %v", err)
	}
	if !bytes.Contains(output.Bytes(), []byte(`"attachment":{"Name":"attachment_demo","FromVersion":0,"ToVersion":1}`)) {
		t.Fatalf("demo reset summary = %s", output.String())
	}
	database, err = store.Open(context.Background(), configuration.DatabasePath, store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var orders, refunds, attachments, mappings int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&orders); err != nil || orders != 10 {
		t.Fatalf("reset demo orders = %d, error = %v", orders, err)
	}
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM refunds`).Scan(&refunds); err != nil || refunds != 5 {
		t.Fatalf("reset demo refunds = %d, error = %v", refunds, err)
	}
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM attachments WHERE id = 'att_00000000000000000000000000000001' AND status = 'BOUND' AND expires_at IS NULL`).Scan(&attachments); err != nil || attachments != 1 {
		t.Fatalf("reset demo attachments = %d, error = %v", attachments, err)
	}
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM order_attachments WHERE attachment_id = 'att_00000000000000000000000000000001' AND order_id = 'ord_00000000000000000000000000000001' AND position = 0`).Scan(&mappings); err != nil || mappings != 1 {
		t.Fatalf("reset demo mappings = %d, error = %v", mappings, err)
	}
	if _, err := os.Stat(filepath.Join(configuration.DataDir, "attachments", "content", "att_00000000000000000000000000000001")); err != nil {
		t.Fatalf("reset demo attachment file missing: %v", err)
	}
}

func TestResetRemovesOnlyAttachmentRoot(t *testing.T) {
	configuration := developmentConfig(t)
	if err := admin.Run(context.Background(), configuration, []string{"migrate"}, io.Discard, nil); err != nil {
		t.Fatal(err)
	}
	attachmentFile := filepath.Join(configuration.DataDir, "attachments", "content", "stale")
	if err := os.MkdirAll(filepath.Dir(attachmentFile), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(attachmentFile, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	unrelatedDir := filepath.Join(configuration.DataDir, "exports")
	if err := os.MkdirAll(unrelatedDir, 0o750); err != nil {
		t.Fatal(err)
	}
	unrelated := filepath.Join(unrelatedDir, "keep.txt")
	if err := os.WriteFile(unrelated, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := admin.Run(context.Background(), configuration, []string{"reset"}, io.Discard, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(configuration.DataDir, "attachments")); !os.IsNotExist(err) {
		t.Fatalf("attachment root remained after reset: %v", err)
	}
	contents, err := os.ReadFile(unrelated)
	if err != nil || string(contents) != "keep" {
		t.Fatalf("unrelated data changed: %q, %v", contents, err)
	}
}

func TestCleanupAttachmentsCommandUsesExistingDatabaseAndLocalStore(t *testing.T) {
	configuration := developmentConfig(t)
	if err := admin.Run(context.Background(), configuration, []string{"migrate"}, io.Discard, nil); err != nil {
		t.Fatal(err)
	}
	database, err := store.Open(context.Background(), configuration.DatabasePath, store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.SQL().Exec(`
		INSERT INTO users(id, username, password_hash, role, created_at, updated_at)
		VALUES ('user-operator', 'operator', 'hash', 'operator', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')
	`); err != nil {
		database.Close()
		t.Fatal(err)
	}
	fileStore, err := files.NewLocal(configuration.DataDir)
	if err != nil {
		database.Close()
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	expiredContent := []byte("%PDF-1.4\nexpired\n%%EOF\n")
	boundContent := []byte("%PDF-1.4\nbound\n%%EOF\n")
	expiredStored, err := fileStore.Write("att_00000000000000000000000000000010", "expired.pdf", expiredContent)
	if err != nil {
		database.Close()
		t.Fatal(err)
	}
	boundStored, err := fileStore.Write("att_00000000000000000000000000000011", "bound.pdf", boundContent)
	if err != nil {
		database.Close()
		t.Fatal(err)
	}
	createdAt := now.Add(-48 * time.Hour)
	expiresAt := createdAt.Add(order.AttachmentUploadLifetime)
	for _, attachment := range []order.Attachment{
		{ID: "att_00000000000000000000000000000010", StorageKey: "att_00000000000000000000000000000010", FileName: expiredStored.FileName, ContentType: expiredStored.ContentType, SizeBytes: expiredStored.SizeBytes, SHA256: expiredStored.SHA256, Status: order.AttachmentStatusUploaded, CreatedBy: "user-operator", ExpiresAt: &expiresAt, CreatedAt: createdAt, UpdatedAt: createdAt},
		{ID: "att_00000000000000000000000000000011", StorageKey: "att_00000000000000000000000000000011", FileName: boundStored.FileName, ContentType: boundStored.ContentType, SizeBytes: boundStored.SizeBytes, SHA256: boundStored.SHA256, Status: order.AttachmentStatusBound, CreatedBy: "user-operator", CreatedAt: createdAt, UpdatedAt: createdAt},
	} {
		if err := database.CreateAttachment(context.Background(), attachment); err != nil {
			database.Close()
			t.Fatal(err)
		}
	}
	residualID := "att_00000000000000000000000000000012"
	residualPath := filepath.Join(configuration.DataDir, "attachments", "temp", residualID)
	if err := os.WriteFile(residualPath, []byte("residual"), 0o600); err != nil {
		database.Close()
		t.Fatal(err)
	}
	old := now.Add(-48 * time.Hour)
	if err := os.Chtimes(residualPath, old, old); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := admin.Run(context.Background(), configuration, []string{"cleanup-attachments"}, &output, nil); err != nil {
		t.Fatal(err)
	}
	var summary struct {
		Operation string `json:"operation"`
		Result    struct {
			Deleted          int `json:"Deleted"`
			ResidualsDeleted int `json:"ResidualsDeleted"`
		} `json:"result"`
	}
	if err := json.Unmarshal(output.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Operation != "cleanup-attachments" || summary.Result.Deleted != 1 || summary.Result.ResidualsDeleted != 1 {
		t.Fatalf("cleanup summary = %s", output.String())
	}
	for _, removed := range []string{
		filepath.Join(configuration.DataDir, "attachments", "content", "att_00000000000000000000000000000010"),
		residualPath,
	} {
		if _, err := os.Stat(removed); !os.IsNotExist(err) {
			t.Fatalf("cleanup target remains %s: %v", removed, err)
		}
	}
	if contents, err := fileStore.Read("att_00000000000000000000000000000011"); err != nil || string(contents) != string(boundContent) {
		t.Fatalf("bound attachment changed: %q, %v", contents, err)
	}
	database, err = store.Open(context.Background(), configuration.DatabasePath, store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var expiredRows, boundRows int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM attachments WHERE id = 'att_00000000000000000000000000000010'`).Scan(&expiredRows); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM attachments WHERE id = 'att_00000000000000000000000000000011' AND status = 'BOUND'`).Scan(&boundRows); err != nil {
		t.Fatal(err)
	}
	if expiredRows != 0 || boundRows != 1 {
		t.Fatalf("cleanup rows expired=%d bound=%d", expiredRows, boundRows)
	}
}

func TestCleanupAttachmentsRequiresExistingDatabase(t *testing.T) {
	configuration := developmentConfig(t)
	if err := admin.Run(context.Background(), configuration, []string{"cleanup-attachments"}, io.Discard, nil); err == nil {
		t.Fatal("cleanup without database error = nil")
	}
	if _, err := os.Stat(configuration.DatabasePath); !os.IsNotExist(err) {
		t.Fatalf("cleanup created database: %v", err)
	}
}

func TestProductionResetIsRejectedBeforeDeletion(t *testing.T) {
	dataDir := t.TempDir()
	configuration, err := config.Load(mapLookup(map[string]string{"APP_ENV": "production", "PORT": "8080", "DATA_DIR": dataDir}))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configuration.DatabasePath, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := admin.Run(context.Background(), configuration, []string{"reset"}, io.Discard, nil); err == nil {
		t.Fatal("reset error = nil")
	}
	contents, err := os.ReadFile(configuration.DatabasePath)
	if err != nil || string(contents) != "preserve" {
		t.Fatalf("database changed: %q, %v", contents, err)
	}
}

func TestSeedRejectsNewerVersionWithoutModification(t *testing.T) {
	configuration := developmentConfig(t)
	if err := admin.Run(context.Background(), configuration, []string{"migrate"}, io.Discard, nil); err != nil {
		t.Fatal(err)
	}
	database, err := store.Open(context.Background(), configuration.DatabasePath, store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.SQL().Exec("INSERT INTO seed_versions(name, version, applied_at) VALUES ('runtime', 2, 'future')"); err != nil {
		database.Close()
		t.Fatal(err)
	}
	database.Close()

	if err := admin.Run(context.Background(), configuration, []string{"seed"}, io.Discard, nil); err == nil {
		t.Fatal("seed error = nil")
	}
	database, err = store.Open(context.Background(), configuration.DatabasePath, store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var version int
	var appliedAt string
	if err := database.SQL().QueryRow("SELECT version, applied_at FROM seed_versions WHERE name = 'runtime'").Scan(&version, &appliedAt); err != nil {
		t.Fatal(err)
	}
	if version != 2 || appliedAt != "future" {
		t.Fatalf("seed changed to version=%d applied_at=%q", version, appliedAt)
	}
}

func TestUnknownCommandIsRejected(t *testing.T) {
	if err := admin.Run(context.Background(), developmentConfig(t), []string{"unknown"}, io.Discard, nil); err == nil {
		t.Fatal("Run() error = nil")
	}
}

func TestExecuteDevelopmentSeedAndResetRequirePasswordBeforeDatabaseAccess(t *testing.T) {
	dataDir := t.TempDir()
	values := map[string]string{"DATA_DIR": dataDir}
	for _, command := range []string{"seed", "reset"} {
		if err := admin.Execute(context.Background(), mapLookup(values), []string{command}, io.Discard, nil); err == nil {
			t.Fatalf("%s without password error = nil", command)
		}
		if _, err := os.Stat(filepath.Join(dataDir, "allinme.db")); !os.IsNotExist(err) {
			t.Fatalf("%s touched database before validation: %v", command, err)
		}
	}
	values["DEMO_ACCOUNT_PASSWORD"] = "123456789012"
	var output bytes.Buffer
	if err := admin.Execute(context.Background(), mapLookup(values), []string{"migrate"}, &output, nil); err != nil {
		t.Fatal(err)
	}
	if err := admin.Execute(context.Background(), mapLookup(values), []string{"seed"}, &output, nil); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(output.Bytes(), []byte(`"attachment":{"Name":"attachment_demo"`)) {
		t.Fatalf("development seed summary = %s", output.String())
	}
	database, err := store.Open(context.Background(), filepath.Join(dataDir, "allinme.db"), store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var users int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM users`).Scan(&users); err != nil || users != 4 {
		t.Fatalf("users = %d, error = %v", users, err)
	}
	var orders, refunds int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&orders); err != nil || orders != 10 {
		t.Fatalf("orders = %d, error = %v", orders, err)
	}
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM refunds`).Scan(&refunds); err != nil || refunds != 5 {
		t.Fatalf("refunds = %d, error = %v", refunds, err)
	}
}

func TestExecuteSeedReportsCommittedRuntimeWhenAuthGroupFails(t *testing.T) {
	dataDir := t.TempDir()
	values := map[string]string{"DATA_DIR": dataDir, "DEMO_ACCOUNT_PASSWORD": "123456789012"}
	if err := admin.Execute(context.Background(), mapLookup(values), []string{"migrate"}, io.Discard, nil); err != nil {
		t.Fatal(err)
	}
	database, err := store.Open(context.Background(), filepath.Join(dataDir, "allinme.db"), store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := database.SQL().Exec(`
		INSERT INTO users(id, username, password_hash, role, created_at, updated_at)
		VALUES ('conflict', 'viewer', 'invalid', 'viewer', ?, ?)
	`, now, now); err != nil {
		database.Close()
		t.Fatal(err)
	}
	database.Close()
	var output bytes.Buffer
	err = admin.Execute(context.Background(), mapLookup(values), []string{"seed"}, &output, nil)
	if err == nil || !strings.Contains(err.Error(), "runtime seed committed") {
		t.Fatalf("seed error = %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("failed seed reported overall success: %s", output.String())
	}
	database, err = store.Open(context.Background(), filepath.Join(dataDir, "allinme.db"), store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var runtimeVersion int
	if err := database.SQL().QueryRow(`SELECT version FROM seed_versions WHERE name = 'runtime'`).Scan(&runtimeVersion); err != nil || runtimeVersion != 1 {
		t.Fatalf("runtime version = %d, error = %v", runtimeVersion, err)
	}
	var authVersions int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM seed_versions WHERE name = 'auth_demo'`).Scan(&authVersions); err != nil || authVersions != 0 {
		t.Fatalf("auth versions = %d, error = %v", authVersions, err)
	}
}

func TestExecuteProductionBootstrapAdmin(t *testing.T) {
	dataDir := t.TempDir()
	values := map[string]string{
		"APP_ENV": "production", "PORT": "8080", "DATA_DIR": dataDir,
		"BOOTSTRAP_ADMIN_USERNAME": " Root ", "BOOTSTRAP_ADMIN_PASSWORD": "123456789012",
	}
	if err := admin.Execute(context.Background(), mapLookup(values), []string{"migrate"}, io.Discard, nil); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := admin.Execute(context.Background(), mapLookup(values), []string{"bootstrap-admin"}, &output, nil); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(output.Bytes(), []byte(`"username":"root"`)) || bytes.Contains(output.Bytes(), []byte(values["BOOTSTRAP_ADMIN_PASSWORD"])) {
		t.Fatalf("bootstrap output = %s", output.String())
	}
	if err := admin.Execute(context.Background(), mapLookup(values), []string{"bootstrap-admin"}, io.Discard, nil); err == nil {
		t.Fatal("repeat bootstrap error = nil")
	}
	values["DEMO_ACCOUNT_PASSWORD"] = "should-not-be-read"
	if err := admin.Execute(context.Background(), mapLookup(values), []string{"seed"}, io.Discard, nil); err != nil {
		t.Fatalf("production seed error = %v", err)
	}
	database, err := store.Open(context.Background(), filepath.Join(dataDir, "allinme.db"), store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var orders int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&orders); err != nil || orders != 0 {
		t.Fatalf("production seed orders = %d, error = %v", orders, err)
	}
	var refunds int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM refunds`).Scan(&refunds); err != nil || refunds != 0 {
		t.Fatalf("production seed refunds = %d, error = %v", refunds, err)
	}
	var attachments int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM attachments`).Scan(&attachments); err != nil || attachments != 0 {
		t.Fatalf("production seed attachments = %d, error = %v", attachments, err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "attachments")); !os.IsNotExist(err) {
		t.Fatalf("production seed created attachment storage: %v", err)
	}
}

func TestExecuteBootstrapAdminRejectsInvalidBoundariesWithoutCreatingUser(t *testing.T) {
	t.Run("development", func(t *testing.T) {
		values := map[string]string{
			"DATA_DIR": t.TempDir(), "BOOTSTRAP_ADMIN_USERNAME": "admin", "BOOTSTRAP_ADMIN_PASSWORD": "123456789012",
		}
		if err := admin.Execute(context.Background(), mapLookup(values), []string{"bootstrap-admin"}, io.Discard, nil); err == nil {
			t.Fatal("development bootstrap error = nil")
		}
		if _, err := os.Stat(filepath.Join(values["DATA_DIR"], "allinme.db")); !os.IsNotExist(err) {
			t.Fatalf("development bootstrap touched database: %v", err)
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		for _, test := range []struct {
			name     string
			username string
			password string
		}{
			{name: "missing username", password: "123456789012"},
			{name: "blank username", username: "   ", password: "123456789012"},
			{name: "short password", username: "admin", password: "short"},
		} {
			t.Run(test.name, func(t *testing.T) {
				dataDir := t.TempDir()
				values := map[string]string{
					"APP_ENV": "production", "PORT": "8080", "DATA_DIR": dataDir,
					"BOOTSTRAP_ADMIN_USERNAME": test.username, "BOOTSTRAP_ADMIN_PASSWORD": test.password,
				}
				if err := admin.Execute(context.Background(), mapLookup(values), []string{"bootstrap-admin"}, io.Discard, nil); err == nil {
					t.Fatal("invalid bootstrap error = nil")
				}
				if _, err := os.Stat(filepath.Join(dataDir, "allinme.db")); !os.IsNotExist(err) {
					t.Fatalf("invalid bootstrap touched database: %v", err)
				}
			})
		}
	})

	t.Run("unmigrated database", func(t *testing.T) {
		dataDir := t.TempDir()
		path := filepath.Join(dataDir, "allinme.db")
		database, err := store.Open(context.Background(), path, store.OpenCreate)
		if err != nil {
			t.Fatal(err)
		}
		database.Close()
		values := productionBootstrapValues(dataDir)
		if err := admin.Execute(context.Background(), mapLookup(values), []string{"bootstrap-admin"}, io.Discard, nil); err == nil {
			t.Fatal("unmigrated bootstrap error = nil")
		}
	})

	t.Run("nonempty users table", func(t *testing.T) {
		dataDir := t.TempDir()
		values := productionBootstrapValues(dataDir)
		if err := admin.Execute(context.Background(), mapLookup(values), []string{"migrate"}, io.Discard, nil); err != nil {
			t.Fatal(err)
		}
		if err := admin.Execute(context.Background(), mapLookup(values), []string{"bootstrap-admin"}, io.Discard, nil); err != nil {
			t.Fatal(err)
		}
		values["BOOTSTRAP_ADMIN_PASSWORD"] = "different-pass"
		if err := admin.Execute(context.Background(), mapLookup(values), []string{"bootstrap-admin"}, io.Discard, nil); err == nil {
			t.Fatal("nonempty bootstrap error = nil")
		}
		database, err := store.Open(context.Background(), filepath.Join(dataDir, "allinme.db"), store.OpenExisting)
		if err != nil {
			t.Fatal(err)
		}
		defer database.Close()
		var users int
		var username string
		if err := database.SQL().QueryRow(`SELECT COUNT(*), MIN(username) FROM users`).Scan(&users, &username); err != nil {
			t.Fatal(err)
		}
		if users != 1 || username != "admin" {
			t.Fatalf("users = %d, username = %q", users, username)
		}
	})
}

func productionBootstrapValues(dataDir string) map[string]string {
	return map[string]string{
		"APP_ENV": "production", "PORT": "8080", "DATA_DIR": dataDir,
		"BOOTSTRAP_ADMIN_USERNAME": "admin", "BOOTSTRAP_ADMIN_PASSWORD": "123456789012",
	}
}

func developmentConfig(t *testing.T) config.Config {
	t.Helper()
	configuration, err := config.Load(mapLookup(map[string]string{"DATA_DIR": t.TempDir()}))
	if err != nil {
		t.Fatal(err)
	}
	return configuration
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
