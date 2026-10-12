package memory

import (
	"context"
	"sort"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

// CreateOrder inserts a sheet and its items. order_no is unique among live rows. 2026-10-12
func (s *Store) CreateOrder(_ context.Context, o domain.Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.orders[o.OrderID]; ok {
		return port.ErrDuplicateBusinessNo
	}
	for _, ex := range s.orders {
		if ex.IsDeleted {
			continue
		}
		if ex.OrderNo == o.OrderNo {
			return port.ErrDuplicateBusinessNo
		}
	}
	s.orders[o.OrderID] = cloneOrder(o)
	return nil
}

func (s *Store) GetOrder(_ context.Context, orderID string) (*domain.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[orderID]
	if !ok || o.IsDeleted {
		return nil, port.ErrNotFound
	}
	c := cloneOrder(o)
	return &c, nil
}

func (s *Store) SearchOrders(_ context.Context, q port.OrderQuery) ([]domain.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	out := make([]domain.Order, 0)
	for _, o := range s.orders {
		if !orderMatches(o, q) {
			continue
		}
		out = append(out, cloneOrder(o))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].OrderID < out[j].OrderID
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func orderMatches(o domain.Order, q port.OrderQuery) bool {
	if o.IsDeleted {
		return false
	}
	if q.RegisterID != "" && o.RegisterID != q.RegisterID {
		return false
	}
	if q.PatientID != "" && o.PatientID != q.PatientID {
		return false
	}
	if q.OrderNo != "" && o.OrderNo != q.OrderNo {
		return false
	}
	if q.Status != "" && o.Status != q.Status {
		return false
	}
	return true
}

func (s *Store) UpdateOrderDraft(_ context.Context, o domain.Order, replaceItems bool) (*domain.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.orders[o.OrderID]
	if !ok || cur.IsDeleted {
		return nil, port.ErrNotFound
	}
	if cur.Status != domain.OrderStatusSave {
		return nil, port.ErrOrderNotEditable
	}
	cur.PatientID = o.PatientID
	cur.DepartmentID = o.DepartmentID
	cur.DoctorID = o.DoctorID
	cur.DiagnoseName = o.DiagnoseName
	cur.RecipeSort = o.RecipeSort
	cur.TotalFeeCents = o.TotalFeeCents
	cur.UpdatedAt = o.UpdatedAt
	cur.UpdatedBy = o.UpdatedBy
	if replaceItems {
		cur.Items = append([]domain.OrderItem(nil), o.Items...)
	}
	s.orders[o.OrderID] = cur
	c := cloneOrder(cur)
	return &c, nil
}

func (s *Store) CommitOrder(_ context.Context, o domain.Order) (*domain.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.orders[o.OrderID]
	if !ok || cur.IsDeleted {
		return nil, port.ErrNotFound
	}
	if cur.Status != domain.OrderStatusSave {
		return nil, port.ErrOrderNotCommittable
	}
	cur.Status = domain.OrderStatusCommit
	cur.TotalFeeCents = o.TotalFeeCents
	cur.SubmittedAt = o.SubmittedAt
	cur.UpdatedAt = o.UpdatedAt
	cur.UpdatedBy = o.UpdatedBy
	cur.Items = append([]domain.OrderItem(nil), o.Items...)
	s.orders[o.OrderID] = cur
	c := cloneOrder(cur)
	return &c, nil
}

func (s *Store) AbateOrder(_ context.Context, orderID, operatorID string, at time.Time) (*domain.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.orders[orderID]
	if !ok || cur.IsDeleted {
		return nil, port.ErrNotFound
	}
	if cur.Charged {
		return nil, port.ErrOrderChargedNotAbatable
	}
	if cur.Status != domain.OrderStatusSave && cur.Status != domain.OrderStatusCommit {
		return nil, port.ErrIllegalStateTransition
	}
	at = at.UTC()
	cur.Status = domain.OrderStatusAbate
	cur.AbatedAt = at
	cur.AbatedBy = operatorID
	cur.UpdatedAt = at
	cur.UpdatedBy = operatorID
	s.orders[orderID] = cur
	c := cloneOrder(cur)
	return &c, nil
}

func (s *Store) CatalogByItemNo(_ context.Context, itemNo string) (*domain.CatalogItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.catalog[itemNo]
	// Inactive rows are returned so the order use case can emit CATALOG_INACTIVE. 2026-10-13
	if !ok || c.IsDeleted {
		return nil, port.ErrNotFound
	}
	cp := c
	return &cp, nil
}

// SetOrderCharged is a test hook for the charged=true abate refusal. It is not on the port. 2026-10-12
func (s *Store) SetOrderCharged(orderID string, charged bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[orderID]
	if !ok || o.IsDeleted {
		return port.ErrNotFound
	}
	o.Charged = charged
	s.orders[orderID] = o
	return nil
}

// SetOrderStatus overwrites status for PROC/ABATE fixtures. It does not write abate columns. 2026-10-12
func (s *Store) SetOrderStatus(orderID, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[orderID]
	if !ok || o.IsDeleted {
		return port.ErrNotFound
	}
	o.Status = status
	s.orders[orderID] = o
	return nil
}

// SetOrderItemAmountMilli overwrites one line quantity so commit can refuse a non-positive amount. 2026-10-12
func (s *Store) SetOrderItemAmountMilli(orderID string, lineNo int, milli int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[orderID]
	if !ok || o.IsDeleted {
		return port.ErrNotFound
	}
	for i := range o.Items {
		if o.Items[i].LineNo == lineNo {
			o.Items[i].AmountMilli = milli
			s.orders[orderID] = o
			return nil
		}
	}
	return port.ErrNotFound
}

func cloneOrder(o domain.Order) domain.Order {
	if o.Items != nil {
		items := make([]domain.OrderItem, len(o.Items))
		copy(items, o.Items)
		o.Items = items
	} else {
		o.Items = []domain.OrderItem{}
	}
	return o
}
