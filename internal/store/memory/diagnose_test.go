package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

func TestDiagnoseLifecycle(t *testing.T) {
	s := New()
	ctx := context.Background()
	at := time.Date(2026, 10, 15, 1, 2, 3, 0, time.UTC)
	d := domain.Diagnose{DiagnoseID: "diag-1", RegisterID: "reg-1", Status: domain.DiagnoseStatusSave, Content: "上感", CreatedAt: at, UpdatedAt: at}
	if err := s.CreateDiagnose(ctx, d); err != nil {
		t.Fatal(err)
	}
	rows, err := s.ListDiagnoses(ctx, port.DiagnoseQuery{RegisterID: "reg-1"})
	if err != nil || len(rows) != 1 || rows[0].Content != "上感" {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
	got, err := s.CommitDiagnose(ctx, d.DiagnoseID, "doctor-1", at.Add(time.Minute))
	if err != nil || got.Status != domain.DiagnoseStatusCommit || got.UpdatedBy != "doctor-1" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	if _, err := s.CommitDiagnose(ctx, d.DiagnoseID, "doctor-1", at); !errors.Is(err, port.ErrDiagnoseNotCommittable) {
		t.Fatalf("second commit err=%v", err)
	}
	got, err = s.AbateDiagnose(ctx, d.DiagnoseID, "doctor-1", at.Add(2*time.Minute))
	if err != nil || got.Status != domain.DiagnoseStatusAbate {
		t.Fatalf("abate=%+v err=%v", got, err)
	}
	if _, err := s.AbateDiagnose(ctx, d.DiagnoseID, "doctor-1", at); !errors.Is(err, port.ErrDiagnoseNotAbatable) {
		t.Fatalf("second abate err=%v", err)
	}
}
