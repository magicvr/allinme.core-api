package order_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/order"
)

var (
	_ order.AttachmentRepository = (*attachmentRepository)(nil)
	_ order.AttachmentFileStore  = (*attachmentFiles)(nil)
)

func TestAttachmentIdentifiersAndOrderedIDValidation(t *testing.T) {
	for _, status := range []order.AttachmentStatus{order.AttachmentStatusUploaded, order.AttachmentStatusBound, order.AttachmentStatusDeleting} {
		if !status.Valid() {
			t.Errorf("status %q is invalid", status)
		}
	}
	if order.AttachmentStatus("UNKNOWN").Valid() {
		t.Fatal("unknown attachment status is valid")
	}

	id, err := order.NewAttachmentIDFrom(bytes.NewReader(bytes.Repeat([]byte{0xef}, 16)))
	if err != nil {
		t.Fatal(err)
	}
	if id != "att_efefefefefefefefefefefefefefefef" || !order.ValidAttachmentID(id) || !order.ValidAttachmentStorageKey(id) {
		t.Fatalf("attachment ID = %q", id)
	}
	valid := []string{"att_00000000000000000000000000000001", "att_00000000000000000000000000000002"}
	if err := order.ValidateAttachmentIDs(valid); err != nil {
		t.Fatal(err)
	}
	for _, ids := range [][]string{
		{"bad"},
		{valid[0], valid[0]},
		makeAttachmentIDs(11),
	} {
		if err := order.ValidateAttachmentIDs(ids); err == nil {
			t.Fatalf("ValidateAttachmentIDs(%v) error = nil", ids)
		}
	}
}

func TestNormalizeAttachmentFileNameSanitizesDisplayName(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{input: "  ..\\reports/quarter\x00ly.pdf  ", want: "quarter_ly.pdf"},
		{input: ".invoice.pdf", want: "invoice.pdf"},
		{input: "invoice.pdf...", want: "invoice.pdf"},
		{input: "report�.pdf", want: "report�.pdf"},
	} {
		name, err := order.NormalizeAttachmentFileName(test.input)
		if err != nil {
			t.Fatal(err)
		}
		if name != test.want {
			t.Fatalf("NormalizeAttachmentFileName(%q) = %q, want %q", test.input, name, test.want)
		}
	}
	for _, invalid := range []string{" ", "../..", string([]byte{0xff}), string(bytes.Repeat([]byte{'a'}, 256))} {
		if _, err := order.NormalizeAttachmentFileName(invalid); err == nil {
			t.Fatalf("NormalizeAttachmentFileName(%q) error = nil", invalid)
		}
	}
}

func TestUploadAttachmentWritesThenPersistsVerifiedMetadata(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	content := minimalPNG()
	repository := newAttachmentRepository()
	files := newAttachmentFiles()
	service := newAttachmentService(t, repository, files, now)

	result, err := service.UploadAttachment(context.Background(), attachmentPrincipal(auth.RoleOperator, "operator-1"), order.UploadAttachmentCommand{
		FileName: " C:\\fakepath\\receipt.png ",
		Content:  content,
	})
	if err != nil {
		t.Fatal(err)
	}
	id := "att_00000000000000000000000000000001"
	if len(files.writeCalls) != 1 || files.writeCalls[0].storageKey != id || files.writeCalls[0].fileName != "receipt.png" {
		t.Fatalf("write calls = %+v", files.writeCalls)
	}
	created := repository.values[id]
	if created.ID != id || created.StorageKey != id || created.Status != order.AttachmentStatusUploaded || created.CreatedBy != "operator-1" || created.ExpiresAt == nil || !created.ExpiresAt.Equal(now.Add(24*time.Hour)) {
		t.Fatalf("created attachment = %+v", created)
	}
	if result.Attachment.ID != id || result.Attachment.FileName != "receipt.png" || result.Attachment.ContentType != "image/png" || result.Attachment.SizeBytes != int64(len(content)) || result.ExpiresAt != now.Add(24*time.Hour) {
		t.Fatalf("upload result = %+v", result)
	}
}

