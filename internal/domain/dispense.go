package domain

import "time"

// Dispense and inventory values match OpenAPI 0.3.9 and migrations 006/007.
// Quantities are ×1000 (decimal 18,3). No batch, expiry, or FIFO. 2026-10-12
const (
	InventoryStatusActive   = "ACTIVE"
	InventoryStatusInactive = "INACTIVE"

	DispenseStatusDone            = "DONE"
	DispenseStatusPartialReturned = "PARTIAL_RETURNED"
	DispenseStatusReturned        = "RETURNED"

	LineDispenseNone    = "NONE"
	LineDispensePartial = "PARTIAL"
	LineDispenseFull    = "FULL"

	LedgerBizDispense = "DISPENSE"
	LedgerBizReturn   = "RETURN"
	LedgerBizAdjust   = "ADJUST"
)

// InventoryItem is dbo.inventory_item. ItemNo ≡ skuCode (DRUG-A / DRUG-B). TREAT is not stored.
type InventoryItem struct {
	SkuID          string
	ItemNo         string
	ItemName       string
	QtyOnHandMilli int64
	Units          string
	Status         string
	RowVersion     string
	CreatedAt      time.Time
	CreatedBy      string
	UpdatedAt      time.Time
	UpdatedBy      string
	IsDeleted      bool
}

// InventoryLedger is an append-only qty_delta. DISPENSE is negative, RETURN is positive.
type InventoryLedger struct {
	LedgerID      string
	SkuID         string
	QtyDeltaMilli int64
	BizType       string
	BizID         string
	Remark        string
	CreatedAt     time.Time
	CreatedBy     string
}

// DispenseItem is one line of a dispense slip. Qty*Milli is quantity × 1000.
// LineStatus is a snapshot; readers recompute NONE|PARTIAL|FULL from net dispensed.
type DispenseItem struct {
	DispenseItemID    string
	DispenseID        string
	OrderItemID       string
	ChargeItemID      string
	ItemNo            string
	ItemName          string
	Units             string
	QtyChargedMilli   int64
	QtyDispensedMilli int64
	QtyReturnedMilli  int64
	LineStatus        string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Dispense is dbo.dispense. Status is this slip, not the order.
type Dispense struct {
	DispenseID   string
	DispenseNo   string
	OrderID      string
	ChargeID     string
	RegisterID   string
	PatientID    string
	Status       string
	DispensedAt  time.Time
	OperatorID   string
	DeptID       string
	ReturnReason string
	Items        []DispenseItem
	CreatedAt    time.Time
	CreatedBy    string
	UpdatedAt    time.Time
	UpdatedBy    string
	IsDeleted    bool
}

// LineStatus is NONE / PARTIAL / FULL from charged qty and net dispensed (dispensed − returned).
func LineStatus(chargedMilli, netMilli int64) string {
	if netMilli <= 0 {
		return LineDispenseNone
	}
	if chargedMilli > 0 && netMilli >= chargedMilli {
		return LineDispenseFull
	}
	return LineDispensePartial
}

// HeaderStatus is DONE, PARTIAL_RETURNED, or RETURNED for one slip.
func HeaderStatus(items []DispenseItem) string {
	if len(items) == 0 {
		return DispenseStatusDone
	}
	allReturned := true
	anyReturned := false
	for _, it := range items {
		if it.QtyReturnedMilli > 0 {
			anyReturned = true
		}
		if it.QtyReturnedMilli < it.QtyDispensedMilli {
			allReturned = false
		}
	}
	if allReturned {
		return DispenseStatusReturned
	}
	if anyReturned {
		return DispenseStatusPartialReturned
	}
	return DispenseStatusDone
}
