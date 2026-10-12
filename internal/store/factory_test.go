package store

import (
	"context"
	"testing"

	"github.com/cznUSER/B-HIS/internal/config"
	"github.com/cznUSER/B-HIS/internal/store/seed"
)

func TestOpenMemory(t *testing.T) {
	users, _, _, patients, registers, orders, closer, err := Open(config.Config{Database: config.DatabaseConfig{Driver: "memory"}})
	if err != nil {
		t.Fatal(err)
	}
	defer closer.Close()
	u, err := users.FindByUsername(nil, seed.DemoUsername)
	if err != nil || u.Username != "system" || u.DisplayName != seed.DemoDisplayName {
		t.Fatalf("%v %+v", err, u)
	}
	if patients == nil || registers == nil || orders == nil {
		t.Fatal("nil patient, register, or order store")
	}
	ctx := context.Background()
	for _, id := range []string{"dept-1", "dept-2"} {
		ok, err := registers.DepartmentExists(ctx, id)
		if err != nil || !ok {
			t.Fatalf("department %s: %v %v", id, ok, err)
		}
	}
	for _, id := range []string{"doc-1", "doc-2"} {
		ok, err := registers.DoctorExists(ctx, id)
		if err != nil || !ok {
			t.Fatalf("doctor %s: %v %v", id, ok, err)
		}
	}
	for _, no := range []string{"DRUG-A", "DRUG-B", "TREAT-1"} {
		item, err := orders.CatalogByItemNo(ctx, no)
		if err != nil || item.ItemNo != no || item.PriceCents <= 0 {
			t.Fatalf("catalog %s: %v %+v", no, err, item)
		}
	}
	drug, err := orders.CatalogByItemNo(ctx, "DRUG-A")
	if err != nil || drug.CatalogItemID != "cat-drug-a" || drug.PriceCents != 1000 || drug.ItemName != "演示阿莫西林胶囊" {
		t.Fatalf("DRUG-A seed %+v %v", drug, err)
	}
}

func TestOpenSQLServerRequiresDSN(t *testing.T) {
	_, _, _, _, _, _, _, err := Open(config.Config{Database: config.DatabaseConfig{Driver: "sqlserver"}})
	if err == nil {
		t.Fatal("expected error")
	}
}
