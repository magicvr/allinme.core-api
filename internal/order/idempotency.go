package order

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
)

const (
	CreateMethod       = "POST"
	CreateRoute        = "/api/v1/orders"
	SnapshotVersionOne = int64(1)
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

func normalizedDigest(command CreateCommand) ([sha256.Size]byte, error) {
	encoded, err := json.Marshal(createDigest{Operation: CreateMethod + " " + CreateRoute, CustomerName: command.CustomerName, Currency: command.Currency, Items: command.Items})
	if err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("encode normalized create digest: %w", err)
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

func decodeSnapshot(record IdempotencyRecord, expected IdempotencyScope) (Order, error) {
	if record.Scope.PrincipalUserID != expected.PrincipalUserID || record.Scope.Method != CreateMethod || record.Scope.Route != CreateRoute || record.Scope.Key != expected.Key || record.OrderID == "" || record.SnapshotVersion != SnapshotVersionOne {
		return Order{}, ErrInternal
	}
	if !bytes.Equal(record.Scope.RequestDigest[:], expected.RequestDigest[:]) {
		return Order{}, ErrIdempotencyConflict
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
	result := Order{ID: value.ID, CustomerName: value.CustomerName, Status: value.Status, PaymentStatus: value.PaymentStatus, Currency: value.Currency, TotalAmount: value.TotalAmount, Version: value.Version, CreatedAt: createdAt, UpdatedAt: updatedAt, Items: make([]Item, 0, len(value.Items))}
	command := CreateCommand{CustomerName: value.CustomerName, Currency: value.Currency, Items: make([]ItemCommand, 0, len(value.Items))}
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

func sameCreateFacts(left, right CreateCommand) bool {
	if left.CustomerName != right.CustomerName || left.Currency != right.Currency || len(left.Items) != len(right.Items) {
		return false
	}
	for index := range left.Items {
		if left.Items[index] != right.Items[index] {
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
	digest, err := normalizedDigest(normalized)
	if err != nil {
		return Order{}, err
	}
	scope := IdempotencyScope{PrincipalUserID: principal.UserID, Method: CreateMethod, Route: CreateRoute, Key: key, RequestDigest: digest}
	existing, found, err := service.repository.GetIdempotency(ctx, scope)
	if err != nil {
		return Order{}, fmt.Errorf("read idempotency key: %w", err)
	}
	if found {
		return decodeSnapshot(existing, scope)
	}
	orderID, err := service.newOrderID()
	if err != nil {
		return Order{}, fmt.Errorf("generate order ID: %w", err)
	}
	items, err := service.persistenceItems(normalized.Items)
	if err != nil {
		return Order{}, err
	}
	now := UTCNow(service.clock)
	result := Order{ID: orderID, CustomerName: normalized.CustomerName, Status: StatusDraft, PaymentStatus: PaymentStatusUnpaid, Currency: normalized.Currency, TotalAmount: total, Version: 1, CreatedAt: now, UpdatedAt: now, Items: make([]Item, 0, len(items))}
	for _, item := range items {
		result.Items = append(result.Items, Item{ID: item.ID, SKU: item.SKU, Name: item.Name, Quantity: item.Quantity, UnitPrice: item.UnitPrice})
	}
	snapshot, err := encodeSnapshotV1(result)
	if err != nil {
		return Order{}, fmt.Errorf("encode order snapshot: %w", err)
	}
	record, _, err := service.repository.CreateOrderIdempotent(ctx, IdempotentCreatePersistence{Create: CreatePersistence{ID: orderID, CustomerName: normalized.CustomerName, Currency: normalized.Currency, TotalAmount: total, CreatedAt: FormatTime(now), Items: items}, Record: IdempotencyRecord{Scope: scope, OrderID: orderID, SnapshotVersion: SnapshotVersionOne, SnapshotJSON: snapshot, CreatedAt: FormatTime(now)}})
	if err != nil {
		return Order{}, fmt.Errorf("create order: %w", err)
	}
	replayed, err := decodeSnapshot(record, scope)
	if err != nil {
		return Order{}, err
	}
	return replayed, nil
}
