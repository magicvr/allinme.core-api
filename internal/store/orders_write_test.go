package store_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/order"
	"github.com/magicvr/allinme.core-api/internal/store"
)

func TestCreateOrderAndUpdateDraftTransactions(t *testing.T) {
	ctx := context.Background()
	database, err := store.Open(ctx, filepath.Join(t.TempDir(), "orders.db"), store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	persistence := idempotentPersistence("key-1")
	record, created, err := database.CreateOrderIdempotent(ctx, persistence)
	if err != nil {
		t.Fatal(err)
	}
	if !created || record.OrderID != persistence.Create.ID {
		t.Fatalf("created=%v record=%+v", created, record)
	}
	createdOrder, found, err := database.GetOrder(ctx, persistence.Create.ID)
	if err != nil || !found {
		t.Fatal(err)
	}
	updated, err := database.UpdateDraft(ctx, order.UpdateDraftPersistence{
		ID: createdOrder.ID, CustomerName: "Updated", Currency: "CNY", TotalAmount: 900, Version: 1, UpdatedAt: "2026-07-12T01:00:00Z",
		Items: []order.PersistenceItem{{ID: "itm_0000000000000000000000000000000b", Position: 0, SKU: "NEW", Name: "New", Quantity: 3, UnitPrice: 300}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.CustomerName != "Updated" || updated.Version != 2 || updated.TotalAmount != 900 || len(updated.Items) != 1 || updated.Items[0].SKU != "NEW" || updated.CreatedAt.Format("2006-01-02T15:04:05Z") != "2026-07-12T00:00:00Z" {
		t.Fatalf("updated = %+v", updated)
	}
	if _, err := database.UpdateDraft(ctx, order.UpdateDraftPersistence{ID: createdOrder.ID, CustomerName: "Conflict", Currency: "CNY", TotalAmount: 1, Version: 1, UpdatedAt: "2026-07-12T02:00:00Z", Items: []order.PersistenceItem{{ID: "itm_0000000000000000000000000000000c", SKU: "X", Name: "X", Quantity: 1, UnitPrice: 1}}}); !errors.Is(err, order.ErrVersionConflict) {
		t.Fatalf("stale UpdateDraft() error = %v", err)
	}
}

func TestCreateOrderBindsPreparedAttachmentsInOrder(t *testing.T) {
	database := openSeededOrderDatabaseForWrite(t)
	ctx := context.Background()
	insertAttachmentOwner(t, database, "user-1")
	ids := []string{"att_00000000000000000000000000000002", "att_00000000000000000000000000000001"}
	for _, id := range ids {
		insertUploadedAttachment(t, database, id, "user-1", "2026-07-12T00:00:00Z")
	}
	prepared, err := database.PrepareAttachmentsForOrder(ctx, "user-1", ids, time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC))
	if err != nil || len(prepared) != 2 || prepared[0].ID != ids[0] || prepared[1].ID != ids[1] {
		t.Fatalf("prepared = %+v error=%v", prepared, err)
	}
	persistence := idempotentPersistence("key-attachments")
	persistence.Create.AttachmentIDs = ids
	if _, created, err := database.CreateOrderIdempotent(ctx, persistence); err != nil || !created {
		t.Fatalf("create = %v %v", created, err)
	}
	result, found, err := database.GetOrder(ctx, persistence.Create.ID)
	if err != nil || !found || result.AttachmentCount != 2 || len(result.Attachments) != 2 || result.Attachments[0].ID != ids[0] || result.Attachments[1].ID != ids[1] {
		t.Fatalf("order = %+v found=%v error=%v", result, found, err)
	}
	for _, id := range ids {
		var status string
		var expiresAt sql.NullString
		if err := database.SQL().QueryRowContext(ctx, `SELECT status, expires_at FROM attachments WHERE id = ?`, id).Scan(&status, &expiresAt); err != nil {
			t.Fatal(err)
		}
		if status != "BOUND" || expiresAt.Valid {
			t.Fatalf("attachment %s = %s/%+v", id, status, expiresAt)
		}
	}
	updated, err := database.UpdateDraft(ctx, order.UpdateDraftPersistence{ID: result.ID, CustomerName: "Updated with attachments", Currency: "CNY", TotalAmount: 500, Version: 1, UpdatedAt: "2026-07-12T13:00:00Z", Items: []order.PersistenceItem{{ID: "itm_0000000000000000000000000000000b", Position: 0, SKU: "SKU", Name: "Item", Quantity: 2, UnitPrice: 250}}})
	if err != nil || updated.AttachmentCount != 2 || len(updated.Attachments) != 2 || updated.Attachments[0].ID != ids[0] {
		t.Fatalf("updated order = %+v error=%v", updated, err)
	}
	transitioned, err := database.TransitionOrder(ctx, order.TransitionPersistence{ID: result.ID, Version: 2, AllowedSources: []order.Status{order.StatusDraft}, Target: order.StatusConfirmed, UpdatedAt: "2026-07-12T14:00:00Z"})
	if err != nil || transitioned.AttachmentCount != 2 || len(transitioned.Attachments) != 2 || transitioned.Attachments[1].ID != ids[1] {
		t.Fatalf("transitioned order = %+v error=%v", transitioned, err)
	}
	page, err := database.ListOrders(ctx, order.ListQuery{Page: 1, PageSize: 20, Sort: "createdAt"})
	if err != nil {
		t.Fatal(err)
	}
	var listed *order.Order
	for index := range page.Items {
		if page.Items[index].ID == persistence.Create.ID {
			listed = &page.Items[index]
		}
	}
	if listed == nil || listed.AttachmentCount != 2 || listed.Attachments != nil {
		t.Fatalf("listed order = %+v", listed)
	}
}

func TestCreateOrderAttachmentFailureRollsBackAndLoserDoesNotBind(t *testing.T) {
	database := openSeededOrderDatabaseForWrite(t)
	ctx := context.Background()
	insertAttachmentOwner(t, database, "user-1")
	insertAttachmentOwner(t, database, "user-2")
	firstID := "att_00000000000000000000000000000001"
	otherID := "att_00000000000000000000000000000002"
	insertUploadedAttachment(t, database, firstID, "user-1", "2026-07-12T00:00:00Z")
	insertUploadedAttachment(t, database, otherID, "user-2", "2026-07-12T00:00:00Z")
	failed := idempotentPersistence("key-attachment-failure")
	failed.Create.AttachmentIDs = []string{firstID, otherID}
	if _, _, err := database.CreateOrderIdempotent(ctx, failed); err == nil {
		t.Fatal("attachment create error = nil")
	} else if details, ok := order.ValidationDetails(err); !ok || len(details) != 1 || details[0].Field != "attachmentIds[1]" {
		t.Fatalf("attachment create error = %v details=%+v", err, details)
	}
	var orders, mappings int
	if err := database.SQL().QueryRowContext(ctx, `SELECT COUNT(*) FROM orders WHERE id = ?`, failed.Create.ID).Scan(&orders); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRowContext(ctx, `SELECT COUNT(*) FROM order_attachments`).Scan(&mappings); err != nil {
		t.Fatal(err)
	}
	var firstStatus string
	if err := database.SQL().QueryRowContext(ctx, `SELECT status FROM attachments WHERE id = ?`, firstID).Scan(&firstStatus); err != nil {
		t.Fatal(err)
	}
	if orders != 0 || mappings != 0 || firstStatus != "UPLOADED" {
		t.Fatalf("rollback orders=%d mappings=%d first=%s", orders, mappings, firstStatus)
	}

	winner := idempotentPersistence("key-attachment-winner")
	winner.Create.ID, winner.Record.OrderID = "ord_0000000000000000000000000000000b", "ord_0000000000000000000000000000000b"
	if _, created, err := database.CreateOrderIdempotent(ctx, winner); err != nil || !created {
		t.Fatalf("winner = %v %v", created, err)
	}
	candidateID := "att_00000000000000000000000000000003"
	insertUploadedAttachment(t, database, candidateID, "user-1", "2026-07-12T00:00:00Z")
	loser := idempotentPersistence("key-attachment-winner")
	loser.Create.ID, loser.Record.OrderID = "ord_0000000000000000000000000000000c", "ord_0000000000000000000000000000000c"
	loser.Create.AttachmentIDs = []string{candidateID}
	if _, created, err := database.CreateOrderIdempotent(ctx, loser); err != nil || created {
		t.Fatalf("loser = %v %v", created, err)
	}
	var candidateStatus string
	if err := database.SQL().QueryRowContext(ctx, `SELECT status FROM attachments WHERE id = ?`, candidateID).Scan(&candidateStatus); err != nil {
		t.Fatal(err)
	}
	if candidateStatus != "UPLOADED" {
		t.Fatalf("loser candidate status = %s", candidateStatus)
	}
}

func insertAttachmentOwner(t *testing.T, database *store.DB, id string) {
	t.Helper()
	if _, err := database.SQL().Exec(`INSERT INTO users(id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, 'hash', 'operator', '2026-07-12T00:00:00Z', '2026-07-12T00:00:00Z')`, id, id); err != nil {
		t.Fatal(err)
	}
}

func insertUploadedAttachment(t *testing.T, database *store.DB, id, owner, createdAt string) {
	t.Helper()
	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(id))
	if _, err := database.SQL().Exec(`INSERT INTO attachments(id, created_by, status, file_name, storage_key, content_type, size_bytes, sha256, expires_at, created_at, updated_at) VALUES (?, ?, 'UPLOADED', ?, ?, 'application/pdf', 100, ?, ?, ?, ?)`, id, owner, id+".pdf", id, digest[:], created.Add(24*time.Hour).Format(time.RFC3339), createdAt, createdAt); err != nil {
		t.Fatal(err)
	}
}

func TestCreateOrderRollsBackWholeAggregate(t *testing.T) {
	ctx := context.Background()
	database, err := store.Open(ctx, filepath.Join(t.TempDir(), "rollback.db"), store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	persistence := idempotentPersistence("key-rollback")
	persistence.Create.Items[0].ID = "bad"
	_, _, err = database.CreateOrderIdempotent(ctx, persistence)
	if err == nil {
		t.Fatal("CreateOrder() error = nil")
	}
	var orders, items int
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&orders); err != nil {
		t.Fatal(err)
	}
	if err := database.SQL().QueryRow(`SELECT COUNT(*) FROM order_items`).Scan(&items); err != nil {
		t.Fatal(err)
	}
	if orders != 0 || items != 0 {
		t.Fatalf("orders=%d items=%d", orders, items)
	}
}

func TestCreateOrderIdempotentReplaysStoredSnapshot(t *testing.T) {
	database := openSeededOrderDatabaseForWrite(t)
	ctx := context.Background()
	persistence := idempotentPersistence("key-replay")
	persistence.Create.ID = "ord_0000000000000000000000000000000a"
	persistence.Record.OrderID = persistence.Create.ID
	first, created, err := database.CreateOrderIdempotent(ctx, persistence)
	if err != nil || !created {
		t.Fatalf("first create = %+v %v %v", first, created, err)
	}
	persistence.Record.Scope.RequestDigest[0] = 2
	persistence.Create.ID = "ord_0000000000000000000000000000000b"
	second, created, err := database.CreateOrderIdempotent(ctx, persistence)
	if err != nil || created || second.OrderID != first.OrderID || second.Scope.RequestDigest[0] != 1 {
		t.Fatalf("replay = %+v %v %v", second, created, err)
	}
	var count int
	if err := database.SQL().QueryRowContext(ctx, `SELECT COUNT(*) FROM orders WHERE id IN (?, ?)`, first.OrderID, persistence.Create.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("created orders = %d", count)
	}
}

func TestCreateOrderIdempotentConnectionWaitHonorsDeadline(t *testing.T) {
	database := openSeededOrderDatabaseForWrite(t)
	connection, err := database.SQL().Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	persistence := idempotentPersistence("key-deadline")
	persistence.Create.ID = "ord_0000000000000000000000000000000a"
	persistence.Record.OrderID = persistence.Create.ID
	if _, _, err := database.CreateOrderIdempotent(ctx, persistence); !errors.Is(err, context.DeadlineExceeded) || errors.Is(err, order.ErrUnavailable) {
		t.Fatalf("connection wait error = %v", err)
	}
	var count int
	if err := connection.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM orders WHERE id = ?`, persistence.Create.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("orders = %d", count)
	}
}

func TestCreateOrderIdempotentAcrossTwoDatabasesCreatesOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "race.db")
	first, err := store.Open(context.Background(), path, store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	if _, err := first.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	second, err := store.Open(context.Background(), path, store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	persistences := []order.IdempotentCreatePersistence{idempotentPersistence("key-race"), idempotentPersistence("key-race")}
	persistences[0].Create.ID, persistences[0].Record.OrderID = "ord_0000000000000000000000000000000a", "ord_0000000000000000000000000000000a"
	persistences[1].Create.ID, persistences[1].Record.OrderID = "ord_0000000000000000000000000000000b", "ord_0000000000000000000000000000000b"
	databases := []*store.DB{first, second}
	start := make(chan struct{})
	errorsFound := make([]error, 2)
	var wait sync.WaitGroup
	for index := range databases {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			_, _, errorsFound[index] = databases[index].CreateOrderIdempotent(context.Background(), persistences[index])
		}(index)
	}
	close(start)
	wait.Wait()
	for _, err := range errorsFound {
		if err != nil && !errors.Is(err, order.ErrUnavailable) {
			t.Fatalf("race error = %v", err)
		}
	}
	var orders, keys int
	if err := first.SQL().QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&orders); err != nil {
		t.Fatal(err)
	}
	if err := first.SQL().QueryRow(`SELECT COUNT(*) FROM idempotency_keys`).Scan(&keys); err != nil {
		t.Fatal(err)
	}
	if orders != 1 || keys != 1 {
		t.Fatalf("orders=%d keys=%d errors=%v", orders, keys, errorsFound)
	}
}

func TestCreateOrderIdempotentClassifiesSQLiteBusy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "busy.db")
	first, err := store.Open(context.Background(), path, store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	if _, err := first.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := first.SeedOrderDemo(context.Background(), testTime()); err != nil {
		t.Fatal(err)
	}
	second, err := store.Open(context.Background(), path, store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if _, err := second.SQL().Exec(`PRAGMA busy_timeout = 1`); err != nil {
		t.Fatal(err)
	}
	locked, release := make(chan struct{}), make(chan struct{})
	transactionDone := make(chan error, 1)
	go func() {
		transactionDone <- first.WithTx(context.Background(), func(tx *sql.Tx) error {
			if _, err := tx.Exec(`UPDATE orders SET customer_name = customer_name WHERE id = 'ord_00000000000000000000000000000001'`); err != nil {
				return err
			}
			close(locked)
			<-release
			return nil
		})
	}()
	<-locked
	persistence := idempotentPersistence("key-busy")
	persistence.Create.ID, persistence.Record.OrderID = "ord_0000000000000000000000000000000a", "ord_0000000000000000000000000000000a"
	_, _, createErr := second.CreateOrderIdempotent(context.Background(), persistence)
	close(release)
	if err := <-transactionDone; err != nil {
		t.Fatal(err)
	}
	if !errors.Is(createErr, order.ErrUnavailable) {
		t.Fatalf("busy error = %v", createErr)
	}
}

func idempotentPersistence(key string) order.IdempotentCreatePersistence {
	var digest [32]byte
	digest[0] = 1
	return order.IdempotentCreatePersistence{
		Create: order.CreatePersistence{ID: "ord_0000000000000000000000000000000a", CustomerName: "Customer", Currency: "CNY", TotalAmount: 500, CreatedAt: "2026-07-12T00:00:00Z", Items: []order.PersistenceItem{{ID: "itm_0000000000000000000000000000000a", Position: 0, SKU: "SKU", Name: "Item", Quantity: 2, UnitPrice: 250}}},
		Record: order.IdempotencyRecord{Scope: order.IdempotencyScope{PrincipalUserID: "user-1", Method: order.CreateMethod, Route: order.CreateRoute, Key: key, RequestDigest: digest}, OrderID: "ord_0000000000000000000000000000000a", SnapshotVersion: 1, SnapshotJSON: []byte(`{"order":{"id":"ord_0000000000000000000000000000000a"}}`), CreatedAt: "2026-07-12T00:00:00Z"},
	}
}

func TestUpdateDraftClassifiesMissingStateAndRollsBackItems(t *testing.T) {
	database := openSeededOrderDatabaseForWrite(t)
	ctx := context.Background()
	if _, err := database.UpdateDraft(ctx, order.UpdateDraftPersistence{ID: "ord_ffffffffffffffffffffffffffffffff", Version: 1}); !errors.Is(err, order.ErrNotFound) {
		t.Fatalf("missing UpdateDraft() error = %v", err)
	}
	confirmedID := "ord_00000000000000000000000000000002"
	if _, err := database.UpdateDraft(ctx, order.UpdateDraftPersistence{ID: confirmedID, Version: 1}); !errors.Is(err, order.ErrStateConflict) {
		t.Fatalf("confirmed UpdateDraft() error = %v", err)
	}
	draftID := "ord_00000000000000000000000000000001"
	_, err := database.UpdateDraft(ctx, order.UpdateDraftPersistence{ID: draftID, CustomerName: "Should Roll Back", Currency: "CNY", TotalAmount: 1, Version: 1, UpdatedAt: "2026-07-12T03:00:00Z", Items: []order.PersistenceItem{{ID: "bad", SKU: "X", Name: "X", Quantity: 1, UnitPrice: 1}}})
	if err == nil {
		t.Fatal("invalid item UpdateDraft() error = nil")
	}
	got, found, err := database.GetOrder(ctx, draftID)
	if err != nil || !found {
		t.Fatal(err)
	}
	if got.CustomerName != "Draft Demo Customer" || got.Version != 1 || got.Items[0].SKU != "DEMO-DRAFT" || got.UpdatedAt.Format("2006-01-02T15:04:05Z") != "2026-01-01T00:00:00Z" {
		t.Fatalf("rolled back order = %+v", got)
	}
}

func TestTransitionOrderCoversLegalAndIllegalStates(t *testing.T) {
	database := openSeededOrderDatabaseForWrite(t)
	ctx := context.Background()
	tests := []struct {
		id      string
		sources []order.Status
		target  order.Status
	}{
		{"ord_00000000000000000000000000000001", []order.Status{order.StatusDraft}, order.StatusConfirmed},
		{"ord_00000000000000000000000000000002", []order.Status{order.StatusConfirmed}, order.StatusFulfilling},
		{"ord_00000000000000000000000000000003", []order.Status{order.StatusFulfilling}, order.StatusShipped},
		{"ord_00000000000000000000000000000004", []order.Status{order.StatusShipped}, order.StatusCompleted},
	}
	for _, test := range tests {
		result, err := database.TransitionOrder(ctx, order.TransitionPersistence{ID: test.id, Version: 1, AllowedSources: test.sources, Target: test.target, UpdatedAt: "2026-07-13T00:00:00Z"})
		if err != nil || result.Status != test.target || result.Version != 2 || result.UpdatedAt.Format(time.RFC3339) != "2026-07-13T00:00:00Z" {
			t.Fatalf("transition %s = %+v %v", test.id, result, err)
		}
	}
	if _, err := database.TransitionOrder(ctx, order.TransitionPersistence{ID: "ord_ffffffffffffffffffffffffffffffff", Version: 1, AllowedSources: []order.Status{order.StatusDraft}, Target: order.StatusConfirmed, UpdatedAt: "2026-07-13T00:00:00Z"}); !errors.Is(err, order.ErrNotFound) {
		t.Fatalf("missing transition error = %v", err)
	}
	if _, err := database.TransitionOrder(ctx, order.TransitionPersistence{ID: "ord_00000000000000000000000000000005", Version: 2, AllowedSources: []order.Status{order.StatusShipped}, Target: order.StatusCompleted, UpdatedAt: "2026-07-13T00:00:00Z"}); !errors.Is(err, order.ErrVersionConflict) {
		t.Fatalf("version transition error = %v", err)
	}
	if _, err := database.TransitionOrder(ctx, order.TransitionPersistence{ID: "ord_00000000000000000000000000000005", Version: 1, AllowedSources: []order.Status{order.StatusShipped}, Target: order.StatusCompleted, UpdatedAt: "2026-07-13T00:00:00Z"}); !errors.Is(err, order.ErrStateConflict) {
		t.Fatalf("state transition error = %v", err)
	}
}

func TestTransitionOrderSameVersionConcurrentChangesOnce(t *testing.T) {
	database := openSeededOrderDatabaseForWrite(t)
	persistence := order.TransitionPersistence{ID: "ord_00000000000000000000000000000001", Version: 1, AllowedSources: []order.Status{order.StatusDraft}, Target: order.StatusConfirmed, UpdatedAt: "2026-07-13T00:00:00Z"}
	start := make(chan struct{})
	errorsFound := make([]error, 2)
	var wait sync.WaitGroup
	for index := range errorsFound {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			_, errorsFound[index] = database.TransitionOrder(context.Background(), persistence)
		}(index)
	}
	close(start)
	wait.Wait()
	successes, conflicts := 0, 0
	for _, err := range errorsFound {
		if err == nil {
			successes++
		} else if errors.Is(err, order.ErrVersionConflict) {
			conflicts++
		} else {
			t.Fatalf("concurrent transition error = %v", err)
		}
	}
	result, found, err := database.GetOrder(context.Background(), persistence.ID)
	if err != nil || !found || successes != 1 || conflicts != 1 || result.Status != order.StatusConfirmed || result.Version != 2 {
		t.Fatalf("successes=%d conflicts=%d order=%+v found=%v err=%v", successes, conflicts, result, found, err)
	}
}

func TestTransitionOrderSameVersionAcrossTwoDatabasesChangesOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transition-race.db")
	first, err := store.Open(context.Background(), path, store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = first.Close() })
	if _, err := first.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := first.SeedOrderDemo(context.Background(), testTime()); err != nil {
		t.Fatal(err)
	}
	second, err := store.Open(context.Background(), path, store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = second.Close() })

	persistence := order.TransitionPersistence{ID: "ord_00000000000000000000000000000001", Version: 1, AllowedSources: []order.Status{order.StatusDraft}, Target: order.StatusConfirmed, UpdatedAt: "2026-07-13T00:00:00Z"}
	databases := []*store.DB{first, second}
	start := make(chan struct{})
	errorsFound := make([]error, len(databases))
	var wait sync.WaitGroup
	for index := range databases {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, errorsFound[index] = databases[index].TransitionOrder(ctx, persistence)
		}(index)
	}
	close(start)
	wait.Wait()

	successes, conflictsOrUnavailable := 0, 0
	for _, err := range errorsFound {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, order.ErrVersionConflict), errors.Is(err, order.ErrUnavailable):
			conflictsOrUnavailable++
		default:
			t.Fatalf("two-database transition error = %v", err)
		}
	}
	result, found, err := first.GetOrder(context.Background(), persistence.ID)
	if err != nil || !found || successes != 1 || conflictsOrUnavailable != 1 || result.Status != order.StatusConfirmed || result.Version != 2 {
		t.Fatalf("successes=%d conflictsOrUnavailable=%d errors=%v order=%+v found=%v err=%v", successes, conflictsOrUnavailable, errorsFound, result, found, err)
	}
}

func TestTransitionOrderCancelFromEveryLegalSource(t *testing.T) {
	database := openSeededOrderDatabaseForWrite(t)
	for _, id := range []string{
		"ord_00000000000000000000000000000001",
		"ord_00000000000000000000000000000002",
		"ord_00000000000000000000000000000003",
	} {
		result, err := database.TransitionOrder(context.Background(), order.TransitionPersistence{
			ID: id, Version: 1,
			AllowedSources: []order.Status{order.StatusDraft, order.StatusConfirmed, order.StatusFulfilling},
			Target:         order.StatusCancelled, UpdatedAt: "2026-07-13T00:00:00Z",
		})
		if err != nil || result.Status != order.StatusCancelled || result.Version != 2 {
			t.Fatalf("cancel %s = %+v %v", id, result, err)
		}
	}
}

func openSeededOrderDatabaseForWrite(t *testing.T) *store.DB {
	t.Helper()
	database, err := store.Open(context.Background(), filepath.Join(t.TempDir(), "seeded.db"), store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	if _, err := database.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SeedOrderDemo(context.Background(), testTime()); err != nil {
		t.Fatal(err)
	}
	return database
}

func testTime() (value time.Time) {
	return time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)
}
