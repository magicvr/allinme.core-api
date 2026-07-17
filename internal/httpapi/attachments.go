package httpapi

import (
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/order"
)

const attachmentRequestBodyLimit = order.MaxAttachmentSizeBytes + 64*1024

var (
	errInvalidMultipart     = errors.New("invalid multipart body")
	errAttachmentBodyTooBig = errors.New("attachment body too large")
)

type AttachmentService interface {
	UploadAttachment(context.Context, auth.Principal, order.UploadAttachmentCommand) (order.UploadAttachmentResult, error)
	DownloadAttachment(context.Context, auth.Principal, order.DownloadAttachmentCommand) (order.DownloadAttachmentResult, error)
	DeleteAttachment(context.Context, auth.Principal, order.DeleteAttachmentCommand) (order.DeleteAttachmentResult, error)
}

type attachmentSummaryDTO struct {
	ID          string `json:"id"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
	SHA256      string `json:"sha256"`
	CreatedAt   string `json:"createdAt"`
}

type attachmentUploadDTO struct {
	attachmentSummaryDTO
	ExpiresAt string `json:"expiresAt"`
}

func registerAttachmentRoutes(mux *http.ServeMux, authService AuthService, service AttachmentService, disabled bool) {
	if disabled || authService == nil || service == nil {
		return
	}

	uploadHandler := RequireAuthentication(authService)(RequireRoles(auth.RoleOperator, auth.RoleAdmin)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		command, err := decodeAttachmentUpload(response, request)
		if err != nil {
			handleAttachmentInputError(response, request, err)
			return
		}
		principal, _ := PrincipalFromContext(request.Context())
		result, err := service.UploadAttachment(request.Context(), principal, command)
		if handleAttachmentError(response, request, err) {
			return
		}
		writeJSON(response, http.StatusCreated, attachmentUploadDTO{
			attachmentSummaryDTO: makeAttachmentSummaryDTO(result.Attachment),
			ExpiresAt:            order.FormatTime(result.ExpiresAt),
		})
	})))

	downloadHandler := RequireAuthentication(authService)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		id := request.PathValue("attachmentId")
		if !order.ValidAttachmentID(id) {
			writeError(response, request, http.StatusNotFound, "NOT_FOUND", "attachment not found")
			return
		}
		principal, _ := PrincipalFromContext(request.Context())
		result, err := service.DownloadAttachment(request.Context(), principal, order.DownloadAttachmentCommand{ID: id})
		if handleAttachmentError(response, request, err) {
			return
		}
		response.Header().Set("Content-Type", result.Attachment.ContentType)
		response.Header().Set("Content-Length", strconv.Itoa(len(result.Content)))
		response.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": result.Attachment.FileName}))
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.Header().Set("Cache-Control", "private, no-store")
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write(result.Content)
	}))

	deleteHandler := RequireAuthentication(authService)(RequireRoles(auth.RoleOperator, auth.RoleAdmin)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		id := request.PathValue("attachmentId")
		if !order.ValidAttachmentID(id) {
			writeError(response, request, http.StatusNotFound, "NOT_FOUND", "attachment not found")
			return
		}
		principal, _ := PrincipalFromContext(request.Context())
		_, err := service.DeleteAttachment(request.Context(), principal, order.DeleteAttachmentCommand{ID: id})
		if handleAttachmentError(response, request, err) {
			return
		}
		response.WriteHeader(http.StatusNoContent)
	})))

	collectionRoute := attachmentCollectionMetadata()
	detailRoute := attachmentDetailMetadata()
	mux.Handle(collectionRoute.pattern, orderRoute(collectionRoute, map[string]http.Handler{http.MethodPost: uploadHandler}))
	mux.Handle(detailRoute.pattern, orderRoute(detailRoute, map[string]http.Handler{http.MethodGet: downloadHandler, http.MethodDelete: deleteHandler}))
}

func decodeAttachmentUpload(response http.ResponseWriter, request *http.Request) (order.UploadAttachmentCommand, error) {
	mediaType, parameters, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/form-data" {
		return order.UploadAttachmentCommand{}, errUnsupportedMedia
	}
	boundary := parameters["boundary"]
	if boundary == "" {
		return order.UploadAttachmentCommand{}, errInvalidMultipart
	}

	if request.ContentLength > attachmentRequestBodyLimit {
		return order.UploadAttachmentCommand{}, errAttachmentBodyTooBig
	}
	request.Body = http.MaxBytesReader(response, request.Body, attachmentRequestBodyLimit)
	reader := multipart.NewReader(request.Body, boundary)
	part, err := reader.NextPart()
	if err != nil {
		return order.UploadAttachmentCommand{}, classifyMultipartError(err)
	}

	fileName, err := attachmentPartFileName(part)
	if err != nil {
		_ = part.Close()
		return order.UploadAttachmentCommand{}, err
	}
	content, err := io.ReadAll(io.LimitReader(part, order.MaxAttachmentSizeBytes+1))
	if err != nil {
		_ = part.Close()
		return order.UploadAttachmentCommand{}, classifyMultipartError(err)
	}
	if int64(len(content)) > order.MaxAttachmentSizeBytes {
		_ = part.Close()
		return order.UploadAttachmentCommand{}, errAttachmentBodyTooBig
	}
	if err := part.Close(); err != nil {
		return order.UploadAttachmentCommand{}, classifyMultipartError(err)
	}
	if _, err := reader.NextPart(); !errors.Is(err, io.EOF) {
		if err != nil {
			return order.UploadAttachmentCommand{}, classifyMultipartError(err)
		}
		return order.UploadAttachmentCommand{}, errInvalidMultipart
	}
	if _, err := io.Copy(io.Discard, request.Body); err != nil {
		return order.UploadAttachmentCommand{}, classifyMultipartError(err)
	}
	return order.UploadAttachmentCommand{FileName: fileName, Content: content}, nil
}

func attachmentPartFileName(part *multipart.Part) (string, error) {
	values := part.Header.Values("Content-Disposition")
	if len(values) != 1 {
		return "", errInvalidMultipart
	}
	disposition, parameters, err := mime.ParseMediaType(values[0])
	if err != nil || disposition != "form-data" || parameters["name"] != "file" {
		return "", errInvalidMultipart
	}
	fileName, exists := parameters["filename"]
	if !exists || fileName == "" {
		return "", errInvalidMultipart
	}
	return fileName, nil
}

func classifyMultipartError(err error) error {
	var maximum *http.MaxBytesError
	if errors.As(err, &maximum) {
		return errAttachmentBodyTooBig
	}
	return errInvalidMultipart
}

func handleAttachmentInputError(response http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, errUnsupportedMedia):
		writeError(response, request, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "content type must be multipart/form-data")
	case errors.Is(err, errAttachmentBodyTooBig):
		writeError(response, request, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "attachment is too large")
	default:
		writeErrorDetails(response, request, http.StatusBadRequest, "INVALID_REQUEST", "invalid request", []errorDetail{{Field: "file", Message: "must be exactly one multipart file part"}})
	}
}

func handleAttachmentError(response http.ResponseWriter, request *http.Request, err error) bool {
	if details, ok := order.ValidationDetails(err); ok {
		mapped := make([]order.FieldError, 0, len(details))
		for _, detail := range details {
			mapped = append(mapped, order.FieldError{Field: "file", Message: detail.Message})
		}
		err = &order.ValidationError{Details: mapped}
	}
	return handleOrderErrorWithOptions(response, request, err, orderErrorOptions{
		notFoundMessage:      "attachment not found",
		stateConflictMessage: "attachment state conflict",
	})
}

func makeAttachmentSummaryDTO(value order.AttachmentSummary) attachmentSummaryDTO {
	return attachmentSummaryDTO{
		ID: value.ID, FileName: value.FileName, ContentType: value.ContentType,
		SizeBytes: value.SizeBytes, SHA256: value.SHA256, CreatedAt: order.FormatTime(value.CreatedAt),
	}
}
