package memory

import (
	"context"
	"sort"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

func (s *Store) CreateDiagnose(_ context.Context, d domain.Diagnose) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.diagnoses[d.DiagnoseID]; exists {
		return port.ErrDuplicateBusinessNo
	}
	s.diagnoses[d.DiagnoseID] = d
	return nil
}

func (s *Store) GetDiagnose(_ context.Context, diagnoseID string) (*domain.Diagnose, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.diagnoses[diagnoseID]
	if !ok || d.IsDeleted {
		return nil, port.ErrNotFound
	}
	return &d, nil
}

func (s *Store) ListDiagnoses(_ context.Context, q port.DiagnoseQuery) ([]domain.Diagnose, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	out := make([]domain.Diagnose, 0)
	for _, d := range s.diagnoses {
		if d.IsDeleted || (q.RegisterID != "" && d.RegisterID != q.RegisterID) {
			continue
		}
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].DiagnoseID < out[j].DiagnoseID
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *Store) CommitDiagnose(_ context.Context, diagnoseID, operatorID string, at time.Time) (*domain.Diagnose, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.diagnoses[diagnoseID]
	if !ok || d.IsDeleted {
		return nil, port.ErrNotFound
	}
	if d.Status != domain.DiagnoseStatusSave {
		return nil, port.ErrDiagnoseNotCommittable
	}
	d.Status = domain.DiagnoseStatusCommit
	d.UpdatedAt = at.UTC()
	d.UpdatedBy = operatorID
	s.diagnoses[diagnoseID] = d
	return &d, nil
}

func (s *Store) AbateDiagnose(_ context.Context, diagnoseID, operatorID string, at time.Time) (*domain.Diagnose, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.diagnoses[diagnoseID]
	if !ok || d.IsDeleted {
		return nil, port.ErrNotFound
	}
	if d.Status != domain.DiagnoseStatusSave && d.Status != domain.DiagnoseStatusCommit {
		return nil, port.ErrDiagnoseNotAbatable
	}
	d.Status = domain.DiagnoseStatusAbate
	d.UpdatedAt = at.UTC()
	d.UpdatedBy = operatorID
	s.diagnoses[diagnoseID] = d
	return &d, nil
}
