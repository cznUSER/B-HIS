package mssql

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/google/uuid"
)

func TestListInventoryLedgerSQLBalanceFiltersAndCursor(t *testing.T) {
	dsn := os.Getenv("BHIS_DASHBOARD_TEST_DSN")
	if dsn == "" {
		t.Skip("set BHIS_DASHBOARD_TEST_DSN to the isolated BHIS_PHASE2_TEST_20261015 database")
	}

	db, err := Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var databaseName string
	if err := db.QueryRowContext(context.Background(), `SELECT DB_NAME()`).Scan(&databaseName); err != nil {
		t.Fatal(err)
	}
	if databaseName != "BHIS_PHASE2_TEST_20261015" {
		t.Fatalf("refusing to write inventory fixture to database %q", databaseName)
	}

	ctx := context.Background()
	suffix := uuid.NewString()[:26]
	skuID := "test-sku-" + suffix
	skuCode := "T" + suffix
	ledgerNew := "test-ledger-new-" + suffix
	ledgerOld := "test-ledger-old-" + suffix
	cleanup := func() {
		if _, err := db.ExecContext(ctx, `DELETE FROM dbo.inventory_ledger WHERE ledger_id IN (@new_id, @old_id)`,
			sql.Named("new_id", ledgerNew), sql.Named("old_id", ledgerOld)); err != nil {
			t.Errorf("cleanup inventory ledger fixture rows: %v", err)
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM dbo.inventory_item WHERE sku_id=@sku_id`,
			sql.Named("sku_id", skuID)); err != nil {
			t.Errorf("cleanup inventory item fixture: %v", err)
		}
	}
	cleanup()
	defer cleanup()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO dbo.inventory_item(sku_id,item_no,item_name,qty_on_hand,units)
		VALUES(@sku_id,@sku_code,N'Inventory Ledger Test',10.500,N'unit')`,
		sql.Named("sku_id", skuID), sql.Named("sku_code", skuCode)); err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 10, 15, 1, 0, 0, 0, time.UTC)
	for _, row := range []struct {
		id, kind, biz string
		qty           string
		at            time.Time
	}{
		{ledgerOld, "ADJUST", "adjust-" + suffix, "1.000", base},
		{ledgerNew, "RETURN", "return-" + suffix, "-0.500", base.Add(time.Minute)},
	} {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO dbo.inventory_ledger(ledger_id,sku_id,qty_delta,biz_type,biz_id,created_at)
			VALUES(@ledger_id,@sku_id,@qty,@biz_type,@biz_id,@created_at)`,
			sql.Named("ledger_id", row.id), sql.Named("sku_id", skuID), sql.Named("qty", row.qty), sql.Named("biz_type", row.kind), sql.Named("biz_id", row.biz), sql.Named("created_at", row.at)); err != nil {
			t.Fatal(err)
		}
	}

	store := New(db)
	query := port.InventoryLedgerQuery{SkuCode: skuCode, Limit: 1}
	items, err := store.ListInventoryLedger(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].LedgerID != ledgerNew || items[0].QtyDeltaMilli != -500 || items[0].QtyAfterMilli != 10500 {
		t.Fatalf("latest ledger page = %+v, want newest -500 with post-transaction balance 10500", items)
	}

	query.HasCursor = true
	query.CursorAt = items[0].CreatedAt
	query.CursorLedgerID = items[0].LedgerID
	items, err = store.ListInventoryLedger(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].LedgerID != ledgerOld || items[0].QtyDeltaMilli != 1000 || items[0].QtyAfterMilli != 11000 {
		t.Fatalf("next ledger page = %+v, want older +1000 with post-transaction balance 11000", items)
	}

	query.HasCursor = false
	query.BizType = "ADJUST"
	items, err = store.ListInventoryLedger(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].LedgerID != ledgerOld {
		t.Fatalf("business type filter = %+v, want only ADJUST entry", items)
	}

	query.BizType = ""
	query.HasFrom = true
	query.FromAt = base.Add(30 * time.Second)
	query.HasTo = true
	query.ToAt = base.Add(2 * time.Minute)
	items, err = store.ListInventoryLedger(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].LedgerID != ledgerNew {
		t.Fatalf("date filter = %+v, want only the entry inside the interval", items)
	}
}
