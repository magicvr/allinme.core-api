package httpapi_test

import (
	"bytes"
	"context"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/httpapi"
	"github.com/magicvr/allinme.core-api/internal/order"
)

const testAttachmentID = "att_00000000000000000000000000000001"

func TestAttachmentUploadStrictMultipartAndErrors(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	service := &fakeAttachmentService{uploadResult: order.UploadAttachmentResult{
		Attachment: order.AttachmentSummary{ID: testAttachmentID, FileName: "invoice.pdf", ContentType: "application/pdf", SizeBytes: 8, SHA256: strings.Repeat("a", 64), CreatedAt: createdAt},
		ExpiresAt:  createdAt.Add(24 * time.Hour),
	}}
	authService := &fakeAuthService{authenticatedRole: auth.RoleOperator}
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: authService, Attachments: service})

	body, contentType := multipartBody(t, func(writer *multipart.Writer) {
		part, err := writer.CreateFormFile("file", "invoice.pdf")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = part.Write([]byte("%PDF-1.7"))
	})
	response := attachmentRequest(handler, http.MethodPost, "/api/v1/attachments", contentType, body, true)
	if response.Code != http.StatusCreated || service.uploadCalls != 1 || service.uploadCommand.FileName != "invoice.pdf" || string(service.uploadCommand.Content) != "%PDF-1.7" {
		t.Fatalf("upload = %d %s command=%+v calls=%d", response.Code, response.Body.String(), service.uploadCommand, service.uploadCalls)
	}
	want := `{"id":"` + testAttachmentID + `","fileName":"invoice.pdf","contentType":"application/pdf","sizeBytes":8,"sha256":"` + strings.Repeat("a", 64) + `","createdAt":"2026-01-01T00:00:00Z","expiresAt":"2026-01-02T00:00:00Z"}`
	if strings.TrimSpace(response.Body.String()) != want || strings.Contains(response.Body.String(), "url") {
		t.Fatalf("upload body = %s", response.Body.String())
	}

	unauthenticated := attachmentRequest(handler, http.MethodPost, "/api/v1/attachments", "text/plain", []byte("bad"), false)
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated = %d %s", unauthenticated.Code, unauthenticated.Body.String())
	}
	authService.authenticatedRole = auth.RoleViewer
	viewer := attachmentRequest(handler, http.MethodPost, "/api/v1/attachments", "text/plain", []byte("bad"), true)
	if viewer.Code != http.StatusForbidden {
		t.Fatalf("viewer = %d %s", viewer.Code, viewer.Body.String())
	}
	authService.authenticatedRole = auth.RoleOperator

	invalid := []struct {
		name        string
		contentType string
		body        []byte
		status      int
	}{
		{name: "unsupported", contentType: "application/json", body: []byte("{}"), status: http.StatusUnsupportedMediaType},
		{name: "missing boundary", contentType: "multipart/form-data", body: []byte("bad"), status: http.StatusBadRequest},
	}
	wrongField, wrongFieldType := multipartBody(t, func(writer *multipart.Writer) {
		part, _ := writer.CreateFormFile("other", "invoice.pdf")
		_, _ = part.Write([]byte("%PDF-1.7"))
	})
	invalid = append(invalid, struct {
		name        string
		contentType string
		body        []byte
		status      int
	}{name: "wrong field", contentType: wrongFieldType, body: wrongField, status: http.StatusBadRequest})
	emptyName, emptyNameType := rawMultipartBody(t, `form-data; name="file"; filename=""`, []byte("%PDF-1.7"))
	invalid = append(invalid, struct {
		name        string
		contentType string
		body        []byte
		status      int
	}{name: "empty filename", contentType: emptyNameType, body: emptyName, status: http.StatusBadRequest})
	extra, extraType := multipartBody(t, func(writer *multipart.Writer) {
		part, _ := writer.CreateFormFile("file", "invoice.pdf")
		_, _ = part.Write([]byte("%PDF-1.7"))
		_ = writer.WriteField("note", "extra")
	})
	invalid = append(invalid, struct {
		name        string
		contentType string
		body        []byte
		status      int
	}{name: "extra part", contentType: extraType, body: extra, status: http.StatusBadRequest})
	tooLarge, tooLargeType := multipartBody(t, func(writer *multipart.Writer) {
		part, _ := writer.CreateFormFile("file", "large.pdf")
		_, _ = part.Write(bytes.Repeat([]byte{'x'}, int(order.MaxAttachmentSizeBytes+1)))
	})
	invalid = append(invalid, struct {
		name        string
		contentType string
		body        []byte
		status      int
	}{name: "file too large", contentType: tooLargeType, body: tooLarge, status: http.StatusRequestEntityTooLarge})
	requestCap := append(append([]byte{}, body...), bytes.Repeat([]byte{'x'}, int(order.MaxAttachmentSizeBytes+64*1024)-len(body)+1)...)
	invalid = append(invalid, struct {
		name        string
		contentType string
		body        []byte
		status      int
	}{name: "request too large", contentType: contentType, body: requestCap, status: http.StatusRequestEntityTooLarge})

	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			response := attachmentRequest(handler, http.MethodPost, "/api/v1/attachments", test.contentType, test.body, true)
			if response.Code != test.status {
				t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
			}
			if test.status == http.StatusBadRequest && !strings.Contains(response.Body.String(), `"field":"file"`) {
				t.Fatalf("body = %s", response.Body.String())
			}
		})
	}

	service.uploadErr = &order.ValidationError{Details: []order.FieldError{{Field: "content", Message: "must be PDF, PNG, or JPEG"}}}
	response = attachmentRequest(handler, http.MethodPost, "/api/v1/attachments", contentType, body, true)
	if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), `"field":"file"`) || strings.Contains(response.Body.String(), `"field":"content"`) {
		t.Fatalf("validation = %d %s", response.Code, response.Body.String())
	}
	service.uploadErr = order.ErrUnavailable
	response = attachmentRequest(handler, http.MethodPost, "/api/v1/attachments", contentType, body, true)
	if response.Code != http.StatusServiceUnavailable || response.Header().Get("Retry-After") != "1" {
		t.Fatalf("unavailable = %d headers=%v %s", response.Code, response.Header(), response.Body.String())
	}
}

