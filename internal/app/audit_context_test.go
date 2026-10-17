package app

import (
	"context"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/buildinfo"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/store/memory"
	"github.com/cznUSER/B-HIS/internal/store/seed"
)

func TestBusinessAuditCapturesStoredDepartmentAndBuildVersion(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	now := time.Date(2026, 10, 14, 1, 2, 3, 0, time.UTC)
	if err := store.CreateRegister(ctx, domain.Register{RegisterID: "reg-audit", RegisterNo: "RA-1", DepartmentID: "dept-register"}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateOrder(ctx, domain.Order{
		OrderID: "order-audit", OrderNo: "OA-1", RegisterID: "reg-audit", DepartmentID: "dept-order",
		Status: domain.OrderStatusCommit,
		Items:  []domain.OrderItem{{OrderItemID: "item-audit", OrderID: "order-audit", ItemNo: "DRUG-A", ItemType: domain.ItemTypeDrug, AmountMilli: 1000}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateCharge(ctx, domain.Charge{
		ChargeID: "charge-audit", SquareNo: "CA-1", OrderID: "order-audit", RegisterID: "reg-audit",
		Status: domain.ChargeStatusNormal, PaidAt: now,
		Items: []domain.ChargeItem{{ChargeItemID: "charge-item-audit", ChargeID: "charge-audit", OrderItemID: "item-audit", ItemNo: "DRUG-A", ItemType: domain.ItemTypeDrug, QtyMilli: 1000}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CommitDispense(ctx, domain.Dispense{
		DispenseID: "dispense-audit", DispenseNo: "DA-1", OrderID: "order-audit", ChargeID: "charge-audit",
		RegisterID: "reg-audit", DeptID: "dept-dispense", DispensedAt: now,
		Items: []domain.DispenseItem{{DispenseItemID: "dispense-item-audit", ChargeItemID: "charge-item-audit", OrderItemID: "item-audit", ItemNo: "DRUG-A", QtyDispensedMilli: 1000}},
	}); err != nil {
		t.Fatal(err)
	}

	svc := &Service{Audit: store, Registers: store, Orders: store, Charges: store, Dispenses: store}
	actor, err := store.FindByID(ctx, seed.DemoUserID)
	if err != nil {
		t.Fatal(err)
	}
	svc.auditRegister(ctx, now, actor, domain.ActionRegisterCreate, "reg-audit", domain.OutcomeSuccess, "", "req-reg", "127.0.0.1")
	svc.auditOrder(ctx, now, actor, domain.ActionOrderCommit, "order-audit", domain.OutcomeSuccess, "", "req-order", "127.0.0.1")
	svc.auditCharge(ctx, now, actor, domain.ActionChargePay, "charge-audit", domain.OutcomeSuccess, "", "req-charge", "127.0.0.1")
	svc.auditDispense(ctx, now, actor, domain.ActionDispenseCreate, "dispense-audit", domain.OutcomeSuccess, "", "req-dispense", "127.0.0.1")

	want := map[string]string{"req-reg": "dept-register", "req-order": "dept-order", "req-charge": "dept-order", "req-dispense": "dept-dispense"}
	for _, event := range store.Audits() {
		if expected, ok := want[event.RequestID]; ok {
			if event.DepartmentID != expected || event.BuildVersion != buildinfo.Version {
				t.Errorf("%s department/version = %q/%q; want %q/%q", event.RequestID, event.DepartmentID, event.BuildVersion, expected, buildinfo.Version)
			}
			delete(want, event.RequestID)
		}
	}
	if len(want) > 0 {
		t.Fatalf("missing business audit events: %v", want)
	}
}
