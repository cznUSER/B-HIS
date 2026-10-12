package memory

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

// CommitDispense checks stock and available qty, then decrements qty_on_hand, appends a
// negative DISPENSE ledger, stores the slip, and moves the order to PROC. Any refusal
// leaves those rows unchanged. 2026-10-12
func (s *Store) CommitDispense(_ context.Context, d domain.Dispense) (*domain.Dispense, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.dispenses[d.DispenseID]; ok {
		return nil, port.ErrDuplicateBusinessNo
	}
	for _, ex := range s.dispenses {
		if !ex.IsDeleted && ex.DispenseNo == d.DispenseNo {
			return nil, port.ErrDuplicateBusinessNo
		}
	}
	order, ok := s.orders[d.OrderID]
	if !ok || order.IsDeleted {
		return nil, port.ErrNotFound
	}
	if order.Status != domain.OrderStatusCommit && order.Status != domain.OrderStatusProc {
		return nil, port.ErrIllegalStateTransition
	}
	ch, ok := s.charges[d.ChargeID]
	if !ok || ch.IsDeleted || ch.OrderID != d.OrderID || ch.Status != domain.ChargeStatusNormal || ch.PaidAt.IsZero() {
		return nil, port.ErrNotCharged
	}
	if len(d.Items) == 0 {
		return nil, port.ErrDispenseExceeds
	}
	pendingNet := map[string]int64{}
	needs := map[string]int64{}
	for i := range d.Items {
		it := &d.Items[i]
		ci, found := chargeItemOn(ch, it.ChargeItemID)
		if !found || (it.OrderItemID != "" && ci.OrderItemID != "" && it.OrderItemID != ci.OrderItemID) {
			return nil, port.ErrIllegalStateTransition
		}
		if ci.ItemType == domain.ItemTypeTreat || !drugLine(order, ci) {
			return nil, port.ErrIllegalStateTransition
		}
		it.OrderItemID = ci.OrderItemID
		it.ItemNo = ci.ItemNo
		if it.ItemName == "" {
			it.ItemName = ci.ItemName
		}
		it.QtyChargedMilli = ci.QtyMilli
		net := s.netMilliLocked(it.ChargeItemID) + pendingNet[it.ChargeItemID]
		if net >= ci.QtyMilli {
			return nil, port.ErrAlreadyDispensed
		}
		if it.QtyDispensedMilli <= 0 || net+it.QtyDispensedMilli > ci.QtyMilli {
			return nil, port.ErrDispenseExceeds
		}
		pendingNet[it.ChargeItemID] += it.QtyDispensedMilli
		needs[it.ItemNo] += it.QtyDispensedMilli
		it.LineStatus = domain.LineStatus(ci.QtyMilli, net+it.QtyDispensedMilli)
		it.QtyReturnedMilli = 0
	}
	for itemNo, qty := range needs {
		on, ok := s.onHandLocked(itemNo)
		if !ok || qty > on {
			return nil, port.ErrInsufficientStock
		}
	}
	now := d.UpdatedAt
	if now.IsZero() {
		now = d.DispensedAt
	}
	for itemNo, qty := range needs {
		row := s.inventory[itemNo]
		row.QtyOnHandMilli -= qty
		row.RowVersion = bumpRowVersion(row.RowVersion)
		row.UpdatedAt = now
		row.UpdatedBy = d.UpdatedBy
		s.inventory[itemNo] = row
	}
	for _, it := range d.Items {
		s.ledgers = append(s.ledgers, domain.InventoryLedger{
			LedgerID: it.DispenseItemID + ":dispense", SkuID: s.inventory[it.ItemNo].SkuID,
			QtyDeltaMilli: -it.QtyDispensedMilli, BizType: domain.LedgerBizDispense, BizID: d.DispenseID,
			Remark: it.DispenseItemID, CreatedAt: now, CreatedBy: d.CreatedBy,
		})
	}
	d.Status = domain.DispenseStatusDone
	stored := cloneDispense(d)
	s.dispenses[d.DispenseID] = stored
	order.Status = domain.OrderStatusProc
	order.UpdatedAt = now
	order.UpdatedBy = d.UpdatedBy
	s.orders[d.OrderID] = order
	out := cloneDispense(stored)
	return &out, nil
}

