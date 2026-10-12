package memory

import (
	"context"
	"sort"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

// CreateRegister inserts a register bill. register_no is unique among live rows.
// A clash is refused so the caller can retry with a new number. 2026-10-12
func (s *Store) CreateRegister(_ context.Context, r domain.Register) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.registers[r.RegisterID]; ok {
		return port.ErrDuplicateBusinessNo
	}
	for _, ex := range s.registers {
		if ex.IsDeleted {
			continue
		}
		if ex.RegisterNo == r.RegisterNo {
			return port.ErrDuplicateBusinessNo
		}
	}
	s.registers[r.RegisterID] = r
	return nil
}

func (s *Store) GetRegister(_ context.Context, registerID string) (*domain.Register, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.registers[registerID]
	if !ok || r.IsDeleted {
		return nil, port.ErrNotFound
	}
	c := r
	return &c, nil
}

func (s *Store) SearchRegisters(_ context.Context, q port.RegisterQuery) ([]domain.Register, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	out := make([]domain.Register, 0)
	for _, r := range s.registers {
		if !registerMatches(r, q) {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RegisteredAt.Equal(out[j].RegisteredAt) {
			return out[i].RegisterID < out[j].RegisterID
		}
		return out[i].RegisteredAt.Before(out[j].RegisteredAt)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func registerMatches(r domain.Register, q port.RegisterQuery) bool {
	if r.IsDeleted {
		return false
	}
	if q.PatientID != "" && r.PatientID != q.PatientID {
		return false
	}
	if q.RegisterNo != "" && r.RegisterNo != q.RegisterNo {
		return false
	}
	if q.VisitDate != "" && r.VisitDate.Format("2006-01-02") != q.VisitDate {
		return false
	}
	return true
}

// CancelRegister accepts only NORMAL. BACK and every other status leave the row unchanged.
func (s *Store) CancelRegister(_ context.Context, registerID, operatorID string, at time.Time) (*domain.Register, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.registers[registerID]
	if !ok || r.IsDeleted {
		return nil, port.ErrNotFound
	}
	switch r.Status {
	case domain.RegisterStatusNormal:
		r.Status = domain.RegisterStatusBack
		r.BackOperatorID = operatorID
		r.BackOperatedAt = at.UTC()
		r.UpdatedAt = at.UTC()
		r.UpdatedBy = operatorID
		s.registers[registerID] = r
		c := r
		return &c, nil
	case domain.RegisterStatusBack:
		return nil, port.ErrRegisterAlreadyBack
	default:
		return nil, port.ErrRegisterNotCancellable
	}
}

// CheckInRegister accepts only NORMAL or USED. Other statuses leave the row unchanged.
// 2026-10-12
func (s *Store) CheckInRegister(_ context.Context, registerID, operatorID string, at time.Time) (*domain.Register, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.registers[registerID]
	if !ok || r.IsDeleted {
		return nil, port.ErrNotFound
	}
	switch r.Status {
	case domain.RegisterStatusNormal, domain.RegisterStatusUsed:
		r.Status = domain.RegisterStatusDiagnosing
		r.UpdatedAt = at.UTC()
		r.UpdatedBy = operatorID
		s.registers[registerID] = r
		c := r
		return &c, nil
	default:
		return nil, port.ErrRegisterNotCheckIn
	}
}

func (s *Store) DepartmentExists(_ context.Context, departmentID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.departments[departmentID]
	return ok && !d.IsDeleted, nil
}

func (s *Store) DoctorExists(_ context.Context, doctorID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.doctors[doctorID]
	return ok && !d.IsDeleted, nil
}

func (s *Store) GetDepartment(_ context.Context, departmentID string) (*domain.Department, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.departments[departmentID]
	if !ok || d.IsDeleted {
		return nil, port.ErrNotFound
	}
	return &d, nil
}

func (s *Store) GetDoctor(_ context.Context, doctorID string) (*domain.Doctor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.doctors[doctorID]
	if !ok || d.IsDeleted {
		return nil, port.ErrNotFound
	}
	return &d, nil
}

// SetRegisterStatus overwrites status for fixtures that need a non-NORMAL bill
// without check-in. It does not write back columns. 2026-10-12
func (s *Store) SetRegisterStatus(registerID, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.registers[registerID]
	if !ok || r.IsDeleted {
		return port.ErrNotFound
	}
	r.Status = status
	s.registers[registerID] = r
	return nil
}
