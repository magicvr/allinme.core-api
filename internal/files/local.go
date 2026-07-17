package files

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/magicvr/allinme.core-api/internal/order"
)

const MaxSizeBytes int64 = 10 * 1024 * 1024

var (
	ErrInvalidStorageKey  = errors.New("invalid attachment storage key")
	ErrFileExists         = errors.New("attachment file already exists")
	ErrFileTooLarge       = errors.New("attachment file exceeds 10 MiB")
	ErrUnsupportedContent = errors.New("unsupported attachment content")
)

type Local struct {
	dataDir string
}

var _ order.AttachmentFileStore = (*Local)(nil)

func NewLocal(dataDir string) (*Local, error) {
	if strings.TrimSpace(dataDir) == "" {
		return nil, fmt.Errorf("initialize attachment storage: DATA_DIR is required")
	}
	absolute, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, safeFileError("resolve attachment storage", err)
	}
	root := filepath.Join(filepath.Clean(absolute), "attachments")
	if err := rejectAttachmentRoot(root); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absolute, 0o750); err != nil {
		return nil, safeFileError("create attachment data directory", err)
	}
	dataRoot, err := os.OpenRoot(absolute)
	if err != nil {
		return nil, safeFileError("open attachment data directory", err)
	}
	defer dataRoot.Close()
	if err := dataRoot.MkdirAll(filepath.Join("attachments", "temp"), 0o750); err != nil {
		return nil, safeFileError("create attachment temp directory", err)
	}
	if err := dataRoot.MkdirAll(filepath.Join("attachments", "content"), 0o750); err != nil {
		return nil, safeFileError("create attachment content directory", err)
	}
	attachmentRoot, err := dataRoot.OpenRoot("attachments")
	if err != nil {
		return nil, safeFileError("open attachment storage", err)
	}
	if err := attachmentRoot.Close(); err != nil {
		return nil, safeFileError("close attachment storage", err)
	}
	return &Local{dataDir: absolute}, nil
}

func (local *Local) openRoot() (*os.Root, error) {
	dataRoot, err := os.OpenRoot(local.dataDir)
	if err != nil {
		return nil, safeFileError("open attachment data directory", err)
	}
	attachmentRoot, err := dataRoot.OpenRoot("attachments")
	closeErr := dataRoot.Close()
	if err != nil {
		return nil, safeFileError("open attachment storage", err)
	}
	if closeErr != nil {
		attachmentRoot.Close()
		return nil, safeFileError("close attachment data directory", closeErr)
	}
	return attachmentRoot, nil
}

func (local *Local) Write(storageKey, fileName string, content []byte) (order.StoredAttachmentFile, error) {
	if err := validateStorageKey(storageKey); err != nil {
		return order.StoredAttachmentFile{}, err
	}
	metadata, err := inspect(fileName, content)
	if err != nil {
		return order.StoredAttachmentFile{}, err
	}

	root, err := local.openRoot()
	if err != nil {
		return order.StoredAttachmentFile{}, err
	}
	defer root.Close()

	finalName := filepath.Join("content", storageKey)
	if _, err := root.Stat(finalName); err == nil {
		return order.StoredAttachmentFile{}, ErrFileExists
	} else if !errors.Is(err, fs.ErrNotExist) {
		return order.StoredAttachmentFile{}, safeFileError("inspect attachment content", err)
	}

	tempName := filepath.Join("temp", storageKey)
	temp, err := root.OpenFile(tempName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return order.StoredAttachmentFile{}, ErrFileExists
		}
		return order.StoredAttachmentFile{}, safeFileError("create attachment temp file", err)
	}
	keepTemp := true
	defer func() {
		if keepTemp {
			_ = temp.Close()
			_ = root.Remove(tempName)
		}
	}()

	if err := writeAll(temp, content); err != nil {
		return order.StoredAttachmentFile{}, safeFileError("write attachment temp file", err)
	}
	if err := temp.Sync(); err != nil {
		return order.StoredAttachmentFile{}, safeFileError("sync attachment temp file", err)
	}
	if err := temp.Close(); err != nil {
		return order.StoredAttachmentFile{}, safeFileError("close attachment temp file", err)
	}
	if _, err := root.Stat(finalName); err == nil {
		return order.StoredAttachmentFile{}, ErrFileExists
	} else if !errors.Is(err, fs.ErrNotExist) {
		return order.StoredAttachmentFile{}, safeFileError("inspect attachment content", err)
	}
	if err := root.Rename(tempName, finalName); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return order.StoredAttachmentFile{}, ErrFileExists
		}
		return order.StoredAttachmentFile{}, safeFileError("publish attachment content", err)
	}
	keepTemp = false
	return metadata, nil
}

func (local *Local) Read(storageKey string) ([]byte, error) {
	if err := validateStorageKey(storageKey); err != nil {
		return nil, err
	}
	root, err := local.openRoot()
	if err != nil {
		return nil, err
	}
	defer root.Close()
	file, err := root.Open(filepath.Join("content", storageKey))
	if err != nil {
		return nil, safeFileError("open attachment content", err)
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, MaxSizeBytes+1))
	if err != nil {
		return nil, safeFileError("read attachment content", err)
	}
	if int64(len(content)) > MaxSizeBytes {
		return nil, ErrFileTooLarge
	}
	return content, nil
}

