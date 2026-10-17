package app

import (
	"context"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/cznUSER/B-HIS/internal/store/memory"
)

func TestCreateRegisterIdempotencySurvivesServiceRestart(t *testing.T) {
	store := memory.New()
	ctx := context.Background()
	firstService := &Service{Users: store, Audit: store, Patients: store, Registers: store, Now: func() time.Time { return patientClock }}
	patient, err := firstService.CreatePatient(ctx, systemClaims(), validInput(), "req-register-idem-patient", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	in := CreateRegisterInput{
		PatientID: patient.PatientID, DepartmentID: "dept-1", DoctorID: "doc-1",
		VisitDate: "2026-10-15", IdempotencyKey: "restart-register-1",
	}
	first, err := firstService.CreateRegister(ctx, systemClaims(), in, "req-register-idem-1", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}

	secondService := &Service{Users: store, Audit: store, Patients: store, Registers: store, Now: func() time.Time { return patientClock }}
	second, err := secondService.CreateRegister(ctx, systemClaims(), in, "req-register-idem-2", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if second.RegisterID != first.RegisterID {
		t.Fatalf("restart replay register = %s, want %s", second.RegisterID, first.RegisterID)
	}
	rows, err := store.SearchRegisters(ctx, port.RegisterQuery{PatientID: patient.PatientID, VisitDate: "2026-10-15", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("register rows after restart replay = %d, want 1", len(rows))
	}
}
