package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/store/memory"
)

func TestDiagnoseWritesHonorRegisterExpiry(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	store := memory.New()
	svc := &Service{
		Users: store, Audit: store, Registers: store, Diagnoses: store,
		RegisterExpireHours: 2, Now: func() time.Time { return now },
	}
	reg := domain.Register{
		RegisterID: "diagnose-expired", RegisterNo: "R-DIAG-EXPIRED", PatientID: "patient-1",
		DepartmentID: "dept-1", DoctorID: "doc-1", VisitDate: now,
		Status: domain.RegisterStatusNormal, RegisteredAt: now.Add(-3 * time.Hour), CreatedAt: now, UpdatedAt: now,
	}
	if err := store.CreateRegister(ctx, reg); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateDiagnose(ctx, systemClaims(), reg.RegisterID, CreateDiagnoseInput{Content: "过期诊断"}, "req-diag-create-expired", "127.0.0.1"); !isRegisterExpiredError(err) {
		t.Fatalf("create expired diagnosis error=%v", err)
	}

	saved := domain.Diagnose{DiagnoseID: "diagnose-expired-save", RegisterID: reg.RegisterID, Status: domain.DiagnoseStatusSave, Content: "已暂存", CreatedAt: now, UpdatedAt: now}
	if err := store.CreateDiagnose(ctx, saved); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CommitDiagnose(ctx, systemClaims(), saved.DiagnoseID, "req-diag-commit-expired", "127.0.0.1"); !isRegisterExpiredError(err) {
		t.Fatalf("commit expired diagnosis error=%v", err)
	}
	got, err := store.GetDiagnose(ctx, saved.DiagnoseID)
	if err != nil || got.Status != domain.DiagnoseStatusSave {
		t.Fatalf("expired commit changed diagnosis: got=%+v err=%v", got, err)
	}
}

func TestDiagnoseAbateHonorsRegisterExpiry(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	store := memory.New()
	svc := &Service{
		Users: store, Audit: store, Registers: store, Diagnoses: store,
		RegisterExpireHours: 2, Now: func() time.Time { return now },
	}
	reg := domain.Register{
		RegisterID: "diagnose-expired-abate", RegisterNo: "R-DIAG-ABATE", PatientID: "patient-1",
		DepartmentID: "dept-1", DoctorID: "doc-1", VisitDate: now,
		Status: domain.RegisterStatusNormal, RegisteredAt: now.Add(-3 * time.Hour), CreatedAt: now, UpdatedAt: now,
	}
	if err := store.CreateRegister(ctx, reg); err != nil {
		t.Fatal(err)
	}
	diagnose := domain.Diagnose{DiagnoseID: "diagnose-expired-commit", RegisterID: reg.RegisterID, Status: domain.DiagnoseStatusCommit, Content: "已提交", CreatedAt: now, UpdatedAt: now}
	if err := store.CreateDiagnose(ctx, diagnose); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AbateDiagnose(ctx, systemClaims(), diagnose.DiagnoseID, "req-diag-abate-expired", "127.0.0.1"); !isRegisterExpiredError(err) {
		t.Fatalf("abate expired diagnosis error=%v", err)
	}
	got, err := store.GetDiagnose(ctx, diagnose.DiagnoseID)
	if err != nil || got.Status != domain.DiagnoseStatusCommit {
		t.Fatalf("expired abate changed diagnosis: got=%+v err=%v", got, err)
	}
}

func isRegisterExpiredError(err error) bool {
	var appErr *Error
	return errors.As(err, &appErr) && appErr.Status == 409 && appErr.Code == CodeRegisterExpired
}
