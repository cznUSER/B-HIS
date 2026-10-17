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

func TestOrderCreateIdempotencySQL(t *testing.T) {
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
		t.Fatalf("refusing order idempotency fixture in %q", databaseName)
	}

	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:20]
	patientID := "idem-order-patient-" + suffix
	registerID := "idem-order-register-" + suffix
	orderID := "idem-order-" + suffix
	orderNo := "OIO" + suffix
	key := "sql-order-idem-" + suffix
	cleanup := func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.idempotency_record WHERE scope=@scope AND idem_key=@key`, sql.Named("scope", "order.create"), sql.Named("key", key))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.order_item WHERE order_id=@id`, sql.Named("id", orderID))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.order_sheet WHERE order_id=@id`, sql.Named("id", orderID))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.register_bill WHERE register_id=@id`, sql.Named("id", registerID))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.patient WHERE patient_id=@id`, sql.Named("id", patientID))
	}
	cleanup()
	defer cleanup()
	when := time.Now().UTC()
	if _, err := db.ExecContext(ctx, `
		INSERT INTO dbo.patient(patient_id,hosp_pk_no,patient_no,name,sex,birthday,id_type,id_number,card_no,card_status,status,created_at,updated_at)
		VALUES(@id,@hosp,@patient_no,N'医嘱幂等测试','M','1990-01-01','OTHER',@id_number,@card,'NOT_SENT','ACTIVE',@at,@at)`,
		sql.Named("id", patientID), sql.Named("hosp", "IOH"+suffix), sql.Named("patient_no", "IOP"+suffix),
		sql.Named("id_number", "ION"+suffix), sql.Named("card", "IOC"+suffix), sql.Named("at", when)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO dbo.register_bill(register_id,register_no,patient_id,department_id,doctor_id,register_sort,visit_date,status,register_fee,register_charged,registered_at,created_at,created_by,updated_at,updated_by,is_deleted)
		VALUES(@id,@no,@patient,@dept,@doctor,'NORMAL',@visit,'NORMAL',0,0,@visit,@visit,'test',@visit,'test',0)`,
		sql.Named("id", registerID), sql.Named("no", "RIO"+suffix), sql.Named("patient", patientID), sql.Named("dept", "dept-1"), sql.Named("doctor", "doc-1"), sql.Named("visit", when)); err != nil {
		t.Fatal(err)
	}
	o := domain.Order{
		OrderID: orderID, OrderNo: orderNo, RegisterID: registerID, PatientID: patientID,
		DepartmentID: "dept-1", DoctorID: "doc-1", RecipeSort: "NORMAL", Status: domain.OrderStatusSave,
		CreatedAt: when, CreatedBy: "test", UpdatedAt: when, UpdatedBy: "test",
		Items: []domain.OrderItem{{OrderItemID: "idem-item-" + suffix, OrderID: orderID, LineNo: 1, ItemNo: "TREAT", ItemType: "TREAT", ItemName: "测试", Units: "次", AmountMilli: 1000, PriceCents: 100, TotalFeeCents: 100}},
	}
	hash := strings.Repeat("a", 64)
	first, replayed, err := New(db).CreateOrderWithIdempotency(ctx, o, "order.create", key, hash, when.Add(time.Hour))
	if err != nil || replayed || first.OrderID != o.OrderID || len(first.Items) != 1 {
		t.Fatalf("first create = %#v, replayed=%v, err=%v", first, replayed, err)
	}
	second, replayed, err := New(db).CreateOrderWithIdempotency(ctx, o, "order.create", key, hash, when.Add(time.Hour))
	if err != nil || !replayed || second.OrderID != o.OrderID || len(second.Items) != 1 {
		t.Fatalf("restart replay = %#v, replayed=%v, err=%v", second, replayed, err)
	}
	changed := o
	changed.DoctorID = "doc-2"
	_, _, err = New(db).CreateOrderWithIdempotency(ctx, changed, "order.create", key, strings.Repeat("b", 64), when.Add(time.Hour))
	if !errors.Is(err, port.ErrIdempotencyConflict) {
		t.Fatalf("changed payload error = %v, want %v", err, port.ErrIdempotencyConflict)
	}
	var orders, items int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.order_sheet WHERE order_id=@id`, sql.Named("id", orderID)).Scan(&orders); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.order_item WHERE order_id=@id`, sql.Named("id", orderID)).Scan(&items); err != nil {
		t.Fatal(err)
	}
	if orders != 1 || items != 1 {
		t.Fatalf("rows after replay = orders %d, items %d; want 1/1", orders, items)
	}

	concurrentKey := key + "-concurrent"
	concurrentIDs := []string{orderID + "-a", orderID + "-b"}
	defer func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.idempotency_record WHERE scope=@scope AND idem_key=@key`, sql.Named("scope", "order.create"), sql.Named("key", concurrentKey))
		for _, id := range concurrentIDs {
			_, _ = db.ExecContext(ctx, `DELETE FROM dbo.order_item WHERE order_id=@id`, sql.Named("id", id))
			_, _ = db.ExecContext(ctx, `DELETE FROM dbo.order_sheet WHERE order_id=@id`, sql.Named("id", id))
		}
	}()
	concurrentOrders := []domain.Order{
		{
			OrderID: concurrentIDs[0], OrderNo: orderNo + "A", RegisterID: registerID, PatientID: patientID,
			DepartmentID: "dept-1", DoctorID: "doc-1", RecipeSort: "NORMAL", Status: domain.OrderStatusSave,
			CreatedAt: when, CreatedBy: "test", UpdatedAt: when, UpdatedBy: "test",
			Items: []domain.OrderItem{{OrderItemID: "idem-item-" + suffix + "-a", OrderID: concurrentIDs[0], LineNo: 1, ItemNo: "TREAT", ItemType: "TREAT", ItemName: "测试", Units: "次", AmountMilli: 1000, PriceCents: 100, TotalFeeCents: 100}},
		},
		{
			OrderID: concurrentIDs[1], OrderNo: orderNo + "B", RegisterID: registerID, PatientID: patientID,
			DepartmentID: "dept-1", DoctorID: "doc-1", RecipeSort: "NORMAL", Status: domain.OrderStatusSave,
			CreatedAt: when, CreatedBy: "test", UpdatedAt: when, UpdatedBy: "test",
			Items: []domain.OrderItem{{OrderItemID: "idem-item-" + suffix + "-b", OrderID: concurrentIDs[1], LineNo: 1, ItemNo: "TREAT", ItemType: "TREAT", ItemName: "测试", Units: "次", AmountMilli: 1000, PriceCents: 100, TotalFeeCents: 100}},
		},
	}
	type result struct {
		order    domain.Order
		replayed bool
		err      error
	}
	results := make([]result, len(concurrentOrders))
	var wg sync.WaitGroup
	for i := range concurrentOrders {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i].order, results[i].replayed, results[i].err = New(db).CreateOrderWithIdempotency(ctx, concurrentOrders[i], "order.create", concurrentKey, strings.Repeat("c", 64), when.Add(time.Hour))
		}(i)
	}
	wg.Wait()
	for i, result := range results {
		if result.err != nil {
			t.Fatalf("concurrent result %d error = %v", i, result.err)
		}
	}
	if results[0].replayed == results[1].replayed || results[0].order.OrderID != results[1].order.OrderID {
		t.Fatalf("concurrent results = %#v, %#v; want one create and one replay of the same order", results[0], results[1])
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.idempotency_record WHERE scope=@scope AND idem_key=@key`, sql.Named("scope", "order.create"), sql.Named("key", concurrentKey)).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("concurrent idempotency rows = %d, want 1", count)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.order_sheet WHERE order_id IN (@id1,@id2)`, sql.Named("id1", concurrentIDs[0]), sql.Named("id2", concurrentIDs[1])).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("concurrent order rows = %d, want 1", count)
	}

	rollbackKey := key + "-rollback"
	rollbackID := orderID + "-rollback"
	defer func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.idempotency_record WHERE scope=@scope AND idem_key=@key`, sql.Named("scope", "order.create"), sql.Named("key", rollbackKey))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.order_item WHERE order_id=@id`, sql.Named("id", rollbackID))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.order_sheet WHERE order_id=@id`, sql.Named("id", rollbackID))
	}()
	rollbackOrder := concurrentOrders[0]
	rollbackOrder.OrderID = rollbackID
	rollbackOrder.OrderNo = orderNo + "R"
	rollbackOrder.Items = []domain.OrderItem{
		{OrderItemID: "idem-item-rollback-" + suffix, OrderID: rollbackID, LineNo: 1, ItemNo: "TREAT", ItemType: "TREAT", ItemName: "测试", Units: "次", AmountMilli: 1000, PriceCents: 100, TotalFeeCents: 100},
		{OrderItemID: "idem-item-rollback-" + suffix, OrderID: rollbackID, LineNo: 2, ItemNo: "TREAT", ItemType: "TREAT", ItemName: "测试", Units: "次", AmountMilli: 1000, PriceCents: 100, TotalFeeCents: 100},
	}
	if _, _, err := New(db).CreateOrderWithIdempotency(ctx, rollbackOrder, "order.create", rollbackKey, strings.Repeat("d", 64), when.Add(time.Hour)); err == nil {
		t.Fatal("duplicate order item unexpectedly succeeded")
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.order_sheet WHERE order_id=@id`, sql.Named("id", rollbackID)).Scan(&orders); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.order_item WHERE order_id=@id`, sql.Named("id", rollbackID)).Scan(&items); err != nil {
		t.Fatal(err)
	}
	if orders != 0 || items != 0 {
		t.Fatalf("failed order rows = orders %d, items %d; want 0/0", orders, items)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.idempotency_record WHERE scope=@scope AND idem_key=@key`, sql.Named("scope", "order.create"), sql.Named("key", rollbackKey)).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("failed idempotency rows = %d, want 0", count)
	}
}
