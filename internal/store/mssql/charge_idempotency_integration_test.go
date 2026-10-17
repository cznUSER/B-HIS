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

func TestChargeCreateIdempotencySQL(t *testing.T) {
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
		t.Fatalf("refusing charge idempotency fixture in %q", databaseName)
	}
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:20]
	patientID, registerID, orderID := "idem-charge-patient-"+suffix, "idem-charge-register-"+suffix, "idem-charge-order-"+suffix
	orderItemID, key := "idem-charge-item-"+suffix, "sql-charge-idem-"+suffix
	chargeID := "idem-charge-" + suffix
	cleanup := func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.payment WHERE charge_id=@id`, sql.Named("id", chargeID))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.idempotency_record WHERE scope IN ('charge.pay','charge.refund') AND (idem_key LIKE @prefix + '%' OR idem_key LIKE @refundPrefix + '%')`, sql.Named("prefix", key), sql.Named("refundPrefix", key+"-refund"))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.idempotency_record WHERE scope=@scope AND idem_key LIKE @prefix + '%'`, sql.Named("scope", "charge.create"), sql.Named("prefix", key))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.charge_item WHERE charge_id=@id`, sql.Named("id", chargeID))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.charge WHERE charge_id=@id`, sql.Named("id", chargeID))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.order_item WHERE order_id=@id`, sql.Named("id", orderID))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.order_sheet WHERE order_id=@id`, sql.Named("id", orderID))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.register_bill WHERE register_id=@id`, sql.Named("id", registerID))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.patient WHERE patient_id=@id`, sql.Named("id", patientID))
	}
	cleanup()
	defer cleanup()
	when := time.Now().UTC()
	if _, err := db.ExecContext(ctx, `INSERT INTO dbo.patient(patient_id,hosp_pk_no,patient_no,name,sex,birthday,id_type,id_number,card_status,status,created_at,updated_at) VALUES(@id,@hosp,@pno,N'收费幂等测试','M','1990-01-01','OTHER',@num,'NOT_SENT','ACTIVE',@at,@at)`, sql.Named("id", patientID), sql.Named("hosp", "ICH"+suffix), sql.Named("pno", "ICP"+suffix), sql.Named("num", "ICN"+suffix), sql.Named("at", when)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO dbo.register_bill(register_id,register_no,patient_id,department_id,doctor_id,register_sort,visit_date,status,register_fee,register_charged,registered_at,created_at,created_by,updated_at,updated_by,is_deleted) VALUES(@id,@no,@patient,'dept-1','doc-1','NORMAL',@visit,'NORMAL',0,0,@visit,@visit,'test',@visit,'test',0)`, sql.Named("id", registerID), sql.Named("no", "ICR"+suffix), sql.Named("patient", patientID), sql.Named("visit", when)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO dbo.order_sheet(order_id,order_no,register_id,patient_id,department_id,doctor_id,recipe_sort,status,total_fee,charged,created_at,created_by,updated_at,updated_by,is_deleted) VALUES(@id,@no,@register,@patient,'dept-1','doc-1','NORMAL','COMMIT',1,0,@at,'test',@at,'test',0)`, sql.Named("id", orderID), sql.Named("no", "ICO"+suffix), sql.Named("register", registerID), sql.Named("patient", patientID), sql.Named("at", when)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO dbo.order_item(order_item_id,order_id,line_no,item_no,item_type,item_name,units,amount,price,total_fee) VALUES(@id,@order,1,'TREAT','TREAT',N'测试','次',1,1,1)`, sql.Named("id", orderItemID), sql.Named("order", orderID)); err != nil {
		t.Fatal(err)
	}
	c := domain.Charge{ChargeID: chargeID, SquareNo: "ICS" + suffix, OrderID: orderID, RegisterID: registerID, PatientID: patientID, TotalFeeCents: 100, SquareFeeCents: 100, Status: domain.ChargeStatusNormal, CreatedAt: when, UpdatedAt: when, CreatedBy: "test", UpdatedBy: "test", Items: []domain.ChargeItem{{ChargeItemID: "ici-" + suffix, ChargeID: chargeID, OrderItemID: orderItemID, LineNo: 1, ItemNo: "TREAT", ItemName: "测试", QtyMilli: 1000, UnitPriceCents: 100, TotalFeeCents: 100, CanBack: true}}}
	store := New(db)
	hash := strings.Repeat("a", 64)
	first, replayed, err := store.CreateChargeWithIdempotency(ctx, c, "charge.create", key, hash, when.Add(time.Hour))
	if err != nil || replayed || first.ChargeID != chargeID || len(first.Items) != 1 {
		t.Fatalf("first=%#v replayed=%v err=%v", first, replayed, err)
	}
	second, replayed, err := New(db).CreateChargeWithIdempotency(ctx, c, "charge.create", key, hash, when.Add(time.Hour))
	if err != nil || !replayed || second.ChargeID != chargeID || len(second.Items) != 1 {
		t.Fatalf("replay=%#v replayed=%v err=%v", second, replayed, err)
	}
	var wg sync.WaitGroup
	concurrentErrs := make([]error, 2)
	for i := range concurrentErrs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, replayed, err := New(db).CreateChargeWithIdempotency(ctx, c, "charge.create", key, hash, when.Add(time.Hour))
			if err == nil && !replayed {
				concurrentErrs[i] = errors.New("concurrent replay was not marked replayed")
				return
			}
			concurrentErrs[i] = err
		}(i)
	}
	wg.Wait()
	for _, err := range concurrentErrs {
		if err != nil {
			t.Fatalf("concurrent replay=%v", err)
		}
	}
	_, _, err = store.CreateChargeWithIdempotency(ctx, c, "charge.create", key, strings.Repeat("b", 64), when.Add(time.Hour))
	if !errors.Is(err, port.ErrIdempotencyConflict) {
		t.Fatalf("conflict=%v", err)
	}
	rollbackKey := key + "-rollback"
	_, _, err = store.CreateChargeWithIdempotency(ctx, c, "charge.create", rollbackKey, strings.Repeat("c", 64), when.Add(time.Hour))
	if !errors.Is(err, port.ErrChargeExists) {
		t.Fatalf("duplicate order=%v", err)
	}
	var idemCount, chargeCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.idempotency_record WHERE scope='charge.create' AND idem_key=@key`, sql.Named("key", rollbackKey)).Scan(&idemCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.charge WHERE charge_id=@id`, sql.Named("id", chargeID)).Scan(&chargeCount); err != nil {
		t.Fatal(err)
	}
	if idemCount != 0 || chargeCount != 1 {
		t.Fatalf("rollback idem=%d charge=%d", idemCount, chargeCount)
	}
	pay := domain.Payment{PaymentID: "pay-" + suffix, PaymentNo: "PN" + suffix, ChargeID: chargeID, Direction: domain.PaymentDirectionCharge, PayType: domain.PayTypeCash, AmountCents: 100, Status: domain.PaymentStatusSucceeded, PaidAt: when, CreatedAt: when, UpdatedAt: when}
	paid, savedPay, replayed, err := store.PayChargeWithIdempotency(ctx, chargeID, pay, "charge.pay", key, strings.Repeat("p", 64), when.Add(time.Hour))
	if err != nil || replayed || paid == nil || savedPay == nil {
		t.Fatalf("pay first paid=%#v payment=%#v replayed=%v err=%v", paid, savedPay, replayed, err)
	}
	paidReplay, savedPayReplay, replayed, err := New(db).PayChargeWithIdempotency(ctx, chargeID, pay, "charge.pay", key, strings.Repeat("p", 64), when.Add(time.Hour))
	if err != nil || !replayed || paidReplay.ChargeID != chargeID || savedPayReplay.PaymentID != pay.PaymentID {
		t.Fatalf("pay replay paid=%#v payment=%#v replayed=%v err=%v", paidReplay, savedPayReplay, replayed, err)
	}
	_, _, _, err = store.PayChargeWithIdempotency(ctx, chargeID, pay, "charge.pay", key, strings.Repeat("q", 64), when.Add(time.Hour))
	if !errors.Is(err, port.ErrIdempotencyConflict) {
		t.Fatalf("pay conflict=%v", err)
	}
	refund := domain.Payment{PaymentID: "refund-" + suffix, PaymentNo: "RN" + suffix, ChargeID: chargeID, Direction: domain.PaymentDirectionRefund, PayType: domain.PayTypeCash, AmountCents: 100, Status: domain.PaymentStatusSucceeded, PaidAt: when, CreatedAt: when, UpdatedAt: when}
	old, err := New(db).GetCharge(ctx, chargeID)
	if err != nil {
		t.Fatal(err)
	}
	old.Items[0].RefundedQtyMilli = old.Items[0].QtyMilli
	abated, next, savedRefund, replayed, err := store.RefundChargeWithIdempotency(ctx, *old, nil, refund, "charge.refund", key+"-refund", strings.Repeat("r", 64), when.Add(time.Hour))
	if err != nil || replayed || abated == nil || next != nil || savedRefund == nil {
		t.Fatalf("refund first old=%#v next=%#v payment=%#v replayed=%v err=%v", abated, next, savedRefund, replayed, err)
	}
	replayOld, replayNext, replayRefund, replayed, err := New(db).RefundChargeWithIdempotency(ctx, *old, nil, refund, "charge.refund", key+"-refund", strings.Repeat("r", 64), when.Add(time.Hour))
	if err != nil || !replayed || replayOld.ChargeID != chargeID || replayNext != nil || replayRefund.PaymentID != refund.PaymentID {
		t.Fatalf("refund replay old=%#v next=%#v payment=%#v replayed=%v err=%v", replayOld, replayNext, replayRefund, replayed, err)
	}
	_, _, _, _, err = store.RefundChargeWithIdempotency(ctx, *old, nil, refund, "charge.refund", key+"-refund", strings.Repeat("s", 64), when.Add(time.Hour))
	if !errors.Is(err, port.ErrIdempotencyConflict) {
		t.Fatalf("refund conflict=%v", err)
	}
}
