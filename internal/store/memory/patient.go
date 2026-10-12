package memory

import (
	"context"
	"sort"
	"strings"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

// Create inserts a patient. Empty id_number and card_no do not participate in uniqueness.
// A clash is refused; this store never merges records. 2026-10-12
func (s *Store) Create(_ context.Context, p domain.Patient) error {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func patientMatches(p domain.Patient, q port.PatientQuery) bool {
	if p.IsDeleted {
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
