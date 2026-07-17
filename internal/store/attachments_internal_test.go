package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/order"
)

func TestAttachmentRepositoryCreateGetAndStorageKey(t *testing.T) {
	database := openAttachmentRepositoryDB(t, filepath.Join(t.TempDir(), "attachments.db"))
	createdAt := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	attachment := attachmentRepositoryValue("01", "user-owner", order.AttachmentStatusUploaded, createdAt, createdAt)

	if err := database.CreateAttachment(context.Background(), attachment); err != nil {
		t.Fatalf("CreateAttachment() error = %v", err)
	}
	got, found, err := database.GetAttachment(context.Background(), attachment.ID)
	if err != nil || !found {
		t.Fatalf("GetAttachment() found = %t, error = %v", found, err)
	}
	if !reflect.DeepEqual(got, attachment) {
		t.Fatalf("GetAttachment() = %+v, want %+v", got, attachment)
	}
	missing, found, err := database.GetAttachment(context.Background(), "att_000000000000000000000000000000ff")
	if err != nil || found || missing != (order.Attachment{}) {
		t.Fatalf("missing GetAttachment() = %+v, %t, %v", missing, found, err)
	}
	for key, want := range map[string]bool{attachment.StorageKey: true, "att_000000000000000000000000000000ff": false} {
		exists, err := database.AttachmentStorageKeyExists(context.Background(), key)
		if err != nil || exists != want {
			t.Fatalf("AttachmentStorageKeyExists(%q) = %t, %v, want %t", key, exists, err, want)
		}
	}
}

func TestAttachmentRepositoryStatusAndDeleteCompareAndSwap(t *testing.T) {
	database := openAttachmentRepositoryDB(t, filepath.Join(t.TempDir(), "attachments.db"))
	createdAt := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	attachment := attachmentRepositoryValue("02", "user-owner", order.AttachmentStatusUploaded, createdAt, createdAt)
	if err := database.CreateAttachment(context.Background(), attachment); err != nil {
		t.Fatal(err)
	}

	updatedAt := createdAt.Add(time.Hour)
	updated, err := database.SetAttachmentStatus(context.Background(), attachment.ID, order.AttachmentStatusBound, order.AttachmentStatusDeleting, updatedAt)
	if err != nil || updated {
		t.Fatalf("stale SetAttachmentStatus() = %t, %v", updated, err)
	}
	updated, err = database.SetAttachmentStatus(context.Background(), attachment.ID, order.AttachmentStatusUploaded, order.AttachmentStatusDeleting, updatedAt)
	if err != nil || !updated {
		t.Fatalf("SetAttachmentStatus() = %t, %v", updated, err)
	}
	got, found, err := database.GetAttachment(context.Background(), attachment.ID)
	if err != nil || !found {
		t.Fatalf("GetAttachment() found = %t, error = %v", found, err)
	}
	if got.Status != order.AttachmentStatusDeleting || got.ExpiresAt != nil || !got.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("transitioned attachment = %+v", got)
	}
	updated, err = database.SetAttachmentStatus(context.Background(), attachment.ID, order.AttachmentStatusUploaded, order.AttachmentStatusDeleting, updatedAt.Add(time.Hour))
	if err != nil || updated {
		t.Fatalf("replayed SetAttachmentStatus() = %t, %v", updated, err)
	}

	deleted, err := database.DeleteAttachment(context.Background(), attachment.ID, order.AttachmentStatusUploaded)
	if err != nil || deleted {
		t.Fatalf("stale DeleteAttachment() = %t, %v", deleted, err)
	}
	deleted, err = database.DeleteAttachment(context.Background(), attachment.ID, order.AttachmentStatusDeleting)
	if err != nil || !deleted {
		t.Fatalf("DeleteAttachment() = %t, %v", deleted, err)
	}
	deleted, err = database.DeleteAttachment(context.Background(), attachment.ID, order.AttachmentStatusDeleting)
	if err != nil || deleted {
		t.Fatalf("replayed DeleteAttachment() = %t, %v", deleted, err)
	}
}