func TestUploadAttachmentAcceptsSupportedDetectedTypes(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	for name, contentTypeAndContent := range map[string]struct {
		contentType string
		content     []byte
	}{
		"document.pdf": {contentType: "application/pdf", content: []byte("%PDF-1.7\n")},
		"photo.jpg":    {contentType: "image/jpeg", content: minimalJPEG()},
		"image.png":    {contentType: "image/png", content: minimalPNG()},
	} {
		t.Run(name, func(t *testing.T) {
			service := newAttachmentService(t, newAttachmentRepository(), newAttachmentFiles(), now)
			result, err := service.UploadAttachment(context.Background(), attachmentPrincipal(auth.RoleOperator, "operator"), order.UploadAttachmentCommand{FileName: name, Content: contentTypeAndContent.content})
			if err != nil {
				t.Fatal(err)
			}
			if result.Attachment.ContentType != contentTypeAndContent.contentType {
				t.Fatalf("content type = %q", result.Attachment.ContentType)
			}
		})
	}
}

func TestUploadAttachmentValidatesContentAndCompensatesMetadataFailure(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	principal := attachmentPrincipal(auth.RoleAdmin, "admin-1")
	for _, command := range []order.UploadAttachmentCommand{
		{FileName: "empty.pdf"},
		{FileName: "fake.pdf", Content: []byte("not a supported file")},
		{FileName: "fake.png", Content: []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}},
		{FileName: "fake.jpg", Content: []byte{0xff, 0xd8, 0xff}},
		{FileName: "large.png", Content: make([]byte, order.MaxAttachmentSizeBytes+1)},
	} {
		files := newAttachmentFiles()
		service := newAttachmentService(t, newAttachmentRepository(), files, now)
		if _, err := service.UploadAttachment(context.Background(), principal, command); err == nil {
			t.Fatalf("UploadAttachment(%q) error = nil", command.FileName)
		}
		if len(files.writeCalls) != 0 {
			t.Fatalf("invalid upload wrote file: %+v", files.writeCalls)
		}
	}

	repository := newAttachmentRepository()
	repository.createErr = errors.New("database failed")
	files := newAttachmentFiles()
	service := newAttachmentService(t, repository, files, now)
	if _, err := service.UploadAttachment(context.Background(), principal, order.UploadAttachmentCommand{FileName: "receipt.png", Content: minimalPNG()}); err == nil {
		t.Fatal("metadata failure error = nil")
	}
	if len(files.deleteCalls) != 1 || files.deleteCalls[0] != "att_00000000000000000000000000000001" {
		t.Fatalf("compensation deletes = %v", files.deleteCalls)
	}
}

func TestUploadAttachmentWriteFailureDoesNotDeleteExistingFile(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	files := newAttachmentFiles()
	id := "att_00000000000000000000000000000001"
	existing := []byte("%PDF-1.4\nexisting")
	files.contents[id] = bytes.Clone(existing)
	files.writeErr = errors.New("file exists")
	service := newAttachmentService(t, newAttachmentRepository(), files, now)
	if _, err := service.UploadAttachment(context.Background(), attachmentPrincipal(auth.RoleOperator, "operator"), order.UploadAttachmentCommand{FileName: "new.pdf", Content: []byte("%PDF-1.4\nnew")}); err == nil {
		t.Fatal("UploadAttachment() write error = nil")
	}
	if len(files.deleteCalls) != 0 || !bytes.Equal(files.contents[id], existing) {
		t.Fatalf("write failure changed existing file: deletes=%v content=%q", files.deleteCalls, files.contents[id])
	}
}

