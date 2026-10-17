package mssql

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/google/uuid"
)

func TestDispenseIdempotencySQL(t *testing.T) {
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
		t.Fatalf("refusing dispense idempotency fixture in %q", databaseName)
	}
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:20]
	patientID, registerID, orderID := "idem-dispense-patient-"+suffix, "idem-dispense-register-"+suffix, "idem-dispense-order-"+suffix
	orderItemID, chargeID, chargeItemID := "idem-dispense-order-item-"+suffix, "idem-dispense-charge-"+suffix, "idem-dispense-charge-item-"+suffix
	skuID := "idem-dispense-sku-" + suffix
	dispenseID, dispenseItemID := "idem-dispense-"+suffix, "idem-dispense-item-"+suffix
	dispenseID2, dispenseItemID2 := "idem-dispense-concurrent-"+suffix, "idem-dispense-concurrent-item-"+suffix
	key := "sql-dispense-idem-" + suffix
	itemNo := "D" + suffix
	cleanup := func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.idempotency_record WHERE scope IN ('dispense.create','dispense.return') AND idem_key LIKE @prefix + '%'`, sql.Named("prefix", key))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.inventory_ledger WHERE sku_id=@id`, sql.Named("id", skuID))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.dispense_item WHERE dispense_id=@id`, sql.Named("id", dispenseID2))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.dispense WHERE dispense_id=@id`, sql.Named("id", dispenseID2))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.dispense_item WHERE dispense_id=@id`, sql.Named("id", dispenseID))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.dispense WHERE dispense_id=@id`, sql.Named("id", dispenseID))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.charge_item WHERE charge_id=@id`, sql.Named("id", chargeID))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.charge WHERE charge_id=@id`, sql.Named("id", chargeID))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.order_item WHERE order_id=@id`, sql.Named("id", orderID))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.order_sheet WHERE order_id=@id`, sql.Named("id", orderID))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.register_bill WHERE register_id=@id`, sql.Named("id", registerID))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.patient WHERE patient_id=@id`, sql.Named("id", patientID))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.inventory_item WHERE sku_id=@id`, sql.Named("id", skuID))
	}
	cleanup()
	defer cleanup()
	when := time.Now().UTC()
	if _, err := db.ExecContext(ctx, `INSERT INTO dbo.patient(patient_id,hosp_pk_no,patient_no,name,sex,birthday,id_type,id_number,card_status,status,created_at,updated_at) VALUES(@id,@hosp,@pno,N'发药幂等测试','M','1990-01-01','OTHER',@num,'NOT_SENT','ACTIVE',@at,@at)`, sql.Named("id", patientID), sql.Named("hosp", "IDH"+suffix), sql.Named("pno", "IDP"+suffix), sql.Named("num", "IDN"+suffix), sql.Named("at", when)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO dbo.register_bill(register_id,register_no,patient_id,department_id,doctor_id,register_sort,visit_date,status,register_fee,register_charged,registered_at,created_at,created_by,updated_at,updated_by,is_deleted) VALUES(@id,@no,@patient,'dept-1','doc-1','NORMAL',@visit,'NORMAL',0,0,@visit,@visit,'test',@visit,'test',0)`, sql.Named("id", registerID), sql.Named("no", "IDR"+suffix), sql.Named("patient", patientID), sql.Named("visit", when)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO dbo.order_sheet(order_id,order_no,register_id,patient_id,department_id,doctor_id,recipe_sort,status,total_fee,charged,created_at,created_by,updated_at,updated_by,is_deleted) VALUES(@id,@no,@register,@patient,'dept-1','doc-1','NORMAL','COMMIT',1,0,@at,'test',@at,'test',0)`, sql.Named("id", orderID), sql.Named("no", "IDO"+suffix), sql.Named("register", registerID), sql.Named("patient", patientID), sql.Named("at", when)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO dbo.order_item(order_item_id,order_id,line_no,item_no,item_type,item_name,units,amount,price,total_fee) VALUES(@id,@order,1,@item_no,'DRUG',N'测试药品',N'盒',1,1,1)`, sql.Named("id", orderItemID), sql.Named("order", orderID), sql.Named("item_no", itemNo)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO dbo.charge(charge_id,square_no,order_id,register_id,patient_id,total_fee,square_fee,favourable_fee,status,paid_at,created_at,created_by,updated_at,updated_by,is_deleted) VALUES(@id,@no,@order,@register,@patient,1,1,0,'NORMAL',@at,@at,'test',@at,'test',0)`, sql.Named("id", chargeID), sql.Named("no", "IDS"+suffix), sql.Named("order", orderID), sql.Named("register", registerID), sql.Named("patient", patientID), sql.Named("at", when)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO dbo.charge_item(charge_item_id,charge_id,order_item_id,line_no,item_no,item_name,qty,unit_price,total_fee,can_back,refunded_qty) VALUES(@id,@charge,@order_item,1,@item_no,N'测试药品',1,1,1,1,0)`, sql.Named("id", chargeItemID), sql.Named("charge", chargeID), sql.Named("order_item", orderItemID), sql.Named("item_no", itemNo)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO dbo.inventory_item(sku_id,item_no,item_name,qty_on_hand,units,status,created_at,created_by,updated_at,updated_by,is_deleted) VALUES(@id,@item_no,N'测试药品',5,N'盒','ACTIVE',@at,'test',@at,'test',0)`, sql.Named("id", skuID), sql.Named("item_no", itemNo), sql.Named("at", when)); err != nil {
		t.Fatal(err)
	}
	d := domain.Dispense{DispenseID: dispenseID, DispenseNo: "IDD" + suffix, OrderID: orderID, ChargeID: chargeID, RegisterID: registerID, PatientID: patientID, DispensedAt: when, UpdatedAt: when, CreatedBy: "test", UpdatedBy: "test", Items: []domain.DispenseItem{{DispenseItemID: dispenseItemID, ChargeItemID: chargeItemID, OrderItemID: orderItemID, ItemNo: itemNo, QtyDispensedMilli: 1000}}}
	expires := time.Now().UTC().Add(time.Hour)
	first, replayed, err := New(db).CreateDispenseWithIdempotency(ctx, d, "dispense.create", key, strings.Repeat("a", 64), expires)
	if err != nil || replayed || first.DispenseID != dispenseID {
		t.Fatalf("first=%#v replay=%v err=%v", first, replayed, err)
	}
	second, replayed, err := New(db).CreateDispenseWithIdempotency(ctx, d, "dispense.create", key, strings.Repeat("a", 64), expires)
	if err != nil || !replayed || second.DispenseID != dispenseID {
		t.Fatalf("replay=%#v replay=%v err=%v", second, replayed, err)
	}
	var qty string
	if err := db.QueryRowContext(ctx, `SELECT CONVERT(varchar(40),qty_on_hand) FROM dbo.inventory_item WHERE sku_id=@id`, sql.Named("id", skuID)).Scan(&qty); err != nil {
		t.Fatal(err)
	}
	if qty != "4.000" {
		t.Fatalf("stock after replay=%s", qty)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.inventory_ledger WHERE biz_id=@id`, sql.Named("id", dispenseID)).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("ledger after replay=%d", count)
	}
	if _, _, err := New(db).CreateDispenseWithIdempotency(ctx, d, "dispense.create", key, strings.Repeat("b", 64), expires); !errors.Is(err, port.ErrIdempotencyConflict) {
		t.Fatalf("conflict=%v", err)
	}
	returned, replayed, err := New(db).ReturnDispenseWithIdempotency(ctx, dispenseID, "患者要求", "test", when, []port.ReturnLine{{DispenseItemID: dispenseItemID, ReturnMilli: 1000}}, "dispense.return", key+"-return", strings.Repeat("c", 64), expires)
	if err != nil || replayed || returned.Status != domain.DispenseStatusReturned {
		t.Fatalf("return=%#v replay=%v err=%v", returned, replayed, err)
	}
	_, replayed, err = New(db).ReturnDispenseWithIdempotency(ctx, dispenseID, "患者要求", "test", when, []port.ReturnLine{{DispenseItemID: dispenseItemID, ReturnMilli: 1000}}, "dispense.return", key+"-return", strings.Repeat("c", 64), expires)
	if err != nil || !replayed {
		t.Fatalf("return replay=%v err=%v", replayed, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT CONVERT(varchar(40),qty_on_hand) FROM dbo.inventory_item WHERE sku_id=@id`, sql.Named("id", skuID)).Scan(&qty); err != nil {
		t.Fatal(err)
	}
	if qty != "5.000" {
		t.Fatalf("stock after return replay=%s", qty)
	}
	d2 := d
	d2.DispenseID = dispenseID2
	d2.DispenseNo = "IDC" + suffix
	d2.Items = []domain.DispenseItem{{DispenseItemID: dispenseItemID2, ChargeItemID: chargeItemID, OrderItemID: orderItemID, ItemNo: itemNo, QtyDispensedMilli: 1000}}
	var wg sync.WaitGroup
	results := make([]error, 2)
	concurrentKey := key + "-concurrent"
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _, results[i] = New(db).CreateDispenseWithIdempotency(ctx, d2, "dispense.create", concurrentKey, strings.Repeat("d", 64), expires)
		}(i)
	}
	wg.Wait()
	for _, e := range results {
		if e != nil && !errors.Is(e, port.ErrAlreadyDispensed) {
			t.Fatalf("concurrent error=%v", e)
		}
	}
}
