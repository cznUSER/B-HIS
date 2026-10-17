package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/cznUSER/B-HIS/internal/store/memory"
	"github.com/cznUSER/B-HIS/internal/store/seed"
)

type batchFixtureStore struct{ rows []domain.InventoryBatch }

func (s batchFixtureStore) ListInventoryBatches(_ context.Context, q port.InventoryBatchQuery) ([]domain.InventoryBatch, error) {
	out := make([]domain.InventoryBatch, 0, len(s.rows))
	for _, row := range s.rows {
		if q.CursorBatchID != "" && row.BatchID <= q.CursorBatchID {
			continue
		}
		if q.SkuCode != "" && row.ItemNo != q.SkuCode {
			continue
		}
		if q.StorageUnitID != "" && row.StorageUnitID != q.StorageUnitID {
			continue
		}
		if q.BatchNo != "" && row.BatchNo != q.BatchNo {
			continue
		}
		if q.ExpiryStatus != "" && row.ExpiryStatus != q.ExpiryStatus {
			continue
		}
		out = append(out, row)
		if len(out) == q.Limit {
			break
		}
	}
	return out, nil
}

func TestListInventoryBatchesValidationAndClassificationContract(t *testing.T) {
	_ = memory.New() // compile-time check that the regular memory store remains independent.
	day := func(y, m, d int) *time.Time { v := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC); return &v }
	store := batchFixtureStore{rows: []domain.InventoryBatch{
		{BatchID: "P2-6A-EXPIRED", ItemNo: "P2-6A-DRUG", ExpireDate: day(2026, 10, 14), QtyOnHandMilli: 3000, ExpiryStatus: domain.ExpiryStatusExpired},
		{BatchID: "P2-6A-EXPIRING", ItemNo: "P2-6A-DRUG", ExpireDate: day(2026, 10, 30), QtyOnHandMilli: 5000, ExpiryStatus: domain.ExpiryStatusExpiring},
		{BatchID: "P2-6A-VALID", ItemNo: "P2-6A-DRUG", ExpireDate: day(2027, 1, 14), QtyOnHandMilli: 10000, ExpiryStatus: domain.ExpiryStatusValid},
		{BatchID: "P2-6A-UNKNOWN", ItemNo: "P2-6A-DRUG", QtyOnHandMilli: 7000, ExpiryStatus: domain.ExpiryStatusUnknown},
	}}
	issuer, err := auth.NewIssuer("test-secret-must-be-long-enough-32b!!", 8*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := issuer.Parse(bearerForBatchTest(t, issuer, seed.DemoUser()), time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	svc := &Service{Users: memory.New(), InventoryBatches: store, Master: memory.New()}
	// No actor lookup is possible without the same memory store, so use a seeded store that implements Users and Master.
	st := memory.New()
	svc.Users, svc.Master = st, st
	page, err := svc.ListInventoryBatches(context.Background(), claims, InventoryBatchSearch{AsOfDate: "2026-10-15", ExpiryWindowDays: 30, ExpiryWindowDaysSet: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 4 || page.Summary.QtyTotal != "25.000" || page.Summary.QtyUnknown != "7.000" {
		t.Fatalf("page=%+v", page)
	}
	if page.Items[0].ExpiryStatus != domain.ExpiryStatusExpired || page.Items[3].ExpiryStatus != domain.ExpiryStatusUnknown {
		t.Fatalf("statuses=%+v", page.Items)
	}
	for _, tc := range []struct {
		name string
		q    InventoryBatchSearch
	}{
		{"missing date", InventoryBatchSearch{ExpiryWindowDays: 30, ExpiryWindowDaysSet: true}},
		{"missing window", InventoryBatchSearch{AsOfDate: "2026-10-15"}},
		{"bad date", InventoryBatchSearch{AsOfDate: "2026-13-01", ExpiryWindowDays: 30, ExpiryWindowDaysSet: true}},
		{"negative window", InventoryBatchSearch{AsOfDate: "2026-10-15", ExpiryWindowDays: -1, ExpiryWindowDaysSet: true}},
		{"bad status", InventoryBatchSearch{AsOfDate: "2026-10-15", ExpiryWindowDays: 30, ExpiryWindowDaysSet: true, ExpiryStatus: "NOPE"}},
	} {
		_, err := svc.ListInventoryBatches(context.Background(), claims, tc.q)
		var ae *Error
		if !errors.As(err, &ae) || ae.Status != 400 {
			t.Fatalf("%s err=%v", tc.name, err)
		}
	}
}

func bearerForBatchTest(t *testing.T, issuer *auth.Issuer, user domain.User) string {
	t.Helper()
	tok, _, _, err := issuer.Issue(user, time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return tok
}
