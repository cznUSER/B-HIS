package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

func testIdempotentOrder() domain.Order {
	now := time.Now().UTC()
	return domain.Order{
		OrderID: "idem-order-1", OrderNo: "O-IDEM-1", RegisterID: "register-1", PatientID: "patient-1",
		DepartmentID: "dept-1", DoctorID: "doc-1", RecipeSort: "NORMAL", Status: domain.OrderStatusSave,
		CreatedAt: now, UpdatedAt: now,
		Items: []domain.OrderItem{{OrderItemID: "idem-item-1", OrderID: "idem-order-1", LineNo: 1, ItemNo: "item-1", AmountMilli: 1000, PriceCents: 100}},
	}
}

func TestOrderCreateIdempotencyReplaysAndRejectsChangedPayload(t *testing.T) {
	store := New()
	ctx := context.Background()
	o := testIdempotentOrder()
	expires := time.Now().UTC().Add(time.Hour)
	first, replayed, err := store.CreateOrderWithIdempotency(ctx, o, "order.create", "order-key-1", "hash-1", expires)
	if err != nil || replayed || first.OrderID != o.OrderID || len(first.Items) != 1 {
		t.Fatalf("first create = %#v, replayed=%v, err=%v", first, replayed, err)
	}
	// A second service instance sharing the same store must replay after restart.
	second, replayed, err := store.CreateOrderWithIdempotency(ctx, o, "order.create", "order-key-1", "hash-1", expires)
	if err != nil || !replayed || second.OrderID != o.OrderID || len(second.Items) != 1 {
		t.Fatalf("restart replay = %#v, replayed=%v, err=%v", second, replayed, err)
	}
	changed := o
	changed.DoctorID = "doc-2"
	_, _, err = store.CreateOrderWithIdempotency(ctx, changed, "order.create", "order-key-1", "different-hash", expires)
	if !errors.Is(err, port.ErrIdempotencyConflict) {
		t.Fatalf("changed payload error = %v, want %v", err, port.ErrIdempotencyConflict)
	}
	rows, err := store.SearchOrders(ctx, port.OrderQuery{RegisterID: o.RegisterID, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || len(rows[0].Items) != 1 {
		t.Fatalf("order rows after replay = %#v, want one order with one item", rows)
	}
}

func TestOrderCreateIdempotencyRejectsConcurrentPendingKey(t *testing.T) {
	store := New()
	now := time.Now().UTC()
	store.idempotency[idempotencyMapKey("order.create", "order-key-pending")] = idempotencyRecord{
		scope: "order.create", key: "order-key-pending", requestHash: "hash-pending",
		state: idempotencyPending, expiresAt: now.Add(time.Hour),
	}
	_, _, err := store.CreateOrderWithIdempotency(context.Background(), testIdempotentOrder(), "order.create", "order-key-pending", "hash-pending", now.Add(time.Hour))
	if !errors.Is(err, port.ErrIdempotencyInProgress) {
		t.Fatalf("pending error = %v, want %v", err, port.ErrIdempotencyInProgress)
	}
}
