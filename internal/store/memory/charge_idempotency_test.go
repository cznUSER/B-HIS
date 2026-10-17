package memory

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

func testIdempotentCharge() domain.Charge {
	return domain.Charge{ChargeID: "charge-idem-1", SquareNo: "S-IDEM-1", OrderID: "order-idem-1", RegisterID: "register-idem-1", PatientID: "patient-idem-1", TotalFeeCents: 100, SquareFeeCents: 100, Status: domain.ChargeStatusNormal, Items: []domain.ChargeItem{{ChargeItemID: "charge-item-idem-1", ChargeID: "charge-idem-1", LineNo: 1, ItemNo: "TREAT", ItemName: "测试", QtyMilli: 1000, UnitPriceCents: 100, TotalFeeCents: 100, CanBack: true}}}
}

func TestChargeCreateIdempotencyReplayConflictAndPending(t *testing.T) {
	s := New()
	ctx := context.Background()
	c := testIdempotentCharge()
	c.OrderID = ""
	expires := time.Now().Add(time.Hour)
	first, replayed, err := s.CreateChargeWithIdempotency(ctx, c, "charge.create", "charge-key", "hash-a", expires)
	if err != nil || replayed || first.ChargeID != c.ChargeID {
		t.Fatalf("first = %#v replayed=%v err=%v", first, replayed, err)
	}
	second, replayed, err := s.CreateChargeWithIdempotency(ctx, c, "charge.create", "charge-key", "hash-a", expires)
	if err != nil || !replayed || second.ChargeID != c.ChargeID || len(second.Items) != 1 {
		t.Fatalf("replay = %#v replayed=%v err=%v", second, replayed, err)
	}
	_, _, err = s.CreateChargeWithIdempotency(ctx, c, "charge.create", "charge-key", "hash-b", expires)
	if !errors.Is(err, port.ErrIdempotencyConflict) {
		t.Fatalf("conflict = %v", err)
	}
	s.idempotency[idempotencyMapKey("charge.create", "pending-key")] = idempotencyRecord{requestHash: "hash-p", state: idempotencyPending, expiresAt: expires}
	_, _, err = s.CreateChargeWithIdempotency(ctx, c, "charge.create", "pending-key", "hash-p", expires)
	if !errors.Is(err, port.ErrIdempotencyInProgress) {
		t.Fatalf("pending = %v", err)
	}
}

func TestChargeCreateIdempotencyConcurrentSameKey(t *testing.T) {
	s := New()
	ctx := context.Background()
	wg := sync.WaitGroup{}
	results := make([]error, 2)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c := testIdempotentCharge()
			c.ChargeID = "charge-idem-concurrent"
			results[i] = func() error {
				_, _, err := s.CreateChargeWithIdempotency(ctx, c, "charge.create", "concurrent-key", "hash-c", time.Now().Add(time.Hour))
				return err
			}()
		}(i)
	}
	wg.Wait()
	for _, err := range results {
		if err != nil && !errors.Is(err, port.ErrIdempotencyInProgress) {
			t.Fatalf("concurrent err = %v", err)
		}
	}
	rows, err := s.SearchCharges(ctx, port.ChargeQuery{OrderID: "order-idem-1", Limit: 10})
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows = %d err=%v", len(rows), err)
	}
}

func TestChargePayAndRefundIdempotencyReplayConflict(t *testing.T) {
	s := New()
	ctx := context.Background()
	when := time.Now().UTC()
	c := testIdempotentCharge()
	c.OrderID = ""
	c.Status = ""
	c.PaidAt = time.Time{}
	if err := s.CreateCharge(ctx, c); err != nil {
		t.Fatal(err)
	}
	if got, err := s.GetCharge(ctx, c.ChargeID); err != nil || got == nil {
		t.Fatalf("created charge lookup=%#v err=%v", got, err)
	}
	pay := domain.Payment{PaymentID: "pay-idem-1", PaymentNo: "pay-no-idem-1", ChargeID: c.ChargeID, Direction: domain.PaymentDirectionCharge, PayType: domain.PayTypeCash, AmountCents: c.SquareFeeCents, Status: domain.PaymentStatusSucceeded, PaidAt: when, CreatedAt: when, UpdatedAt: when}
	old, savedPay, replayed, err := s.PayChargeWithIdempotency(ctx, c.ChargeID, pay, "charge.pay", "pay-key", "hash-a", when.Add(time.Hour))
	if err != nil || replayed || old == nil || savedPay == nil {
		t.Fatalf("pay first old=%#v pay=%#v replayed=%v err=%v", old, savedPay, replayed, err)
	}
	replayOld, replayPay, replayed, err := s.PayChargeWithIdempotency(ctx, c.ChargeID, pay, "charge.pay", "pay-key", "hash-a", when.Add(time.Hour))
	if err != nil || !replayed || replayOld.ChargeID != c.ChargeID || replayPay.PaymentID != pay.PaymentID {
		t.Fatalf("pay replay old=%#v pay=%#v replayed=%v err=%v", replayOld, replayPay, replayed, err)
	}
	_, _, _, err = s.PayChargeWithIdempotency(ctx, c.ChargeID, pay, "charge.pay", "pay-key", "hash-b", when.Add(time.Hour))
	if !errors.Is(err, port.ErrIdempotencyConflict) {
		t.Fatalf("pay conflict=%v", err)
	}
	refund := domain.Payment{PaymentID: "refund-idem-1", PaymentNo: "refund-no-idem-1", ChargeID: c.ChargeID, Direction: domain.PaymentDirectionRefund, PayType: domain.PayTypeCash, AmountCents: c.SquareFeeCents, Status: domain.PaymentStatusSucceeded, PaidAt: when, CreatedAt: when, UpdatedAt: when}
	abated, next, savedRefund, replayed, err := s.RefundChargeWithIdempotency(ctx, *old, nil, refund, "charge.refund", "refund-key", "hash-r", when.Add(time.Hour))
	if err != nil || replayed || abated == nil || next != nil || savedRefund == nil {
		t.Fatalf("refund first old=%#v next=%#v pay=%#v replayed=%v err=%v", abated, next, savedRefund, replayed, err)
	}
	replayAbated, replayNext, replayRefund, replayed, err := s.RefundChargeWithIdempotency(ctx, *old, nil, refund, "charge.refund", "refund-key", "hash-r", when.Add(time.Hour))
	if err != nil || !replayed || replayAbated.ChargeID != c.ChargeID || replayNext != nil || replayRefund.PaymentID != refund.PaymentID {
		t.Fatalf("refund replay old=%#v next=%#v pay=%#v replayed=%v err=%v", replayAbated, replayNext, replayRefund, replayed, err)
	}
	_, _, _, _, err = s.RefundChargeWithIdempotency(ctx, *old, nil, refund, "charge.refund", "refund-key", "hash-other", when.Add(time.Hour))
	if !errors.Is(err, port.ErrIdempotencyConflict) {
		t.Fatalf("refund conflict=%v", err)
	}
}
