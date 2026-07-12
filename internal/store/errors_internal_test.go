package store

import (
	"context"
	"errors"
	"testing"

	"github.com/magicvr/allinme.core-api/internal/order"
)

func TestClassifyOrderErrorUsesSQLiteCodes(t *testing.T) {
	for _, code := range []int{5, 6, 261, 262, 517, 518, 773} {
		if !unavailableSQLiteCode(code) {
			t.Fatalf("code %d not unavailable", code)
		}
	}
	if unavailableSQLiteCode(19) {
		t.Fatal("constraint classified unavailable")
	}
	for _, err := range []error{context.Canceled, context.DeadlineExceeded, order.ErrNotFound} {
		if classified := classifyOrderError(err); !errors.Is(classified, err) || errors.Is(classified, order.ErrUnavailable) {
			t.Fatalf("classified %v as %v", err, classified)
		}
	}
}
