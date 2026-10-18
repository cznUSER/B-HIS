package memory

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

// Create inserts a patient. Empty id_number and card_no do not participate in uniqueness.
// A clash is refused; this store never merges records. 2026-10-12
func (s *Store) Create(_ context.Context, p domain.Patient) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createPatientLocked(p)
}

func (s *Store) createPatientLocked(p domain.Patient) error {
	if _, ok := s.patients[p.PatientID]; ok {
		return port.ErrDuplicateBusinessNo
	}
	for _, ex := range s.patients {
		if ex.IsDeleted {
			continue
		}
		if ex.HospPkNo == p.HospPkNo {
			return port.ErrDuplicateBusinessNo
		}
		if p.IDNumber != "" && ex.IDNumber == p.IDNumber {
			return port.ErrDuplicateIDNumber
		}
		if p.CardNo != "" && ex.CardNo == p.CardNo {
			return port.ErrDuplicateCardNo
		}
	}
	s.patients[p.PatientID] = p
	return nil
}

// CreateWithIdempotency keeps the key and patient mutation under the same
// memory lock, matching the SQL transaction contract used by the MSSQL store.
func (s *Store) CreateWithIdempotency(_ context.Context, p domain.Patient, scope, key, requestHash string, expiresAt time.Time) (domain.Patient, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	mapKey := idempotencyMapKey(scope, key)
	now := time.Now().UTC()
	if record, ok := s.idempotency[mapKey]; ok {
		if !record.expiresAt.IsZero() && !now.Before(record.expiresAt) {
			delete(s.idempotency, mapKey)
		} else {
			if record.requestHash != requestHash {
				return domain.Patient{}, false, port.ErrIdempotencyConflict
			}
			switch record.state {
			case idempotencyPending:
				return domain.Patient{}, false, port.ErrIdempotencyInProgress
			case idempotencyCompleted:
				existing, ok := s.patients[record.resourceID]
				if !ok || existing.IsDeleted {
					return domain.Patient{}, false, port.ErrNotFound
				}
				return existing, true, nil
			}
		}
	}
	s.idempotency[mapKey] = idempotencyRecord{scope: scope, key: key, requestHash: requestHash, state: idempotencyPending, expiresAt: expiresAt.UTC()}
	if err := s.createPatientLocked(p); err != nil {
		delete(s.idempotency, mapKey)
		return domain.Patient{}, false, err
	}
	s.idempotency[mapKey] = idempotencyRecord{scope: scope, key: key, requestHash: requestHash, state: idempotencyCompleted, resourceID: p.PatientID, expiresAt: expiresAt.UTC()}
	return p, false, nil
}

func (s *Store) Get(_ context.Context, patientID string) (*domain.Patient, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.patients[patientID]
	if !ok || p.IsDeleted {
		return nil, port.ErrNotFound
	}
	c := p
	return &c, nil
}

func (s *Store) Search(_ context.Context, q port.PatientQuery) ([]domain.Patient, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	out := make([]domain.Patient, 0)
	for _, p := range s.patients {
		if !patientMatches(p, q) {
			continue
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].PatientID < out[j].PatientID
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	start := q.Offset
	if start < 0 {
		start = 0
	}
	if start > len(out) {
		start = len(out)
	}
	end := start + limit
	if end > len(out) {
		end = len(out)
	}
	return out[start:end], nil
}

func patientMatches(p domain.Patient, q port.PatientQuery) bool {
	if p.IsDeleted {
		return false
	}
	if q.PatientNo != "" && p.PatientNo != q.PatientNo {
		return false
	}
	if q.CardNo != "" && p.CardNo != q.CardNo {
		return false
	}
	if q.IDNumber != "" && p.IDNumber != q.IDNumber {
		return false
	}
	if q.HospPkNo != "" && p.HospPkNo != q.HospPkNo {
		return false
	}
	if q.Name != "" && !strings.Contains(p.Name, q.Name) {
		return false
	}
	if q.Address != "" && !strings.Contains(p.Address, q.Address) {
		return false
	}
	return true
}
