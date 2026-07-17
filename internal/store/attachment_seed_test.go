package store_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/files"
	"github.com/magicvr/allinme.core-api/internal/order"
	"github.com/magicvr/allinme.core-api/internal/store"
)

const attachmentDemoID = "att_00000000000000000000000000000001"

func TestAttachmentDemoSeedCreatesAndValidatesFixedAttachment(t *testing.T) {
	database := openMigrated(t)
	seedAttachmentPrerequisites(t, database)
	fileStore, err := files.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	appliedAt := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)

	first, err := database.SeedAttachmentDemo(context.Background(), fileStore, appliedAt)
	if err != nil {
		t.Fatal(err)
	}
	if first.Name != "attachment_demo" || first.FromVersion != 0 || first.ToVersion != 1 {
		t.Fatalf("first seed = %+v", first)
	}
	second, err := database.SeedAttachmentDemo(context.Background(), fileStore, appliedAt.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if second.FromVersion != 1 || second.ToVersion != 1 {
		t.Fatalf("second seed = %+v", second)
	}

	var fileName, storageKey, contentType, status, username string
	var sizeBytes int64
	var digest []byte
	var expiresAt any
	if err := database.SQL().QueryRow(`
		SELECT a.file_name, a.storage_key, a.content_type, a.size_bytes, a.sha256, a.status, a.expires_at, u.username
		FROM attachments a JOIN users u ON u.id = a.created_by WHERE a.id = ?
	`, attachmentDemoID).Scan(&fileName, &storageKey, &contentType, &sizeBytes, &digest, &status, &expiresAt, &username); err != nil {
		t.Fatal(err)
	}
	content, err := fileStore.Read(attachmentDemoID)
	if err != nil {
		t.Fatal(err)
	}
	if fileName != "demo-invoice.pdf" || storageKey != attachmentDemoID || contentType != "application/pdf" || sizeBytes != int64(len(content)) || len(digest) != 32 || status != "BOUND" || expiresAt != nil || username != "operator" || !strings.HasPrefix(string(content), "%PDF-") {
		t.Fatalf("attachment contract = %q %q %q %d %d %q %v %q content=%q", fileName, storageKey, contentType, sizeBytes, len(digest), status, expiresAt, username, content)
	}
	var orderID, boundAt, storedAppliedAt string
	var position int
	if err := database.SQL().QueryRow(`SELECT order_id, position, bound_at FROM order_attachments WHERE attachment_id = ?`, attachmentDemoID).Scan(&orderID, &position, &boundAt); err != nil {
		t.Fatal(err)
	}
	if orderID != "ord_00000000000000000000000000000001" || position != 0 || boundAt != "2026-01-01T00:00:00Z" {
		t.Fatalf("mapping = %q %d %q", orderID, position, boundAt)
	}
	if err := database.SQL().QueryRow(`SELECT applied_at FROM seed_versions WHERE name = 'attachment_demo'`).Scan(&storedAppliedAt); err != nil {
		t.Fatal(err)
	}
	if storedAppliedAt != appliedAt.Format(time.RFC3339) {
		t.Fatalf("applied_at = %q", storedAppliedAt)
	}
}

func TestAttachmentDemoSeedRejectsMetadataAndFileTampering(t *testing.T) {
	for _, test := range []struct {
		name   string
		tamper func(*testing.T, *store.DB, *files.Local)
	}{
		{
			name: "metadata",
			tamper: func(t *testing.T, database *store.DB, _ *files.Local) {
				if _, err := database.SQL().Exec(`UPDATE attachments SET file_name = 'tampered.pdf' WHERE id = ?`, attachmentDemoID); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "mapping",
			tamper: func(t *testing.T, database *store.DB, _ *files.Local) {
				if _, err := database.SQL().Exec(`UPDATE order_attachments SET bound_at = '2026-01-01T00:00:01Z' WHERE attachment_id = ?`, attachmentDemoID); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "bytes",
			tamper: func(t *testing.T, _ *store.DB, fileStore *files.Local) {
				if err := fileStore.Delete(attachmentDemoID); err != nil {
					t.Fatal(err)
				}
				if _, err := fileStore.Write(attachmentDemoID, "demo-invoice.pdf", []byte("%PDF-1.4\ntampered\n%%EOF\n")); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			database := openMigrated(t)
			seedAttachmentPrerequisites(t, database)
			fileStore, err := files.NewLocal(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := database.SeedAttachmentDemo(context.Background(), fileStore, time.Now()); err != nil {
				t.Fatal(err)
			}
			test.tamper(t, database, fileStore)
			if _, err := database.SeedAttachmentDemo(context.Background(), fileStore, time.Now()); err == nil || !strings.Contains(err.Error(), "reset is required") {
				t.Fatalf("tampered replay error = %v", err)
			}
		})
	}
}

func TestAttachmentDemoSeedCompensatesFileWhenDatabaseFails(t *testing.T) {
	database := openMigrated(t)
	seedAttachmentPrerequisites(t, database)
	local, err := files.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	fileStore := &closingAttachmentFiles{AttachmentFileStore: local, close: database.Close}
	if _, err := database.SeedAttachmentDemo(context.Background(), fileStore, time.Now()); err == nil {
		t.Fatal("seed error = nil")
	}
	if _, err := local.Read(attachmentDemoID); err == nil {
		t.Fatal("attachment file remained after database failure")
	}
}

func TestAttachmentDemoSeedDoesNotDeletePreexistingMatchingFileOnDatabaseFailure(t *testing.T) {
	database := openMigrated(t)
	seedAttachmentPrerequisites(t, database)
	dataDir := t.TempDir()
	local, err := files.NewLocal(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	content := []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Kids [] /Count 0 >>\nendobj\nxref\n0 3\n0000000000 65535 f \n0000000009 00000 n \n0000000058 00000 n \ntrailer\n<< /Size 3 /Root 1 0 R >>\nstartxref\n110\n%%EOF\n")
	if _, err := local.Write(attachmentDemoID, "demo-invoice.pdf", content); err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SeedAttachmentDemo(context.Background(), local, time.Now()); err == nil {
		t.Fatal("seed error = nil")
	}
	if got, err := os.ReadFile(filepath.Join(dataDir, "attachments", "content", attachmentDemoID)); err != nil || string(got) != string(content) {
		t.Fatalf("preexisting file changed: %q, %v", got, err)
	}
}

func seedAttachmentPrerequisites(t *testing.T, database *store.DB) {
	t.Helper()
	if _, err := database.SQL().Exec(`
		INSERT INTO users(id, username, password_hash, role, created_at, updated_at)
		VALUES ('user-operator', 'operator', 'hash', 'operator', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SeedOrderDemo(context.Background(), time.Now()); err != nil {
		t.Fatal(err)
	}
}

type closingAttachmentFiles struct {
	order.AttachmentFileStore
	close func() error
}

func (files *closingAttachmentFiles) Write(storageKey, fileName string, content []byte) (order.StoredAttachmentFile, error) {
	stored, err := files.AttachmentFileStore.Write(storageKey, fileName, content)
	if err == nil {
		err = files.close()
	}
	return stored, err
}

var _ order.AttachmentFileStore = (*closingAttachmentFiles)(nil)
