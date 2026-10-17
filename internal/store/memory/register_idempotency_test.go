package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

func TestRegisterCreateIdempotencyReplaysAndRejectsChangedPayload(t *testing.T) {
	store := New()
	ctx := context.Background()
	now := time.Now().UTC()
	reg := domain.Register{
		RegisterID: "idem-register-1", RegisterNo: "R-IDEM-1", PatientID: "patient-1",
		DepartmentID: "dept-1", DoctorID: "doc-1", VisitDate: now,
		Status: domain.RegisterStatusNormal, RegisteredAt: now, CreatedAt: now, UpdatedAt: now,
	}
	expires := now.Add(time.Hour)
	first, replayed, err := store.CreateRegisterWithIdempotency(ctx, reg, "register.create", "register-key-1", "hash-1", expires)
	if err != nil || replayed || first.RegisterID != reg.RegisterID {
		t.Fatalf("first create = %#v, replayed=%v, err=%v", first, replayed, err)
	}
	second, replayed, err := store.CreateRegisterWithIdempotency(ctx, reg, "register.create", "register-key-1", "hash-1", expires)
	if err != nil || !replayed || second.RegisterID != reg.RegisterID {
		t.Fatalf("replay = %#v, replayed=%v, err=%v", second, replayed, err)
	}
	changed := reg
	changed.DoctorID = "doc-2"
	_, _, err = store.CreateRegisterWithIdempotency(ctx, changed, "register.create", "register-key-1", "different-hash", expires)
	if !errors.Is(err, port.ErrIdempotencyConflict) {
		t.Fatalf("changed payload error = %v, want %v", err, port.ErrIdempotencyConflict)
	}
	rows, err := store.SearchRegisters(ctx, port.RegisterQuery{PatientID: reg.PatientID, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("register rows after replay = %d, want 1", len(rows))
	}
}

func TestRegisterCreateIdempotencyRejectsConcurrentPendingKey(t *testing.T) {
	store := New()
	now := time.Now().UTC()
	store.idempotency[idempotencyMapKey("register.create", "register-key-pending")] = idempotencyRecord{
		scope: "register.create", key: "register-key-pending", requestHash: "hash-pending",
		state: idempotencyPending, expiresAt: now.Add(time.Hour),
	}
	_, _, err := store.CreateRegisterWithIdempotency(context.Background(), domain.Register{
		RegisterID: "pending-register", RegisterNo: "R-PENDING", PatientID: "patient-1",
		DepartmentID: "dept-1", DoctorID: "doc-1", VisitDate: now, Status: domain.RegisterStatusNormal,
	}, "register.create", "register-key-pending", "hash-pending", now.Add(time.Hour))
	if !errors.Is(err, port.ErrIdempotencyInProgress) {
		t.Fatalf("pending error = %v, want %v", err, port.ErrIdempotencyInProgress)
	}
}
