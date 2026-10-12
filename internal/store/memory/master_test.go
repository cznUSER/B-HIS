package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

func TestAdjustLedgerAndCatalogStock(t *testing.T) {
	ctx := context.Background()
	s := New()
	now := time.Date(2026, 10, 13, 1, 0, 0, 0, time.UTC)

	got, err := s.AdjustInventory(ctx, "sku-drug-a", -500, "盘亏", "system", "1", "led-1", now)
	if err != nil {
		t.Fatal(err)
	}
	if got.QtyOnHandMilli != domain.DefaultDrugOnHandMilli-500 || got.RowVersion != "2" {
		t.Fatalf("qty/version %+v", got)
	}
	adjusts := 0
	for _, row := range s.ledgers {
		if row.BizType == "INIT" {
			t.Fatal("ledger must not use INIT")
		}
		if row.BizType == domain.LedgerBizAdjust && row.Remark == "盘亏" && row.QtyDeltaMilli == -500 {
			adjusts++
		}
	}
	if adjusts != 1 {
		t.Fatalf("adjust rows %d", adjusts)
	}
	if _, err := s.AdjustInventory(ctx, "sku-drug-a", -1_000_000_000, "超额", "system", "", "led-2", now); !errors.Is(err, port.ErrAdjustNegative) {
		t.Fatalf("negative: %v", err)
	}
	if _, err := s.AdjustInventory(ctx, "no-such-sku", 1, "期初", "system", "", "led-3", now); !errors.Is(err, port.ErrNotDrugSKU) {
		t.Fatalf("missing: %v", err)
	}
	if _, err := s.AdjustInventory(ctx, "sku-drug-a", 1, "过期版本", "system", "1", "led-4", now); !errors.Is(err, port.ErrConflict) {
		t.Fatalf("conflict: %v", err)
	}

	drug, err := s.CreateCatalog(ctx, domain.CatalogItem{
		CatalogItemID: "cat-drug-c", ItemNo: "DRUG-C", ItemName: "丙", ItemType: domain.ItemTypeDrug,
		Units: "盒", PriceCents: 100, Status: domain.StatusActive, CreatedAt: now, UpdatedAt: now,
	}, "sku-drug-c")
	if err != nil || drug.ItemNo != "DRUG-C" {
		t.Fatalf("drug %v %+v", err, drug)
	}
	on, ok := s.inventory["DRUG-C"]
	if !ok || on.QtyOnHandMilli != 0 || on.SkuID != "sku-drug-c" {
		t.Fatalf("drug stock %+v %v", on, ok)
	}
	if _, err := s.CreateCatalog(ctx, domain.CatalogItem{
		CatalogItemID: "cat-treat-9", ItemNo: "TREAT-9", ItemName: "治疗", ItemType: domain.ItemTypeTreat,
		Units: "次", PriceCents: 200, Status: domain.StatusActive,
	}, "ignored"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.inventory["TREAT-9"]; ok {
		t.Fatal("TREAT must not create inventory")
	}
}
