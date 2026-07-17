package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/order"
)

const (
	attachmentDemoSeedName    = "attachment_demo"
	attachmentDemoSeedVersion = 1
	attachmentDemoID          = "att_00000000000000000000000000000001"
	attachmentDemoOrderID     = "ord_00000000000000000000000000000001"
	attachmentDemoFileName    = "demo-invoice.pdf"
	attachmentDemoTimestamp   = "2026-01-01T00:00:00Z"
)

var attachmentDemoContent = []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Kids [] /Count 0 >>\nendobj\nxref\n0 3\n0000000000 65535 f \n0000000009 00000 n \n0000000058 00000 n \ntrailer\n<< /Size 3 /Root 1 0 R >>\nstartxref\n110\n%%EOF\n")

type AttachmentSeedResult struct {
	Name        string
	FromVersion int
	ToVersion   int
}

type attachmentDemoState struct {
	version int
	ownerID string
}

func (database *DB) SeedAttachmentDemo(ctx context.Context, fileStore order.AttachmentFileStore, appliedAt time.Time) (AttachmentSeedResult, error) {
	if fileStore == nil {
		return AttachmentSeedResult{}, errors.New("attachment file store is required")
	}
	result := AttachmentSeedResult{Name: attachmentDemoSeedName, ToVersion: attachmentDemoSeedVersion}
	state, err := database.inspectAttachmentDemoSeed(ctx)
	if err != nil {
		return AttachmentSeedResult{}, err
	}
	result.FromVersion = state.version
	if state.version == attachmentDemoSeedVersion {
		if err := validateAttachmentDemoFile(fileStore); err != nil {
			return AttachmentSeedResult{}, fmt.Errorf("attachment demo seed is inconsistent; reset is required: %w", err)
		}
		return result, nil
	}

	stored, err := fileStore.Write(attachmentDemoID, attachmentDemoFileName, attachmentDemoContent)
	wroteFile := err == nil
	if err != nil {
		content, readErr := fileStore.Read(attachmentDemoID)
		if readErr != nil || !bytes.Equal(content, attachmentDemoContent) {
			return AttachmentSeedResult{}, fmt.Errorf("write attachment demo file: %w", err)
		}
		stored = attachmentDemoStoredFile()
	}
	if stored != attachmentDemoStoredFile() {
		if wroteFile {
			_ = fileStore.Delete(attachmentDemoID)
		}
		return AttachmentSeedResult{}, errors.New("attachment demo file metadata differs from seed contract")
	}

	err = database.WithTx(ctx, func(transaction *sql.Tx) error {
		current, err := readAttachmentDemoSeedVersion(ctx, transaction)
		if err != nil {
			return err
		}
		if current > attachmentDemoSeedVersion {
			return fmt.Errorf("seed %q version %d is newer than supported version %d", attachmentDemoSeedName, current, attachmentDemoSeedVersion)
		}
		ownerID, err := validateAttachmentDemoPrerequisites(ctx, transaction)
		if err != nil {
			return err
		}
		if current == attachmentDemoSeedVersion {
			return validateAttachmentDemoMetadata(ctx, transaction, ownerID)
		}
		timestamp, err := time.Parse(time.RFC3339, attachmentDemoTimestamp)
		if err != nil {
			return err
		}
		attachment := order.Attachment{
			ID: attachmentDemoID, StorageKey: attachmentDemoID, FileName: stored.FileName,
			ContentType: stored.ContentType, SizeBytes: stored.SizeBytes, SHA256: stored.SHA256,
			Status: order.AttachmentStatusBound, CreatedBy: ownerID, CreatedAt: timestamp, UpdatedAt: timestamp,
		}
		if err := order.ValidateAttachment(attachment); err != nil {
			return fmt.Errorf("validate attachment demo metadata: %w", err)
		}
		if _, err := transaction.ExecContext(ctx, `
			INSERT INTO attachments(id, file_name, storage_key, content_type, size_bytes, sha256, status, created_by, expires_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?)
		`, attachment.ID, attachment.FileName, attachment.StorageKey, attachment.ContentType, attachment.SizeBytes, attachment.SHA256[:], attachment.Status, attachment.CreatedBy, attachmentDemoTimestamp, attachmentDemoTimestamp); err != nil {
			return fmt.Errorf("insert attachment demo metadata: %w", err)
		}
		if _, err := transaction.ExecContext(ctx, `
			INSERT INTO order_attachments(attachment_id, order_id, position, bound_at)
			VALUES (?, ?, 0, ?)
		`, attachmentDemoID, attachmentDemoOrderID, attachmentDemoTimestamp); err != nil {
			return fmt.Errorf("bind attachment demo metadata: %w", err)
		}
		if err := validateAttachmentDemoMetadata(ctx, transaction, ownerID); err != nil {
			return fmt.Errorf("validate newly inserted attachment demo seed: %w", err)
		}
		if _, err := transaction.ExecContext(ctx, `INSERT INTO seed_versions(name, version, applied_at) VALUES (?, ?, ?)`, attachmentDemoSeedName, attachmentDemoSeedVersion, appliedAt.UTC().Format(time.RFC3339)); err != nil {
			return fmt.Errorf("record attachment demo seed: %w", err)
		}
		return nil
	})
	if err != nil {
		if wroteFile {
			if compensationErr := fileStore.Delete(attachmentDemoID); compensationErr != nil {
				return AttachmentSeedResult{}, errors.Join(err, fmt.Errorf("compensate attachment demo file: %w", compensationErr))
			}
		}
		return AttachmentSeedResult{}, err
	}
	return result, nil
}

