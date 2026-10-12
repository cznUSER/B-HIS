package seed

import "github.com/cznUSER/B-HIS/internal/domain"

// CatalogItems matches migrations/sqlserver/core/004_order_seed.up.sql.
// 内存库与 SQL 种子使用同一组 id / item_no / 单价。2026-10-12
func CatalogItems() []domain.CatalogItem {
	return []domain.CatalogItem{
		{CatalogItemID: "cat-drug-a", ItemNo: "DRUG-A", ItemName: "演示阿莫西林胶囊", ItemType: domain.ItemTypeDrug, Spec: "0.25g×24", Units: "盒", PriceCents: 1000, Status: "ACTIVE"},
		{CatalogItemID: "cat-drug-b", ItemNo: "DRUG-B", ItemName: "演示布洛芬片", ItemType: domain.ItemTypeDrug, Spec: "0.2g×20", Units: "盒", PriceCents: 800, Status: "ACTIVE"},
		{CatalogItemID: "cat-treat-1", ItemNo: "TREAT-1", ItemName: "演示诊查费", ItemType: domain.ItemTypeTreat, Units: "次", PriceCents: 1500, Status: "ACTIVE"},
	}
}
