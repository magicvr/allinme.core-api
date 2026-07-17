package files_test

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/magicvr/allinme.core-api/internal/files"
)

const (
	keyOne   = "att_00000000000000000000000000000001"
	keyTwo   = "att_00000000000000000000000000000002"
	keyThree = "att_00000000000000000000000000000003"
)

func TestLocalWriteReadDelete(t *testing.T) {
	dataDir := t.TempDir()
	store, err := files.NewLocal(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	content := []byte("%PDF-1.7\nattachment")
	stored, err := store.Write(keyOne, "../customer\\\r\ninvoice.pdf", content)
	if err != nil {
		t.Fatal(err)
	}
	if stored.FileName != "invoice.pdf" || stored.ContentType != "application/pdf" || stored.SizeBytes != int64(len(content)) || stored.SHA256 != sha256.Sum256(content) {
		t.Fatalf("Write() metadata = %+v", stored)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "attachments", "temp", keyOne)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("temp file error = %v", err)
	}
	persisted, err := os.ReadFile(filepath.Join(dataDir, "attachments", "content", keyOne))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(persisted, content) {
		t.Fatalf("persisted content = %q", persisted)
	}
	read, err := store.Read(keyOne)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(read, content) {
		t.Fatalf("Read() = %q", read)
	}
	if err := store.Delete(keyOne); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(keyOne); err != nil {
		t.Fatalf("second Delete() error = %v", err)
	}
	if _, err := store.Read(keyOne); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Read() after delete error = %v", err)
	}
}

func TestLocalDetectsSupportedContentAndValidatesImageHeaders(t *testing.T) {
	store, err := files.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	pngContent := encodePNG(t)
	jpegContent := encodeJPEG(t)
	for _, test := range []struct {
		key         string
		fileName    string
		content     []byte
		contentType string
	}{
		{keyOne, "scan.bin", []byte("%PDF-1.4\nbody"), "application/pdf"},
		{keyTwo, "photo.dat", pngContent, "image/png"},
		{keyThree, "photo.exe", jpegContent, "image/jpeg"},
	} {
		stored, err := store.Write(test.key, test.fileName, test.content)
		if err != nil {
			t.Fatalf("Write(%q) error = %v", test.fileName, err)
		}
		if stored.ContentType != test.contentType {
			t.Fatalf("Write(%q) content type = %q", test.fileName, stored.ContentType)
		}
	}

	for index, content := range [][]byte{
		nil,
		[]byte("plain text"),
		[]byte("\x89PNG\r\n\x1a\ninvalid"),
		[]byte{0xff, 0xd8, 0xff, 0x00},
	} {
		key := "att_fffffffffffffffffffffffffffffff" + string(rune('0'+index))
		if _, err := store.Write(key, "invalid.bin", content); !errors.Is(err, files.ErrUnsupportedContent) {
			t.Fatalf("Write(invalid %d) error = %v", index, err)
		}
	}
}

func TestLocalEnforcesMaximumSize(t *testing.T) {
	store, err := files.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	maximum := make([]byte, files.MaxSizeBytes)
	copy(maximum, "%PDF-")
	if _, err := store.Write(keyOne, "maximum.pdf", maximum); err != nil {
		t.Fatalf("Write(maximum) error = %v", err)
	}
	tooLarge := append(maximum, 0)
	if _, err := store.Write(keyTwo, "large.pdf", tooLarge); !errors.Is(err, files.ErrFileTooLarge) {
		t.Fatalf("Write(too large) error = %v", err)
	}
}