func TestDeleteAttachmentUsesHiddenNotFoundAndStateSequence(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	principal := attachmentPrincipal(auth.RoleOperator, "owner")

	t.Run("uploaded owner", func(t *testing.T) {
		repository := newAttachmentRepository()
		attachment := uploadedAttachment("01", "owner", now)
		repository.values[attachment.ID] = attachment
		files := newAttachmentFiles()
		files.contents[attachment.StorageKey] = minimalPNG()
		service := newAttachmentService(t, repository, files, now)
		result, err := service.DeleteAttachment(context.Background(), principal, order.DeleteAttachmentCommand{ID: attachment.ID})
		if err != nil {
			t.Fatal(err)
		}
		if result.ID != attachment.ID || len(repository.transitions) != 1 || repository.transitions[0].from != order.AttachmentStatusUploaded || repository.transitions[0].to != order.AttachmentStatusDeleting || len(files.deleteCalls) != 1 {
			t.Fatalf("delete sequence result=%+v transitions=%+v files=%v", result, repository.transitions, files.deleteCalls)
		}
		if _, exists := repository.values[attachment.ID]; exists {
			t.Fatal("metadata remains after delete")
		}
	})

	for name, test := range map[string]struct {
		attachment order.Attachment
		id         string
		want       error
	}{
		"invalid":        {id: "bad", want: order.ErrNotFound},
		"missing":        {id: "att_00000000000000000000000000000002", want: order.ErrNotFound},
		"other owner":    {attachment: uploadedAttachment("03", "other", now), id: "att_00000000000000000000000000000003", want: order.ErrNotFound},
		"bound owner":    {attachment: boundAttachment("04", "owner", now), id: "att_00000000000000000000000000000004", want: order.ErrStateConflict},
		"deleting owner": {attachment: deletingAttachment("05", "owner", now), id: "att_00000000000000000000000000000005", want: order.ErrNotFound},
	} {
		t.Run(name, func(t *testing.T) {
			repository := newAttachmentRepository()
			if test.attachment.ID != "" {
				repository.values[test.attachment.ID] = test.attachment
			}
			files := newAttachmentFiles()
			service := newAttachmentService(t, repository, files, now)
			if _, err := service.DeleteAttachment(context.Background(), principal, order.DeleteAttachmentCommand{ID: test.id}); !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
			if len(files.deleteCalls) != 0 {
				t.Fatalf("hidden delete touched file store: %v", files.deleteCalls)
			}
		})
	}
}

func TestDownloadAttachmentRequiresBoundAndVerifiesContent(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	content := minimalPNG()
	attachment := boundAttachment("06", "owner", now)
	repository := newAttachmentRepository()
	repository.values[attachment.ID] = attachment
	files := newAttachmentFiles()
	files.contents[attachment.StorageKey] = content
	service := newAttachmentService(t, repository, files, now)

	result, err := service.DownloadAttachment(context.Background(), attachmentPrincipal(auth.RoleViewer, "viewer"), order.DownloadAttachmentCommand{ID: attachment.ID})
	if err != nil {
		t.Fatal(err)
	}
	if result.Attachment.ID != attachment.ID || !bytes.Equal(result.Content, content) {
		t.Fatalf("download result = %+v", result)
	}

	files.contents[attachment.StorageKey] = append(content, 0)
	if _, err := service.DownloadAttachment(context.Background(), attachmentPrincipal(auth.RoleViewer, "viewer"), order.DownloadAttachmentCommand{ID: attachment.ID}); !errors.Is(err, order.ErrInternal) {
		t.Fatalf("corrupt download error = %v", err)
	}
	unbound := uploadedAttachment("07", "owner", now)
	repository.values[unbound.ID] = unbound
	if _, err := service.DownloadAttachment(context.Background(), attachmentPrincipal(auth.RoleAdmin, "admin"), order.DownloadAttachmentCommand{ID: unbound.ID}); !errors.Is(err, order.ErrNotFound) {
		t.Fatalf("unbound download error = %v", err)
	}
	if _, err := service.DownloadAttachment(context.Background(), auth.Principal{}, order.DownloadAttachmentCommand{ID: attachment.ID}); !errors.Is(err, order.ErrForbidden) {
		t.Fatalf("unauthorized download error = %v", err)
	}
}