func TestAttachmentDownloadDeleteAndDisabledRoutes(t *testing.T) {
	content := []byte("%PDF-1.7")
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	service := &fakeAttachmentService{downloadResult: order.DownloadAttachmentResult{
		Attachment: order.AttachmentSummary{ID: testAttachmentID, FileName: "invoice \"demo\".pdf", ContentType: "application/pdf", SizeBytes: int64(len(content)), SHA256: strings.Repeat("b", 64), CreatedAt: createdAt},
		Content:    content,
	}}
	authService := &fakeAuthService{authenticatedRole: auth.RoleViewer}
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: authService, Attachments: service})

	unauthenticated := attachmentRequest(handler, http.MethodGet, "/api/v1/attachments/"+testAttachmentID, "", nil, false)
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated = %d %s", unauthenticated.Code, unauthenticated.Body.String())
	}
	response := attachmentRequest(handler, http.MethodGet, "/api/v1/attachments/"+testAttachmentID, "", nil, true)
	if response.Code != http.StatusOK || !bytes.Equal(response.Body.Bytes(), content) || response.Header().Get("Content-Type") != "application/pdf" || response.Header().Get("Content-Length") != "8" || response.Header().Get("X-Content-Type-Options") != "nosniff" || response.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("download = %d headers=%v body=%q", response.Code, response.Header(), response.Body.Bytes())
	}
	disposition, parameters, err := mime.ParseMediaType(response.Header().Get("Content-Disposition"))
	if err != nil || disposition != "attachment" || parameters["filename"] != `invoice "demo".pdf` {
		t.Fatalf("Content-Disposition = %q (%q %+v, %v)", response.Header().Get("Content-Disposition"), disposition, parameters, err)
	}
	if service.downloadCommand.ID != testAttachmentID {
		t.Fatalf("download command = %+v", service.downloadCommand)
	}

	invalid := attachmentRequest(handler, http.MethodGet, "/api/v1/attachments/not-an-attachment", "", nil, true)
	if invalid.Code != http.StatusNotFound || service.downloadCalls != 1 {
		t.Fatalf("invalid id = %d calls=%d %s", invalid.Code, service.downloadCalls, invalid.Body.String())
	}
	wrongMethod := attachmentRequest(handler, http.MethodPatch, "/api/v1/attachments/"+testAttachmentID, "", nil, true)
	if wrongMethod.Code != http.StatusMethodNotAllowed || wrongMethod.Header().Get("Allow") != "GET, DELETE" {
		t.Fatalf("wrong method = %d Allow=%q", wrongMethod.Code, wrongMethod.Header().Get("Allow"))
	}

	viewerDelete := attachmentRequest(handler, http.MethodDelete, "/api/v1/attachments/"+testAttachmentID, "", nil, true)
	if viewerDelete.Code != http.StatusForbidden {
		t.Fatalf("viewer delete = %d %s", viewerDelete.Code, viewerDelete.Body.String())
	}
	authService.authenticatedRole = auth.RoleAdmin
	deleted := attachmentRequest(handler, http.MethodDelete, "/api/v1/attachments/"+testAttachmentID, "", nil, true)
	if deleted.Code != http.StatusNoContent || deleted.Body.Len() != 0 || service.deleteCommand.ID != testAttachmentID {
		t.Fatalf("delete = %d body=%q command=%+v", deleted.Code, deleted.Body.String(), service.deleteCommand)
	}
	service.deleteErr = order.ErrStateConflict
	conflict := attachmentRequest(handler, http.MethodDelete, "/api/v1/attachments/"+testAttachmentID, "", nil, true)
	if conflict.Code != http.StatusConflict || !strings.Contains(conflict.Body.String(), `"code":"STATE_CONFLICT"`) {
		t.Fatalf("delete conflict = %d %s", conflict.Code, conflict.Body.String())
	}

	orders := &fakeOrderService{page: order.Page{Items: []order.Order{}, Page: 1, PageSize: 20}}
	disabled := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: authService, Orders: orders, Attachments: service, DisableAttachmentRoutes: true})
	attachment := attachmentRequest(disabled, http.MethodGet, "/api/v1/attachments/"+testAttachmentID, "", nil, true)
	if attachment.Code != http.StatusNotFound {
		t.Fatalf("disabled attachment = %d %s", attachment.Code, attachment.Body.String())
	}
	ordersResponse := attachmentRequest(disabled, http.MethodGet, "/api/v1/orders", "", nil, true)
	if ordersResponse.Code != http.StatusOK {
		t.Fatalf("disabled orders = %d %s", ordersResponse.Code, ordersResponse.Body.String())
	}
}

