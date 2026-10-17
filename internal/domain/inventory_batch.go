package domain

import "time"

const (
	PermInventoryBatchRead = "inventory:batch-read"

	ExpiryStatusUnknown  = "UNKNOWN"
	ExpiryStatusValid    = "VALID"
	ExpiryStatusExpiring = "EXPIRING"
	ExpiryStatusExpired  = "EXPIRED"
)

// ClassifyExpiry is shared by memory and SQL stores so a fixed as-of date has
// identical boundary behavior in tests and production reads.
func ClassifyExpiry(expireDate *time.Time, asOfDate time.Time, windowDays int) string {
	if expireDate == nil {
		return ExpiryStatusUnknown
	}
	date := dateOnly(*expireDate)
	asOf := dateOnly(asOfDate)
	windowEnd := asOf.AddDate(0, 0, windowDays)
	switch {
	case date.Before(asOf):
		return ExpiryStatusExpired
	case !date.After(windowEnd):
		return ExpiryStatusExpiring
	default:
		return ExpiryStatusValid
	}
}

func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

// InventoryBatch is an isolated, read-only batch snapshot. It does not map to
// inventory_item and is intentionally not consumed by dispense/return logic.
type InventoryBatch struct {
	BatchID        string
	SkuID          string
	ItemNo         string
	ItemName       string
	StorageUnitID  string
	BatchNo        string
	ProductNo      string
	SupplierID     string
	ProductDate    *time.Time
	ExpireDate     *time.Time
	QtyOnHandMilli int64
	Units          string
	SourceRef      string
	ExpiryStatus   string
	RowVersion     string
}