func TestCleanupAttachmentsConvergesExpiredDeletingAndResiduals(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	repository := newAttachmentRepository()
	expired := uploadedAttachment("08", "owner", now.Add(-25*time.Hour))
	deleting := deletingAttachment("09", "owner", now.Add(-48*time.Hour))
	bound := boundAttachment("0a", "owner", now.Add(-48*time.Hour))
	repository.values[expired.ID] = expired
	repository.values[deleting.ID] = deleting
	repository.values[bound.ID] = bound
	files := newAttachmentFiles()
	files.contents[expired.StorageKey] = minimalPNG()
	files.contents[deleting.StorageKey] = minimalPNG()
	files.contents[bound.StorageKey] = minimalPNG()
	residual := "att_0000000000000000000000000000000b"
	files.contents[residual] = minimalPNG()
	files.residuals = []string{residual, bound.StorageKey, "unsafe-key"}
	service := newAttachmentService(t, repository, files, now)

	result, err := service.CleanupAttachments(context.Background(), order.CleanupAttachmentsCommand{BatchSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if result.Scanned != 2 || result.Deleted != 2 || result.ResidualsScanned != 2 || result.ResidualsDeleted != 1 || result.Skipped != 1 || result.Failed != 0 {
		t.Fatalf("cleanup result = %+v", result)
	}
	if _, exists := repository.values[bound.ID]; !exists {
		t.Fatal("cleanup deleted bound attachment")
	}
	if _, exists := files.contents[bound.StorageKey]; !exists {
		t.Fatal("cleanup deleted bound file")
	}

	result, err = service.CleanupAttachments(context.Background(), order.CleanupAttachmentsCommand{BatchSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if result.Deleted != 0 || result.ResidualsDeleted != 0 {
		t.Fatalf("repeated cleanup result = %+v", result)
	}
}

func makeAttachmentIDs(count int) []string {
	result := make([]string, count)
	for index := range result {
		result[index] = "att_0000000000000000000000000000000" + string(rune('a'+index))
	}
	return result
}

func attachmentPrincipal(role auth.Role, userID string) auth.Principal {
	return auth.Principal{Role: role, UserID: userID}
}

func newAttachmentService(t *testing.T, repository order.AttachmentRepository, files order.AttachmentFileStore, now time.Time) *order.AttachmentService {
	t.Helper()
	service, err := order.NewAttachmentServiceWithDependencies(repository, files, func() time.Time { return now }, func() (string, error) {
		return "att_00000000000000000000000000000001", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func minimalPNG() []byte {
	var encoded bytes.Buffer
	imageValue := image.NewRGBA(image.Rect(0, 0, 1, 1))
	imageValue.Set(0, 0, color.RGBA{R: 0x22, G: 0x88, B: 0xcc, A: 0xff})
	if err := png.Encode(&encoded, imageValue); err != nil {
		panic(err)
	}
	return encoded.Bytes()
}

func minimalJPEG() []byte {
	var encoded bytes.Buffer
	imageValue := image.NewRGBA(image.Rect(0, 0, 1, 1))
	imageValue.Set(0, 0, color.RGBA{R: 0x22, G: 0x88, B: 0xcc, A: 0xff})
	if err := jpeg.Encode(&encoded, imageValue, nil); err != nil {
		panic(err)
	}
	return encoded.Bytes()
}

func uploadedAttachment(suffix, owner string, createdAt time.Time) order.Attachment {
	createdAt = createdAt.UTC().Truncate(time.Second)
	expiresAt := createdAt.Add(24 * time.Hour)
	content := minimalPNG()
	return order.Attachment{
		ID: "att_000000000000000000000000000000" + suffix, StorageKey: "att_000000000000000000000000000000" + suffix,
		FileName: "receipt.png", ContentType: "image/png", SizeBytes: int64(len(content)), SHA256: sha256.Sum256(content),
		Status: order.AttachmentStatusUploaded, CreatedBy: owner, ExpiresAt: &expiresAt, CreatedAt: createdAt, UpdatedAt: createdAt,
	}
}

func boundAttachment(suffix, owner string, createdAt time.Time) order.Attachment {
	value := uploadedAttachment(suffix, owner, createdAt)
	value.Status = order.AttachmentStatusBound
	value.ExpiresAt = nil
	return value
}

func deletingAttachment(suffix, owner string, createdAt time.Time) order.Attachment {
	value := uploadedAttachment(suffix, owner, createdAt)
	value.Status = order.AttachmentStatusDeleting
	value.ExpiresAt = nil
	return value
}

type attachmentTransition struct {
	id       string
	from, to order.AttachmentStatus
}

type attachmentRepository struct {
	values      map[string]order.Attachment
	createErr   error
	transitions []attachmentTransition
}

func newAttachmentRepository() *attachmentRepository {
	return &attachmentRepository{values: make(map[string]order.Attachment)}
}

func (repository *attachmentRepository) CreateAttachment(_ context.Context, value order.Attachment) error {
	if repository.createErr != nil {
		return repository.createErr
	}
	repository.values[value.ID] = value
	return nil
}

func (repository *attachmentRepository) GetAttachment(_ context.Context, id string) (order.Attachment, bool, error) {
	value, found := repository.values[id]
	return value, found, nil
}

func (repository *attachmentRepository) SetAttachmentStatus(_ context.Context, id string, from, to order.AttachmentStatus, now time.Time) (bool, error) {
	value, found := repository.values[id]
	if !found || value.Status != from {
		return false, nil
	}
	value.Status = to
	value.UpdatedAt = now
	if to == order.AttachmentStatusDeleting {
		value.ExpiresAt = nil
	}
	repository.values[id] = value
	repository.transitions = append(repository.transitions, attachmentTransition{id: id, from: from, to: to})
	return true, nil
}

func (repository *attachmentRepository) DeleteAttachment(_ context.Context, id string, status order.AttachmentStatus) (bool, error) {
	value, found := repository.values[id]
	if !found || value.Status != status {
		return false, nil
	}
	delete(repository.values, id)
	return true, nil
}

func (repository *attachmentRepository) ListAttachmentsForCleanup(_ context.Context, now time.Time, limit int) ([]order.Attachment, error) {
	result := make([]order.Attachment, 0, limit)
	for _, value := range repository.values {
		if value.Status == order.AttachmentStatusDeleting || (value.Status == order.AttachmentStatusUploaded && value.ExpiresAt != nil && !value.ExpiresAt.After(now)) {
			result = append(result, value)
			if len(result) == limit {
				break
			}
		}
	}
	return result, nil
}

func (repository *attachmentRepository) AttachmentStorageKeyExists(_ context.Context, storageKey string) (bool, error) {
	for _, value := range repository.values {
		if value.StorageKey == storageKey {
			return true, nil
		}
	}
	return false, nil
}

type attachmentWriteCall struct {
	storageKey string
	fileName   string
}

type attachmentFiles struct {
	contents    map[string][]byte
	writeCalls  []attachmentWriteCall
	deleteCalls []string
	residuals   []string
	writeErr    error
}

func newAttachmentFiles() *attachmentFiles {
	return &attachmentFiles{contents: make(map[string][]byte)}
}

func (files *attachmentFiles) Write(storageKey, fileName string, content []byte) (order.StoredAttachmentFile, error) {
	files.writeCalls = append(files.writeCalls, attachmentWriteCall{storageKey: storageKey, fileName: fileName})
	if files.writeErr != nil {
		return order.StoredAttachmentFile{}, files.writeErr
	}
	files.contents[storageKey] = bytes.Clone(content)
	contentType := "image/png"
	if bytes.HasPrefix(content, []byte("%PDF-")) {
		contentType = "application/pdf"
	} else if bytes.HasPrefix(content, []byte{0xff, 0xd8, 0xff}) {
		contentType = "image/jpeg"
	}
	return order.StoredAttachmentFile{FileName: fileName, ContentType: contentType, SizeBytes: int64(len(content)), SHA256: sha256.Sum256(content)}, nil
}

func (files *attachmentFiles) Read(storageKey string) ([]byte, error) {
	content, found := files.contents[storageKey]
	if !found {
		return nil, errors.New("file not found")
	}
	return bytes.Clone(content), nil
}

func (files *attachmentFiles) Delete(storageKey string) error {
	files.deleteCalls = append(files.deleteCalls, storageKey)
	delete(files.contents, storageKey)
	return nil
}

func (files *attachmentFiles) DeleteResidual(storageKey string) error {
	return files.Delete(storageKey)
}

func (files *attachmentFiles) ListResiduals(time.Time) ([]string, error) {
	result := make([]string, 0, len(files.residuals))
	for _, storageKey := range files.residuals {
		if _, exists := files.contents[storageKey]; exists {
			result = append(result, storageKey)
		}
	}
	return result, nil
}
