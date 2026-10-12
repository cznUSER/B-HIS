package memory

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

// dispenseRec is the process-local dispensed/returned stub. Not a SQL column. 2026-10-12
type dispenseRec struct {
	dispensed int64
	returned  int64
}

// chargeZone matches app.shanghai (UTC+8, no DST). List filters use this calendar day. 2026-10-12
var chargeZone = time.FixedZone("Asia/Shanghai", 8*60*60)

// CreateCharge inserts a pending square. One live or historical charge per order (1:1). 2026-10-12
func (s *Store) CreateCharge(_ context.Context, c domain.Charge) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.charges[c.ChargeID]; ok {
		return port.ErrDuplicateBusinessNo
	}
	for _, ex := range s.charges {
		if ex.IsDeleted {
			continue
		}
		if ex.SquareNo == c.SquareNo {
			return port.ErrDuplicateBusinessNo
		}
		if c.OrderID != "" && ex.OrderID == c.OrderID {
			return port.ErrChargeExists
		}
	}
	s.charges[c.ChargeID] = cloneCharge(c)
	return nil
}

func (s *Store) GetCharge(_ context.Context, chargeID string) (*domain.Charge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.charges[chargeID]
	if !ok || c.IsDeleted {
		return nil, port.ErrNotFound
	}
	out := cloneCharge(c)
	return &out, nil
}

