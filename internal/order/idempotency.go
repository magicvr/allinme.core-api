package order

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
	"unicode/utf8"

	"github.com/magicvr/allinme.core-api/internal/auth"
)

const (
	CreateMethod       = "POST"
	CreateRoute        = "/api/v1/orders"
	SnapshotVersionOne = int64(1)
	SnapshotVersionTwo = int64(2)
)

type IdempotencyScope struct {
	PrincipalUserID string
	Method          string
	Route           string
	Key             string
	RequestDigest   [sha256.Size]byte
}

type IdempotencyRecord struct {
	Scope           IdempotencyScope
	OrderID         string
	SnapshotVersion int64
	SnapshotJSON    []byte
	SnapshotDigest  [sha256.Size]byte
	CreatedAt       string
}

type IdempotentCreatePersistence struct {
	Create CreatePersistence
	Record IdempotencyRecord
}

type createDigest struct {
	Operation    string        `json:"operation"`
	CustomerName string        `json:"customerName"`
	Currency     string        `json:"currency"`
	Items        []ItemCommand `json:"items"`
}

type createDigestV2 struct {
	Operation     string        `json:"operation"`
	CustomerName  string        `json:"customerName"`
	Currency      string        `json:"currency"`
	Items         []ItemCommand `json:"items"`
	AttachmentIDs []string      `json:"attachmentIds"`
}

type snapshotV1 struct {
	Order snapshotOrderV1 `json:"order"`
}

type snapshotOrderV1 struct {
	ID            string           `json:"id"`
	CustomerName  string           `json:"customerName"`
	Status        Status           `json:"status"`
	PaymentStatus PaymentStatus    `json:"paymentStatus"`
	Currency      string           `json:"currency"`
	TotalAmount   int64            `json:"totalAmount"`
	Version       int64            `json:"version"`
	CreatedAt     string           `json:"createdAt"`
	UpdatedAt     string           `json:"updatedAt"`
	Items         []snapshotItemV1 `json:"items"`
}

type snapshotItemV1 struct {
	ID        string `json:"id"`
	SKU       string `json:"sku"`
	Name      string `json:"name"`
	Quantity  int64  `json:"quantity"`
	UnitPrice int64  `json:"unitPrice"`
}

type snapshotV2 struct {
	Order snapshotOrderV2 `json:"order"`
}

type snapshotOrderV2 struct {
	ID                    string                 `json:"id"`
	CustomerName          string                 `json:"customerName"`
	Status                Status                 `json:"status"`
	PaymentStatus         PaymentStatus          `json:"paymentStatus"`
	Currency              string                 `json:"currency"`
	TotalAmount           int64                  `json:"totalAmount"`
	AvailableRefundAmount int64                  `json:"availableRefundAmount"`
	Version               int64                  `json:"version"`
	CreatedAt             string                 `json:"createdAt"`
	UpdatedAt             string                 `json:"updatedAt"`
	AttachmentCount       int64                  `json:"attachmentCount"`
	Items                 []snapshotItemV1       `json:"items"`
	Attachments           []snapshotAttachmentV2 `json:"attachments"`
}

