package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

func memoryDispenseFixture(t *testing.T) (*Store, context.Context, time.Time) {
	t.Helper()
	ctx := context.Background()
	s := New()
	now := time.Date(2026, 10, 15, 3, 0, 0, 0, time.UTC)
	if err := s.CreateOrder(ctx, domain.Order{
		OrderID: "idem-o1", OrderNo: "idem-O-1", RegisterID: "idem-r1", PatientID: "idem-p1", Status: domain.OrderStatusCommit,
		Items: []domain.OrderItem{{OrderItemID: "idem-oi1", OrderID: "idem-o1", ItemNo: "DRUG-A", ItemType: domain.ItemTypeDrug, ItemName: "药品", Units: "盒", AmountMilli: 1000}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateCharge(ctx, domain.Charge{
		ChargeID: "idem-c1", SquareNo: "idem-S-1", OrderID: "idem-o1", RegisterID: "idem-r1", PatientID: "idem-p1", Status: domain.ChargeStatusNormal, PaidAt: now,
		Items: []domain.ChargeItem{{ChargeItemID: "idem-ci1", ChargeID: "idem-c1", OrderItemID: "idem-oi1", ItemNo: "DRUG-A", ItemType: domain.ItemTypeDrug, QtyMilli: 1000}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetDrugOnHandMilli("DRUG-A", 5000); err != nil {
		t.Fatal(err)
	}
	return s, ctx, now
}

func TestDispenseIdempotencyCreateAndReturn(t *testing.T) {
	s, ctx, now := memoryDispenseFixture(t)
	expires := time.Now().UTC().Add(time.Hour)
	d := domain.Dispense{
		DispenseID: "idem-d1", DispenseNo: "idem-D-1", OrderID: "idem-o1", ChargeID: "idem-c1", RegisterID: "idem-r1", PatientID: "idem-p1", DispensedAt: now, UpdatedAt: now, CreatedBy: "u", UpdatedBy: "u",
		Items: []domain.DispenseItem{{DispenseItemID: "idem-di1", ChargeItemID: "idem-ci1", OrderItemID: "idem-oi1", ItemNo: "DRUG-A", QtyDispensedMilli: 1000}},
	}
	first, replayed, err := s.CreateDispenseWithIdempotency(ctx, d, "dispense.create", "key-1", "hash-1", expires)
	if err != nil || replayed || first.DispenseID != d.DispenseID {
		t.Fatalf("first %+v replay=%v err=%v", first, replayed, err)
	}
	on, _ := s.QtyOnHandMilli(ctx, "DRUG-A")
	if on != 4000 || len(s.ledgers) != 1 {
		t.Fatalf("create side effects on=%d ledgers=%d", on, len(s.ledgers))
	}
	replay, replayed, err := s.CreateDispenseWithIdempotency(ctx, d, "dispense.create", "key-1", "hash-1", expires)
	if err != nil || !replayed || replay.DispenseID != d.DispenseID {
		t.Fatalf("replay %+v replay=%v err=%v", replay, replayed, err)
	}
	on, _ = s.QtyOnHandMilli(ctx, "DRUG-A")
	if on != 4000 || len(s.ledgers) != 1 {
		t.Fatalf("replay side effects on=%d ledgers=%d", on, len(s.ledgers))
	}
	if _, _, err := s.CreateDispenseWithIdempotency(ctx, d, "dispense.create", "key-1", "hash-other", expires); !errors.Is(err, port.ErrIdempotencyConflict) {
		t.Fatalf("conflict %v", err)
	}
	returned, replayed, err := s.ReturnDispenseWithIdempotency(ctx, d.DispenseID, "患者要求", "u", now, []port.ReturnLine{{DispenseItemID: "idem-di1", ReturnMilli: 1000}}, "dispense.return", "ret-key-1", "ret-hash-1", expires)
	if err != nil || replayed || returned.Status != domain.DispenseStatusReturned {
		t.Fatalf("return %+v replay=%v err=%v", returned, replayed, err)
	}
	on, _ = s.QtyOnHandMilli(ctx, "DRUG-A")
	if on != 5000 || len(s.ledgers) != 2 {
		t.Fatalf("return side effects on=%d ledgers=%d", on, len(s.ledgers))
	}
	returned, replayed, err = s.ReturnDispenseWithIdempotency(ctx, d.DispenseID, "患者要求", "u", now, []port.ReturnLine{{DispenseItemID: "idem-di1", ReturnMilli: 1000}}, "dispense.return", "ret-key-1", "ret-hash-1", expires)
	if err != nil || !replayed || returned.Status != domain.DispenseStatusReturned {
		t.Fatalf("return replay %+v replay=%v err=%v", returned, replayed, err)
	}
	on, _ = s.QtyOnHandMilli(ctx, "DRUG-A")
	if on != 5000 || len(s.ledgers) != 2 {
		t.Fatalf("return replay side effects on=%d ledgers=%d", on, len(s.ledgers))
	}
}

func TestDispenseIdempotencyFailureReleasesKey(t *testing.T) {
	s, ctx, now := memoryDispenseFixture(t)
	if err := s.SetDrugOnHandMilli("DRUG-A", 0); err != nil {
		t.Fatal(err)
	}
	d := domain.Dispense{DispenseID: "bad-d", DispenseNo: "bad-D", OrderID: "idem-o1", ChargeID: "idem-c1", DispensedAt: now, UpdatedAt: now, CreatedBy: "u", UpdatedBy: "u", Items: []domain.DispenseItem{{DispenseItemID: "bad-di", ChargeItemID: "idem-ci1", OrderItemID: "idem-oi1", ItemNo: "DRUG-A", QtyDispensedMilli: 1000}}}
	if _, _, err := s.CreateDispenseWithIdempotency(ctx, d, "dispense.create", "bad-key", "bad-hash", time.Now().UTC().Add(time.Hour)); !errors.Is(err, port.ErrInsufficientStock) {
		t.Fatalf("expected stock failure, got %v", err)
	}
	if _, ok := s.idempotency[idempotencyMapKey("dispense.create", "bad-key")]; ok {
		t.Fatal("failed create retained idempotency marker")
	}
	if len(s.ledgers) != 0 {
		t.Fatalf("failed create wrote ledger: %d", len(s.ledgers))
	}
}
