package mssql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/app"
	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/cznUSER/B-HIS/internal/store/seed"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TestRegisterLifecycleSQLMatrix verifies the SQL store's register state
// transitions without involving the HTTP or in-memory implementations. It is
// intentionally guarded to the disposable phase-two SQL database.
func TestRegisterLifecycleSQLMatrix(t *testing.T) {
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
		t.Fatalf("refusing to write register lifecycle fixture to database %q", databaseName)
	}

	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:20]
	patientID := "lc-patient-" + suffix
	registerPrefix := "lc-register-" + suffix + "-"
	// register_no is varchar(32); keep the generated fixture number below that
	// limit even when the descriptive lifecycle case name is long.
	registerNoPrefix := "LCR" + suffix
	registerNoSequence := 0
	cleanup := func() {
		if _, err := db.ExecContext(ctx, `DELETE FROM dbo.register_bill WHERE register_id LIKE @prefix`, sql.Named("prefix", registerPrefix+"%")); err != nil {
			t.Errorf("cleanup register lifecycle fixtures: %v", err)
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM dbo.patient WHERE patient_id=@patient_id`, sql.Named("patient_id", patientID)); err != nil {
			t.Errorf("cleanup lifecycle patient: %v", err)
		}
	}
	cleanup()
	defer cleanup()

	when := time.Date(2026, 10, 15, 2, 0, 0, 0, time.UTC)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO dbo.patient(patient_id,hosp_pk_no,patient_no,name,sex,birthday,id_type,id_number,card_no,card_status,status,created_at,updated_at)
		VALUES(@id,@hosp,@patient_no,N'挂号生命周期测试','M','1990-01-01','OTHER','',@card,'NOT_SENT','ACTIVE',@at,@at)`,
		sql.Named("id", patientID), sql.Named("hosp", "LCH"+suffix), sql.Named("patient_no", "LCP"+suffix),
		sql.Named("card", "LCC"+suffix), sql.Named("at", when)); err != nil {
		t.Fatal(err)
	}

	store := New(db)
	insert := func(name, status string) string {
		registerID := registerPrefix + name
		registerNoSequence++
		r := domain.Register{
			RegisterID: registerID, RegisterNo: fmt.Sprintf("%s%02d", registerNoPrefix, registerNoSequence), PatientID: patientID,
			DepartmentID: "dept-1", DoctorID: "doc-1", RegisterSort: "sort-normal",
			VisitDate: when, Status: status, RegisteredAt: when, CreatedAt: when, UpdatedAt: when,
		}
		if err := insertRegister(ctx, db, r); err != nil {
			t.Fatalf("insert %s register: %v", status, err)
		}
		return registerID
	}
	assertUnchanged := func(registerID, want string) {
		got, err := store.GetRegister(ctx, registerID)
		if err != nil {
			t.Fatalf("read %s: %v", registerID, err)
		}
		if got.Status != want {
			t.Fatalf("register %s changed to %s, want %s", registerID, got.Status, want)
		}
	}

	cancelNormal := insert("cancel-normal", domain.RegisterStatusNormal)
	got, err := store.CancelRegister(ctx, cancelNormal, "lifecycle-operator", when.Add(time.Minute))
	if err != nil || got.Status != domain.RegisterStatusBack || got.BackOperatorID != "lifecycle-operator" {
		t.Fatalf("NORMAL cancel = %+v, err=%v", got, err)
	}
	if _, err := store.CancelRegister(ctx, cancelNormal, "second-operator", when.Add(2*time.Minute)); !errors.Is(err, port.ErrRegisterAlreadyBack) {
		t.Fatalf("second NORMAL cancel error = %v, want ErrRegisterAlreadyBack", err)
	}
	assertUnchanged(cancelNormal, domain.RegisterStatusBack)

	for _, status := range []string{domain.RegisterStatusUsed, domain.RegisterStatusDiagnosing, domain.RegisterStatusDiagnosed, domain.RegisterStatusPaused} {
		registerID := insert("cancel-"+strings.ToLower(status), status)
		if _, err := store.CancelRegister(ctx, registerID, "lifecycle-operator", when.Add(time.Minute)); !errors.Is(err, port.ErrRegisterNotCancellable) {
			t.Fatalf("cancel %s error = %v, want ErrRegisterNotCancellable", status, err)
		}
		assertUnchanged(registerID, status)
	}

	for _, status := range []string{domain.RegisterStatusNormal, domain.RegisterStatusUsed} {
		registerID := insert("check-in-"+strings.ToLower(status), status)
		got, err := store.CheckInRegister(ctx, registerID, "lifecycle-operator", when.Add(time.Minute))
		if err != nil || got.Status != domain.RegisterStatusDiagnosing {
			t.Fatalf("check-in %s = %+v, err=%v", status, got, err)
		}
	}
	for _, status := range []string{domain.RegisterStatusBack, domain.RegisterStatusDiagnosing, domain.RegisterStatusDiagnosed, domain.RegisterStatusPaused} {
		registerID := insert("check-in-"+strings.ToLower(status), status)
		if _, err := store.CheckInRegister(ctx, registerID, "lifecycle-operator", when.Add(time.Minute)); !errors.Is(err, port.ErrRegisterNotCheckIn) {
			t.Fatalf("check-in %s error = %v, want ErrRegisterNotCheckIn", status, err)
		}
		assertUnchanged(registerID, status)
	}

	completeDiagnosing := insert("complete-diagnosing", domain.RegisterStatusDiagnosing)
	got, err = store.CompleteRegister(ctx, completeDiagnosing, "lifecycle-operator", when.Add(time.Minute))
	if err != nil || got.Status != domain.RegisterStatusDiagnosed {
		t.Fatalf("DIAGNOSING complete = %+v, err=%v", got, err)
	}
	var updatedBy string
	if err := db.QueryRowContext(ctx, `SELECT updated_by FROM dbo.register_bill WHERE register_id=@register_id`, sql.Named("register_id", completeDiagnosing)).Scan(&updatedBy); err != nil {
		t.Fatal(err)
	}
	if updatedBy != "lifecycle-operator" {
		t.Fatalf("completed register updated_by=%q, want lifecycle-operator", updatedBy)
	}
	if _, err := store.CompleteRegister(ctx, completeDiagnosing, "second-operator", when.Add(2*time.Minute)); !errors.Is(err, port.ErrRegisterNotComplete) {
		t.Fatalf("second complete error = %v, want ErrRegisterNotComplete", err)
	}
	assertUnchanged(completeDiagnosing, domain.RegisterStatusDiagnosed)

	for _, status := range []string{domain.RegisterStatusNormal, domain.RegisterStatusUsed, domain.RegisterStatusBack, domain.RegisterStatusDiagnosed, domain.RegisterStatusPaused} {
		registerID := insert("complete-"+strings.ToLower(status), status)
		if _, err := store.CompleteRegister(ctx, registerID, "lifecycle-operator", when.Add(time.Minute)); !errors.Is(err, port.ErrRegisterNotComplete) {
			t.Fatalf("complete %s error = %v, want ErrRegisterNotComplete", status, err)
		}
		assertUnchanged(registerID, status)
	}
}

