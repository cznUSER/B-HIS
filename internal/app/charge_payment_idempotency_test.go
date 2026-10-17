package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/cznUSER/B-HIS/internal/store/memory"
)

func TestChargePayAndRefundIdempotencySurvivesServiceRestart(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	when := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)
	newService := func() *Service {
		return &Service{Users: store, Audit: store, Patients: store, Registers: store, Orders: store, Charges: store, Now: func() time.Time { return when }}
	}
	patient, err := newService().CreatePatient(ctx, systemClaims(), validInput(), "pay-idem-patient", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	reg := domain.Register{RegisterID: "pay-idem-register", RegisterNo: "R-pay-idem", PatientID: patient.PatientID, DepartmentID: "dept-1", DoctorID: "doc-1", VisitDate: when, Status: domain.RegisterStatusNormal, RegisteredAt: when, CreatedAt: when, UpdatedAt: when}
	if err := store.CreateRegister(ctx, reg); err != nil {
		t.Fatal(err)
	}
	order := domain.Order{OrderID: "pay-idem-order", OrderNo: "O-pay-idem", RegisterID: reg.RegisterID, PatientID: patient.PatientID, DepartmentID: "dept-1", DoctorID: "doc-1", Status: domain.OrderStatusCommit, TotalFeeCents: 100, CreatedAt: when, UpdatedAt: when, Items: []domain.OrderItem{{OrderItemID: "pay-idem-item", OrderID: "pay-idem-order", LineNo: 1, ItemNo: "TREAT", ItemType: domain.ItemTypeTreat, ItemName: "测试", AmountMilli: 1000, PriceCents: 100, TotalFeeCents: 100}}}
	if err := store.CreateOrder(ctx, order); err != nil {
		t.Fatal(err)
	}
	charge, err := newService().CreateCharge(ctx, systemClaims(), CreateChargeInput{OrderID: order.OrderID}, "pay-idem-charge", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	payment, err := newService().PayCharge(ctx, systemClaims(), charge.ChargeID, PayInput{PayType: domain.PayTypeCash, Amount: 1, IdempotencyKey: "pay-idem-key"}, "pay-1", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := newService().PayCharge(ctx, systemClaims(), charge.ChargeID, PayInput{PayType: domain.PayTypeCash, Amount: 1, IdempotencyKey: "pay-idem-key"}, "pay-2", "127.0.0.1")
	if err != nil || replayed.PaymentID != payment.PaymentID {
		t.Fatalf("pay replay=%#v err=%v first=%s", replayed, err, payment.PaymentID)
	}
	_, err = newService().PayCharge(ctx, systemClaims(), charge.ChargeID, PayInput{PayType: domain.PayTypeCash, Amount: 2, IdempotencyKey: "pay-idem-key"}, "pay-3", "127.0.0.1")
	var conflict *Error
	if !errors.As(err, &conflict) || conflict.Code != CodeIdempotencyConflict {
		t.Fatalf("pay conflict=%v", err)
	}
	refundInput := RefundInput{PayType: domain.PayTypeCash, Items: []RefundItemInput{{ChargeItemID: charge.Items[0].ChargeItemID, RefundQty: 1}}, IdempotencyKey: "refund-idem-key"}
	refund, err := newService().RefundCharge(ctx, systemClaims(), charge.ChargeID, refundInput, "refund-1", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	replayedRefund, err := newService().RefundCharge(ctx, systemClaims(), charge.ChargeID, refundInput, "refund-2", "127.0.0.1")
	if err != nil || replayedRefund.RefundPayment.PaymentID != refund.RefundPayment.PaymentID || replayedRefund.AbatedCharge.ChargeID != refund.AbatedCharge.ChargeID {
		t.Fatalf("refund replay=%#v err=%v first=%s", replayedRefund, err, refund.RefundPayment.PaymentID)
	}
	rows, err := store.SearchCharges(ctx, port.ChargeQuery{OrderID: order.OrderID, Limit: 10})
	if err != nil || len(rows) != 1 {
		t.Fatalf("charge rows=%d err=%v", len(rows), err)
	}
}
