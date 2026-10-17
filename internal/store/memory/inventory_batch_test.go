package memory

import (
	"context"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

func TestListInventoryBatchesClassifiesFixedDateAndFilters(t *testing.T) {
	s := New()
	date := func(y, m, d int) *time.Time {
		v := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
		return &v
	}
	for _, row := range []domain.InventoryBatch{
		{BatchID: "P2-6A-EXPIRED", ItemNo: "BATCH-DRUG", StorageUnitID: "pharmacy-a", ExpireDate: date(2026, 10, 14), QtyOnHandMilli: 3000},
		{BatchID: "P2-6A-EXPIRING", ItemNo: "BATCH-DRUG", StorageUnitID: "pharmacy-a", ExpireDate: date(2026, 10, 30), QtyOnHandMilli: 5000},
		{BatchID: "P2-6A-VALID", ItemNo: "BATCH-DRUG", StorageUnitID: "pharmacy-a", ExpireDate: date(2027, 1, 14), QtyOnHandMilli: 10000},
		{BatchID: "P2-6A-UNKNOWN", ItemNo: "BATCH-DRUG", StorageUnitID: "pharmacy-a", QtyOnHandMilli: 7000},
	} {
		s.AddInventoryBatch(row)
	}
	rows, err := s.ListInventoryBatches(context.Background(), port.InventoryBatchQuery{AsOfDate: time.Date(2026, 10, 15, 0, 0, 0, 0, time.FixedZone("CST", 8*3600)), ExpiryWindowDays: 30, Limit: 10})
	if err != nil || len(rows) != 4 {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
	want := []string{domain.ExpiryStatusExpired, domain.ExpiryStatusExpiring, domain.ExpiryStatusUnknown, domain.ExpiryStatusValid}
	for i, row := range rows {
		if row.ExpiryStatus != want[i] {
			t.Fatalf("row %d status=%q want %q", i, row.ExpiryStatus, want[i])
		}
	}
	filtered, err := s.ListInventoryBatches(context.Background(), port.InventoryBatchQuery{AsOfDate: time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC), ExpiryWindowDays: 30, ExpiryStatus: domain.ExpiryStatusExpiring, StorageUnitID: "pharmacy-a", Limit: 10})
	if err != nil || len(filtered) != 1 || filtered[0].BatchID != "P2-6A-EXPIRING" {
		t.Fatalf("filtered=%+v err=%v", filtered, err)
	}
}