// CommitReturn credits stock, appends a positive RETURN ledger, and updates qty_returned
// plus the slip status. A refused line changes nothing. 2026-10-12
func (s *Store) CommitReturn(_ context.Context, dispenseID, reason, operatorID string, at time.Time, lines []port.ReturnLine) (*domain.Dispense, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored, ok := s.dispenses[dispenseID]
	if !ok || stored.IsDeleted {
		return nil, port.ErrNotFound
	}
	if len(lines) == 0 {
		return nil, port.ErrNotDispensed
	}
	cur := cloneDispense(stored)
	index := map[string]int{}
	for i, it := range cur.Items {
		index[it.DispenseItemID] = i
	}
	seen := map[string]bool{}
	type applied struct {
		idx   int
		milli int64
	}
	plan := make([]applied, 0, len(lines))
	needs := map[string]int64{}
	for _, ln := range lines {
		if ln.DispenseItemID == "" || seen[ln.DispenseItemID] {
			return nil, port.ErrNotDispensed
		}
		seen[ln.DispenseItemID] = true
		idx, found := index[ln.DispenseItemID]
		if !found {
			return nil, port.ErrNotDispensed
		}
		it := cur.Items[idx]
		remain := it.QtyDispensedMilli - it.QtyReturnedMilli
		if remain <= 0 {
			return nil, port.ErrNotDispensed
		}
		if ln.ReturnMilli <= 0 || ln.ReturnMilli > remain {
			return nil, port.ErrReturnExceeds
		}
		if _, ok := s.onHandLocked(it.ItemNo); !ok {
			return nil, port.ErrInsufficientStock
		}
		plan = append(plan, applied{idx: idx, milli: ln.ReturnMilli})
		needs[it.ItemNo] += ln.ReturnMilli
	}
	now := at
	if now.IsZero() {
		now = time.Now().UTC()
	}
	for itemNo, qty := range needs {
		row := s.inventory[itemNo]
		row.QtyOnHandMilli += qty
		row.RowVersion = bumpRowVersion(row.RowVersion)
		row.UpdatedAt = now
		row.UpdatedBy = operatorID
		s.inventory[itemNo] = row
	}
	// Map still has the pre-return quantities. adjust is this transaction's extra returns.
	adjust := map[string]int64{}
	for _, step := range plan {
		it := &cur.Items[step.idx]
		newNet := s.netMilliLocked(it.ChargeItemID) - adjust[it.ChargeItemID] - step.milli
		it.QtyReturnedMilli += step.milli
		it.LineStatus = domain.LineStatus(it.QtyChargedMilli, newNet)
		it.UpdatedAt = now
		adjust[it.ChargeItemID] += step.milli
		s.ledgers = append(s.ledgers, domain.InventoryLedger{
			LedgerID: it.DispenseItemID + ":return:" + strconv.Itoa(len(s.ledgers)),
			SkuID:    s.inventory[it.ItemNo].SkuID, QtyDeltaMilli: step.milli,
			BizType: domain.LedgerBizReturn, BizID: cur.DispenseID, Remark: it.DispenseItemID,
			CreatedAt: now, CreatedBy: operatorID,
		})
	}
	cur.Status = domain.HeaderStatus(cur.Items)
	cur.ReturnReason = reason
	cur.UpdatedAt = now
	cur.UpdatedBy = operatorID
	s.dispenses[cur.DispenseID] = cloneDispense(cur)
	out := cloneDispense(cur)
	return &out, nil
}

func (s *Store) GetDispense(_ context.Context, dispenseID string) (*domain.Dispense, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.dispenses[dispenseID]
	if !ok || d.IsDeleted {
		return nil, port.ErrNotFound
	}
	out := cloneDispense(d)
	return &out, nil
}