type snapshotAttachmentV2 struct {
	ID          string `json:"id"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
	SHA256      string `json:"sha256"`
	CreatedAt   string `json:"createdAt"`
}

func normalizedDigest(command CreateCommand) ([sha256.Size]byte, error) {
	encoded, err := json.Marshal(createDigest{Operation: CreateMethod + " " + CreateRoute, CustomerName: command.CustomerName, Currency: command.Currency, Items: command.Items})
	if err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("encode normalized create digest: %w", err)
	}
	return sha256.Sum256(encoded), nil
}

func normalizedDigestV2(command CreateCommand) ([sha256.Size]byte, error) {
	encoded, err := json.Marshal(createDigestV2{Operation: CreateMethod + " " + CreateRoute, CustomerName: command.CustomerName, Currency: command.Currency, Items: command.Items, AttachmentIDs: command.AttachmentIDs})
	if err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("encode normalized create v2 digest: %w", err)
	}
	return sha256.Sum256(encoded), nil
}

func encodeSnapshotV1(value Order) ([]byte, error) {
	items := make([]snapshotItemV1, 0, len(value.Items))
	for _, item := range value.Items {
		items = append(items, snapshotItemV1{ID: item.ID, SKU: item.SKU, Name: item.Name, Quantity: item.Quantity, UnitPrice: item.UnitPrice})
	}
	return json.Marshal(snapshotV1{Order: snapshotOrderV1{ID: value.ID, CustomerName: value.CustomerName, Status: value.Status, PaymentStatus: value.PaymentStatus, Currency: value.Currency, TotalAmount: value.TotalAmount, Version: value.Version, CreatedAt: FormatTime(value.CreatedAt), UpdatedAt: FormatTime(value.UpdatedAt), Items: items}})
}

func encodeSnapshotV2(value Order) ([]byte, error) {
	items := make([]snapshotItemV1, 0, len(value.Items))
	for _, item := range value.Items {
		items = append(items, snapshotItemV1{ID: item.ID, SKU: item.SKU, Name: item.Name, Quantity: item.Quantity, UnitPrice: item.UnitPrice})
	}
	attachments := make([]snapshotAttachmentV2, 0, len(value.Attachments))
	for _, attachment := range value.Attachments {
		attachments = append(attachments, snapshotAttachmentV2{ID: attachment.ID, FileName: attachment.FileName, ContentType: attachment.ContentType, SizeBytes: attachment.SizeBytes, SHA256: hex.EncodeToString(attachment.SHA256[:]), CreatedAt: FormatTime(attachment.CreatedAt)})
	}
	return json.Marshal(snapshotV2{Order: snapshotOrderV2{
		ID: value.ID, CustomerName: value.CustomerName, Status: value.Status, PaymentStatus: value.PaymentStatus,
		Currency: value.Currency, TotalAmount: value.TotalAmount, AvailableRefundAmount: value.AvailableRefundAmount,
		Version: value.Version, CreatedAt: FormatTime(value.CreatedAt), UpdatedAt: FormatTime(value.UpdatedAt),
		AttachmentCount: value.AttachmentCount, Items: items, Attachments: attachments,
	}})
}

func decodeSnapshot(record IdempotencyRecord, expected IdempotencyScope, command CreateCommand) (Order, error) {
	if record.Scope.PrincipalUserID != expected.PrincipalUserID || record.Scope.Method != CreateMethod || record.Scope.Route != CreateRoute || record.Scope.Key != expected.Key || record.OrderID == "" {
		return Order{}, ErrInternal
	}
	switch record.SnapshotVersion {
	case SnapshotVersionOne:
		if len(command.AttachmentIDs) != 0 {
			return Order{}, ErrIdempotencyConflict
		}
		digest, err := normalizedDigest(command)
		if err != nil {
			return Order{}, ErrInternal
		}
		if !bytes.Equal(record.Scope.RequestDigest[:], digest[:]) {
			return Order{}, ErrIdempotencyConflict
		}
		return decodeSnapshotV1(record)
	case SnapshotVersionTwo:
		digest, err := normalizedDigestV2(command)
		if err != nil {
			return Order{}, ErrInternal
		}
		if !bytes.Equal(record.Scope.RequestDigest[:], digest[:]) {
			return Order{}, ErrIdempotencyConflict
		}
		return decodeSnapshotV2(record)
	default:
		return Order{}, ErrInternal
	}
}

func validateSnapshotDigest(record IdempotencyRecord) bool {
	snapshotDigest := sha256.Sum256(record.SnapshotJSON)
	return bytes.Equal(record.SnapshotDigest[:], snapshotDigest[:])
}

func decodeSnapshotV1(record IdempotencyRecord) (Order, error) {
	if !validateSnapshotDigest(record) {
		return Order{}, ErrInternal
	}
	decoder := json.NewDecoder(bytes.NewReader(record.SnapshotJSON))
	decoder.DisallowUnknownFields()
	var snapshot snapshotV1
	if err := decoder.Decode(&snapshot); err != nil {
		return Order{}, ErrInternal
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Order{}, ErrInternal
	}
	value := snapshot.Order
	createdAt, createdErr := time.Parse(time.RFC3339, value.CreatedAt)
	updatedAt, updatedErr := time.Parse(time.RFC3339, value.UpdatedAt)
	if createdErr != nil || updatedErr != nil || FormatTime(createdAt) != value.CreatedAt || FormatTime(updatedAt) != value.UpdatedAt || record.CreatedAt != value.CreatedAt || value.ID != record.OrderID || !ValidOrderID(value.ID) || value.Status != StatusDraft || value.PaymentStatus != PaymentStatusUnpaid || value.Version != 1 || !createdAt.Equal(updatedAt) {
		return Order{}, ErrInternal
	}
	result := Order{ID: value.ID, CustomerName: value.CustomerName, Status: value.Status, PaymentStatus: value.PaymentStatus, Currency: value.Currency, TotalAmount: value.TotalAmount, Version: value.Version, CreatedAt: createdAt, UpdatedAt: updatedAt, AttachmentCount: 0, Items: make([]Item, 0, len(value.Items)), Attachments: []Attachment{}}
	command := CreateCommand{CustomerName: value.CustomerName, Currency: value.Currency, Items: make([]ItemCommand, 0, len(value.Items)), AttachmentIDs: []string{}}
	itemIDs := make(map[string]bool, len(value.Items))
	for _, item := range value.Items {
		if !ValidItemID(item.ID) || itemIDs[item.ID] {
			return Order{}, ErrInternal
		}
		itemIDs[item.ID] = true
		command.Items = append(command.Items, ItemCommand{SKU: item.SKU, Name: item.Name, Quantity: item.Quantity, UnitPrice: item.UnitPrice})
		result.Items = append(result.Items, Item{ID: item.ID, SKU: item.SKU, Name: item.Name, Quantity: item.Quantity, UnitPrice: item.UnitPrice})
	}
	normalized, total, err := validateFacts(command.CustomerName, command.Currency, command.Items)
	if err != nil || !sameCreateFacts(normalized, command) || total != value.TotalAmount {
		return Order{}, ErrInternal
	}
	digest, err := normalizedDigest(normalized)
	if err != nil || !bytes.Equal(digest[:], record.Scope.RequestDigest[:]) {
		return Order{}, ErrInternal
	}
	return result, nil
}

func decodeSnapshotV2(record IdempotencyRecord) (Order, error) {
	if !validateSnapshotDigest(record) {
		return Order{}, ErrInternal
	}
	decoder := json.NewDecoder(bytes.NewReader(record.SnapshotJSON))
	decoder.DisallowUnknownFields()
	var snapshot snapshotV2
	if err := decoder.Decode(&snapshot); err != nil {
		return Order{}, ErrInternal
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Order{}, ErrInternal
	}
	value := snapshot.Order
	createdAt, createdErr := parseSnapshotTime(value.CreatedAt)
	updatedAt, updatedErr := parseSnapshotTime(value.UpdatedAt)
	if createdErr != nil || updatedErr != nil || record.CreatedAt != value.CreatedAt || value.ID != record.OrderID || !ValidOrderID(value.ID) || value.Status != StatusDraft || value.PaymentStatus != PaymentStatusUnpaid || value.AvailableRefundAmount != 0 || value.Version != 1 || !createdAt.Equal(updatedAt) || value.AttachmentCount != int64(len(value.Attachments)) || value.AttachmentCount < 0 || value.AttachmentCount > MaxOrderAttachments {
		return Order{}, ErrInternal
	}
	result := Order{ID: value.ID, CustomerName: value.CustomerName, Status: value.Status, PaymentStatus: value.PaymentStatus, Currency: value.Currency, TotalAmount: value.TotalAmount, AvailableRefundAmount: value.AvailableRefundAmount, Version: value.Version, CreatedAt: createdAt, UpdatedAt: updatedAt, AttachmentCount: value.AttachmentCount, Items: make([]Item, 0, len(value.Items)), Attachments: make([]Attachment, 0, len(value.Attachments))}
	command := CreateCommand{CustomerName: value.CustomerName, Currency: value.Currency, Items: make([]ItemCommand, 0, len(value.Items)), AttachmentIDs: make([]string, 0, len(value.Attachments))}
	itemIDs := make(map[string]bool, len(value.Items))
	for _, item := range value.Items {
		if !ValidItemID(item.ID) || itemIDs[item.ID] {
			return Order{}, ErrInternal
		}
		itemIDs[item.ID] = true
		command.Items = append(command.Items, ItemCommand{SKU: item.SKU, Name: item.Name, Quantity: item.Quantity, UnitPrice: item.UnitPrice})
		result.Items = append(result.Items, Item{ID: item.ID, SKU: item.SKU, Name: item.Name, Quantity: item.Quantity, UnitPrice: item.UnitPrice})
	}
	for _, attachment := range value.Attachments {
		decoded, err := decodeSnapshotAttachment(attachment)
		if err != nil {
			return Order{}, ErrInternal
		}
		command.AttachmentIDs = append(command.AttachmentIDs, decoded.ID)
		result.Attachments = append(result.Attachments, decoded)
	}
	if err := ValidateAttachmentIDs(command.AttachmentIDs); err != nil {
		return Order{}, ErrInternal
	}
	normalized, total, err := validateFacts(command.CustomerName, command.Currency, command.Items)
	if err != nil || total != value.TotalAmount {
		return Order{}, ErrInternal
	}
	normalized.AttachmentIDs = append([]string{}, command.AttachmentIDs...)
	if !sameCreateFacts(normalized, command) {
		return Order{}, ErrInternal
	}
	digest, err := normalizedDigestV2(normalized)
	if err != nil || !bytes.Equal(digest[:], record.Scope.RequestDigest[:]) {
		return Order{}, ErrInternal
	}
	return result, nil
}

func parseSnapshotTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil || FormatTime(parsed) != value {
		return time.Time{}, errors.New("invalid snapshot time")
	}
	return parsed, nil
}

func decodeSnapshotAttachment(value snapshotAttachmentV2) (Attachment, error) {
	createdAt, err := parseSnapshotTime(value.CreatedAt)
	if err != nil || !ValidAttachmentID(value.ID) || !utf8.ValidString(value.FileName) || len([]byte(value.FileName)) < 1 || len([]byte(value.FileName)) > MaxAttachmentFileNameBytes || value.SizeBytes < 1 || value.SizeBytes > MaxAttachmentSizeBytes || !snapshotAttachmentContentType(value.ContentType) {
		return Attachment{}, errors.New("invalid snapshot attachment")
	}
	fileName, err := NormalizeAttachmentFileName(value.FileName)
	if err != nil || fileName != value.FileName {
		return Attachment{}, errors.New("invalid snapshot attachment file name")
	}
	digest, err := hex.DecodeString(value.SHA256)
	if err != nil || len(digest) != sha256.Size || hex.EncodeToString(digest) != value.SHA256 || bytes.Equal(digest, make([]byte, sha256.Size)) {
		return Attachment{}, errors.New("invalid snapshot attachment digest")
	}
	result := Attachment{ID: value.ID, FileName: value.FileName, ContentType: value.ContentType, SizeBytes: value.SizeBytes, CreatedAt: createdAt}
	copy(result.SHA256[:], digest)
	return result, nil
}

func snapshotAttachmentContentType(value string) bool {
	return value == "application/pdf" || value == "image/png" || value == "image/jpeg"
}

func sameCreateFacts(left, right CreateCommand) bool {
	if left.CustomerName != right.CustomerName || left.Currency != right.Currency || len(left.Items) != len(right.Items) || len(left.AttachmentIDs) != len(right.AttachmentIDs) {
		return false
	}
	for index := range left.Items {
		if left.Items[index] != right.Items[index] {
			return false
		}
	}
	for index := range left.AttachmentIDs {
		if left.AttachmentIDs[index] != right.AttachmentIDs[index] {
			return false
		}
	}
	return true
}

func (service *Service) Create(ctx context.Context, principal auth.Principal, key string, command CreateCommand) (Order, error) {
	if !CanWrite(principal) {
		return Order{}, ErrForbidden
	}
	normalized, total, err := validateFacts(command.CustomerName, command.Currency, command.Items)
	if err != nil {
		return Order{}, err
	}
	if err := ValidateAttachmentIDs(command.AttachmentIDs); err != nil {
		return Order{}, err
	}
	normalized.AttachmentIDs = append([]string{}, command.AttachmentIDs...)
	digest, err := normalizedDigestV2(normalized)
	if err != nil {
		return Order{}, err
	}
	scope := IdempotencyScope{PrincipalUserID: principal.UserID, Method: CreateMethod, Route: CreateRoute, Key: key, RequestDigest: digest}
	existing, found, err := service.repository.GetIdempotency(ctx, scope)
	if err != nil {
		return Order{}, fmt.Errorf("read idempotency key: %w", err)
	}
	if found {
		return decodeSnapshot(existing, scope, normalized)
	}
	now := UTCNow(service.clock).Truncate(time.Second)
	attachments, err := service.repository.PrepareAttachmentsForOrder(ctx, principal.UserID, normalized.AttachmentIDs, now)
	if err != nil {
		return service.replayCreateAfterPreparationFailure(ctx, scope, normalized, fmt.Errorf("prepare order attachments: %w", err))
	}
	if err := validatePreparedAttachments(normalized.AttachmentIDs, attachments); err != nil {
		return service.replayCreateAfterPreparationFailure(ctx, scope, normalized, err)
	}
	orderID, err := service.newOrderID()
	if err != nil {
		return Order{}, fmt.Errorf("generate order ID: %w", err)
	}
	items, err := service.persistenceItems(normalized.Items)
	if err != nil {
		return Order{}, err
	}
	result := Order{ID: orderID, CustomerName: normalized.CustomerName, Status: StatusDraft, PaymentStatus: PaymentStatusUnpaid, Currency: normalized.Currency, TotalAmount: total, Version: 1, CreatedAt: now, UpdatedAt: now, AttachmentCount: int64(len(attachments)), Items: make([]Item, 0, len(items)), Attachments: append([]Attachment{}, attachments...)}
	for _, item := range items {
		result.Items = append(result.Items, Item{ID: item.ID, SKU: item.SKU, Name: item.Name, Quantity: item.Quantity, UnitPrice: item.UnitPrice})
	}
	snapshot, err := encodeSnapshotV2(result)
	if err != nil {
		return Order{}, fmt.Errorf("encode order snapshot: %w", err)
	}
	record, _, err := service.repository.CreateOrderIdempotent(ctx, IdempotentCreatePersistence{Create: CreatePersistence{ID: orderID, CustomerName: normalized.CustomerName, Currency: normalized.Currency, TotalAmount: total, CreatedAt: FormatTime(now), Items: items, AttachmentIDs: append([]string{}, normalized.AttachmentIDs...)}, Record: IdempotencyRecord{Scope: scope, OrderID: orderID, SnapshotVersion: SnapshotVersionTwo, SnapshotJSON: snapshot, SnapshotDigest: sha256.Sum256(snapshot), CreatedAt: FormatTime(now)}})
	if err != nil {
		return Order{}, fmt.Errorf("create order: %w", err)
	}
	return decodeSnapshot(record, scope, normalized)
}

func (service *Service) replayCreateAfterPreparationFailure(ctx context.Context, scope IdempotencyScope, command CreateCommand, preparationErr error) (Order, error) {
	existing, found, err := service.repository.GetIdempotency(ctx, scope)
	if err != nil {
		return Order{}, fmt.Errorf("recheck idempotency key after attachment preparation: %w", err)
	}
	if found {
		return decodeSnapshot(existing, scope, command)
	}
	return Order{}, preparationErr
}

func validatePreparedAttachments(ids []string, attachments []Attachment) error {
	if len(attachments) != len(ids) {
		return Internal(errors.New("prepared attachment count mismatch"))
	}
	for index, attachment := range attachments {
		if attachment.ID != ids[index] {
			return Internal(errors.New("prepared attachment order mismatch"))
		}
		encoded := snapshotAttachmentV2{ID: attachment.ID, FileName: attachment.FileName, ContentType: attachment.ContentType, SizeBytes: attachment.SizeBytes, SHA256: hex.EncodeToString(attachment.SHA256[:]), CreatedAt: FormatTime(attachment.CreatedAt)}
		if _, err := decodeSnapshotAttachment(encoded); err != nil {
			return Internal(fmt.Errorf("invalid prepared attachment: %w", err))
		}
	}
	return nil
}
