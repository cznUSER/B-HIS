package app

import (
	"context"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/cznUSER/B-HIS/internal/store/memory"
)

func TestCreateChargeIdempotencySurvivesServiceRestart(t *testing.T) {
	store := memory.New()
	ctx := context.Background()
	patientSvc := &Service{Users: store, Audit: store, Patients: store, Now: func() time.Time { return patientClock }}
	patient, err := patientSvc.CreatePatient(ctx, systemClaims(), validInput(), "charge-idem-patient", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	when := patientClock
	register := domain.Register{RegisterID: "charge-idem-register", RegisterNo: "R-charge-idem", PatientID: patient.PatientID, DepartmentID: "dept-1", DoctorID: "doc-1", VisitDate: when, Status: domain.RegisterStatusNormal, RegisteredAt: when, CreatedAt: when, UpdatedAt: when}
	if err := store.CreateRegister(ctx, register); err != nil {
		t.Fatal(err)
	}
	order := domain.Order{OrderID: "charge-idem-order", OrderNo: "O-charge-idem", RegisterID: register.RegisterID, PatientID: patient.PatientID, DepartmentID: "dept-1", DoctorID: "doc-1", Status: domain.OrderStatusCommit, TotalFeeCents: 100, CreatedAt: when, UpdatedAt: when, Items: []domain.OrderItem{{OrderItemID: "charge-idem-item", OrderID: "charge-idem-order", LineNo: 1, ItemNo: "TREAT", ItemType: domain.ItemTypeTreat, ItemName: "测试", AmountMilli: 1000, PriceCents: 100, TotalFeeCents: 100}}}
	if err := store.CreateOrder(ctx, order); err != nil {
		t.Fatal(err)
	}
	newService := func() *Service {
		return &Service{Users: store, Audit: store, Patients: store, Registers: store, Orders: store, Charges: store, Now: func() time.Time { return when }}
	}
	in := CreateChargeInput{OrderID: order.OrderID, IdempotencyKey: "charge-create-restart"}
	first, err := newService().CreateCharge(ctx, systemClaims(), in, "charge-req-1", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := newService().CreateCharge(ctx, systemClaims(), in, "charge-req-2", "127.0.0.1")
	if err != nil || second.ChargeID != first.ChargeID {
		t.Fatalf("replay = %#v err=%v first=%s", second, err, first.ChargeID)
	}
	rows, err := store.SearchCharges(ctx, port.ChargeQuery{OrderID: order.OrderID, Limit: 10})
	if err != nil || len(rows) != 1 {
		t.Fatalf("charge rows = %d err=%v", len(rows), err)
	}
}