func (s *Store) SearchCharges(_ context.Context, q port.ChargeQuery) ([]domain.Charge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	out := make([]domain.Charge, 0)
	for _, c := range s.charges {
		if !chargeMatches(c, q) {
			continue
		}
		out = append(out, cloneCharge(c))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ChargeID < out[j].ChargeID
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func chargeMatches(c domain.Charge, q port.ChargeQuery) bool {
	if c.IsDeleted {
		return false
	}
	if q.PatientID != "" && c.PatientID != q.PatientID {
		return false
	}
	if q.RegisterID != "" && c.RegisterID != q.RegisterID {
		return false
	}
	if q.OrderID != "" && c.OrderID != q.OrderID {
		return false
	}
	day := chargeDay(c.CreatedAt)
	if q.FromDate != "" && day < q.FromDate {
		return false
	}
	if q.ToDate != "" && day > q.ToDate {
		return false
	}
	return true
}

func chargeDay(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	tt := t.In(chargeZone)
	return fmt.Sprintf("%04d-%02d-%02d", tt.Year(), int(tt.Month()), tt.Day())
}

// PayCharge sets paid_at, stores the CHARGE payment, and marks the order charged under one lock. 2026-10-12
func (s *Store) PayCharge(_ context.Context, chargeID string, payment domain.Payment) (*domain.Charge, *domain.Payment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.charges[chargeID]
	if !ok || cur.IsDeleted {
		return nil, nil, port.ErrNotFound
	}
	if cur.Status == domain.ChargeStatusAbate {
		return nil, nil, port.ErrChargeAlreadyAbated
	}
	if !cur.PaidAt.IsZero() {
		return nil, nil, port.ErrChargeAlreadyPaid
	}
	if err := reservePaymentLocked(s, payment); err != nil {
		return nil, nil, err
	}
	if cur.OrderID != "" {
		if o, ok := s.orders[cur.OrderID]; !ok || o.IsDeleted {
			return nil, nil, port.ErrNotFound
		}
	}
	cur.Status = domain.ChargeStatusNormal
	cur.PaidAt = payment.PaidAt
	cur.UpdatedAt = payment.UpdatedAt
	cur.UpdatedBy = payment.UpdatedBy
	s.charges[chargeID] = cur
	s.payments[payment.PaymentID] = payment
	if cur.OrderID != "" {
		o := s.orders[cur.OrderID]
		o.Charged = true
		o.UpdatedAt = payment.UpdatedAt
		o.UpdatedBy = payment.UpdatedBy
		s.orders[cur.OrderID] = o
	}
	out := cloneCharge(cur)
	pay := payment
	return &out, &pay, nil
}

// RefundCharge abates old, optionally inserts the remainder square, and stores the REFUND payment. 2026-10-12
func (s *Store) RefundCharge(_ context.Context, old domain.Charge, newCharge *domain.Charge, payment domain.Payment) (*domain.Charge, *domain.Charge, *domain.Payment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.charges[old.ChargeID]
	if !ok || cur.IsDeleted {
		return nil, nil, nil, port.ErrNotFound
	}
	if cur.Status == domain.ChargeStatusAbate {
		return nil, nil, nil, port.ErrChargeAlreadyAbated
	}
	if cur.PaidAt.IsZero() {
		return nil, nil, nil, port.ErrChargeNotPaid
	}
	if newCharge != nil {
		if _, exists := s.charges[newCharge.ChargeID]; exists {
			return nil, nil, nil, port.ErrDuplicateBusinessNo
		}
		for _, ex := range s.charges {
			if !ex.IsDeleted && ex.SquareNo == newCharge.SquareNo {
				return nil, nil, nil, port.ErrDuplicateBusinessNo
			}
		}
	}
	if err := reservePaymentLocked(s, payment); err != nil {
		return nil, nil, nil, err
	}
	old.CreatedAt = cur.CreatedAt
	old.CreatedBy = cur.CreatedBy
	if old.PaidAt.IsZero() {
		old.PaidAt = cur.PaidAt
	}
	s.charges[old.ChargeID] = cloneCharge(old)
	var savedNew *domain.Charge
	if newCharge != nil {
		cp := cloneCharge(*newCharge)
		s.charges[newCharge.ChargeID] = cp
		out := cloneCharge(cp)
		savedNew = &out
	}
	s.payments[payment.PaymentID] = payment
	outOld := cloneCharge(s.charges[old.ChargeID])
	pay := payment
	return &outOld, savedNew, &pay, nil
}

func reservePaymentLocked(s *Store, payment domain.Payment) error {
	if _, ok := s.payments[payment.PaymentID]; ok {
		return port.ErrDuplicateBusinessNo
	}
	for _, ex := range s.payments {
		if ex.PaymentNo == payment.PaymentNo {
			return port.ErrDuplicateBusinessNo
		}
	}
	return nil
}

// DrugOnHandMilli reads inventory_item.qty_on_hand. Missing or inactive items are 0.
// Charge uses this as a check only. 2026-10-12
func (s *Store) DrugOnHandMilli(_ context.Context, itemNo string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	q, ok := s.onHandLocked(itemNo)
	if !ok {
		return 0, nil
	}
	return q, nil
}

// SetDrugOnHandMilli overwrites qty_on_hand (quantity × 1000) so charge and dispense share one row.
// milli 0 makes the next drug charge fail. Not an HTTP API. 2026-10-12
func (s *Store) SetDrugOnHandMilli(itemNo string, milli int64) error {
	if itemNo == "" || milli < 0 {
		return fmt.Errorf("invalid on-hand")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.inventory[itemNo]
	if !ok {
		it = domain.InventoryItem{
			SkuID: "sku-" + itemNo, ItemNo: itemNo, ItemName: itemNo, Units: "盒",
			Status: domain.InventoryStatusActive,
		}
	}
	it.QtyOnHandMilli = milli
	it.IsDeleted = false
	it.Status = domain.InventoryStatusActive
	if it.RowVersion == "" {
		it.RowVersion = "1"
	} else {
		it.RowVersion = bumpRowVersion(it.RowVersion)
	}
	s.inventory[itemNo] = it
	return nil
}

// DispenseLedger prefers real dispense_item totals. The test stub applies only when no slip exists.
// 2026-10-12
func (s *Store) DispenseLedger(_ context.Context, chargeItemID string) (int64, int64, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	disp, ret, n := s.sumDispenseLocked(func(it domain.DispenseItem) bool {
		return it.ChargeItemID == chargeItemID
	})
	if n > 0 {
		return disp, ret, true, nil
	}
	rec, ok := s.dispenseStub[chargeItemID]
	if !ok {
		return 0, 0, false, nil
	}
	return rec.dispensed, rec.returned, true, nil
}

// SetDispenseMilli records a charge-item stub when the caller is not going through dispense HTTP.
// Real slips, when present, hide this stub. 2026-10-12
func (s *Store) SetDispenseMilli(chargeItemID string, dispensedMilli, returnedMilli int64) error {
	if chargeItemID == "" || dispensedMilli < 0 || returnedMilli < 0 || returnedMilli > dispensedMilli {
		return fmt.Errorf("invalid dispense qty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := findChargeItemLocked(s, chargeItemID); !ok {
		return port.ErrNotFound
	}
	if s.dispenseStub == nil {
		s.dispenseStub = map[string]dispenseRec{}
	}
	s.dispenseStub[chargeItemID] = dispenseRec{dispensed: dispensedMilli, returned: returnedMilli}
	return nil
}

// SetChargeItemCanBack flips can_back for the ITEM_NOT_CAN_BACK fixture. Not an HTTP API. 2026-10-12
func (s *Store) SetChargeItemCanBack(chargeItemID string, canBack bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, idx, ok := findChargeItemIndexLocked(s, chargeItemID)
	if !ok {
		return port.ErrNotFound
	}
	c.Items[idx].CanBack = canBack
	s.charges[c.ChargeID] = c
	return nil
}

func findChargeItemLocked(s *Store, chargeItemID string) (domain.ChargeItem, bool) {
	c, idx, ok := findChargeItemIndexLocked(s, chargeItemID)
	if !ok {
		return domain.ChargeItem{}, false
	}
	return c.Items[idx], true
}

func findChargeItemIndexLocked(s *Store, chargeItemID string) (domain.Charge, int, bool) {
	for _, c := range s.charges {
		if c.IsDeleted {
			continue
		}
		for i := range c.Items {
			if c.Items[i].ChargeItemID == chargeItemID {
				return c, i, true
			}
		}
	}
	return domain.Charge{}, 0, false
}

func cloneCharge(c domain.Charge) domain.Charge {
	if c.Items != nil {
		items := make([]domain.ChargeItem, len(c.Items))
		copy(items, c.Items)
		c.Items = items
	} else {
		c.Items = []domain.ChargeItem{}
	}
	return c
}
