package store

import (
	"math"
	"testing"
)

func TestCheckedRefundSumRejectsOverflow(t *testing.T) {
	if got, err := checkedRefundSum(math.MaxInt64-5, 5); err != nil || got != math.MaxInt64 {
		t.Fatalf("boundary sum = %d, %v", got, err)
	}
	if _, err := checkedRefundSum(math.MaxInt64-5, 6); err == nil {
		t.Fatal("overflow sum error = nil")
	}
	if _, err := checkedRefundSum(0, -1); err == nil {
		t.Fatal("negative sum error = nil")
	}
}