func (database *DB) inspectAttachmentDemoSeed(ctx context.Context) (attachmentDemoState, error) {
	state := attachmentDemoState{}
	err := database.WithTx(ctx, func(transaction *sql.Tx) error {
		current, err := readAttachmentDemoSeedVersion(ctx, transaction)
		if err != nil {
			return err
		}
		state.version = current
		if current > attachmentDemoSeedVersion {
			return fmt.Errorf("seed %q version %d is newer than supported version %d", attachmentDemoSeedName, current, attachmentDemoSeedVersion)
		}
		ownerID, err := validateAttachmentDemoPrerequisites(ctx, transaction)
		if err != nil {
			return err
		}
		state.ownerID = ownerID
		if current == attachmentDemoSeedVersion {
			if err := validateAttachmentDemoMetadata(ctx, transaction, ownerID); err != nil {
				return fmt.Errorf("attachment demo seed is inconsistent; reset is required: %w", err)
			}
			return nil
		}
		var conflicts int
		if err := transaction.QueryRowContext(ctx, `
			SELECT
				(SELECT COUNT(*) FROM attachments WHERE id = ? OR storage_key = ?) +
				(SELECT COUNT(*) FROM order_attachments WHERE attachment_id = ? OR (order_id = ? AND position = 0))
		`, attachmentDemoID, attachmentDemoID, attachmentDemoID, attachmentDemoOrderID).Scan(&conflicts); err != nil {
			return fmt.Errorf("inspect attachment demo conflicts: %w", err)
		}
		if conflicts != 0 {
			return errors.New("attachment demo seed conflicts with existing metadata; reset is required")
		}
		return nil
	})
	if err != nil {
		return attachmentDemoState{}, err
	}
	return state, nil
}

func readAttachmentDemoSeedVersion(ctx context.Context, transaction *sql.Tx) (int, error) {
	var current int
	err := transaction.QueryRowContext(ctx, `SELECT version FROM seed_versions WHERE name = ?`, attachmentDemoSeedName).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read seed version %q: %w", attachmentDemoSeedName, err)
	}
	return current, nil
}

func validateAttachmentDemoPrerequisites(ctx context.Context, transaction *sql.Tx) (string, error) {
	if err := validateOrderDemoSeed(ctx, transaction); err != nil {
		return "", fmt.Errorf("order demo seed prerequisite is inconsistent: %w", err)
	}
	var id, username, role string
	var disabledAt sql.NullString
	if err := transaction.QueryRowContext(ctx, `SELECT id, username, role, disabled_at FROM users WHERE username = 'operator'`).Scan(&id, &username, &role, &disabledAt); err != nil {
		return "", fmt.Errorf("read attachment demo owner: %w", err)
	}
	if id == "" || username != "operator" || role != string(auth.RoleOperator) || disabledAt.Valid {
		return "", errors.New("attachment demo owner differs from seed contract")
	}
	return id, nil
}

func validateAttachmentDemoMetadata(ctx context.Context, transaction *sql.Tx, ownerID string) error {
	attachment, err := scanAttachment(transaction.QueryRowContext(ctx, `SELECT `+attachmentColumns+` FROM attachments WHERE id = ?`, attachmentDemoID))
	if err != nil {
		return fmt.Errorf("read attachment demo metadata: %w", err)
	}
	timestamp, err := time.Parse(time.RFC3339, attachmentDemoTimestamp)
	if err != nil {
		return err
	}
	expected := attachmentDemoStoredFile()
	if attachment.ID != attachmentDemoID || attachment.StorageKey != attachmentDemoID || attachment.FileName != expected.FileName || attachment.ContentType != expected.ContentType || attachment.SizeBytes != expected.SizeBytes || attachment.SHA256 != expected.SHA256 || attachment.Status != order.AttachmentStatusBound || attachment.CreatedBy != ownerID || attachment.ExpiresAt != nil || !attachment.CreatedAt.Equal(timestamp) || !attachment.UpdatedAt.Equal(timestamp) {
		return errors.New("attachment metadata differs from seed contract")
	}
	var orderID, boundAt string
	var position int
	if err := transaction.QueryRowContext(ctx, `SELECT order_id, position, bound_at FROM order_attachments WHERE attachment_id = ?`, attachmentDemoID).Scan(&orderID, &position, &boundAt); err != nil {
		return fmt.Errorf("read attachment demo mapping: %w", err)
	}
	if orderID != attachmentDemoOrderID || position != 0 || boundAt != attachmentDemoTimestamp {
		return errors.New("attachment mapping differs from seed contract")
	}
	return nil
}

func validateAttachmentDemoFile(fileStore order.AttachmentFileStore) error {
	content, err := fileStore.Read(attachmentDemoID)
	if err != nil {
		return fmt.Errorf("read attachment demo file: %w", err)
	}
	if !bytes.Equal(content, attachmentDemoContent) {
		return errors.New("attachment bytes differ from seed contract")
	}
	return nil
}

func attachmentDemoStoredFile() order.StoredAttachmentFile {
	return order.StoredAttachmentFile{
		FileName: attachmentDemoFileName, ContentType: "application/pdf",
		SizeBytes: int64(len(attachmentDemoContent)), SHA256: sha256.Sum256(attachmentDemoContent),
	}
}
