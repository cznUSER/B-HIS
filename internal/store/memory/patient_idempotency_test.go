package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

func TestPatientCreateIdempotencyReplaysAndRejectsChangedPayload(t *testing.T) {
	store := New()
	p := domain.Patient{
		PatientID:    "patient-idem-1",
		HospPkNo:     "H-IDEM-1",
		PatientNo:    "P-IDEM-1",
		Name:         "幂等患者",
		Sex:          domain.SexUnknown,
		Birthday:     time.Date(1990, 1, 2, 0, 0, 0, 0, time.UTC),
		FareIdentity: "SELF",
		Status:       domain.PatientStatusActive,
	}

	first, replayed, err := store.CreateWithIdempotency(context.Background(), p, "patient.create", "key-1", "hash-1", time.Now().Add(time.Hour))
	if err != nil || replayed || first.PatientID != p.PatientID {
		t.Fatalf("first create = %#v, replayed=%v, err=%v", first, replayed, err)
	}

	second, replayed, err := store.CreateWithIdempotency(context.Background(), p, "patient.create", "key-1", "hash-1", time.Now().Add(time.Hour))
	if err != nil || !replayed || second.PatientID != p.PatientID {
		t.Fatalf("replay = %#v, replayed=%v, err=%v", second, replayed, err)
	}

	p.Name = "改过的幂等患者"
	_, _, err = store.CreateWithIdempotency(context.Background(), p, "patient.create", "key-1", "different-hash", time.Now().Add(time.Hour))
	if !errors.Is(err, port.ErrIdempotencyConflict) {
		t.Fatalf("changed payload error = %v, want %v", err, port.ErrIdempotencyConflict)
	}
}

func TestPatientCreateIdempotencyRejectsConcurrentPendingKey(t *testing.T) {
	store := New()
	p := domain.Patient{PatientID: "patient-idem-2", HospPkNo: "H-IDEM-2", PatientNo: "P-IDEM-2", Name: "处理中", Sex: domain.SexUnknown, Birthday: time.Date(1990, 1, 2, 0, 0, 0, 0, time.UTC), FareIdentity: "SELF", Status: domain.PatientStatusActive}
	store.mu.Lock()
	store.idempotency[idempotencyMapKey("patient.create", "key-pending")] = idempotencyRecord{scope: "patient.create", key: "key-pending", requestHash: "hash-pending", state: idempotencyPending, expiresAt: time.Now().Add(time.Hour)}
	store.mu.Unlock()
	_, _, err := store.CreateWithIdempotency(context.Background(), p, "patient.create", "key-pending", "hash-pending", time.Now().Add(time.Hour))
	if !errors.Is(err, port.ErrIdempotencyInProgress) {
		t.Fatalf("pending error = %v, want %v", err, port.ErrIdempotencyInProgress)
	}
}
