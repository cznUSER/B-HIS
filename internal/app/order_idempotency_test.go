package app

import (
	"context"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/cznUSER/B-HIS/internal/store/memory"
)

func TestCreateOrderIdempotencySurvivesServiceRestart(t *testing.T) {
	store := memory.New()
	ctx := context.Background()
	now := time.Date(2026, 10, 15, 4, 0, 0, 0, time.UTC)
	firstService := &Service{Users: store, Audit: store, Patients: store, Registers: store, Orders: store, Now: func() time.Time { return now }}
	patient, err := firstService.CreatePatient(ctx, systemClaims(), validInput(), "req-order-idem-patient", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	reg, err := firstService.CreateRegister(ctx, systemClaims(), CreateRegisterInput{
		PatientID: patient.PatientID, DepartmentID: "dept-1", DoctorID: "doc-1", VisitDate: "2026-10-15",
	}, "req-order-idem-register", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	in := CreateOrderInput{RegisterID: reg.RegisterID, IdempotencyKey: "restart-order-1"}
	first, err := firstService.CreateOrder(ctx, systemClaims(), in, "req-order-idem-1", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	secondService := &Service{Users: store, Audit: store, Patients: store, Registers: store, Orders: store, Now: func() time.Time { return now }}
	second, err := secondService.CreateOrder(ctx, systemClaims(), in, "req-order-idem-2", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if second.OrderID != first.OrderID || second.OrderNo != first.OrderNo {
		t.Fatalf("restart replay order = %#v, want order %s", second, first.OrderID)
	}
	rows, err := store.SearchOrders(ctx, port.OrderQuery{RegisterID: reg.RegisterID, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("order rows after restart replay = %d, want 1", len(rows))
	}
}