func (s *Store) SearchDispenses(_ context.Context, q port.DispenseQuery) ([]domain.Dispense, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	out := make([]domain.Dispense, 0)
	for _, d := range s.dispenses {
		if !dispenseMatches(d, q) {
			continue
		}
		out = append(out, cloneDispense(d))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DispensedAt.Equal(out[j].DispensedAt) {
			return out[i].DispenseID < out[j].DispenseID
		}
		return out[i].DispensedAt.Before(out[j].DispensedAt)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *Store) ListInventory(_ context.Context, itemNo string) ([]domain.InventoryItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.InventoryItem, 0)
	for _, it := range s.inventory {
		if it.IsDeleted {
			continue
		}
		if itemNo != "" && it.ItemNo != itemNo {
			continue
		}
		out = append(out, it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ItemNo < out[j].ItemNo })
	return out, nil
}

func (s *Store) QtyOnHandMilli(ctx context.Context, itemNo string) (int64, error) {
	return s.DrugOnHandMilli(ctx, itemNo)
}

func (s *Store) OrderHasDispense(_ context.Context, orderID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range s.dispenses {
		if !d.IsDeleted && d.OrderID == orderID {
			return true, nil
		}
	}
	return false, nil
}

func (s *Store) NetByChargeItem(ctx context.Context, chargeItemID string) (int64, int64, bool, error) {
	return s.DispenseLedger(ctx, chargeItemID)
}

func (s *Store) NetByOrderItem(_ context.Context, orderItemID string) (int64, int64, bool, error) {
	if orderItemID == "" {
		return 0, 0, false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	disp, ret, n := s.sumDispenseLocked(func(it domain.DispenseItem) bool {
		return it.OrderItemID == orderItemID
	})
	if n == 0 {
		return 0, 0, false, nil
	}
	return disp, ret, true, nil
}

func (s *Store) onHandLocked(itemNo string) (int64, bool) {
	it, ok := s.inventory[itemNo]
	if !ok || it.IsDeleted || it.Status != domain.InventoryStatusActive {
		return 0, false
	}
	return it.QtyOnHandMilli, true
}

func (s *Store) sumDispenseLocked(match func(domain.DispenseItem) bool) (disp, ret int64, n int) {
	for _, d := range s.dispenses {
		if d.IsDeleted {
			continue
		}
		for _, it := range d.Items {
			if match(it) {
				disp += it.QtyDispensedMilli
				ret += it.QtyReturnedMilli
				n++
			}
		}
	}
	return disp, ret, n
}

func (s *Store) netMilliLocked(chargeItemID string) int64 {
	disp, ret, _ := s.sumDispenseLocked(func(it domain.DispenseItem) bool {
		return it.ChargeItemID == chargeItemID
	})
	return disp - ret
}

func chargeItemOn(ch domain.Charge, chargeItemID string) (domain.ChargeItem, bool) {
	for _, it := range ch.Items {
		if it.ChargeItemID == chargeItemID {
			return it, true
		}
	}
	return domain.ChargeItem{}, false
}

func drugLine(order domain.Order, ci domain.ChargeItem) bool {
	if ci.ItemType == domain.ItemTypeDrug {
		return true
	}
	if ci.ItemType == domain.ItemTypeTreat {
		return false
	}
	for _, oi := range order.Items {
		if oi.OrderItemID != "" && oi.OrderItemID == ci.OrderItemID {
			return oi.ItemType == domain.ItemTypeDrug
		}
	}
	return ci.ItemNo != ""
}

func dispenseMatches(d domain.Dispense, q port.DispenseQuery) bool {
	if d.IsDeleted {
		return false
	}
	if q.PatientID != "" && d.PatientID != q.PatientID {
		return false
	}
	if q.RegisterID != "" && d.RegisterID != q.RegisterID {
		return false
	}
	if q.ChargeID != "" && d.ChargeID != q.ChargeID {
		return false
	}
	if q.OrderID != "" && d.OrderID != q.OrderID {
		return false
	}
	day := chargeDay(d.DispensedAt)
	if q.FromDate != "" && day < q.FromDate {
		return false
	}
	if q.ToDate != "" && day > q.ToDate {
		return false
	}
	return true
}

func cloneDispense(d domain.Dispense) domain.Dispense {
	if d.Items != nil {
		items := make([]domain.DispenseItem, len(d.Items))
		copy(items, d.Items)
		d.Items = items
	}
	return d
}