func TestLocalRejectsNoncanonicalStorageKeys(t *testing.T) {
	dataDir := t.TempDir()
	store, err := files.NewLocal(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	invalid := []string{
		"", "att_1", "ATT_00000000000000000000000000000001",
		"att_0000000000000000000000000000000A",
		"att_000000000000000000000000000000001",
		"../" + keyOne, keyOne + "/other", keyOne + "\\other",
		filepath.Join(dataDir, keyOne), "C:\\" + keyOne,
	}
	for _, key := range invalid {
		if _, err := store.Write(key, "file.pdf", []byte("%PDF-")); !errors.Is(err, files.ErrInvalidStorageKey) {
			t.Errorf("Write(%q) error = %v", key, err)
		}
		if _, err := store.Read(key); !errors.Is(err, files.ErrInvalidStorageKey) {
			t.Errorf("Read(%q) error = %v", key, err)
		}
		if err := store.Delete(key); !errors.Is(err, files.ErrInvalidStorageKey) {
			t.Errorf("Delete(%q) error = %v", key, err)
		}
		if err := store.DeleteResidual(key); !errors.Is(err, files.ErrInvalidStorageKey) {
			t.Errorf("DeleteResidual(%q) error = %v", key, err)
		}
	}
	entries, err := os.ReadDir(filepath.Join(dataDir, "attachments", "content"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("content entries = %v", entries)
	}
}

func TestLocalWriteIsExclusive(t *testing.T) {
	dataDir := t.TempDir()
	store, err := files.NewLocal(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	contentDir := filepath.Join(dataDir, "attachments", "content")
	tempDir := filepath.Join(dataDir, "attachments", "temp")
	if err := os.WriteFile(filepath.Join(contentDir, keyOne), []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Write(keyOne, "replacement.pdf", []byte("%PDF-new")); !errors.Is(err, files.ErrFileExists) {
		t.Fatalf("Write(existing final) error = %v", err)
	}
	unchanged, err := os.ReadFile(filepath.Join(contentDir, keyOne))
	if err != nil {
		t.Fatal(err)
	}
	if string(unchanged) != "existing" {
		t.Fatalf("existing final = %q", unchanged)
	}
	if err := os.WriteFile(filepath.Join(tempDir, keyTwo), []byte("residual"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Write(keyTwo, "new.pdf", []byte("%PDF-new")); !errors.Is(err, files.ErrFileExists) {
		t.Fatalf("Write(existing temp) error = %v", err)
	}
}

func TestLocalSanitizesDisplayFilename(t *testing.T) {
	store, err := files.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	stored, err := store.Write(keyOne, "\x00\r\n..", []byte("%PDF-1.7"))
	if err != nil {
		t.Fatal(err)
	}
	if stored.FileName != "attachment.pdf" {
		t.Fatalf("fallback FileName = %q", stored.FileName)
	}
	longName := strings.Repeat("界", 100) + ".png"
	stored, err = store.Write(keyTwo, longName, encodePNG(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.FileName) > 255 || !utf8.ValidString(stored.FileName) {
		t.Fatalf("long FileName has %d bytes and valid=%v", len(stored.FileName), utf8.ValidString(stored.FileName))
	}
}

func TestLocalListsStableCanonicalResiduals(t *testing.T) {
	dataDir := t.TempDir()
	store, err := files.NewLocal(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	tempDir := filepath.Join(dataDir, "attachments", "temp")
	stableBefore := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	for _, residual := range []struct {
		name     string
		modified time.Time
	}{
		{keyTwo, stableBefore.Add(-time.Hour)},
		{keyOne, stableBefore},
		{keyThree, stableBefore.Add(time.Second)},
		{"att_NOT_CANONICAL", stableBefore.Add(-time.Hour)},
	} {
		path := filepath.Join(tempDir, residual.name)
		if err := os.WriteFile(path, []byte("temp"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, residual.modified, residual.modified); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(tempDir, "att_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), 0o700); err != nil {
		t.Fatal(err)
	}
	residuals, err := store.ListResiduals(stableBefore)
	if err != nil {
		t.Fatal(err)
	}
	if expected := []string{keyOne, keyTwo}; !reflect.DeepEqual(residuals, expected) {
		t.Fatalf("ListResiduals() = %v, want %v", residuals, expected)
	}
	if err := store.DeleteResidual(keyOne); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteResidual(keyOne); err != nil {
		t.Fatalf("second DeleteResidual() error = %v", err)
	}
	residuals, err = store.ListResiduals(stableBefore)
	if err != nil {
		t.Fatal(err)
	}
	if expected := []string{keyTwo}; !reflect.DeepEqual(residuals, expected) {
		t.Fatalf("ListResiduals() after delete = %v, want %v", residuals, expected)
	}
}

func TestLocalRejectsAttachmentRootLink(t *testing.T) {
	dataDir := t.TempDir()
	target := t.TempDir()
	root := filepath.Join(dataDir, "attachments")
	if err := os.Symlink(target, root); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}
	if _, err := files.NewLocal(dataDir); err == nil {
		t.Fatal("NewLocal() accepted attachment root link")
	}
	if _, err := os.Stat(filepath.Join(target, "content")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("linked target was modified: %v", err)
	}
}

func TestLocalRejectsAttachmentRootReplacementEscape(t *testing.T) {
	dataDir := t.TempDir()
	store, err := files.NewLocal(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dataDir, "attachments")
	original := filepath.Join(dataDir, "attachments-original")
	target := t.TempDir()
	if err := os.Rename(root, original); err != nil {
		t.Skipf("attachment root replacement is unavailable: %v", err)
	}
	if err := os.Symlink(target, root); err != nil {
		if restoreErr := os.Rename(original, root); restoreErr != nil {
			t.Fatalf("symlink unavailable (%v) and root restore failed: %v", err, restoreErr)
		}
		t.Skipf("symlink creation is unavailable: %v", err)
	}
	if _, err := store.Write(keyOne, "escape.pdf", []byte("%PDF-1.7\nbody")); err == nil {
		t.Fatal("Write() followed replacement attachment root")
	}
	if _, err := os.Stat(filepath.Join(target, "content", keyOne)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("replacement target was modified: %v", err)
	}
}

func TestLocalPreservesValidReplacementCharacterInDisplayFilename(t *testing.T) {
	store, err := files.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	stored, err := store.Write(keyOne, "report�.pdf", []byte("%PDF-1.7\nbody"))
	if err != nil {
		t.Fatal(err)
	}
	if stored.FileName != "report�.pdf" {
		t.Fatalf("FileName = %q", stored.FileName)
	}
}

func TestLocalErrorsDoNotExposeAbsolutePaths(t *testing.T) {
	dataDir := t.TempDir()
	blocked := filepath.Join(dataDir, "private-data-root")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := files.NewLocal(blocked)
	if err == nil {
		t.Fatal("NewLocal() error = nil")
	}
	if strings.Contains(err.Error(), dataDir) || strings.Contains(err.Error(), blocked) {
		t.Fatalf("NewLocal() leaked path in %q", err)
	}

	store, err := files.NewLocal(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Read(keyOne)
	if err == nil {
		t.Fatal("Read(missing) error = nil")
	}
	if strings.Contains(err.Error(), dataDir) {
		t.Fatalf("Read() leaked path in %q", err)
	}
}

func encodePNG(t *testing.T) []byte {
	t.Helper()
	var output bytes.Buffer
	picture := image.NewRGBA(image.Rect(0, 0, 2, 2))
	picture.Set(0, 0, color.RGBA{R: 255, A: 255})
	if err := png.Encode(&output, picture); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func encodeJPEG(t *testing.T) []byte {
	t.Helper()
	var output bytes.Buffer
	picture := image.NewRGBA(image.Rect(0, 0, 2, 2))
	picture.Set(0, 0, color.RGBA{G: 255, A: 255})
	if err := jpeg.Encode(&output, picture, nil); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
