package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/magicvr/allinme.core-api/internal/order"
)

const attachmentColumns = `id, file_name, storage_key, content_type, size_bytes, sha256, status, created_by, expires_at, created_at, updated_at`

func (database *DB) CreateAttachment(ctx context.Context, attachment order.Attachment) (resultErr error) {
	defer func() { resultErr = classifyOrderError(resultErr) }()
	if err := order.ValidateAttachment(attachment); err != nil {
		return err
	}
	var expiresAt any
	if attachment.ExpiresAt != nil {
		expiresAt = order.FormatTime(*attachment.ExpiresAt)
	}
	_, err := database.sql.ExecContext(ctx, `
		INSERT INTO attachments(
			id, file_name, storage_key, content_type, size_bytes, sha256, status,
			created_by, expires_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, attachment.ID, attachment.FileName, attachment.StorageKey, attachment.ContentType,
		attachment.SizeBytes, attachment.SHA256[:], attachment.Status, attachment.CreatedBy,
		expiresAt, order.FormatTime(attachment.CreatedAt), order.FormatTime(attachment.UpdatedAt))
	if err != nil {
		return fmt.Errorf("insert attachment: %w", err)
	}
	return nil
}

func (database *DB) GetAttachment(ctx context.Context, id string) (result order.Attachment, found bool, resultErr error) {
	defer func() { resultErr = classifyOrderError(resultErr) }()
	result, err := scanAttachment(database.sql.QueryRowContext(ctx, `SELECT `+attachmentColumns+` FROM attachments WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return order.Attachment{}, false, nil
	}
	if err != nil {
		return order.Attachment{}, false, fmt.Errorf("read attachment: %w", err)
	}
	return result, true, nil
}

func (database *DB) SetAttachmentStatus(ctx context.Context, id string, expected, target order.AttachmentStatus, updatedAt time.Time) (updated bool, resultErr error) {
	defer func() { resultErr = classifyOrderError(resultErr) }()
	if !expected.Valid() || !target.Valid() || !validAttachmentRepositoryTime(updatedAt) {
		return false, order.Internal(errors.New("invalid attachment status transition"))
	}
	result, err := database.sql.ExecContext(ctx, `
		UPDATE attachments
		SET status = ?, expires_at = NULL, updated_at = ?
		WHERE id = ? AND status = ?
	`, target, order.FormatTime(updatedAt), id, expected)
	if err != nil {
		return false, fmt.Errorf("update attachment status: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read updated attachment rows: %w", err)
	}
	return affected == 1, nil
}

func (database *DB) DeleteAttachment(ctx context.Context, id string, expected order.AttachmentStatus) (deleted bool, resultErr error) {
	defer func() { resultErr = classifyOrderError(resultErr) }()
	if !expected.Valid() {
		return false, order.Internal(errors.New("invalid expected attachment status"))
	}
	result, err := database.sql.ExecContext(ctx, `DELETE FROM attachments WHERE id = ? AND status = ?`, id, expected)
	if err != nil {
		return false, fmt.Errorf("delete attachment: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read deleted attachment rows: %w", err)
	}
	return affected == 1, nil
}

func (database *DB) ListAttachmentsForCleanup(ctx context.Context, expiredAt time.Time, limit int) (result []order.Attachment, resultErr error) {
	defer func() { resultErr = classifyOrderError(resultErr) }()
	if !validAttachmentRepositoryTime(expiredAt) || limit < 1 || limit > order.MaxAttachmentCleanupBatch {
		return nil, order.Internal(errors.New("invalid attachment cleanup query"))
	}
	rows, err := database.sql.QueryContext(ctx, `
		SELECT `+attachmentColumns+`
		FROM attachments
		WHERE status = 'DELETING' OR (status = 'UPLOADED' AND expires_at <= ?)
		ORDER BY
			CASE WHEN status = 'UPLOADED' THEN 0 ELSE 1 END,
			CASE WHEN status = 'UPLOADED' THEN expires_at ELSE updated_at END,
			id
		LIMIT ?
	`, order.FormatTime(expiredAt), limit)
	if err != nil {
		return nil, fmt.Errorf("query attachments for cleanup: %w", err)
	}
	defer rows.Close()
	result = make([]order.Attachment, 0, limit)
	for rows.Next() {
		attachment, scanErr := scanAttachment(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, attachment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate attachments for cleanup: %w", err)
	}
	return result, nil
}

func (database *DB) AttachmentStorageKeyExists(ctx context.Context, storageKey string) (exists bool, resultErr error) {
	defer func() { resultErr = classifyOrderError(resultErr) }()
	var marker int
	err := database.sql.QueryRowContext(ctx, `SELECT 1 FROM attachments WHERE storage_key = ?`, storageKey).Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check attachment storage key: %w", err)
	}
	return marker == 1, nil
}

func validAttachmentRepositoryTime(value time.Time) bool {
	return !value.IsZero() && value.Location() == time.UTC && value.Nanosecond() == 0
}
