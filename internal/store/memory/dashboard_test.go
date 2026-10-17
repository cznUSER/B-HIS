package memory

import (
	"context"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

func TestSummarizeDashboardUsesShanghaiWindowAndDepartmentScopes(t *testing.T) {
	ctx := context.Background()
	st := New()
	start := time.Date(2026, 10, 12, 0, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	inside := start.Add(30 * time.Minute)
	outside := start.Add(-time.Nanosecond)

	for _, row := range []domain.Register{
		{RegisterID: "enc-1", RegisterNo: "reg-1", DepartmentID: "dept-1", RegisteredAt: inside, Status: domain.RegisterStatusNormal},
		{RegisterID: "enc-2", RegisterNo: "reg-2", DepartmentID: "dept-1", RegisteredAt: inside.Add(time.Hour), Status: domain.RegisterStatusUsed},
		{RegisterID: "enc-other-dept", RegisterNo: "reg-other", DepartmentID: "dept-2", RegisteredAt: inside, Status: domain.RegisterStatusNormal},
		{RegisterID: "enc-outside", RegisterNo: "reg-outside", DepartmentID: "dept-1", RegisteredAt: outside, Status: domain.RegisterStatusNormal},
		{RegisterID: "enc-back", RegisterNo: "reg-back", DepartmentID: "dept-1", RegisteredAt: inside, Status: domain.RegisterStatusBack},
	} {
		if err := st.CreateRegister(ctx, row); err != nil {
			t.Fatal(err)
		}
	}

	paidRefunded := dashboardTestCharge("charge-refunded", "enc-1", 2000, inside)
	paidOpen := dashboardTestCharge("charge-open", "enc-2", 3500, inside)
	paidOtherDept := dashboardTestCharge("charge-other", "enc-other-dept", 9000, inside)
	paidOutside := dashboardTestCharge("charge-outside", "enc-outside", 7000, outside)
	pending := dashboardTestCharge("charge-pending", "enc-1", 1200, inside)
	for _, charge := range []domain.Charge{paidRefunded, paidOpen, paidOtherDept, paidOutside, pending} {
		if err := st.CreateCharge(ctx, charge); err != nil {
			t.Fatal(err)
		}
	}
	for _, charge := range []domain.Charge{paidRefunded, paidOpen, paidOtherDept, paidOutside} {
		pay := domain.Payment{
			PaymentID: "payment-" + charge.ChargeID, PaymentNo: "pay-" + charge.ChargeID,
			ChargeID: charge.ChargeID, Direction: domain.PaymentDirectionCharge,
			PayType: domain.PayTypeCash, AmountCents: charge.TotalFeeCents,
			Status: domain.PaymentStatusSucceeded, PaidAt: charge.CreatedAt,
		}
		if _, _, err := st.PayCharge(ctx, charge.ChargeID, pay); err != nil {
			t.Fatal(err)
		}
	}
	refund := domain.Payment{
		PaymentID: "payment-refund", PaymentNo: "pay-refund", ChargeID: paidRefunded.ChargeID,
		Direction: domain.PaymentDirectionRefund, PayType: domain.PayTypeCash, AmountCents: 2000,
		Status: domain.PaymentStatusSucceeded, PaidAt: inside.Add(2 * time.Hour),
	}
	old := paidRefunded
	old.Status = domain.ChargeStatusAbate
	if _, _, _, err := st.RefundCharge(ctx, old, nil, refund); err != nil {
		t.Fatal(err)
	}

	got, err := st.SummarizeDashboard(ctx, port.DashboardQuery{
		StartAt: start, EndAt: start.AddDate(0, 0, 1),
		ChargeDepartmentIDs:    []string{"dept-1"},
		EncounterDepartmentIDs: []string{"dept-1"},
		DispenseDepartmentIDs:  []string{"dept-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.RevenueCents != 5500 || got.RefundCents != 2000 || got.NetRevenueCents != 3500 {
		t.Fatalf("financial totals include out-of-scope or out-of-period rows: %+v", got)
	}
	if got.ChargeCount != 2 || got.RefundCount != 1 || got.EncounterCount != 2 || got.PendingChargeCount != 1 || got.PendingDispenseCount != 2 {
		t.Fatalf("unexpected counts: %+v", got)
	}
}

func dashboardTestCharge(id, registerID string, amount int64, at time.Time) domain.Charge {
	return domain.Charge{
		ChargeID: id, SquareNo: "square-" + id, RegisterID: registerID,
		TotalFeeCents: amount, SquareFeeCents: amount, Status: domain.ChargeStatusNormal,
		CreatedAt: at, Items: []domain.ChargeItem{{
			ChargeItemID: "item-" + id, ChargeID: id, ItemNo: "DRUG-A", ItemType: domain.ItemTypeDrug,
			QtyMilli: 1000, UnitPriceCents: amount, TotalFeeCents: amount,
		}},
	}
}
