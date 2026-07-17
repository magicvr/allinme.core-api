package order_test

import (
	"bytes"
	"context"
	"crypto/sha256"
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
	unknownVersion.SnapshotVersion = 3
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
		{name: "item ID", replacements: [][2]string{{`itm_0000000000000000000000000000000a`, `itm_0000000000000000000000000000000b`}}},
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

func TestCreateV2SnapshotDigestAndAttachmentReplay(t *testing.T) {
	attachmentID := "att_00000000000000000000000000000001"
	attachmentDigest := sha256.Sum256([]byte("attachment"))
	repository := &writeRepository{prepared: []order.Attachment{{ID: attachmentID, FileName: "invoice.pdf", ContentType: "application/pdf", SizeBytes: 10, SHA256: attachmentDigest, CreatedAt: time.Date(2026, 7, 12, 11, 0, 0, 0, time.UTC)}}}
	service, err := order.NewServiceWithDependencies(repository, func() time.Time { return time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC) }, func() (string, error) { return "ord_0000000000000000000000000000000a", nil }, func() (string, error) { return "itm_0000000000000000000000000000000a", nil })
	if err != nil {
		t.Fatal(err)
	}
	principal := auth.Principal{UserID: "user-1", Role: auth.RoleOperator}
	command := order.CreateCommand{CustomerName: " Alice ", Currency: "CNY", Items: []order.ItemCommand{{SKU: " SKU ", Name: " Item ", Quantity: 1, UnitPrice: 100}}, AttachmentIDs: []string{attachmentID}}
	first, err := service.Create(context.Background(), principal, "key-v2", command)
	if err != nil {
		t.Fatal(err)
	}
	wantDigestJSON := `{"operation":"POST /api/v1/orders","customerName":"Alice","currency":"CNY","items":[{"sku":"SKU","name":"Item","quantity":1,"unitPrice":100}],"attachmentIds":["att_00000000000000000000000000000001"]}`
	wantDigest := sha256.Sum256([]byte(wantDigestJSON))
	if repository.created.Record.SnapshotVersion != order.SnapshotVersionTwo || repository.created.Record.Scope.RequestDigest != wantDigest || repository.created.Create.AttachmentIDs[0] != attachmentID {
		t.Fatalf("persistence = %+v", repository.created)
	}
	if first.AttachmentCount != 1 || len(first.Attachments) != 1 || first.Attachments[0].ID != attachmentID {
		t.Fatalf("created order = %+v", first)
	}
	repository.existing = &repository.created.Record
	second, err := service.Create(context.Background(), principal, "key-v2", command)
	if err != nil {
		t.Fatal(err)
	}
	if second.Attachments[0].FileName != "invoice.pdf" || repository.prepareCalls != 1 {
		t.Fatalf("replay = %+v prepare calls=%d", second, repository.prepareCalls)
	}
	conflict := command
	conflict.AttachmentIDs = []string{"att_00000000000000000000000000000002"}
	if _, err := service.Create(context.Background(), principal, "key-v2", conflict); !errors.Is(err, order.ErrIdempotencyConflict) {
		t.Fatalf("attachment conflict error = %v", err)
	}
}

func TestCreateLegacyV1ReplayAllowsOnlyEmptyAttachments(t *testing.T) {
	repository := &writeRepository{}
	service, err := order.NewService(repository)
	if err != nil {
		t.Fatal(err)
	}
	requestJSON := `{"operation":"POST /api/v1/orders","customerName":"Legacy","currency":"CNY","items":[{"sku":"SKU","name":"Item","quantity":1,"unitPrice":100}]}`
	snapshotJSON := `{"order":{"id":"ord_00000000000000000000000000000001","customerName":"Legacy","status":"DRAFT","paymentStatus":"UNPAID","currency":"CNY","totalAmount":100,"version":1,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","items":[{"id":"itm_00000000000000000000000000000001","sku":"SKU","name":"Item","quantity":1,"unitPrice":100}]}}`
	record := order.IdempotencyRecord{Scope: order.IdempotencyScope{PrincipalUserID: "user-1", Method: order.CreateMethod, Route: order.CreateRoute, Key: "legacy", RequestDigest: sha256.Sum256([]byte(requestJSON))}, OrderID: "ord_00000000000000000000000000000001", SnapshotVersion: order.SnapshotVersionOne, SnapshotJSON: []byte(snapshotJSON), SnapshotDigest: sha256.Sum256([]byte(snapshotJSON)), CreatedAt: "2026-01-01T00:00:00Z"}
	repository.existing = &record
	command := order.CreateCommand{CustomerName: "Legacy", Currency: "CNY", Items: []order.ItemCommand{{SKU: "SKU", Name: "Item", Quantity: 1, UnitPrice: 100}}}
	result, err := service.Create(context.Background(), auth.Principal{UserID: "user-1", Role: auth.RoleOperator}, "legacy", command)
	if err != nil || result.AttachmentCount != 0 || result.Attachments == nil {
		t.Fatalf("legacy replay = %+v error=%v", result, err)
	}
	command.AttachmentIDs = []string{"att_00000000000000000000000000000001"}
	if _, err := service.Create(context.Background(), auth.Principal{UserID: "user-1", Role: auth.RoleOperator}, "legacy", command); !errors.Is(err, order.ErrIdempotencyConflict) {
		t.Fatalf("legacy attachment conflict = %v", err)
	}
}

func TestCreatePreparationFailurePrioritizesLateWinner(t *testing.T) {
	winnerRepository := &writeRepository{}
	service, err := order.NewServiceWithDependencies(winnerRepository, func() time.Time { return time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC) }, func() (string, error) { return "ord_0000000000000000000000000000000a", nil }, func() (string, error) { return "itm_0000000000000000000000000000000a", nil })
	if err != nil {
		t.Fatal(err)
	}
	command := order.CreateCommand{CustomerName: "Alice", Currency: "CNY", Items: []order.ItemCommand{{SKU: "SKU", Name: "Item", Quantity: 1, UnitPrice: 100}}}
	principal := auth.Principal{UserID: "user-1", Role: auth.RoleOperator}
	winner, err := service.Create(context.Background(), principal, "key-late", command)
	if err != nil {
		t.Fatal(err)
	}
	lateRepository := &writeRepository{prepareErr: errors.New("attachment read failed"), lateExisting: &winnerRepository.created.Record}
	lateService, err := order.NewService(lateRepository)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := lateService.Create(context.Background(), principal, "key-late", command)
	if err != nil || replayed.ID != winner.ID || lateRepository.idempotencyCalls != 2 {
		t.Fatalf("late replay = %+v calls=%d error=%v", replayed, lateRepository.idempotencyCalls, err)
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
