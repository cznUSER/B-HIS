package app

import (
	"context"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/store/memory"
)

func TestRegisterFeeCashPayAndFullRefundUpdatesRegister(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	when := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	svc := &Service{Users: store, Audit: store, Registers: store, Charges: store, Now: func() time.Time { return when }}
	reg := domain.Register{
		RegisterID: "register-fee-1", RegisterNo: "R-fee-1", PatientID: "patient-fee-1",
		DepartmentID: "dept-1", DoctorID: "doc-1", VisitDate: when,
		Status: domain.RegisterStatusNormal, RegisterFeeCents: 1250,
		RegisteredAt: when, CreatedAt: when, UpdatedAt: when,
	}
	if err := store.CreateRegister(ctx, reg); err != nil {
		t.Fatal(err)
	}

	charge, err := svc.CreateRegisterFeeCharge(ctx, systemClaims(), reg.RegisterID, "register-fee-create", "req-create", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := svc.CreateRegisterFeeCharge(ctx, systemClaims(), reg.RegisterID, "register-fee-create", "req-create-replay", "127.0.0.1")
	if err != nil || replayed.ChargeID != charge.ChargeID {
		t.Fatalf("create replay=%#v err=%v", replayed, err)
	}

	if _, err := svc.PayCharge(ctx, systemClaims(), charge.ChargeID, PayInput{
		PayType: domain.PayTypeCash, Amount: 12.50, IdempotencyKey: "register-fee-pay",
	}, "req-pay", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	afterPay, err := store.GetRegister(ctx, reg.RegisterID)
	if err != nil || !afterPay.RegisterCharged {
		t.Fatalf("register after pay=%#v err=%v", afterPay, err)
	}

	refund, err := svc.RefundCharge(ctx, systemClaims(), charge.ChargeID, RefundInput{
		PayType:        domain.PayTypeCash,
		Items:          []RefundItemInput{{ChargeItemID: charge.Items[0].ChargeItemID, RefundQty: 1}},
		IdempotencyKey: "register-fee-refund",
	}, "req-refund", "127.0.0.1")
	if err != nil || refund.NewSquare != nil || refund.RefundPayment.Amount != 12.50 {
		t.Fatalf("refund=%#v err=%v", refund, err)
	}
	afterRefund, err := store.GetRegister(ctx, reg.RegisterID)
	if err != nil || afterRefund.RegisterCharged {
		t.Fatalf("register after refund=%#v err=%v", afterRefund, err)
	}
}

func TestPersistedRegisterFeeDoesNotRequireDrugStock(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	svc := &Service{Charges: store, Orders: store}
	fee := chargeFromRegisterFee(domain.Register{RegisterFeeCents: 1250}, "fee", "F1", "test", time.Now())
	// SQL charge_item has no item_type column. Its reload must still recognize
	// the dedicated registration fee when its transient classification is gone.
	fee.Items[0].ItemType = ""
	lines, err := svc.chargeStockLines(ctx, fee)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ensureStock(ctx, lines); err != nil {
		t.Fatalf("registration fee required drug stock: %v", err)
	}
	// A prescription item with the same code must retain the ordinary stock guard.
	fee.OrderID = "prescription"
	lines, err = svc.chargeStockLines(ctx, fee)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ensureStock(ctx, lines); err == nil {
		t.Fatal("order-backed charge bypassed stock validation")
	}
}

func TestRegisterFeeZeroIsNotChargeable(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	when := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	svc := &Service{Users: store, Audit: store, Registers: store, Charges: store, Now: func() time.Time { return when }}
	reg := domain.Register{RegisterID: "register-fee-zero", RegisterNo: "R-fee-zero", PatientID: "patient-fee-zero", DepartmentID: "dept-1", DoctorID: "doc-1", VisitDate: when, Status: domain.RegisterStatusNormal, RegisteredAt: when, CreatedAt: when, UpdatedAt: when}
	if err := store.CreateRegister(ctx, reg); err != nil {
		t.Fatal(err)
	}
	_, err := svc.CreateRegisterFeeCharge(ctx, systemClaims(), reg.RegisterID, "", "req", "127.0.0.1")
	appErr, ok := err.(*Error)
	if !ok || appErr.Code != CodeRegisterFeeNotChargeable {
		t.Fatalf("err=%v", err)
	}
}