func TestAttachmentRepositoryCleanupBoundaryReplayAndBatch(t *testing.T) {
	database := openAttachmentRepositoryDB(t, filepath.Join(t.TempDir(), "attachments.db"))
	cutoff := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	values := []order.Attachment{
		attachmentRepositoryValue("10", "user-owner", order.AttachmentStatusDeleting, cutoff.Add(-72*time.Hour), cutoff.Add(-2*time.Hour)),
		attachmentRepositoryValue("11", "user-owner", order.AttachmentStatusUploaded, cutoff.Add(-48*time.Hour), cutoff.Add(-48*time.Hour)),
		attachmentRepositoryValue("12", "user-owner", order.AttachmentStatusUploaded, cutoff.Add(-24*time.Hour), cutoff.Add(-24*time.Hour)),
		attachmentRepositoryValue("13", "user-owner", order.AttachmentStatusUploaded, cutoff.Add(-23*time.Hour), cutoff.Add(-23*time.Hour)),
		attachmentRepositoryValue("14", "user-owner", order.AttachmentStatusBound, cutoff.Add(-72*time.Hour), cutoff.Add(-time.Hour)),
	}
	for _, attachment := range values {
		if err := database.CreateAttachment(context.Background(), attachment); err != nil {
			t.Fatalf("CreateAttachment(%s) error = %v", attachment.ID, err)
		}
	}

	want := []string{values[1].ID, values[2].ID, values[0].ID}
	for attempt := 0; attempt < 2; attempt++ {
		got, err := database.ListAttachmentsForCleanup(context.Background(), cutoff, len(want))
		if err != nil {
			t.Fatalf("ListAttachmentsForCleanup() attempt %d error = %v", attempt, err)
		}
		if len(got) != len(want) {
			t.Fatalf("cleanup attempt %d count = %d, want %d", attempt, len(got), len(want))
		}
		for index := range want {
			if got[index].ID != want[index] {
				t.Fatalf("cleanup attempt %d IDs[%d] = %q, want %q", attempt, index, got[index].ID, want[index])
			}
		}
	}

	batched, err := database.ListAttachmentsForCleanup(context.Background(), cutoff, 2)
	if err != nil || len(batched) != 2 || batched[0].ID != want[0] || batched[1].ID != want[1] {
		t.Fatalf("batched cleanup = %+v, error = %v", batched, err)
	}
}

func TestAttachmentRepositoryCleanupDoesNotStarveExpiredUploads(t *testing.T) {
	database := openAttachmentRepositoryDB(t, filepath.Join(t.TempDir(), "attachments.db"))
	cutoff := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	for index := 0; index < 4; index++ {
		suffix := string(rune('a'+index)) + "0"
		status := order.AttachmentStatusDeleting
		createdAt := cutoff.Add(-72 * time.Hour)
		if index >= 2 {
			status = order.AttachmentStatusUploaded
			createdAt = cutoff.Add(-48 * time.Hour)
		}
		if err := database.CreateAttachment(context.Background(), attachmentRepositoryValue(suffix, "user-owner", status, createdAt, createdAt)); err != nil {
			t.Fatal(err)
		}
	}
	got, err := database.ListAttachmentsForCleanup(context.Background(), cutoff, 2)
	if err != nil {
		t.Fatal(err)
	}
	statuses := map[order.AttachmentStatus]int{}
	for _, value := range got {
		statuses[value.Status]++
	}
	if statuses[order.AttachmentStatusUploaded] != 2 || statuses[order.AttachmentStatusDeleting] != 0 {
		t.Fatalf("cleanup statuses = %v", statuses)
	}
}