func TestRegisterExpirySQLBoundary(t *testing.T) {
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
		t.Fatalf("refusing to write register expiry fixture to database %q", databaseName)
	}

	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:20]
	patientID := "expiry-patient-" + suffix
	registerPrefix := "expiry-register-" + suffix + "-"
	expiredRequestID := "expiry-boundary-" + suffix
	openRequestID := "expiry-disabled-" + suffix
	cleanup := func() {
		if _, err := db.ExecContext(ctx, `DELETE FROM dbo.audit_log WHERE request_id IN (@expired_request, @open_request)`, sql.Named("expired_request", expiredRequestID), sql.Named("open_request", openRequestID)); err != nil {
			t.Errorf("cleanup register expiry audit rows: %v", err)
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM dbo.register_bill WHERE register_id LIKE @prefix`, sql.Named("prefix", registerPrefix+"%")); err != nil {
			t.Errorf("cleanup register expiry fixtures: %v", err)
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM dbo.patient WHERE patient_id=@patient_id`, sql.Named("patient_id", patientID)); err != nil {
			t.Errorf("cleanup expiry patient: %v", err)
		}
	}
	cleanup()
	defer cleanup()

	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO dbo.patient(patient_id,hosp_pk_no,patient_no,name,sex,birthday,id_type,id_number,card_no,card_status,status,created_at,updated_at)
		VALUES(@id,@hosp,@patient_no,N'挂号有效期测试','M','1990-01-01','OTHER','',@card,'NOT_SENT','ACTIVE',@at,@at)`,
		sql.Named("id", patientID), sql.Named("hosp", "EXH"+suffix), sql.Named("patient_no", "EXP"+suffix),
		sql.Named("card", "EXC"+suffix), sql.Named("at", now)); err != nil {
		t.Fatal(err)
	}

	store := New(db)
	insert := func(name string, registeredAt time.Time) string {
		registerID := registerPrefix + name
		r := domain.Register{
			RegisterID: registerID, RegisterNo: "EXP" + suffix + name, PatientID: patientID,
			DepartmentID: "dept-1", DoctorID: "doc-1", VisitDate: now,
			Status: domain.RegisterStatusNormal, RegisteredAt: registeredAt, CreatedAt: registeredAt, UpdatedAt: registeredAt,
		}
		if err := insertRegister(ctx, db, r); err != nil {
			t.Fatalf("insert expiry register: %v", err)
		}
		return registerID
	}
	claims := &auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: seed.DemoUserID},
		Username:         seed.DemoUsername,
		Roles:            []string{string(domain.RoleSystem)},
	}

	expiredID := insert("old", now.Add(-2*time.Hour))
	svc := &app.Service{Users: store, Registers: store, Audit: store, RegisterExpireHours: 2, Now: func() time.Time { return now }}
	_, err = svc.CheckInRegister(ctx, claims, expiredID, expiredRequestID, "127.0.0.1")
	var appErr *app.Error
	if !errors.As(err, &appErr) || appErr.Code != app.CodeRegisterExpired || appErr.Status != 409 {
		t.Fatalf("boundary check-in error = %v, want REGISTER_EXPIRED/409", err)
	}
	got, err := store.GetRegister(ctx, expiredID)
	if err != nil || got.Status != domain.RegisterStatusNormal {
		t.Fatalf("expired register after rejection = %+v, err=%v", got, err)
	}

	openID := insert("disabled", now.Add(-24*time.Hour))
	noExpiry := &app.Service{Users: store, Registers: store, Audit: store, Now: func() time.Time { return now }}
	dto, err := noExpiry.CheckInRegister(ctx, claims, openID, openRequestID, "127.0.0.1")
	if err != nil || dto.Status != domain.RegisterStatusDiagnosing {
		t.Fatalf("disabled expiry check-in = %+v, err=%v", dto, err)
	}
}
