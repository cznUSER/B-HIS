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

func TestRegisterCreateIdempotencySQL(t *testing.T) {
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
		t.Fatalf("refusing register idempotency fixture in %q", databaseName)
	}

	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:20]
	patientID := "idem-register-patient-" + suffix
	registerID := "idem-register-" + suffix
	registerNo := "RIR" + suffix
	key := "sql-register-idem-" + suffix
	cleanup := func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.idempotency_record WHERE scope=@scope AND idem_key=@key`, sql.Named("scope", "register.create"), sql.Named("key", key))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.register_bill WHERE register_id=@id`, sql.Named("id", registerID))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.patient WHERE patient_id=@id`, sql.Named("id", patientID))
	}
	cleanup()
	defer cleanup()

	when := time.Now().UTC()
	if _, err := db.ExecContext(ctx, `
		INSERT INTO dbo.patient(patient_id,hosp_pk_no,patient_no,name,sex,birthday,id_type,id_number,card_no,card_status,status,created_at,updated_at)
		VALUES(@id,@hosp,@patient_no,N'挂号幂等测试','M','1990-01-01','OTHER',@id_number,@card,'NOT_SENT','ACTIVE',@at,@at)`,
		sql.Named("id", patientID), sql.Named("hosp", "IRH"+suffix), sql.Named("patient_no", "IRP"+suffix),
		sql.Named("id_number", "IRN"+suffix), sql.Named("card", "IRC"+suffix), sql.Named("at", when)); err != nil {
		t.Fatal(err)
	}

	r := domain.Register{
		RegisterID: registerID, RegisterNo: registerNo, PatientID: patientID,
		DepartmentID: "dept-1", DoctorID: "doc-1", VisitDate: when,
		Status: domain.RegisterStatusNormal, RegisteredAt: when, CreatedAt: when, UpdatedAt: when,
	}
	hash := strings.Repeat("a", 64)
	store := New(db)
	first, replayed, err := store.CreateRegisterWithIdempotency(ctx, r, "register.create", key, hash, when.Add(time.Hour))
	if err != nil || replayed || first.RegisterID != r.RegisterID {
		t.Fatalf("first create = %#v, replayed=%v, err=%v", first, replayed, err)
	}
	second, replayed, err := New(db).CreateRegisterWithIdempotency(ctx, r, "register.create", key, hash, when.Add(time.Hour))
	if err != nil || !replayed || second.RegisterID != r.RegisterID {
		t.Fatalf("restart replay = %#v, replayed=%v, err=%v", second, replayed, err)
	}
	changed := r
	changed.DoctorID = "doc-2"
	_, _, err = New(db).CreateRegisterWithIdempotency(ctx, changed, "register.create", key, strings.Repeat("b", 64), when.Add(time.Hour))
	if !errors.Is(err, port.ErrIdempotencyConflict) {
		t.Fatalf("changed payload error = %v, want %v", err, port.ErrIdempotencyConflict)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.register_bill WHERE register_id=@id`, sql.Named("id", registerID)).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("register rows after replay = %d, want 1", count)
	}
}

func TestRegisterCreateIdempotencySQLConcurrent(t *testing.T) {
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
		t.Fatalf("refusing register idempotency fixture in %q", databaseName)
	}
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:20]
	patientID := "idem-register-concurrent-patient-" + suffix
	key := "sql-register-concurrent-" + suffix
	cleanup := func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.idempotency_record WHERE scope=@scope AND idem_key=@key`, sql.Named("scope", "register.create"), sql.Named("key", key))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.register_bill WHERE register_id LIKE @prefix`, sql.Named("prefix", "idem-register-concurrent-"+suffix+"-%"))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.patient WHERE patient_id=@id`, sql.Named("id", patientID))
	}
	cleanup()
	defer cleanup()
	when := time.Now().UTC()
	if _, err := db.ExecContext(ctx, `
		INSERT INTO dbo.patient(patient_id,hosp_pk_no,patient_no,name,sex,birthday,id_type,id_number,card_no,card_status,status,created_at,updated_at)
		VALUES(@id,@hosp,@patient_no,N'挂号幂等并发测试','M','1990-01-01','OTHER',@id_number,@card,'NOT_SENT','ACTIVE',@at,@at)`,
		sql.Named("id", patientID), sql.Named("hosp", "IRCH"+suffix), sql.Named("patient_no", "IRCP"+suffix),
		sql.Named("id_number", "IRCN"+suffix), sql.Named("card", "IRCC"+suffix), sql.Named("at", when)); err != nil {
		t.Fatal(err)
	}
	hash := strings.Repeat("c", 64)
	registers := []domain.Register{
		{RegisterID: "idem-register-concurrent-" + suffix + "-a", RegisterNo: "RICA" + suffix, PatientID: patientID, DepartmentID: "dept-1", DoctorID: "doc-1", VisitDate: when, Status: domain.RegisterStatusNormal, RegisteredAt: when, CreatedAt: when, UpdatedAt: when},
		{RegisterID: "idem-register-concurrent-" + suffix + "-b", RegisterNo: "RICB" + suffix, PatientID: patientID, DepartmentID: "dept-1", DoctorID: "doc-1", VisitDate: when, Status: domain.RegisterStatusNormal, RegisteredAt: when, CreatedAt: when, UpdatedAt: when},
	}
	type result struct {
		reg      domain.Register
		replayed bool
		err      error
	}
	results := make([]result, 2)
	var wg sync.WaitGroup
	for i := range registers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i].reg, results[i].replayed, results[i].err = New(db).CreateRegisterWithIdempotency(ctx, registers[i], "register.create", key, hash, when.Add(time.Hour))
		}(i)
	}
	wg.Wait()
	for i, result := range results {
		if result.err != nil {
			t.Fatalf("concurrent result %d error = %v", i, result.err)
		}
	}
	if results[0].replayed == results[1].replayed {
		t.Fatalf("concurrent replay flags = %v, %v; want one create and one replay", results[0].replayed, results[1].replayed)
	}
	if results[0].reg.RegisterID != results[1].reg.RegisterID {
		t.Fatalf("concurrent register IDs = %s, %s", results[0].reg.RegisterID, results[1].reg.RegisterID)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.idempotency_record WHERE scope=@scope AND idem_key=@key`, sql.Named("scope", "register.create"), sql.Named("key", key)).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("idempotency rows = %d, want 1", count)
	}
}