func TestAttachmentRepositoryCleanupBatchOnePrefersExpiredUpload(t *testing.T) {
	database := openAttachmentRepositoryDB(t, filepath.Join(t.TempDir(), "attachments.db"))
	cutoff := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	deleting := attachmentRepositoryValue("a0", "user-owner", order.AttachmentStatusDeleting, cutoff.Add(-72*time.Hour), cutoff.Add(-72*time.Hour))
	expired := attachmentRepositoryValue("b0", "user-owner", order.AttachmentStatusUploaded, cutoff.Add(-48*time.Hour), cutoff.Add(-48*time.Hour))
	for _, attachment := range []order.Attachment{deleting, expired} {
		if err := database.CreateAttachment(context.Background(), attachment); err != nil {
			t.Fatal(err)
		}
	}
	got, err := database.ListAttachmentsForCleanup(context.Background(), cutoff, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != expired.ID || got[0].Status != order.AttachmentStatusUploaded {
		t.Fatalf("cleanup batch = %+v, want expired upload %q", got, expired.ID)
	}
}

func TestAttachmentRepositoryRejectsCorruptLoadedData(t *testing.T) {
	database := openAttachmentRepositoryDB(t, filepath.Join(t.TempDir(), "attachments.db"))
	createdAt := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	attachment := attachmentRepositoryValue("20", "user-owner", order.AttachmentStatusUploaded, createdAt, createdAt)
	if err := database.CreateAttachment(context.Background(), attachment); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SQL().Exec(`PRAGMA ignore_check_constraints = ON; UPDATE attachments SET sha256 = zeroblob(32) WHERE id = ?; PRAGMA ignore_check_constraints = OFF;`, attachment.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := database.GetAttachment(context.Background(), attachment.ID); !errors.Is(err, order.ErrInternal) {
		t.Fatalf("GetAttachment() corrupt error = %v", err)
	}
	if _, err := database.ListAttachmentsForCleanup(context.Background(), createdAt.Add(24*time.Hour), 10); !errors.Is(err, order.ErrInternal) {
		t.Fatalf("ListAttachmentsForCleanup() corrupt error = %v", err)
	}
}

func TestAttachmentRepositoryClassifiesSQLiteBusy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "busy.db")
	first := openAttachmentRepositoryDB(t, path)
	attachment := attachmentRepositoryValue("30", "user-owner", order.AttachmentStatusUploaded, time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC), time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC))
	if err := first.CreateAttachment(context.Background(), attachment); err != nil {
		t.Fatal(err)
	}
	second, err := Open(context.Background(), path, OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = second.Close() })
	if _, err := second.SQL().Exec(`PRAGMA busy_timeout = 1`); err != nil {
		t.Fatal(err)
	}

	locked, release := make(chan struct{}), make(chan struct{})
	transactionDone := make(chan error, 1)
	go func() {
		transactionDone <- first.WithTx(context.Background(), func(transaction *sql.Tx) error {
			if _, err := transaction.Exec(`UPDATE attachments SET updated_at = updated_at WHERE id = ?`, attachment.ID); err != nil {
				return err
			}
			close(locked)
			<-release
			return nil
		})
	}()
	<-locked
	_, transitionErr := second.SetAttachmentStatus(context.Background(), attachment.ID, order.AttachmentStatusUploaded, order.AttachmentStatusDeleting, attachment.UpdatedAt.Add(time.Hour))
	close(release)
	if err := <-transactionDone; err != nil {
		t.Fatal(err)
	}
	if !errors.Is(transitionErr, order.ErrUnavailable) {
		t.Fatalf("SetAttachmentStatus() busy error = %v", transitionErr)
	}
}

func openAttachmentRepositoryDB(t *testing.T, path string) *DB {
	t.Helper()
	database, err := Open(context.Background(), path, OpenCreate)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if _, err := database.SQL().ExecContext(context.Background(), `
		INSERT INTO users(id, username, password_hash, role, created_at, updated_at)
		VALUES ('user-owner', 'attachment-owner', 'hash', 'operator', '2026-07-01T00:00:00Z', '2026-07-01T00:00:00Z')
	`); err != nil {
		t.Fatalf("insert attachment owner: %v", err)
	}
	return database
}

func attachmentRepositoryValue(suffix, owner string, status order.AttachmentStatus, createdAt, updatedAt time.Time) order.Attachment {
	content := []byte("attachment repository test content")
	value := order.Attachment{
		ID:          "att_000000000000000000000000000000" + suffix,
		StorageKey:  "att_000000000000000000000000000000" + suffix,
		FileName:    "receipt.png",
		ContentType: "image/png",
		SizeBytes:   int64(len(content)),
		SHA256:      sha256.Sum256(content),
		Status:      status,
		CreatedBy:   owner,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
	if status == order.AttachmentStatusUploaded {
		expiresAt := createdAt.Add(order.AttachmentUploadLifetime)
		value.ExpiresAt = &expiresAt
	}
	return value
}
