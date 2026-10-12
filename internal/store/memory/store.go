package memory

import (
	"context"
	"sync"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/cznUSER/B-HIS/internal/store/seed"
)

// Store is the zero-DB identity store for local demo and tests.
type Store struct {
	mu           sync.Mutex
	byID         map[string]domain.User
	byName       map[string]string
	revoked      map[string]time.Time
	audits       []domain.AuditEvent
	patients     map[string]domain.Patient
	departments  map[string]domain.Department
	doctors      map[string]domain.Doctor
	registers    map[string]domain.Register
	catalog      map[string]domain.CatalogItem
	orders       map[string]domain.Order
	charges      map[string]domain.Charge
	payments     map[string]domain.Payment
	inventory    map[string]domain.InventoryItem
	ledgers      []domain.InventoryLedger
	dispenses    map[string]domain.Dispense
	dispenseStub map[string]dispenseRec
}

func New() *Store {
	s := &Store{
		byID:         make(map[string]domain.User),
		byName:       make(map[string]string),
		revoked:      make(map[string]time.Time),
		patients:     make(map[string]domain.Patient),
		departments:  make(map[string]domain.Department),
		doctors:      make(map[string]domain.Doctor),
		registers:    make(map[string]domain.Register),
		catalog:      make(map[string]domain.CatalogItem),
		orders:       make(map[string]domain.Order),
		charges:      make(map[string]domain.Charge),
		payments:     make(map[string]domain.Payment),
		inventory:    make(map[string]domain.InventoryItem),
		dispenses:    make(map[string]domain.Dispense),
		dispenseStub: make(map[string]dispenseRec),
	}
	s.Add(seed.DemoUser())
	s.Add(seed.DisabledUser())
	for _, d := range seed.Departments() {
		s.departments[d.DepartmentID] = d
	}
	for _, d := range seed.Doctors() {
		s.doctors[d.DoctorID] = d
	}
	for _, c := range seed.CatalogItems() {
		s.catalog[c.ItemNo] = c
	}
	for _, it := range seed.InventoryItems() {
		s.inventory[it.ItemNo] = it
	}
	return s
}

func (s *Store) Add(u domain.User) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[u.ID] = clone(u)
	s.byName[u.Username] = u.ID
}

func (s *Store) SetStatus(id, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.byID[id]
	if !ok {
		return
	}
	u.Status = status
	s.byID[id] = u
}

func (s *Store) FindByUsername(_ context.Context, username string) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.byName[username]
	if !ok {
		return nil, port.ErrNotFound
	}
	u := clone(s.byID[id])
	return &u, nil
}

func (s *Store) FindByID(_ context.Context, id string) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.byID[id]
	if !ok {
		return nil, port.ErrNotFound
	}
	c := clone(u)
	return &c, nil
}

func (s *Store) Revoke(_ context.Context, jti, _ string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revoked[jti] = expiresAt.UTC()
	return nil
}

func (s *Store) IsRevoked(_ context.Context, jti string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.revoked[jti]
	if !ok {
		return false, nil
	}
	if !exp.IsZero() && !exp.After(time.Now().UTC()) {
		delete(s.revoked, jti)
		return false, nil
	}
	return true, nil
}

func (s *Store) Append(_ context.Context, event domain.AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audits = append(s.audits, event)
	return nil
}

func (s *Store) Audits() []domain.AuditEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.AuditEvent, len(s.audits))
	copy(out, s.audits)
	return out
}

func clone(u domain.User) domain.User {
	if u.Roles != nil {
		roles := make([]domain.Role, len(u.Roles))
		copy(roles, u.Roles)
		u.Roles = roles
	}
	return u
}