func TestAttachmentCORSRoutesAndDisableMetadata(t *testing.T) {
	service := &fakeAttachmentService{}
	authService := &fakeAuthService{authenticatedRole: auth.RoleAdmin}
	handler := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: authService, Attachments: service, CORSAllowedOrigin: testAllowedOrigin})
	preflight := httptest.NewRequest(http.MethodOptions, "/api/v1/attachments/"+testAttachmentID, nil)
	preflight.Header.Set("Origin", testAllowedOrigin)
	preflight.Header.Set("Access-Control-Request-Method", http.MethodDelete)
	preflight.Header.Set("Access-Control-Request-Headers", "Authorization")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, preflight)
	if response.Code != http.StatusNoContent || response.Header().Get("Access-Control-Allow-Methods") != "GET, POST, PATCH, DELETE, OPTIONS" || response.Header().Get("Access-Control-Expose-Headers") != "X-Request-ID, Content-Disposition" {
		t.Fatalf("preflight = %d headers=%v %s", response.Code, response.Header(), response.Body.String())
	}

	disabled := httpapi.NewHandler(httpapi.Dependencies{Logger: discardLogger(), Auth: authService, Attachments: service, DisableAttachmentRoutes: true, CORSAllowedOrigin: testAllowedOrigin})
	response = httptest.NewRecorder()
	disabled.ServeHTTP(response, preflight.Clone(preflight.Context()))
	if response.Code != http.StatusNotFound || response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("disabled preflight = %d headers=%v %s", response.Code, response.Header(), response.Body.String())
	}
}

func multipartBody(t *testing.T, write func(*multipart.Writer)) ([]byte, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	write(writer)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return body.Bytes(), writer.FormDataContentType()
}

func rawMultipartBody(t *testing.T, disposition string, content []byte) ([]byte, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", disposition)
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(content)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return body.Bytes(), writer.FormDataContentType()
}

func attachmentRequest(handler http.Handler, method, path, contentType string, body []byte, authenticated bool) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	if authenticated {
		request.Header.Set("Authorization", "Bearer access-token")
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

type fakeAttachmentService struct {
	uploadResult    order.UploadAttachmentResult
	downloadResult  order.DownloadAttachmentResult
	uploadErr       error
	downloadErr     error
	deleteErr       error
	uploadCommand   order.UploadAttachmentCommand
	downloadCommand order.DownloadAttachmentCommand
	deleteCommand   order.DeleteAttachmentCommand
	uploadCalls     int
	downloadCalls   int
	deleteCalls     int
}

func (service *fakeAttachmentService) UploadAttachment(_ context.Context, _ auth.Principal, command order.UploadAttachmentCommand) (order.UploadAttachmentResult, error) {
	service.uploadCalls++
	service.uploadCommand = command
	return service.uploadResult, service.uploadErr
}
func (service *fakeAttachmentService) DownloadAttachment(_ context.Context, _ auth.Principal, command order.DownloadAttachmentCommand) (order.DownloadAttachmentResult, error) {
	service.downloadCalls++
	service.downloadCommand = command
	return service.downloadResult, service.downloadErr
}
func (service *fakeAttachmentService) DeleteAttachment(_ context.Context, _ auth.Principal, command order.DeleteAttachmentCommand) (order.DeleteAttachmentResult, error) {
	service.deleteCalls++
	service.deleteCommand = command
	return order.DeleteAttachmentResult{ID: command.ID}, service.deleteErr
}
