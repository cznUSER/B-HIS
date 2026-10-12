package seed

import "github.com/cznUSER/B-HIS/internal/domain"

// InventoryItems matches migrations/sqlserver/core/006_inventory_seed.up.sql.
// TREAT-1 is not stocked. qty_on_hand starts at 100.000. 2026-10-12
func InventoryItems() []domain.InventoryItem {
	return []domain.InventoryItem{
		{SkuID: "sku-drug-a", ItemNo: "DRUG-A", ItemName: "演示阿莫西林胶囊", QtyOnHandMilli: domain.DefaultDrugOnHandMilli, Units: "盒", Status: domain.InventoryStatusActive},
		{SkuID: "sku-drug-b", ItemNo: "DRUG-B", ItemName: "演示布洛芬片", QtyOnHandMilli: domain.DefaultDrugOnHandMilli, Units: "盒", Status: domain.InventoryStatusActive},
	}
}
