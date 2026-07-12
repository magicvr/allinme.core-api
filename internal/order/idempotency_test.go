package order_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/order"
)

func TestCreateReplayUsesImmutableSnapshotWithoutGeneratingFacts(t *testing.T) {
	repository := &writeRepository{}
	orderIDs, itemIDs, clockCalls := 0, 0, 0
	service, err := order.NewServiceWithDependencies(repository, func() time.Time {
		clockCalls++
		return time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	}, func() (string, error) {
		orderIDs++
		return "ord_0000000000000000000000000000000a", nil
	}, func() (string, error) {
		itemIDs++
		return fmt.Sprintf("itm_%032x", itemIDs), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	principal := auth.Principal{UserID: "user-1", Role: auth.RoleOperator}
	command := order.CreateCommand{CustomerName: " Alice ", Currency: "CNY", Items: []order.ItemCommand{{SKU: " SKU ", Name: " Item ", Quantity: 2, UnitPrice: 300}}}
	first, err := service.Create(context.Background(), principal, "key-1", command)
	if err != nil {
		t.Fatal(err)
	}
	repository.existing = &repository.created.Record
	second, err := service.Create(context.Background(), principal, "key-1", command)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || second.CustomerName != "Alice" || orderIDs != 1 || itemIDs != 1 || clockCalls != 1 {
		t.Fatalf("first=%+v second=%+v calls=%d/%d/%d", first, second, orderIDs, itemIDs, clockCalls)
	}
	conflict := command
	conflict.CustomerName = "Bob"
	if _, err := service.Create(context.Background(), principal, "key-1", conflict); !errors.Is(err, order.ErrIdempotencyConflict) {
		t.Fatalf("conflict error = %v", err)
	}
}

func TestCreateReplayRejectsUnknownOrCorruptSnapshot(t *testing.T) {
	repository := &writeRepository{}
	service, err := order.NewServiceWithDependencies(repository, func() time.Time { return time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC) }, func() (string, error) { return "ord_0000000000000000000000000000000a", nil }, func() (string, error) { return "itm_0000000000000000000000000000000a", nil })
	if err != nil {
		t.Fatal(err)
	}
	principal := auth.Principal{UserID: "user-1", Role: auth.RoleAdmin}
	command := order.CreateCommand{CustomerName: "Alice", Currency: "CNY", Items: []order.ItemCommand{{SKU: "SKU", Name: "Item", Quantity: 1, UnitPrice: 100}}}
	if _, err := service.Create(context.Background(), principal, "key-1", command); err != nil {
		t.Fatal(err)
	}
	record := repository.created.Record
	repository.existing = &record
	repository.existing.SnapshotVersion = 2
	if _, err := service.Create(context.Background(), principal, "key-1", command); !errors.Is(err, order.ErrInternal) {
		t.Fatalf("unknown snapshot error = %v", err)
	}
	repository.existing.SnapshotVersion = 1
	repository.existing.SnapshotJSON = []byte(`{"order":{"id":"wrong"}}`)
	if _, err := service.Create(context.Background(), principal, "key-1", command); !errors.Is(err, order.ErrInternal) {
		t.Fatalf("corrupt snapshot error = %v", err)
	}
}
