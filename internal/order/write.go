package order

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/magicvr/allinme.core-api/internal/auth"
)

var integerLexeme = regexp.MustCompile(`^-?(0|[1-9][0-9]*)$`)

const (
	MaxCustomerNameBytes = 120
	MaxSKUBytes          = 64
	MaxItemNameBytes     = 160
	MaxItems             = 100
	MaxQuantity          = int64(10000)
	MaxAmount            = int64(9999999999)
)

type ItemCommand struct {
	SKU       string `json:"sku"`
	Name      string `json:"name"`
	Quantity  int64  `json:"quantity"`
	UnitPrice int64  `json:"unitPrice"`
}

type CreateCommand struct {
	CustomerName string
	Currency     string
	Items        []ItemCommand
}

type EditCommand struct {
	CustomerName string
	Currency     string
	Items        []ItemCommand
	Version      int64
}

type FieldError struct {
	Field   string
	Message string
}

type ValidationError struct {
	Details []FieldError
}

func (err *ValidationError) Error() string {
	return "order validation failed"
}

type PersistenceItem struct {
	ID        string
	Position  int64
	SKU       string
	Name      string
	Quantity  int64
	UnitPrice int64
}

type CreatePersistence struct {
	ID           string
	CustomerName string
	Currency     string
	TotalAmount  int64
	CreatedAt    string
	Items        []PersistenceItem
}

type UpdateDraftPersistence struct {
	ID           string
	CustomerName string
	Currency     string
	TotalAmount  int64
	Version      int64
	UpdatedAt    string
	Items        []PersistenceItem
}

func CanWrite(principal auth.Principal) bool {
	return auth.RoleAllowed(principal.Role, auth.RoleOperator, auth.RoleAdmin)
}

func ParseIntegerLexeme(value string) (int64, error) {
	if !integerLexeme.MatchString(value) {
		return 0, errors.New("invalid integer lexeme")
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, errors.New("integer is outside int64 range")
	}
	return parsed, nil
}

func (service *Service) Edit(ctx context.Context, principal auth.Principal, id string, command EditCommand) (Order, error) {
	if !CanWrite(principal) {
		return Order{}, ErrForbidden
	}
	if command.Version <= 0 {
		return Order{}, &ValidationError{Details: []FieldError{{Field: "version", Message: "must be greater than 0"}}}
	}
	normalized, total, err := validateFacts(command.CustomerName, command.Currency, command.Items)
	if err != nil {
		return Order{}, err
	}
	items, err := service.persistenceItems(normalized.Items)
	if err != nil {
		return Order{}, err
	}
	result, err := service.repository.UpdateDraft(ctx, UpdateDraftPersistence{
		ID: id, CustomerName: normalized.CustomerName, Currency: normalized.Currency,
		TotalAmount: total, Version: command.Version, UpdatedAt: FormatTime(UTCNow(service.clock)), Items: items,
	})
	if err != nil {
		return Order{}, fmt.Errorf("edit order: %w", err)
	}
	return result, nil
}

func (service *Service) persistenceItems(commands []ItemCommand) ([]PersistenceItem, error) {
	items := make([]PersistenceItem, 0, len(commands))
	for position, command := range commands {
		id, err := service.newItemID()
		if err != nil {
			return nil, fmt.Errorf("generate item ID: %w", err)
		}
		items = append(items, PersistenceItem{ID: id, Position: int64(position), SKU: command.SKU, Name: command.Name, Quantity: command.Quantity, UnitPrice: command.UnitPrice})
	}
	return items, nil
}

func validateFacts(customerName, currency string, items []ItemCommand) (CreateCommand, int64, error) {
	normalized := CreateCommand{CustomerName: strings.TrimSpace(customerName), Currency: currency, Items: make([]ItemCommand, len(items))}
	details := make([]FieldError, 0)
	validateString(&details, "customerName", normalized.CustomerName, MaxCustomerNameBytes)
	if currency != "CNY" {
		details = append(details, FieldError{Field: "currency", Message: "must be CNY"})
	}
	if len(items) < 1 || len(items) > MaxItems {
		details = append(details, FieldError{Field: "items", Message: "must contain between 1 and 100 items"})
	}
	var total int64
	for index, item := range items {
		normalized.Items[index] = ItemCommand{SKU: strings.TrimSpace(item.SKU), Name: strings.TrimSpace(item.Name), Quantity: item.Quantity, UnitPrice: item.UnitPrice}
		prefix := fmt.Sprintf("items[%d]", index)
		validateString(&details, prefix+".sku", normalized.Items[index].SKU, MaxSKUBytes)
		validateString(&details, prefix+".name", normalized.Items[index].Name, MaxItemNameBytes)
		if item.Quantity < 1 || item.Quantity > MaxQuantity {
			details = append(details, FieldError{Field: prefix + ".quantity", Message: "must be between 1 and 10000"})
		}
		if item.UnitPrice < 1 || item.UnitPrice > MaxAmount {
			details = append(details, FieldError{Field: prefix + ".unitPrice", Message: "must be between 1 and 9999999999"})
		}
		if item.Quantity > 0 && item.UnitPrice > 0 {
			if item.Quantity > math.MaxInt64/item.UnitPrice {
				details = append(details, FieldError{Field: "items", Message: "total amount is too large"})
				continue
			}
			line := item.Quantity * item.UnitPrice
			if total > math.MaxInt64-line {
				details = append(details, FieldError{Field: "items", Message: "total amount is too large"})
				continue
			}
			total += line
		}
	}
	if total < 1 || total > MaxAmount {
		details = append(details, FieldError{Field: "items", Message: "total amount must be between 1 and 9999999999"})
	}
	if len(details) > 0 {
		return CreateCommand{}, 0, &ValidationError{Details: details}
	}
	return normalized, total, nil
}

func validateString(details *[]FieldError, field, value string, maximum int) {
	if !utf8.ValidString(value) || len([]byte(value)) < 1 || len([]byte(value)) > maximum {
		*details = append(*details, FieldError{Field: field, Message: fmt.Sprintf("must be between 1 and %d UTF-8 bytes", maximum)})
	}
}

func ValidationDetails(err error) ([]FieldError, bool) {
	var validation *ValidationError
	if !errors.As(err, &validation) {
		return nil, false
	}
	return validation.Details, true
}
