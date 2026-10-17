package mssql

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/google/uuid"
)

func TestCompleteRegisterSQLTransition(t *testing.T) {
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
		t.Fatalf("refusing to write register fixture to database %q", databaseName)
	}

	suffix := uuid.NewString()[:20]
	patientID := "rc-patient-" + suffix
	registerID := "rc-register-" + suffix
	registerNo := "RCR" + suffix
	cleanup := func() {
		if _, err := db.ExecContext(ctx, `DELETE FROM dbo.register_bill WHERE register_id=@register_id`, sql.Named("register_id", registerID)); err != nil {
			t.Errorf("cleanup register fixture: %v", err)
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM dbo.patient WHERE patient_id=@patient_id`, sql.Named("patient_id", patientID)); err != nil {
			t.Errorf("cleanup patient fixture: %v", err)
		}
	}
	cleanup()
	defer cleanup()

	when := time.Now().UTC()
	if _, err := db.ExecContext(ctx, `
		INSERT INTO dbo.patient(patient_id,hosp_pk_no,patient_no,name,sex,birthday,id_type,id_number,card_no,created_at,updated_at)
		VALUES(@id,@hosp,@patient_no,N'完成就诊测试','M','1990-01-01','OTHER','',@card,@at,@at)`,
		sql.Named("id", patientID), sql.Named("hosp", "RCH"+suffix), sql.Named("patient_no", "RCP"+suffix), sql.Named("card", "RCC"+suffix), sql.Named("at", when)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO dbo.register_bill(register_id,register_no,patient_id,department_id,doctor_id,visit_date,status,register_fee,register_charged,registered_at,created_at,updated_at)
		VALUES(@id,@no,@patient,'dept-1','doc-1',CONVERT(date,SYSUTCDATETIME()),'DIAGNOSING',0,0,@at,@at,@at)`,
		sql.Named("id", registerID), sql.Named("no", registerNo), sql.Named("patient", patientID), sql.Named("at", when)); err != nil {
		t.Fatal(err)
	}

	store := New(db)
	got, err := store.CompleteRegister(ctx, registerID, "integration-operator", when.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.RegisterStatusDiagnosed {
		t.Fatalf("completed row = %+v", got)
	}
	if _, err := store.CompleteRegister(ctx, registerID, "second-operator", when.Add(2*time.Minute)); !errors.Is(err, port.ErrRegisterNotComplete) {
		t.Fatalf("second completion error = %v", err)
	}
	var status, updatedBy string
	if err := db.QueryRowContext(ctx, `SELECT status,updated_by FROM dbo.register_bill WHERE register_id=@register_id`, sql.Named("register_id", registerID)).Scan(&status, &updatedBy); err != nil {
		t.Fatal(err)
	}
	if status != domain.RegisterStatusDiagnosed || updatedBy != "integration-operator" {
		t.Fatalf("second completion changed row: status=%s updated_by=%s", status, updatedBy)
	}
}
