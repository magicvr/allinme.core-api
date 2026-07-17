package order

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image/jpeg"
	"image/png"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/magicvr/allinme.core-api/internal/auth"
)

const (
	MaxAttachmentFileNameBytes = 255
	MaxAttachmentSizeBytes     = int64(10 * 1024 * 1024)
	MaxAttachmentCleanupBatch  = 1000
	DefaultAttachmentBatchSize = 100
	AttachmentUploadLifetime   = 24 * time.Hour
)

type AttachmentStatus string

const (
	AttachmentStatusUploaded AttachmentStatus = "UPLOADED"
	AttachmentStatusBound    AttachmentStatus = "BOUND"
	AttachmentStatusDeleting AttachmentStatus = "DELETING"
)

func (status AttachmentStatus) Valid() bool {
	switch status {
	case AttachmentStatusUploaded, AttachmentStatusBound, AttachmentStatusDeleting:
		return true
	default:
		return false
	}
}

type Attachment struct {
	ID          string
	StorageKey  string
	FileName    string
	ContentType string
	SizeBytes   int64
	SHA256      [32]byte
	Status      AttachmentStatus
	CreatedBy   string
	ExpiresAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AttachmentSummary struct {
	ID          string
	FileName    string
	ContentType string
	SizeBytes   int64
	SHA256      string
	CreatedAt   time.Time
}

type UploadAttachmentCommand struct {
	FileName string
	Content  []byte
}

type UploadAttachmentResult struct {
	Attachment AttachmentSummary
	ExpiresAt  time.Time
}

type DeleteAttachmentCommand struct {
	ID string
}

type DeleteAttachmentResult struct {
	ID string
}

type DownloadAttachmentCommand struct {
	ID string
}

type DownloadAttachmentResult struct {
	Attachment AttachmentSummary
	Content    []byte
}

type CleanupAttachmentsCommand struct {
	BatchSize int
}

type CleanupAttachmentsResult struct {
	Scanned          int
	Deleted          int
	ResidualsScanned int
	ResidualsDeleted int
	Skipped          int
	Failed           int
}

type StoredAttachmentFile struct {
	FileName    string
	ContentType string
	SizeBytes   int64
	SHA256      [32]byte
}

type AttachmentFileStore interface {
	Write(storageKey, fileName string, content []byte) (StoredAttachmentFile, error)
	Read(storageKey string) ([]byte, error)
	Delete(storageKey string) error
	DeleteResidual(storageKey string) error
	ListResiduals(stableBefore time.Time) ([]string, error)
}

type AttachmentRepository interface {
	CreateAttachment(context.Context, Attachment) error
	GetAttachment(context.Context, string) (Attachment, bool, error)
	SetAttachmentStatus(context.Context, string, AttachmentStatus, AttachmentStatus, time.Time) (bool, error)
	DeleteAttachment(context.Context, string, AttachmentStatus) (bool, error)
	ListAttachmentsForCleanup(context.Context, time.Time, int) ([]Attachment, error)
	AttachmentStorageKeyExists(context.Context, string) (bool, error)
}

type AttachmentService struct {
	repository AttachmentRepository
	files      AttachmentFileStore
	clock      Clock
	newID      func() (string, error)
}

func NewAttachmentService(repository AttachmentRepository, files AttachmentFileStore) (*AttachmentService, error) {
	return NewAttachmentServiceWithDependencies(repository, files, nil, nil)
}

func NewAttachmentServiceWithDependencies(repository AttachmentRepository, files AttachmentFileStore, clock Clock, newID func() (string, error)) (*AttachmentService, error) {
	if repository == nil {
		return nil, errors.New("attachment repository is required")
	}
	if files == nil {
		return nil, errors.New("attachment file store is required")
	}
	if newID == nil {
		newID = NewAttachmentID
	}
	return &AttachmentService{repository: repository, files: files, clock: clock, newID: newID}, nil
}

func (service *AttachmentService) UploadAttachment(ctx context.Context, principal auth.Principal, command UploadAttachmentCommand) (UploadAttachmentResult, error) {
	if !CanWrite(principal) || principal.UserID == "" {
		return UploadAttachmentResult{}, ErrForbidden
	}
	fileName, err := NormalizeAttachmentFileName(command.FileName)
	if err != nil {
		return UploadAttachmentResult{}, err
	}
	contentType, digest, err := inspectAttachmentContent(command.Content)
	if err != nil {
		return UploadAttachmentResult{}, err
	}
	id, err := service.newID()
	if err != nil {
		return UploadAttachmentResult{}, Internal(fmt.Errorf("generate attachment ID: %w", err))
	}
	if !ValidAttachmentID(id) {
		return UploadAttachmentResult{}, Internal(errors.New("generated invalid attachment ID"))
	}
	storageKey := id

	stored, err := service.files.Write(storageKey, fileName, command.Content)
	if err != nil {
		return UploadAttachmentResult{}, fmt.Errorf("write attachment file: %w", err)
	}
	if stored.FileName != fileName || stored.ContentType != contentType || stored.SizeBytes != int64(len(command.Content)) || stored.SHA256 != digest {
		_ = service.files.Delete(storageKey)
		return UploadAttachmentResult{}, Internal(errors.New("attachment file store verification failed"))
	}

	now := UTCNow(service.clock).Truncate(time.Second)
	expiresAt := now.Add(AttachmentUploadLifetime)
	attachment := Attachment{
		ID: id, StorageKey: storageKey, FileName: fileName, ContentType: contentType,
		SizeBytes: stored.SizeBytes, SHA256: stored.SHA256, Status: AttachmentStatusUploaded,
		CreatedBy: principal.UserID, ExpiresAt: &expiresAt, CreatedAt: now, UpdatedAt: now,
	}
	if err := ValidateAttachment(attachment); err != nil {
		_ = service.files.Delete(storageKey)
		return UploadAttachmentResult{}, err
	}
	if err := service.repository.CreateAttachment(ctx, attachment); err != nil {
		if compensationErr := service.files.Delete(storageKey); compensationErr != nil {
			return UploadAttachmentResult{}, Internal(errors.Join(fmt.Errorf("create attachment metadata: %w", err), fmt.Errorf("compensate attachment file: %w", compensationErr)))
		}
		return UploadAttachmentResult{}, fmt.Errorf("create attachment metadata: %w", err)
	}
	return UploadAttachmentResult{Attachment: attachment.Summary(), ExpiresAt: expiresAt}, nil
}

func (service *AttachmentService) DeleteAttachment(ctx context.Context, principal auth.Principal, command DeleteAttachmentCommand) (DeleteAttachmentResult, error) {
	if !CanWrite(principal) || principal.UserID == "" {
		return DeleteAttachmentResult{}, ErrForbidden
	}
	if !ValidAttachmentID(command.ID) {
		return DeleteAttachmentResult{}, ErrNotFound
	}
	attachment, found, err := service.repository.GetAttachment(ctx, command.ID)
	if err != nil {
		return DeleteAttachmentResult{}, fmt.Errorf("get attachment before delete: %w", err)
	}
	if !found || attachment.CreatedBy != principal.UserID {
		return DeleteAttachmentResult{}, ErrNotFound
	}
	if attachment.Status == AttachmentStatusBound {
		return DeleteAttachmentResult{}, ErrStateConflict
	}
	if attachment.Status != AttachmentStatusUploaded {
		return DeleteAttachmentResult{}, ErrNotFound
	}
	if err := ValidateAttachment(attachment); err != nil {
		return DeleteAttachmentResult{}, err
	}
	claimed, err := service.repository.SetAttachmentStatus(ctx, attachment.ID, AttachmentStatusUploaded, AttachmentStatusDeleting, UTCNow(service.clock).Truncate(time.Second))
	if err != nil {
		return DeleteAttachmentResult{}, fmt.Errorf("mark attachment deleting: %w", err)
	}
	if !claimed {
		return DeleteAttachmentResult{}, ErrNotFound
	}
	if err := service.files.Delete(attachment.StorageKey); err != nil {
		return DeleteAttachmentResult{}, fmt.Errorf("delete attachment file: %w", err)
	}
	removed, err := service.repository.DeleteAttachment(ctx, attachment.ID, AttachmentStatusDeleting)
	if err != nil {
		return DeleteAttachmentResult{}, fmt.Errorf("delete attachment metadata: %w", err)
	}
	if !removed {
		return DeleteAttachmentResult{}, Internal(errors.New("deleting attachment metadata disappeared"))
	}
	return DeleteAttachmentResult{ID: attachment.ID}, nil
}

func (service *AttachmentService) DownloadAttachment(ctx context.Context, principal auth.Principal, command DownloadAttachmentCommand) (DownloadAttachmentResult, error) {
	if !CanRead(principal) {
		return DownloadAttachmentResult{}, ErrForbidden
	}
	if !ValidAttachmentID(command.ID) {
		return DownloadAttachmentResult{}, ErrNotFound
	}
	attachment, found, err := service.repository.GetAttachment(ctx, command.ID)
	if err != nil {
		return DownloadAttachmentResult{}, fmt.Errorf("get attachment for download: %w", err)
	}
	if !found || attachment.Status != AttachmentStatusBound {
		return DownloadAttachmentResult{}, ErrNotFound
	}
	if err := ValidateAttachment(attachment); err != nil {
		return DownloadAttachmentResult{}, err
	}
	content, err := service.files.Read(attachment.StorageKey)
	if err != nil {
		return DownloadAttachmentResult{}, fmt.Errorf("read attachment file: %w", err)
	}
	contentType, digest, err := inspectAttachmentContent(content)
	if err != nil || int64(len(content)) != attachment.SizeBytes || contentType != attachment.ContentType || digest != attachment.SHA256 {
		return DownloadAttachmentResult{}, Internal(errors.New("stored attachment content verification failed"))
	}
	return DownloadAttachmentResult{Attachment: attachment.Summary(), Content: content}, nil
}

func (service *AttachmentService) CleanupAttachments(ctx context.Context, command CleanupAttachmentsCommand) (CleanupAttachmentsResult, error) {
	batchSize, err := normalizeAttachmentBatchSize(command.BatchSize)
	if err != nil {
		return CleanupAttachmentsResult{}, err
	}
	now := UTCNow(service.clock).Truncate(time.Second)
	candidates, err := service.repository.ListAttachmentsForCleanup(ctx, now, batchSize)
	if err != nil {
		return CleanupAttachmentsResult{}, fmt.Errorf("list attachments for cleanup: %w", err)
	}
	result := CleanupAttachmentsResult{}
	for _, attachment := range candidates {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		result.Scanned++
		if !ValidAttachmentID(attachment.ID) || !ValidAttachmentStorageKey(attachment.StorageKey) {
			result.Failed++
			continue
		}
		if attachment.Status == AttachmentStatusUploaded {
			if attachment.ExpiresAt == nil || attachment.ExpiresAt.After(now) {
				result.Skipped++
				continue
			}
			claimed, claimErr := service.repository.SetAttachmentStatus(ctx, attachment.ID, AttachmentStatusUploaded, AttachmentStatusDeleting, now)
			if claimErr != nil {
				result.Failed++
				continue
			}
			if !claimed {
				result.Skipped++
				continue
			}
		} else if attachment.Status != AttachmentStatusDeleting {
			result.Skipped++
			continue
		}
		if err := service.files.Delete(attachment.StorageKey); err != nil {
			result.Failed++
			continue
		}
		removed, removeErr := service.repository.DeleteAttachment(ctx, attachment.ID, AttachmentStatusDeleting)
		if removeErr != nil || !removed {
			result.Failed++
			continue
		}
		result.Deleted++
	}

	residuals, err := service.files.ListResiduals(now.Add(-AttachmentUploadLifetime))
	if err != nil {
		return result, fmt.Errorf("list residual attachment files: %w", err)
	}
	for _, storageKey := range residuals {
		if result.ResidualsScanned >= batchSize {
			break
		}
		if err := ctx.Err(); err != nil {
			return result, err
		}
		result.ResidualsScanned++
		if !ValidAttachmentStorageKey(storageKey) {
			result.Skipped++
			continue
		}
		exists, existsErr := service.repository.AttachmentStorageKeyExists(ctx, storageKey)
		if existsErr != nil {
			result.Failed++
			continue
		}
		if exists {
			result.Skipped++
			continue
		}
		if err := service.files.DeleteResidual(storageKey); err != nil {
			result.Failed++
			continue
		}
		result.ResidualsDeleted++
	}
	return result, nil
}

func NormalizeAttachmentFileName(fileName string) (string, error) {
	if !utf8.ValidString(fileName) {
		return "", attachmentValidation("fileName", "must be valid UTF-8")
	}
	fileName = strings.TrimSpace(strings.ReplaceAll(fileName, "\\", "/"))
	if separator := strings.LastIndexByte(fileName, '/'); separator >= 0 {
		fileName = fileName[separator+1:]
	}
	fileName = strings.Map(func(value rune) rune {
		if value == 0 || unicode.IsControl(value) {
			return '_'
		}
		return value
	}, fileName)
	fileName = strings.Trim(strings.TrimSpace(fileName), ".")
	if fileName == "" {
		return "", attachmentValidation("fileName", "must contain a display name")
	}
	if len([]byte(fileName)) > MaxAttachmentFileNameBytes {
		return "", attachmentValidation("fileName", "must be at most 255 UTF-8 bytes")
	}
	return fileName, nil
}

func ValidateAttachmentIDs(ids []string) error {
	if len(ids) > 10 {
		return attachmentValidation("attachmentIds", "must contain at most 10 IDs")
	}
	seen := make(map[string]struct{}, len(ids))
	for index, id := range ids {
		field := fmt.Sprintf("attachmentIds[%d]", index)
		if !ValidAttachmentID(id) {
			return attachmentValidation(field, "must be an att_ identifier")
		}
		if _, exists := seen[id]; exists {
			return attachmentValidation(field, "must be unique")
		}
		seen[id] = struct{}{}
	}
	return nil
}

func ValidateAttachment(value Attachment) error {
	if !ValidAttachmentID(value.ID) || !ValidAttachmentStorageKey(value.StorageKey) || !value.Status.Valid() || value.CreatedBy == "" {
		return Internal(errors.New("invalid attachment identity or status"))
	}
	fileName, err := NormalizeAttachmentFileName(value.FileName)
	if err != nil || fileName != value.FileName || !allowedAttachmentContentType(value.ContentType) || value.SizeBytes < 1 || value.SizeBytes > MaxAttachmentSizeBytes {
		return Internal(errors.New("invalid attachment file metadata"))
	}
	if value.SHA256 == ([32]byte{}) || !validAttachmentTime(value.CreatedAt) || !validAttachmentTime(value.UpdatedAt) || value.UpdatedAt.Before(value.CreatedAt) {
		return Internal(errors.New("invalid attachment audit metadata"))
	}
	if value.Status == AttachmentStatusUploaded {
		if value.ExpiresAt == nil || !validAttachmentTime(*value.ExpiresAt) || !value.ExpiresAt.Equal(value.CreatedAt.Add(AttachmentUploadLifetime)) {
			return Internal(errors.New("invalid uploaded attachment expiry"))
		}
	} else if value.ExpiresAt != nil {
		return Internal(errors.New("bound or deleting attachment must not expire"))
	}
	return nil
}

func (attachment Attachment) Summary() AttachmentSummary {
	return AttachmentSummary{
		ID: attachment.ID, FileName: attachment.FileName, ContentType: attachment.ContentType,
		SizeBytes: attachment.SizeBytes, SHA256: hex.EncodeToString(attachment.SHA256[:]), CreatedAt: attachment.CreatedAt,
	}
}

func inspectAttachmentContent(content []byte) (string, [32]byte, error) {
	if len(content) < 1 || int64(len(content)) > MaxAttachmentSizeBytes {
		return "", [32]byte{}, attachmentValidation("content", "must be between 1 byte and 10 MiB")
	}
	contentType := ""
	switch {
	case bytes.HasPrefix(content, []byte("%PDF-")):
		contentType = "application/pdf"
	case bytes.HasPrefix(content, []byte("\x89PNG\r\n\x1a\n")):
		if _, err := png.DecodeConfig(bytes.NewReader(content)); err != nil {
			return "", [32]byte{}, attachmentValidation("content", "must be PDF, PNG, or JPEG")
		}
		contentType = "image/png"
	case len(content) >= 3 && content[0] == 0xff && content[1] == 0xd8 && content[2] == 0xff:
		if _, err := jpeg.DecodeConfig(bytes.NewReader(content)); err != nil {
			return "", [32]byte{}, attachmentValidation("content", "must be PDF, PNG, or JPEG")
		}
		contentType = "image/jpeg"
	default:
		return "", [32]byte{}, attachmentValidation("content", "must be PDF, PNG, or JPEG")
	}
	return contentType, sha256.Sum256(content), nil
}

func allowedAttachmentContentType(contentType string) bool {
	return contentType == "application/pdf" || contentType == "image/png" || contentType == "image/jpeg"
}

func normalizeAttachmentBatchSize(batchSize int) (int, error) {
	if batchSize == 0 {
		return DefaultAttachmentBatchSize, nil
	}
	if batchSize < 1 || batchSize > MaxAttachmentCleanupBatch {
		return 0, attachmentValidation("batchSize", "must be between 1 and 1000")
	}
	return batchSize, nil
}

func attachmentValidation(field, message string) error {
	return &ValidationError{Details: []FieldError{{Field: field, Message: message}}}
}

func validAttachmentTime(value time.Time) bool {
	return !value.IsZero() && value.Location() == time.UTC && value.Nanosecond() == 0
}
