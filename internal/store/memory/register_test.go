package memory

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

func TestRegisterMasterSeedMatchesSQL(t *testing.T) {
	st := New()
	raw, err := os.ReadFile("../../../migrations/sqlserver/core/003_register_seed.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sqlText := string(raw)
	depts := map[string]string{"dept-1": "内科", "dept-2": "外科"}
	for id, name := range depts {
		d, ok := st.departments[id]
		if !ok || d.DeptName != name || d.DeptCode == "" || d.Status != "ACTIVE" || d.IsDeleted {
			t.Fatalf("%s %+v", id, d)
		}
		if !strings.Contains(sqlText, id) || !strings.Contains(sqlText, name) {
			t.Fatalf("sql seed missing %s %s", id, name)
		}
	}
	docs := map[string]struct{ dept, name string }{
		"doc-1": {"dept-1", "张医生"},
		"doc-2": {"dept-2", "李医生"},
	}
	for id, want := range docs {
		d, ok := st.doctors[id]
		if !ok || d.DoctorName != want.name || d.DepartmentID != want.dept || d.Status != "ACTIVE" || d.IsDeleted {
			t.Fatalf("%s %+v", id, d)
		}
		if !strings.Contains(sqlText, id) || !strings.Contains(sqlText, want.name) {
			t.Fatalf("sql seed missing %s %s", id, want.name)
		}
	}
}

func TestCheckInRegister(t *testing.T) {
	st := New()
	ctx := context.Background()
	now := time.Date(2026, 10, 12, 4, 0, 0, 0, time.UTC)
	insert := func(id, no, status string) {
		t.Helper()
		err := st.CreateRegister(ctx, domain.Register{
			RegisterID: id, RegisterNo: no, PatientID: "p1",
			DepartmentID: "dept-1", DoctorID: "doc-1",
			VisitDate: now, Status: status, RegisteredAt: now,
			CreatedAt: now, UpdatedAt: now,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	insert("r-normal", "RN", domain.RegisterStatusNormal)
	at := now.Add(time.Minute)
	got, err := st.CheckInRegister(ctx, "r-normal", "op-1", at)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.RegisterStatusDiagnosing || got.UpdatedBy != "op-1" || !got.UpdatedAt.Equal(at) {
		t.Fatalf("%+v", got)
	}
	if got.BackOperatorID != "" || !got.BackOperatedAt.IsZero() {
		t.Fatalf("back columns %+v", got)
	}
	if _, err := st.CheckInRegister(ctx, "r-normal", "op-2", at.Add(time.Minute)); !errors.Is(err, port.ErrRegisterNotCheckIn) {
		t.Fatalf("second %v", err)
	}
	again, err := st.GetRegister(ctx, "r-normal")
	if err != nil || again.Status != domain.RegisterStatusDiagnosing || again.UpdatedBy != "op-1" {
		t.Fatalf("%v %+v", err, again)
	}

	if _, err := st.CheckInRegister(ctx, "missing", "op-1", at); !errors.Is(err, port.ErrNotFound) {
		t.Fatalf("missing %v", err)
	}

	insert("r-used", "RU", domain.RegisterStatusUsed)
	got, err = st.CheckInRegister(ctx, "r-used", "op-1", at)
	if err != nil || got.Status != domain.RegisterStatusDiagnosing {
		t.Fatalf("%v %+v", err, got)
	}

	for _, status := range []string{domain.RegisterStatusBack, domain.RegisterStatusDiagnosed, domain.RegisterStatusPaused} {
		id := "r-" + status
		insert(id, "R"+status, status)
		if _, err := st.CheckInRegister(ctx, id, "op-9", at); !errors.Is(err, port.ErrRegisterNotCheckIn) {
			t.Fatalf("%s %v", status, err)
		}
		row, err := st.GetRegister(ctx, id)
		if err != nil || row.Status != status || row.UpdatedBy != "" {
			t.Fatalf("%s changed %+v %v", status, row, err)
		}
	}
}
