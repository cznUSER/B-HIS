package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/store/memory"
)

// TestPayChargeZeroAmountIsInvalid covers the boundary where a charge has
// squareFee=0 (zero-price TREAT item). The contract says amount ≤ 0 is
// INVALID_AMOUNT (H08 / 接口契约 §1.3), so paying 0 must be rejected even
// though it equals squareFee. 2026-10-18
func TestPayChargeZeroAmountIsInvalid(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	when := time.Date(2026, 10, 18, 8, 0, 0, 0, time.UTC)
	svc := &Service{
		Users: store, Audit: store, Patients: store, Registers: store,
		Orders: store, Charges: store, Now: func() time.Time { return when },
	}

	patient, err := svc.CreatePatient(ctx, systemClaims(), validInput(), "zero-pay-patient", "127.0.0.1")
	if err != nil {
		t.Fatalf("create patient: %v", err)
	}
	reg := domain.Register{
		RegisterID: "zero-pay-reg", RegisterNo: "R-zero-pay", PatientID: patient.PatientID,
		DepartmentID: "dept-1", DoctorID: "doc-1", VisitDate: when,
		Status: domain.RegisterStatusNormal, RegisteredAt: when, CreatedAt: when, UpdatedAt: when,
	}
	if err := store.CreateRegister(ctx, reg); err != nil {
		t.Fatalf("create register: %v", err)
	}
	order := domain.Order{
		OrderID: "zero-pay-order", OrderNo: "O-zero-pay", RegisterID: reg.RegisterID,
		PatientID: patient.PatientID, DepartmentID: "dept-1", DoctorID: "doc-1",
		Status: domain.OrderStatusCommit, TotalFeeCents: 0,
		CreatedAt: when, UpdatedAt: when,
		Items: []domain.OrderItem{{
			OrderItemID: "zero-pay-item", OrderID: "zero-pay-order", LineNo: 1,
			ItemNo: "TREAT-1", ItemType: domain.ItemTypeTreat, ItemName: "免费治疗",
			Units: "次", AmountMilli: 1000, PriceCents: 0, TotalFeeCents: 0,
		}},
	}
	if err := store.CreateOrder(ctx, order); err != nil {
		t.Fatalf("create order: %v", err)
	}

	charge, err := svc.CreateCharge(ctx, systemClaims(), CreateChargeInput{OrderID: order.OrderID}, "zero-pay-create", "127.0.0.1")
	if err != nil {
		t.Fatalf("create charge: %v", err)
	}
	if charge.SquareFee != 0 {
		t.Fatalf("expected zero square fee, got %v", charge.SquareFee)
	}

	_, err = svc.PayCharge(ctx, systemClaims(), charge.ChargeID, PayInput{
		PayType: domain.PayTypeCash, Amount: 0,
	}, "zero-pay-pay", "127.0.0.1")
	var appErr *Error
	if err == nil || !errors.As(err, &appErr) || appErr.Code != CodeInvalidAmount {
		t.Fatalf("expected INVALID_AMOUNT, got %v", err)
	}
}
