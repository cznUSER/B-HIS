package mssql

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/port"
)

func TestListInventoryBatchesSQLFixedDateAndIsolation(t *testing.T) {
	dsn := os.Getenv("BHIS_DASHBOARD_TEST_DSN")
	if dsn == "" {
		t.Skip("set BHIS_DASHBOARD_TEST_DSN to isolated BHIS_PHASE2_TEST_20261015")
	}
	db, err := Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	var databaseName string
	if err := db.QueryRowContext(ctx, `SELECT DB_NAME()`).Scan(&databaseName); err != nil {
		t.Fatal(err)
	}
	if databaseName != "BHIS_PHASE2_TEST_20261015" {
		t.Fatalf("refusing batch fixture in %q", databaseName)
	}

	const prefix = "P2-6A-SQL-"
	cleanup := func() {
		if _, err := db.ExecContext(ctx, `DELETE FROM dbo.inventory_batch WHERE batch_id LIKE @prefix`, sql.Named("prefix", prefix+"%")); err != nil {
			t.Errorf("cleanup batch fixture: %v", err)
		}
	}
	cleanup()
	defer cleanup()
	rows := []struct {
		id, batch, expire string
		qty               string
	}{
		{prefix + "EXPIRED", "B-EXPIRED", "2026-10-14", "3.000"},
		{prefix + "EXPIRING-ASOF", "B-EXPIRING-ASOF", "2026-10-15", "1.000"},
		{prefix + "EXPIRING-END", "B-EXPIRING-END", "2026-11-14", "4.000"},
		{prefix + "VALID", "B-VALID", "2026-11-15", "10.000"},
		{prefix + "UNKNOWN", "B-UNKNOWN", "", "7.000"},
	}
	for i, row := range rows {
		var expire any
		if row.expire != "" {
			expire = row.expire
		}
		if _, err := db.ExecContext(ctx, `
			INSERT INTO dbo.inventory_batch(
				batch_id,sku_id,item_no,item_name,storage_unit_id,batch_no,expire_date,qty_on_hand,units,source_ref,
				normalized_storage_unit_id,normalized_batch_no,normalized_source_ref,row_version)
			VALUES(@id,@sku,@item,N'批次测试药品',N'fixture-pharmacy',@batch,@expire,@qty,N'盒',@source,N'fixture-pharmacy',@batch,@source,@version)`,
			sql.Named("id", row.id), sql.Named("sku", prefix+"SKU"), sql.Named("item", prefix+"DRUG"), sql.Named("batch", row.batch), sql.Named("expire", expire), sql.Named("qty", row.qty), sql.Named("source", prefix+"SRC"+string(rune('A'+i))), sql.Named("version", "fixture-1")); err != nil {
			t.Fatal(err)
		}
	}
	store := New(db)
	items, err := store.ListInventoryBatches(ctx, port.InventoryBatchQuery{SkuCode: prefix + "DRUG", AsOfDate: time.Date(2026, 10, 15, 0, 0, 0, 0, time.FixedZone("CST", 8*3600)), ExpiryWindowDays: 30, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 5 {
		t.Fatalf("items=%+v", items)
	}
	statuses := map[string]string{}
	for _, item := range items {
		statuses[item.BatchID] = item.ExpiryStatus
	}
	want := map[string]string{prefix + "EXPIRED": "EXPIRED", prefix + "EXPIRING-ASOF": "EXPIRING", prefix + "EXPIRING-END": "EXPIRING", prefix + "VALID": "VALID", prefix + "UNKNOWN": "UNKNOWN"}
	for id, status := range want {
		if statuses[id] != status {
			t.Fatalf("%s status=%q want %q", id, statuses[id], status)
		}
	}
	filtered, err := store.ListInventoryBatches(ctx, port.InventoryBatchQuery{SkuCode: prefix + "DRUG", ExpiryStatus: "EXPIRING", AsOfDate: time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC), ExpiryWindowDays: 30, Limit: 10})
	if err != nil || len(filtered) != 2 {
		t.Fatalf("filtered=%+v err=%v", filtered, err)
	}
}
