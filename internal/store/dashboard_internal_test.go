package store

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/order"
)

func TestCheckedDashboardArithmeticAllowsNegativeNetAndRejectsOverflow(t *testing.T) {
	if value, err := checkedDashboardSubtract(0, 120000); err != nil || value != -120000 {
		t.Fatalf("negative net = %d, %v", value, err)
	}
	if _, err := checkedDashboardAdd(math.MaxInt64, 1); err == nil {
		t.Fatal("dashboard add overflow error = nil")
	}
	if _, err := checkedDashboardSubtract(math.MinInt64, 1); err == nil {
		t.Fatal("dashboard subtract underflow error = nil")
	}
}

func TestDashboardFixedSeedSummaryStatusAndTrend(t *testing.T) {
	database := openRefundCapabilityDatabase(t)
	queries := 0
	database.queryObserver = func() { queries++ }
	summary, err := database.DashboardSummary(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if summary != (order.DashboardSummary{OrderCount: 10, GrossAmount: 460000, CompletedRefundAmount: 120000, NetAmount: 340000, Currency: "CNY"}) {
		t.Fatalf("summary = %+v", summary)
	}
	if queries != 3 {
		t.Fatalf("summary queries = %d", queries)
	}
	queries = 0
	status, err := database.DashboardOrderStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	wantStatus := []order.DashboardStatusItem{
		{Status: order.StatusDraft, Count: 1}, {Status: order.StatusConfirmed, Count: 1}, {Status: order.StatusFulfilling, Count: 2},
		{Status: order.StatusShipped, Count: 2}, {Status: order.StatusCompleted, Count: 2}, {Status: order.StatusCancelled, Count: 2},
	}
	if len(status.Items) != len(wantStatus) {
		t.Fatalf("status = %+v", status)
	}
	for index := range wantStatus {
		if status.Items[index] != wantStatus[index] {
			t.Fatalf("status[%d] = %+v, want %+v", index, status.Items[index], wantStatus[index])
		}
	}
	if queries != 3 {
		t.Fatalf("status queries = %d", queries)
	}
	queries = 0
	trend, err := database.DashboardTrend(context.Background(), 7, time.Date(2026, 1, 7, 12, 0, 0, 0, time.FixedZone("CST", 8*60*60)))
	if err != nil {
		t.Fatal(err)
	}
	if trend.Days != 7 || trend.StartDate != "2026-01-01" || trend.EndDate != "2026-01-07" || len(trend.Items) != 7 {
		t.Fatalf("trend envelope = %+v", trend)
	}
	if trend.Items[0].OrderCount != 10 || trend.Items[0].GrossAmount != 460000 || trend.Items[0].CompletedRefundAmount != 0 || trend.Items[0].NetAmount != 460000 {
		t.Fatalf("trend first bucket = %+v", trend.Items[0])
	}
	if trend.Items[1].CompletedRefundAmount != 120000 || trend.Items[1].NetAmount != -120000 {
		t.Fatalf("trend second bucket = %+v", trend.Items[1])
	}
	for index := 2; index < len(trend.Items); index++ {
		if trend.Items[index].OrderCount != 0 || trend.Items[index].GrossAmount != 0 || trend.Items[index].CompletedRefundAmount != 0 || trend.Items[index].NetAmount != 0 {
			t.Fatalf("trend zero bucket[%d] = %+v", index, trend.Items[index])
		}
	}
	if queries != 3 {
		t.Fatalf("trend queries = %d", queries)
	}
	trend30, err := database.DashboardTrend(context.Background(), 30, time.Date(2026, 1, 7, 23, 59, 59, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if trend30.StartDate != "2025-12-09" || trend30.EndDate != "2026-01-07" || len(trend30.Items) != 30 {
		t.Fatalf("30 day trend = %+v", trend30)
	}
}

func TestDashboardTrendUsesUTCDateAndAllowsWindowOutsideNegativeBucket(t *testing.T) {
	database := openRefundCapabilityDatabase(t)
	if _, err := database.SQL().Exec(`
		UPDATE orders SET created_at = '2025-12-31T23:00:00Z', updated_at = '2026-01-02T00:00:00Z', payment_status = 'PARTIALLY_REFUNDED', version = 2
		WHERE id = 'ord_00000000000000000000000000000007';
		INSERT INTO refunds(id, order_id, amount, reason, status, version, requested_by, decided_by, created_at, updated_at, decided_at)
		VALUES ('rfd_ffffffffffffffffffffffffffffffff', 'ord_00000000000000000000000000000007', 10000, 'outside order', 'COMPLETED', 2, 'user-operator', 'user-approver', '2025-12-31T23:30:00Z', '2026-01-02T01:00:00Z', '2026-01-02T01:00:00Z')
	`); err != nil {
		t.Fatal(err)
	}
	trend, err := database.DashboardTrend(context.Background(), 7, time.Date(2026, 1, 1, 8, 30, 0, 0, time.FixedZone("CST", 8*60*60)))
	if err != nil {
		t.Fatal(err)
	}
	if trend.StartDate != "2025-12-26" || trend.EndDate != "2026-01-01" {
		t.Fatalf("UTC trend dates = %+v", trend)
	}
	// Move the clock to Jan 7 so the Jan 2 refund is inside the window while its order remains outside.
	trend, err = database.DashboardTrend(context.Background(), 7, time.Date(2026, 1, 7, 8, 30, 0, 0, time.FixedZone("CST", 8*60*60)))
	if err != nil {
		t.Fatal(err)
	}
	if trend.Items[1].GrossAmount != 0 || trend.Items[1].CompletedRefundAmount != 130000 || trend.Items[1].NetAmount != -130000 {
		t.Fatalf("negative outside-order bucket = %+v", trend.Items[1])
	}
}

func TestDashboardRejectsCorruptFactsAndClosedDatabase(t *testing.T) {
	database := openRefundCapabilityDatabase(t)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := database.DashboardSummary(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled dashboard error = %v", err)
	}
	if _, err := database.SQL().Exec(`UPDATE orders SET total_amount = total_amount + 1 WHERE id = 'ord_00000000000000000000000000000007'`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.DashboardSummary(context.Background()); !errors.Is(err, order.ErrInternal) {
		t.Fatalf("corrupt summary error = %v", err)
	}
	if _, err := database.DashboardOrderStatus(context.Background()); !errors.Is(err, order.ErrInternal) {
		t.Fatalf("corrupt status error = %v", err)
	}
	if _, err := database.DashboardTrend(context.Background(), 7, time.Date(2026, 1, 7, 0, 0, 0, 0, time.UTC)); !errors.Is(err, order.ErrInternal) {
		t.Fatalf("corrupt trend error = %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := database.DashboardSummary(context.Background()); !errors.Is(err, order.ErrInternal) {
		t.Fatalf("closed dashboard error = %v", err)
	}
}

func TestDashboardResponseUsesSingleSQLiteSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dashboard-snapshot.db")
	first := openRefundCapabilityDatabaseAt(t, path)
	second, err := Open(context.Background(), path, OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = second.Close() })
	var mutationErr error
	queries := 0
	first.queryObserver = func() {
		queries++
		if queries != 1 {
			return
		}
		mutationErr = second.WithTx(context.Background(), func(transaction *sql.Tx) error {
			if _, err := transaction.Exec(`UPDATE orders SET total_amount = 71000 WHERE id = 'ord_00000000000000000000000000000007'`); err != nil {
				return err
			}
			_, err := transaction.Exec(`UPDATE order_items SET unit_price = 71000 WHERE order_id = 'ord_00000000000000000000000000000007'`)
			return err
		})
	}
	summary, err := first.DashboardSummary(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if mutationErr != nil {
		t.Fatal(mutationErr)
	}
	if summary.GrossAmount != 460000 || summary.NetAmount != 340000 {
		t.Fatalf("snapshot summary = %+v", summary)
	}
	first.queryObserver = nil
	summary, err = first.DashboardSummary(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if summary.GrossAmount != 461000 || summary.NetAmount != 341000 {
		t.Fatalf("post-commit summary = %+v", summary)
	}
}
