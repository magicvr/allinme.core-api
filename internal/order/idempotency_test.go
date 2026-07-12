package order_test

import (
	"bytes"
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
	unknownVersion := record
	unknownVersion.SnapshotVersion = 2
	repository.existing = &unknownVersion
	if _, err := service.Create(context.Background(), principal, "key-1", command); !errors.Is(err, order.ErrInternal) {
		t.Fatalf("unknown snapshot error = %v", err)
	}
	corrupt := record
	corrupt.SnapshotJSON = []byte(`{"order":{"id":"wrong"}}`)
	repository.existing = &corrupt
	if _, err := service.Create(context.Background(), principal, "key-1", command); !errors.Is(err, order.ErrInternal) {
		t.Fatalf("corrupt snapshot error = %v", err)
	}

	for _, test := range []struct {
		name         string
		replacements [][2]string
	}{
		{name: "customer", replacements: [][2]string{{`"customerName":"Alice"`, `"customerName":"Bob"`}}},
		{name: "item", replacements: [][2]string{{`"sku":"SKU"`, `"sku":"OTHER"`}}},
		{name: "amount", replacements: [][2]string{{`"totalAmount":100`, `"totalAmount":200`}, {`"unitPrice":100`, `"unitPrice":200`}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			tampered := append([]byte(nil), record.SnapshotJSON...)
			for _, replacement := range test.replacements {
				tampered = bytes.Replace(tampered, []byte(replacement[0]), []byte(replacement[1]), 1)
			}
			tamperedRecord := record
			tamperedRecord.SnapshotJSON = tampered
			repository.existing = &tamperedRecord
			if _, err := service.Create(context.Background(), principal, "key-1", command); !errors.Is(err, order.ErrInternal) {
				t.Fatalf("tampered snapshot error = %v", err)
			}
		})
	}
}

func TestCreateReplayRejectsDuplicateSnapshotItemIDs(t *testing.T) {
	repository := &writeRepository{}
	itemID := 0
	service, err := order.NewServiceWithDependencies(repository, func() time.Time { return time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC) }, func() (string, error) { return "ord_0000000000000000000000000000000a", nil }, func() (string, error) {
		itemID++
		return fmt.Sprintf("itm_%032x", itemID), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	principal := auth.Principal{UserID: "user-1", Role: auth.RoleAdmin}
	command := order.CreateCommand{CustomerName: "Alice", Currency: "CNY", Items: []order.ItemCommand{{SKU: "ONE", Name: "One", Quantity: 1, UnitPrice: 100}, {SKU: "TWO", Name: "Two", Quantity: 1, UnitPrice: 200}}}
	if _, err := service.Create(context.Background(), principal, "key-duplicate", command); err != nil {
		t.Fatal(err)
	}
	record := repository.created.Record
	record.SnapshotJSON = bytes.Replace(record.SnapshotJSON, []byte("itm_00000000000000000000000000000002"), []byte("itm_00000000000000000000000000000001"), 1)
	repository.existing = &record
	if _, err := service.Create(context.Background(), principal, "key-duplicate", command); !errors.Is(err, order.ErrInternal) {
		t.Fatalf("duplicate item ID error = %v", err)
	}
}
