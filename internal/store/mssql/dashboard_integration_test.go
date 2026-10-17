package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/google/uuid"
)

func TestSummarizeDashboardSQLAggregatesIndependentScopesAndSnapshots(t *testing.T) {
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
		t.Fatalf("refusing to write dashboard fixture to database %q", databaseName)
	}

	ctx := context.Background()
	prefix := "D" + uuid.NewString()[:26]
	cleanupDashboardFixture(t, db, prefix)
	defer cleanupDashboardFixture(t, db, prefix)
	if err := seedDashboardFixture(ctx, db, prefix); err != nil {
		t.Fatal(err)
	}

	start := time.Date(2026, 10, 15, 0, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	end := start.AddDate(0, 0, 1)
	query := port.DashboardQuery{
		StartAt:                start,
		EndAt:                  end,
		EncounterDepartmentIDs: []string{"dept-a-" + prefix},
		ChargeDepartmentIDs:    []string{"dept-b-" + prefix},
		DispenseDepartmentIDs:  []string{"dept-b-" + prefix},
	}
	got, err := New(db).SummarizeDashboard(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	want := (port.DashboardMetrics{
		RevenueCents:         700,
		RefundCents:          0,
		NetRevenueCents:      700,
		ChargeCount:          1,
		RefundCount:          0,
		EncounterCount:       4,
		PendingChargeCount:   0,
		PendingDispenseCount: 0,
	})
	if got != want {
		t.Fatalf("SummarizeDashboard() = %+v, want %+v", got, want)
	}

	query.DispenseDepartmentIDs = []string{"dept-a-" + prefix}
	got, err = New(db).SummarizeDashboard(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	want.PendingDispenseCount = 1
	if got != want {
		t.Fatalf("SummarizeDashboard() with independent dispense scope = %+v, want %+v", got, want)
	}

	query.EncounterDepartmentIDs = nil
	query.ChargeDepartmentIDs = nil
	query.DispenseDepartmentIDs = nil
	got, err = New(db).SummarizeDashboard(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	if got != (port.DashboardMetrics{}) {
		t.Fatalf("empty scopes must return zero aggregates, got %+v", got)
	}
}

func seedDashboardFixture(ctx context.Context, db *sql.DB, prefix string) error {
	departmentA := "dept-a-" + prefix
	departmentB := "dept-b-" + prefix
	doctorA := "doctor-a-" + prefix
	doctorB := "doctor-b-" + prefix
	patient := "patient-" + prefix
	register := func(suffix string) string { return "reg-" + prefix + "-" + suffix }
	order := func(suffix string) string { return "order-" + prefix + "-" + suffix }
	charge := func(suffix string) string { return "charge-" + prefix + "-" + suffix }
	orderItem := func(suffix string) string { return "oi-" + prefix + "-" + suffix }
	chargeItem := func(suffix string) string { return "ci-" + prefix + "-" + suffix }
	dispense := func(suffix string) string { return "disp-" + prefix + "-" + suffix }
	base := time.Date(2026, 10, 14, 16, 0, 0, 0, time.UTC)
	inside := base.Add(time.Hour)
	end := base.Add(24 * time.Hour)
	paidOutsideWindow := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO dbo.patient(patient_id,hosp_pk_no,name,birthday) VALUES(@id,@hosp,N'Dashboard Test','2000-01-01')`, []any{sql.Named("id", patient), sql.Named("hosp", prefix)}},
		{`INSERT INTO dbo.department(department_id,dept_code,dept_name) VALUES(@id,@code,N'Dashboard Test A'),(@other,@other_code,N'Dashboard Test B')`, []any{sql.Named("id", departmentA), sql.Named("code", prefix+"A"), sql.Named("other", departmentB), sql.Named("other_code", prefix+"B")}},
		{`INSERT INTO dbo.doctor(doctor_id,department_id,doctor_code,doctor_name) VALUES(@id,@department,@code,N'Dashboard Test A'),(@other,@other_department,@other_code,N'Dashboard Test B')`, []any{sql.Named("id", doctorA), sql.Named("department", departmentA), sql.Named("code", prefix+"DA"), sql.Named("other", doctorB), sql.Named("other_department", departmentB), sql.Named("other_code", prefix+"DB")}},
	}
	for _, suffix := range []string{"a1", "a2", "a3", "a4", "end", "back", "deleted", "other"} {
		department, doctor := departmentA, doctorA
		status, deleted := "NORMAL", false
		registeredAt := base
		switch suffix {
		case "end":
			registeredAt = end
		case "back":
			status = "BACK"
		case "deleted":
			deleted = true
		case "other":
			department, doctor = departmentB, doctorB
		}
		statements = append(statements, struct {
			query string
			args  []any
		}{
			`INSERT INTO dbo.register_bill(register_id,register_no,patient_id,department_id,doctor_id,visit_date,status,registered_at,is_deleted) VALUES(@id,@no,@patient,@department,@doctor,'2026-10-15',@status,@registered,@deleted)`,
			[]any{sql.Named("id", register(suffix)), sql.Named("no", prefix[:18]+"R"+suffix), sql.Named("patient", patient), sql.Named("department", department), sql.Named("doctor", doctor), sql.Named("status", status), sql.Named("registered", registeredAt), sql.Named("deleted", deleted)},
		})
	}
	for _, suffix := range []string{"a1", "a2", "a4", "other"} {
		department, doctor := departmentA, doctorA
		if suffix == "other" {
			department, doctor = departmentB, doctorB
		}
		statements = append(statements, struct {
			query string
			args  []any
		}{
			`INSERT INTO dbo.order_sheet(order_id,order_no,register_id,patient_id,department_id,doctor_id) VALUES(@id,@no,@register,@patient,@department,@doctor)`,
			[]any{sql.Named("id", order(suffix)), sql.Named("no", prefix[:18]+"O"+suffix), sql.Named("register", register(suffix)), sql.Named("patient", patient), sql.Named("department", department), sql.Named("doctor", doctor)},
		})
	}
	for _, item := range []struct {
		suffix string
		order  string
		kind   string
		line   int
	}{
		{"drug-1", "a1", "DRUG", 1}, {"drug-2", "a1", "DRUG", 2}, {"treat-1", "a1", "TREAT", 3},
		{"drug-3", "a2", "DRUG", 1}, {"treat-2", "a4", "TREAT", 1},
	} {
		statements = append(statements, struct {
			query string
			args  []any
		}{
			`INSERT INTO dbo.order_item(order_item_id,order_id,line_no,item_no,item_type,item_name,units,amount,price,total_fee) VALUES(@id,@order,@line,@item,@kind,N'Dashboard Test',N'次',@amount,1,1)`,
			[]any{sql.Named("id", orderItem(item.suffix)), sql.Named("order", order(item.order)), sql.Named("line", item.line), sql.Named("item", prefix[:18]+item.suffix), sql.Named("kind", item.kind), sql.Named("amount", 1)},
		})
	}
	for _, row := range []struct {
		suffix string
		reg    string
		ord    string
		paid   any
	}{
		{"a1", "a1", "a1", paidOutsideWindow}, {"a2", "a2", "a2", inside},
		{"unpaid", "a3", "", nil}, {"treat", "a4", "a4", inside},
		{"other", "other", "other", inside}, {"end", "end", "", nil},
		{"deleted-register", "deleted", "", inside}, {"deleted-charge", "a1", "", inside},
	} {
		var paidAt any = row.paid
		deleted := row.suffix == "deleted-charge"
		var orderID any
		if row.ord != "" {
			orderID = order(row.ord)
		}
		statements = append(statements, struct {
			query string
			args  []any
		}{
			`INSERT INTO dbo.charge(charge_id,square_no,order_id,register_id,patient_id,total_fee,square_fee,favourable_fee,status,paid_at,is_deleted) VALUES(@id,@no,@order,@register,@patient,10,10,0,'NORMAL',@paid,@deleted)`,
			[]any{sql.Named("id", charge(row.suffix)), sql.Named("no", prefix[:13]+"C"+row.suffix), sql.Named("order", orderID), sql.Named("register", register(row.reg)), sql.Named("patient", patient), sql.Named("paid", paidAt), sql.Named("deleted", deleted)},
		})
	}
	for _, item := range []struct {
		suffix, charge, orderItem string
		line                      int
		qty                       string
	}{
		{"drug-1", "a1", "drug-1", 1, "2.000"}, {"drug-2", "a1", "drug-2", 2, "2.000"},
		{"treat-1", "a1", "treat-1", 3, "1.000"}, {"drug-3", "a2", "drug-3", 1, "1.000"},
		{"treat-2", "treat", "treat-2", 1, "1.000"},
	} {
		statements = append(statements, struct {
			query string
			args  []any
		}{
			`INSERT INTO dbo.charge_item(charge_item_id,charge_id,order_item_id,line_no,item_no,item_name,qty,unit_price,total_fee) VALUES(@id,@charge,@order_item,@line,@item,N'Dashboard Test',@qty,1,1)`,
			[]any{sql.Named("id", chargeItem(item.suffix)), sql.Named("charge", charge(item.charge)), sql.Named("order_item", orderItem(item.orderItem)), sql.Named("line", item.line), sql.Named("item", prefix[:18]+item.suffix), sql.Named("qty", item.qty)},
		})
	}
	for _, payment := range []struct {
		suffix, charge, direction, status string
		amount                            string
		paidAt                            time.Time
	}{
		{"charge", "a1", "CHARGE", "SUCCEEDED", "5.00", base},
		{"refund", "a1", "REFUND", "SUCCEEDED", "1.25", end.Add(-time.Second)},
		{"failed", "a1", "CHARGE", "FAILED", "0.90", inside},
		{"at-end", "a2", "CHARGE", "SUCCEEDED", "9.00", end},
		{"other", "other", "CHARGE", "SUCCEEDED", "7.00", inside},
		{"deleted-register", "deleted-register", "CHARGE", "SUCCEEDED", "3.33", inside},
		{"deleted-charge", "deleted-charge", "CHARGE", "SUCCEEDED", "4.44", inside},
	} {
		statements = append(statements, struct {
			query string
			args  []any
		}{
			`INSERT INTO dbo.payment(payment_id,payment_no,charge_id,direction,pay_type,amount,status,paid_at) VALUES(@id,@no,@charge,@direction,'CASH',@amount,@status,@paid)`,
			[]any{sql.Named("id", "payment-"+prefix+"-"+payment.suffix), sql.Named("no", prefix[:10]+"P"+payment.suffix), sql.Named("charge", charge(payment.charge)), sql.Named("direction", payment.direction), sql.Named("amount", payment.amount), sql.Named("status", payment.status), sql.Named("paid", payment.paidAt)},
		})
	}
	for _, row := range []struct {
		suffix, charge, register string
	}{
		{"a1", "a1", "a1"}, {"a2", "a2", "a2"},
	} {
		statements = append(statements, struct {
			query string
			args  []any
		}{
			`INSERT INTO dbo.dispense(dispense_id,dispense_no,order_id,charge_id,register_id,patient_id,dispensed_at) VALUES(@id,@no,@order,@charge,@register,@patient,@at)`,
			[]any{sql.Named("id", dispense(row.suffix)), sql.Named("no", prefix[:18]+"D"+row.suffix), sql.Named("order", order(row.suffix)), sql.Named("charge", charge(row.charge)), sql.Named("register", register(row.register)), sql.Named("patient", patient), sql.Named("at", inside)},
		})
	}
	for _, row := range []struct {
		suffix, dispense, orderItem, chargeItem, itemNo, charged, dispensed, returned string
	}{
		{"drug-1", "a1", "drug-1", "drug-1", "drug-1", "2.000", "2.000", "0.500"},
		{"drug-2", "a1", "drug-2", "drug-2", "drug-2", "2.000", "2.000", "0.000"},
		{"drug-3", "a2", "drug-3", "drug-3", "drug-3", "1.000", "1.000", "0.000"},
	} {
		statements = append(statements, struct {
			query string
			args  []any
		}{
			`INSERT INTO dbo.dispense_item(dispense_item_id,dispense_id,order_item_id,charge_item_id,item_no,item_name,qty_charged,qty_dispensed,qty_returned,line_status) VALUES(@id,@dispense,@order_item,@charge_item,@item,N'Dashboard Test',@charged,@dispensed,@returned,'FULL')`,
			[]any{sql.Named("id", "di-"+prefix+"-"+row.suffix), sql.Named("dispense", dispense(row.dispense)), sql.Named("order_item", orderItem(row.orderItem)), sql.Named("charge_item", chargeItem(row.chargeItem)), sql.Named("item", prefix[:18]+row.itemNo), sql.Named("charged", row.charged), sql.Named("dispensed", row.dispensed), sql.Named("returned", row.returned)},
		})
	}

	for i, statement := range statements {
		if _, err := db.ExecContext(ctx, statement.query, statement.args...); err != nil {
			return fmt.Errorf("seed dashboard fixture statement %d: %w", i+1, err)
		}
	}
	return nil
}

func cleanupDashboardFixture(t *testing.T, db *sql.DB, prefix string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, query := range []string{
		`DELETE FROM dbo.dispense_item WHERE dispense_id LIKE '%' + @prefix + '%'`,
		`DELETE FROM dbo.dispense WHERE dispense_id LIKE '%' + @prefix + '%'`,
		`DELETE FROM dbo.payment WHERE payment_id LIKE '%' + @prefix + '%'`,
		`DELETE FROM dbo.charge_item WHERE charge_item_id LIKE '%' + @prefix + '%'`,
		`DELETE FROM dbo.charge WHERE charge_id LIKE '%' + @prefix + '%'`,
		`DELETE FROM dbo.order_item WHERE order_item_id LIKE '%' + @prefix + '%'`,
		`DELETE FROM dbo.order_sheet WHERE order_id LIKE '%' + @prefix + '%'`,
		`DELETE FROM dbo.register_bill WHERE register_id LIKE '%' + @prefix + '%'`,
		`DELETE FROM dbo.doctor WHERE doctor_id LIKE '%' + @prefix + '%'`,
		`DELETE FROM dbo.department WHERE department_id LIKE '%' + @prefix + '%'`,
		`DELETE FROM dbo.patient WHERE patient_id LIKE '%' + @prefix + '%'`,
	} {
		if _, err := db.ExecContext(ctx, query, sql.Named("prefix", prefix)); err != nil {
			t.Errorf("cleanup dashboard fixture: %v", err)
			return
		}
	}
}
