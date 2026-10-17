package mssql

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/google/uuid"
)

func TestCreateChargeSerializesConcurrentOrderCreation(t *testing.T) {
	dsn := os.Getenv("BHIS_DASHBOARD_TEST_DSN")
	if dsn == "" {
		t.Skip("set BHIS_DASHBOARD_TEST_DSN to the isolated BHIS_PHASE2_TEST_20261015 database")
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
		t.Fatalf("refusing to write charge concurrency fixture to database %q", databaseName)
	}
	suffix := uuid.NewString()[:20]
	patientID := "cc-patient-" + suffix
	registerID := "cc-reg-" + suffix
	registerNo := "CCR" + suffix[:20]
	orderID := "cc-order-" + suffix
	orderNo := "CCO" + suffix[:20]
	orderItemID := "cc-item-" + suffix
	charge1 := "cc-charge-1-" + suffix
	charge2 := "cc-charge-2-" + suffix
	chargeNo1 := "CCS1" + suffix[:20]
	chargeNo2 := "CCS2" + suffix[:20]
	cleanup := func() {
		for _, statement := range []struct {
			query string
			args  []any
		}{
			{`DELETE FROM dbo.charge_item WHERE charge_id IN (@c1,@c2)`, []any{sql.Named("c1", charge1), sql.Named("c2", charge2)}},
			{`DELETE FROM dbo.charge WHERE charge_id IN (@c1,@c2)`, []any{sql.Named("c1", charge1), sql.Named("c2", charge2)}},
			{`DELETE FROM dbo.order_item WHERE order_item_id=@id`, []any{sql.Named("id", orderItemID)}},
			{`DELETE FROM dbo.order_sheet WHERE order_id=@id`, []any{sql.Named("id", orderID)}},
			{`DELETE FROM dbo.register_bill WHERE register_id=@id`, []any{sql.Named("id", registerID)}},
			{`DELETE FROM dbo.patient WHERE patient_id=@id`, []any{sql.Named("id", patientID)}},
		} {
			if _, err := db.ExecContext(ctx, statement.query, statement.args...); err != nil {
				t.Errorf("cleanup charge concurrency fixture: %v", err)
			}
		}
	}
	cleanup()
	defer cleanup()
	when := time.Now().UTC()
	if _, err := db.ExecContext(ctx, `
		INSERT INTO dbo.patient(patient_id,hosp_pk_no,patient_no,name,sex,birthday,id_type,id_number,card_no,created_at,updated_at)
		VALUES(@id,@hosp,@patient_no,N'并发收费测试','M','1990-01-01','OTHER','',@card,@at,@at)`,
		sql.Named("id", patientID), sql.Named("hosp", "CCH"+suffix[:20]), sql.Named("patient_no", "CCP"+suffix[:20]), sql.Named("card", "CCC"+suffix[:20]), sql.Named("at", when)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO dbo.register_bill(register_id,register_no,patient_id,department_id,doctor_id,visit_date,status,register_fee,register_charged,registered_at,created_at,updated_at)
		VALUES(@id,@no,@patient,'dept-1','doc-1',CONVERT(date,SYSUTCDATETIME()),'DIAGNOSING',0,0,@at,@at,@at)`,
		sql.Named("id", registerID), sql.Named("no", registerNo), sql.Named("patient", patientID), sql.Named("at", when)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO dbo.order_sheet(order_id,order_no,register_id,patient_id,department_id,doctor_id,status,total_fee,charged,created_at,updated_at)
		VALUES(@id,@no,@register,@patient,'dept-1','doc-1','COMMIT',1,0,@at,@at)`,
		sql.Named("id", orderID), sql.Named("no", orderNo), sql.Named("register", registerID), sql.Named("patient", patientID), sql.Named("at", when)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO dbo.order_item(order_item_id,order_id,line_no,item_no,item_type,item_name,units,amount,price,total_fee,created_at,updated_at)
		VALUES(@id,@order,'1','TREAT-1','TREAT',N'并发测试处置',N'次',1,1,1,@at,@at)`,
		sql.Named("id", orderItemID), sql.Named("order", orderID), sql.Named("at", when)); err != nil {
		t.Fatal(err)
	}
	store := New(db)
	charges := []domain.Charge{
		{ChargeID: charge1, SquareNo: chargeNo1, OrderID: orderID, RegisterID: registerID, PatientID: patientID, TotalFeeCents: 100, SquareFeeCents: 100, Status: domain.ChargeStatusNormal, CreatedAt: when, UpdatedAt: when, Items: []domain.ChargeItem{{ChargeItemID: charge1 + "-item", ChargeID: charge1, OrderItemID: orderItemID, LineNo: 1, ItemNo: "TREAT-1", ItemName: "并发测试处置", QtyMilli: 1000, UnitPriceCents: 100, TotalFeeCents: 100, CanBack: true}}},
		{ChargeID: charge2, SquareNo: chargeNo2, OrderID: orderID, RegisterID: registerID, PatientID: patientID, TotalFeeCents: 100, SquareFeeCents: 100, Status: domain.ChargeStatusNormal, CreatedAt: when, UpdatedAt: when, Items: []domain.ChargeItem{{ChargeItemID: charge2 + "-item", ChargeID: charge2, OrderItemID: orderItemID, LineNo: 1, ItemNo: "TREAT-1", ItemName: "并发测试处置", QtyMilli: 1000, UnitPriceCents: 100, TotalFeeCents: 100, CanBack: true}}},
	}
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range charges {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			errs[index] = store.CreateCharge(ctx, charges[index])
		}(i)
	}
	wg.Wait()
	successes := 0
	duplicates := 0
	for _, callErr := range errs {
		if callErr == nil {
			successes++
		} else if errors.Is(callErr, port.ErrChargeExists) {
			duplicates++
		} else {
			t.Fatalf("unexpected concurrent charge error: %v", callErr)
		}
	}
	if successes != 1 || duplicates != 1 {
		t.Fatalf("concurrent charge results = successes %d duplicates %d, want 1/1", successes, duplicates)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM dbo.charge WHERE order_id=@order_id AND is_deleted=0`, sql.Named("order_id", orderID)).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("active charges for order = %d, want 1", count)
	}
}