func (local *Local) Delete(storageKey string) error {
	return local.deleteFrom("content", storageKey)
}

// DeleteResidual removes a canonical temp file discovered by ListResiduals.
// Missing residuals are already clean and therefore succeed.
func (local *Local) DeleteResidual(storageKey string) error {
	return local.deleteFrom("temp", storageKey)
}

func (local *Local) deleteFrom(directory, storageKey string) error {
	if err := validateStorageKey(storageKey); err != nil {
		return err
	}
	root, err := local.openRoot()
	if err != nil {
		return err
	}
	defer root.Close()
	if err := root.Remove(filepath.Join(directory, storageKey)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return safeFileError("delete attachment file", err)
	}
	return nil
}

// ListResiduals returns canonical temp-file keys whose modification time is not
// newer than stableBefore. Results are sorted so one-shot cleanup is repeatable.
func (local *Local) ListResiduals(stableBefore time.Time) ([]string, error) {
	root, err := local.openRoot()
	if err != nil {
		return nil, err
	}
	defer root.Close()
	directory, err := root.Open("temp")
	if err != nil {
		return nil, safeFileError("open attachment temp directory", err)
	}
	defer directory.Close()
	entries, err := directory.ReadDir(-1)
	if err != nil {
		return nil, safeFileError("enumerate attachment residuals", err)
	}
	residuals := make([]string, 0, len(entries))
	for _, entry := range entries {
		if validateStorageKey(entry.Name()) != nil || entry.Type()&fs.ModeSymlink != 0 {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, safeFileError("inspect attachment residual", err)
		}
		if info.Mode().IsRegular() && !info.ModTime().After(stableBefore) {
			residuals = append(residuals, entry.Name())
		}
	}
	sort.Strings(residuals)
	return residuals, nil
}

func validateStorageKey(storageKey string) error {
	if len(storageKey) != 36 || !strings.HasPrefix(storageKey, "att_") {
		return ErrInvalidStorageKey
	}
	for _, character := range storageKey[4:] {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return ErrInvalidStorageKey
		}
	}
	return nil
}

func inspect(fileName string, content []byte) (order.StoredAttachmentFile, error) {
	if int64(len(content)) > MaxSizeBytes {
		return order.StoredAttachmentFile{}, ErrFileTooLarge
	}
	contentType, extension, err := detectContent(content)
	if err != nil {
		return order.StoredAttachmentFile{}, err
	}
	return order.StoredAttachmentFile{
		FileName:    sanitizeDisplayFilename(fileName, extension),
		ContentType: contentType,
		SizeBytes:   int64(len(content)),
		SHA256:      sha256.Sum256(content),
	}, nil
}

func detectContent(content []byte) (string, string, error) {
	switch {
	case bytes.HasPrefix(content, []byte("%PDF-")):
		return "application/pdf", ".pdf", nil
	case bytes.HasPrefix(content, []byte("\x89PNG\r\n\x1a\n")):
		if _, err := png.DecodeConfig(bytes.NewReader(content)); err != nil {
			return "", "", ErrUnsupportedContent
		}
		return "image/png", ".png", nil
	case len(content) >= 3 && content[0] == 0xff && content[1] == 0xd8 && content[2] == 0xff:
		if _, err := jpeg.DecodeConfig(bytes.NewReader(content)); err != nil {
			return "", "", ErrUnsupportedContent
		}
		return "image/jpeg", ".jpg", nil
	default:
		return "", "", ErrUnsupportedContent
	}
}

func sanitizeDisplayFilename(fileName, fallbackExtension string) string {
	fileName = strings.ReplaceAll(fileName, "\\", "/")
	if index := strings.LastIndexByte(fileName, '/'); index >= 0 {
		fileName = fileName[index+1:]
	}
	fileName = strings.Map(func(character rune) rune {
		if unicode.IsControl(character) {
			return -1
		}
		return character
	}, fileName)
	fileName = strings.Trim(strings.TrimSpace(fileName), ".")
	if fileName == "" {
		fileName = "attachment" + fallbackExtension
	}
	return truncateUTF8(fileName, 255)
}

func truncateUTF8(value string, maximumBytes int) string {
	if len(value) <= maximumBytes {
		return value
	}
	value = value[:maximumBytes]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}

func writeAll(writer io.Writer, content []byte) error {
	for len(content) > 0 {
		written, err := writer.Write(content)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		content = content[written:]
	}
	return nil
}

type fileError struct {
	op    string
	cause error
}

func safeFileError(operation string, cause error) error {
	return fileError{op: operation, cause: cause}
}

func (err fileError) Error() string {
	switch {
	case errors.Is(err.cause, fs.ErrNotExist):
		return err.op + ": file does not exist"
	case errors.Is(err.cause, fs.ErrExist):
		return err.op + ": file already exists"
	case errors.Is(err.cause, fs.ErrPermission):
		return err.op + ": permission denied"
	default:
		return err.op + ": filesystem operation failed"
	}
}

func (err fileError) Unwrap() error { return err.cause }
