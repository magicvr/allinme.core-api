package order_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/order"
)

func TestCreateNormalizesCalculatesAndPersists(t *testing.T) {
	repository := &writeRepository{}
	sequence := 0
	service, err := order.NewServiceWithDependencies(repository, func() time.Time {
		return time.Date(2026, 7, 12, 20, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}, func() (string, error) {
		return "ord_0000000000000000000000000000000a", nil
	}, func() (string, error) {
		sequence++
		return fmt.Sprintf("itm_%032x", sequence), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Create(context.Background(), auth.Principal{UserID: "user-1", Role: auth.RoleOperator}, "key-1", order.CreateCommand{
		CustomerName: " Alice ", Currency: "CNY",
		Items: []order.ItemCommand{{SKU: " SKU-1 ", Name: " Item 1 ", Quantity: 2, UnitPrice: 100}, {SKU: "SKU-2", Name: "Item 2", Quantity: 3, UnitPrice: 250}},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := repository.created.Create
	if got.ID != "ord_0000000000000000000000000000000a" || got.CustomerName != "Alice" || got.TotalAmount != 950 || got.CreatedAt != "2026-07-12T12:00:00Z" || len(got.Items) != 2 || got.Items[0].SKU != "SKU-1" || got.Items[1].Position != 1 {
		t.Fatalf("CreatePersistence = %+v", got)
	}
}

func TestCreateAndEditValidationAndAuthorization(t *testing.T) {
	service, err := order.NewServiceWithDependencies(&writeRepository{}, nil, func() (string, error) { return "ord_00000000000000000000000000000001", nil }, func() (string, error) { return "itm_00000000000000000000000000000001", nil })
	if err != nil {
		t.Fatal(err)
	}
	valid := order.CreateCommand{CustomerName: "Customer", Currency: "CNY", Items: []order.ItemCommand{{SKU: "SKU", Name: "Item", Quantity: 1, UnitPrice: 1}}}
	if _, err := service.Create(context.Background(), auth.Principal{UserID: "viewer-1", Role: auth.RoleViewer}, "key-1", valid); !errors.Is(err, order.ErrForbidden) {
		t.Fatalf("viewer Create() error = %v", err)
	}
	invalid := order.CreateCommand{CustomerName: " ", Currency: "USD", Items: []order.ItemCommand{{SKU: " ", Name: strings.Repeat("界", 54), Quantity: 10001, UnitPrice: order.MaxAmount + 1}}}
	if _, err := service.Create(context.Background(), auth.Principal{UserID: "admin-1", Role: auth.RoleAdmin}, "key-1", invalid); err == nil {
		t.Fatal("invalid Create() error = nil")
	} else if details, ok := order.ValidationDetails(err); !ok || len(details) < 7 || details[0].Field != "customerName" || details[1].Field != "currency" {
		t.Fatalf("validation details = %+v, %v", details, ok)
	}
	overflow := order.CreateCommand{CustomerName: "Customer", Currency: "CNY", Items: []order.ItemCommand{{SKU: "SKU", Name: "Item", Quantity: order.MaxQuantity, UnitPrice: order.MaxAmount}}}
	if _, err := service.Create(context.Background(), auth.Principal{UserID: "admin-1", Role: auth.RoleAdmin}, "key-1", overflow); err == nil {
		t.Fatal("overflow Create() error = nil")
	}
	if _, err := service.Edit(context.Background(), auth.Principal{Role: auth.RoleAdmin}, "ord_00000000000000000000000000000001", order.EditCommand{CustomerName: valid.CustomerName, Currency: valid.Currency, Items: valid.Items, Version: 0}); err == nil {
		t.Fatal("zero version Edit() error = nil")
	}
}

func TestParseIntegerLexemeRejectsNonIntegerJSONForms(t *testing.T) {
	for _, valid := range []string{"0", "-0", "1", "-1", "9223372036854775807", "-9223372036854775808"} {
		if _, err := order.ParseIntegerLexeme(valid); err != nil {
			t.Errorf("ParseIntegerLexeme(%q) error = %v", valid, err)
		}
	}
	for _, invalid := range []string{"", "+1", "01", "-01", "1.0", "1e2", `"1"`, "null", "9223372036854775808", "-9223372036854775809"} {
		if _, err := order.ParseIntegerLexeme(invalid); err == nil {
			t.Errorf("ParseIntegerLexeme(%q) error = nil", invalid)
		}
	}
}

func TestEditPassesVersionAndNewTimestamp(t *testing.T) {
	repository := &writeRepository{}
	service, err := order.NewServiceWithDependencies(repository, func() time.Time { return time.Date(2026, 7, 12, 1, 2, 3, 0, time.UTC) }, nil, func() (string, error) { return "itm_00000000000000000000000000000009", nil })
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Edit(context.Background(), auth.Principal{Role: auth.RoleAdmin}, "ord_00000000000000000000000000000001", order.EditCommand{CustomerName: " Updated ", Currency: "CNY", Items: []order.ItemCommand{{SKU: " S ", Name: " N ", Quantity: 2, UnitPrice: 300}}, Version: 7})
	if err != nil {
		t.Fatal(err)
	}
	if repository.updated.ID == "" || repository.updated.Version != 7 || repository.updated.TotalAmount != 600 || repository.updated.UpdatedAt != "2026-07-12T01:02:03Z" || repository.updated.CustomerName != "Updated" {
		t.Fatalf("UpdateDraftPersistence = %+v", repository.updated)
	}
}

type writeRepository struct {
	created  order.IdempotentCreatePersistence
	updated  order.UpdateDraftPersistence
	existing *order.IdempotencyRecord
}

func (repository *writeRepository) ListOrders(context.Context, order.ListQuery) (order.Page, error) {
	return order.Page{}, nil
}
func (repository *writeRepository) GetOrder(context.Context, string) (order.Order, bool, error) {
	return order.Order{}, false, nil
}
func (repository *writeRepository) GetIdempotency(context.Context, order.IdempotencyScope) (order.IdempotencyRecord, bool, error) {
	if repository.existing == nil {
		return order.IdempotencyRecord{}, false, nil
	}
	return *repository.existing, true, nil
}
func (repository *writeRepository) CreateOrderIdempotent(_ context.Context, persistence order.IdempotentCreatePersistence) (order.IdempotencyRecord, bool, error) {
	repository.created = persistence
	return persistence.Record, true, nil
}
func (repository *writeRepository) UpdateDraft(_ context.Context, persistence order.UpdateDraftPersistence) (order.Order, error) {
	repository.updated = persistence
	return order.Order{}, nil
}
