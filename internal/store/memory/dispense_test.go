package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

func TestCommitDispenseDoesNotMutateOnShortStock(t *testing.T) {
	ctx := context.Background()
	s := New()
	now := time.Date(2026, 10, 12, 4, 0, 0, 0, time.UTC)
	if err := s.CreateOrder(ctx, domain.Order{
		OrderID: "o1", OrderNo: "O-1", RegisterID: "r1", PatientID: "p1", Status: domain.OrderStatusCommit,
		Items: []domain.OrderItem{{
			OrderItemID: "oi1", OrderID: "o1", ItemNo: "DRUG-A", ItemType: domain.ItemTypeDrug,
			ItemName: "演示阿莫西林胶囊", Units: "盒", AmountMilli: 2000,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateCharge(ctx, domain.Charge{
		ChargeID: "c1", SquareNo: "S-1", OrderID: "o1", RegisterID: "r1", PatientID: "p1",
		Status: domain.ChargeStatusNormal, PaidAt: now,
		Items: []domain.ChargeItem{{
			ChargeItemID: "ci1", ChargeID: "c1", OrderItemID: "oi1", ItemNo: "DRUG-A",
			ItemType: domain.ItemTypeDrug, QtyMilli: 2000,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetDrugOnHandMilli("DRUG-A", 1000); err != nil {
		t.Fatal(err)
	}
	_, err := s.CommitDispense(ctx, domain.Dispense{
		DispenseID: "d-short", DispenseNo: "D-short", OrderID: "o1", ChargeID: "c1",
		RegisterID: "r1", PatientID: "p1", DispensedAt: now, UpdatedAt: now,
		Items: []domain.DispenseItem{{
			DispenseItemID: "di-short", ChargeItemID: "ci1", OrderItemID: "oi1", ItemNo: "DRUG-A",
			QtyDispensedMilli: 2000,
		}},
	})
	if !errors.Is(err, port.ErrInsufficientStock) {
		t.Fatalf("short %v", err)
	}
	on, err := s.QtyOnHandMilli(ctx, "DRUG-A")
	if err != nil || on != 1000 {
		t.Fatalf("stock %d %v", on, err)
	}
	if _, err := s.GetDispense(ctx, "d-short"); !errors.Is(err, port.ErrNotFound) {
		t.Fatalf("slip %v", err)
	}
	if len(s.ledgers) != 0 {
		t.Fatalf("ledger %d", len(s.ledgers))
	}
	ord, err := s.GetOrder(ctx, "o1")
	if err != nil || ord.Status != domain.OrderStatusCommit {
		t.Fatalf("order %+v %v", ord, err)
	}

	if err := s.SetDrugOnHandMilli("DRUG-A", 100000); err != nil {
		t.Fatal(err)
	}
	got, err := s.CommitDispense(ctx, domain.Dispense{
		DispenseID: "d1", DispenseNo: "D-1", OrderID: "o1", ChargeID: "c1",
		RegisterID: "r1", PatientID: "p1", DispensedAt: now, UpdatedAt: now, CreatedBy: "u",
		Items: []domain.DispenseItem{{
			DispenseItemID: "di1", ChargeItemID: "ci1", OrderItemID: "oi1", ItemNo: "DRUG-A",
			QtyDispensedMilli: 2000,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.DispenseStatusDone || got.Items[0].LineStatus != domain.LineDispenseFull {
		t.Fatalf("%+v", got)
	}
	on, _ = s.QtyOnHandMilli(ctx, "DRUG-A")
	if on != 98000 {
		t.Fatalf("after %d", on)
	}
	if len(s.ledgers) != 1 || s.ledgers[0].BizType != domain.LedgerBizDispense || s.ledgers[0].QtyDeltaMilli != -2000 {
		t.Fatalf("ledger %+v", s.ledgers)
	}
	ord, _ = s.GetOrder(ctx, "o1")
	if ord.Status != domain.OrderStatusProc {
		t.Fatalf("proc %s", ord.Status)
	}

	back, err := s.CommitReturn(ctx, "d1", "患者要求", "u", now, []port.ReturnLine{{DispenseItemID: "di1", ReturnMilli: 2000}})
	if err != nil {
		t.Fatal(err)
	}
	if back.Status != domain.DispenseStatusReturned || back.Items[0].QtyReturnedMilli != 2000 {
		t.Fatalf("return %+v", back)
	}
	on, _ = s.QtyOnHandMilli(ctx, "DRUG-A")
	if on != 100000 {
		t.Fatalf("restock %d", on)
	}
	if s.ledgers[1].BizType != domain.LedgerBizReturn || s.ledgers[1].QtyDeltaMilli != 2000 {
		t.Fatalf("return ledger %+v", s.ledgers[1])
	}
}
