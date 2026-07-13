package order_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/order"
)

func TestRefundCreateNormalizesSnapshotAndReplaysBeforeCurrentFacts(t *testing.T) {
	repository := &refundRepositoryStub{}
	generatorCalls := 0
	service, err := order.NewRefundServiceWithDependencies(repository, func() time.Time {
		return time.Date(2026, 1, 2, 8, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}, func() (string, error) {
		generatorCalls++
		return "rfd_00000000000000000000000000000001", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	principal := auth.Principal{UserID: "user-operator", Username: "operator", Role: auth.RoleOperator}
	command := order.RefundRequestCommand{Amount: 100, Reason: " customer request ", OrderVersion: 3}
	created, err := service.Create(context.Background(), principal, "ord_00000000000000000000000000000001", "refund-key", command)
	if err != nil {
		t.Fatal(err)
	}
	if created.Refund.Reason != "customer request" || created.Refund.Status != order.RefundStatusPending || created.Refund.Version != 1 || created.Capabilities.CanApprove || created.Capabilities.CanReject || generatorCalls != 1 {
		t.Fatalf("created refund = %+v, generator calls = %d", created, generatorCalls)
	}
	wantSnapshot := `{"refund":{"id":"rfd_00000000000000000000000000000001","orderId":"ord_00000000000000000000000000000001","amount":100,"currency":"CNY","reason":"customer request","status":"PENDING","version":1,"requestedBy":{"id":"user-operator","username":"operator"},"decidedBy":null,"createdAt":"2026-01-02T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z","decidedAt":null,"canApprove":false,"canReject":false}}`
	if string(repository.created.Record.SnapshotJSON) != wantSnapshot {
		t.Fatalf("snapshot = %s", repository.created.Record.SnapshotJSON)
	}
	repository.existing = &repository.created.Record
	replayed, err := service.Create(context.Background(), principal, "ord_00000000000000000000000000000001", "refund-key", command)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Refund != created.Refund || generatorCalls != 1 {
		t.Fatalf("replayed = %+v, generator calls = %d", replayed, generatorCalls)
	}
	conflicting := command
	conflicting.Amount++
	if _, err := service.Create(context.Background(), principal, "ord_00000000000000000000000000000001", "refund-key", conflicting); !errors.Is(err, order.ErrIdempotencyConflict) || generatorCalls != 1 {
		t.Fatalf("conflicting replay error = %v, generator calls = %d", err, generatorCalls)
	}
}

func TestRefundCreateAuthorizationAndSnapshotDamage(t *testing.T) {
	principal := auth.Principal{UserID: "user-operator", Username: "operator", Role: auth.RoleOperator}
	command := order.RefundRequestCommand{Amount: 100, Reason: "request", OrderVersion: 1}
	for _, role := range []auth.Role{auth.RoleViewer, auth.RoleApprover} {
		service, _ := order.NewRefundService(&refundRepositoryStub{})
		if _, err := service.Create(context.Background(), auth.Principal{Role: role}, "ord_00000000000000000000000000000001", "key", command); !errors.Is(err, order.ErrForbidden) {
			t.Errorf("role %s error = %v", role, err)
		}
	}
	if service, _ := order.NewRefundService(&refundRepositoryStub{}); service != nil {
		if _, err := service.Create(context.Background(), principal, "bad", "key", command); !errors.Is(err, order.ErrNotFound) {
			t.Fatalf("invalid order ID error = %v", err)
		}
	}

	baseRepository := &refundRepositoryStub{}
	service, err := order.NewRefundServiceWithDependencies(baseRepository, func() time.Time { return time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC) }, func() (string, error) { return "rfd_00000000000000000000000000000001", nil })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(context.Background(), principal, "ord_00000000000000000000000000000001", "key", command); err != nil {
		t.Fatal(err)
	}
	base := baseRepository.created.Record
	mutations := []func(*order.RefundIdempotencyRecord){
		func(record *order.RefundIdempotencyRecord) { record.SnapshotVersion = 2 },
		func(record *order.RefundIdempotencyRecord) {
			record.SnapshotJSON = []byte(`{"refund":{}}`)
			record.SnapshotDigest = sha256.Sum256(record.SnapshotJSON)
		},
		func(record *order.RefundIdempotencyRecord) { record.SnapshotDigest = [sha256.Size]byte{} },
		func(record *order.RefundIdempotencyRecord) { record.RefundID = "rfd_00000000000000000000000000000002" },
		func(record *order.RefundIdempotencyRecord) { record.CreatedAt = "2026-01-02T00:00:01Z" },
	}
	for index, mutate := range mutations {
		record := base
		mutate(&record)
		repository := &refundRepositoryStub{existing: &record}
		damagedService, _ := order.NewRefundService(repository)
		if _, err := damagedService.Create(context.Background(), principal, "ord_00000000000000000000000000000001", "key", command); !errors.Is(err, order.ErrInternal) {
			t.Errorf("damaged record %d error = %v", index, err)
		}
		if repository.createCalls != 0 {
			t.Errorf("damaged record %d recreated refund", index)
		}
	}
}

func TestValidateRefundCreatePersistenceRejectsMismatchedInternalFacts(t *testing.T) {
	repository := &refundRepositoryStub{}
	service, err := order.NewRefundServiceWithDependencies(repository, func() time.Time { return time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC) }, func() (string, error) { return "rfd_00000000000000000000000000000001", nil })
	if err != nil {
		t.Fatal(err)
	}
	principal := auth.Principal{UserID: "user-operator", Username: "operator", Role: auth.RoleOperator}
	if _, err := service.Create(context.Background(), principal, "ord_00000000000000000000000000000001", "key", order.RefundRequestCommand{Amount: 100, Reason: "request", OrderVersion: 1}); err != nil {
		t.Fatal(err)
	}
	valid := repository.created
	if err := order.ValidateRefundCreatePersistence(valid); err != nil {
		t.Fatal(err)
	}
	mutations := []func(*order.RefundCreatePersistence){
		func(value *order.RefundCreatePersistence) { value.OrderVersion = 2 },
		func(value *order.RefundCreatePersistence) { value.Record.Scope.PrincipalUserID = "other" },
		func(value *order.RefundCreatePersistence) { value.Record.Scope.RequestDigest = [sha256.Size]byte{} },
		func(value *order.RefundCreatePersistence) { value.Refund.Amount++ },
	}
	for index, mutate := range mutations {
		value := valid
		mutate(&value)
		if err := order.ValidateRefundCreatePersistence(value); !errors.Is(err, order.ErrInternal) {
			t.Errorf("mismatched persistence %d error = %v", index, err)
		}
	}
}

type refundRepositoryStub struct {
	existing    *order.RefundIdempotencyRecord
	created     order.RefundCreatePersistence
	createCalls int
	err         error
}

func (repository *refundRepositoryStub) GetRefundIdempotency(context.Context, order.RefundIdempotencyScope) (order.RefundIdempotencyRecord, bool, error) {
	if repository.existing == nil {
		return order.RefundIdempotencyRecord{}, false, repository.err
	}
	return *repository.existing, true, repository.err
}

func (repository *refundRepositoryStub) CreateRefundIdempotent(_ context.Context, persistence order.RefundCreatePersistence) (order.RefundIdempotencyRecord, bool, error) {
	repository.createCalls++
	repository.created = persistence
	return persistence.Record, true, repository.err
}
