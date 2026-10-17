package port

import (
	"context"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
)

// InventoryBatchQuery describes the read-only batch snapshot query. Cursor is
// ordered by batch_id and is opaque to callers at the HTTP boundary.
type InventoryBatchQuery struct {
	SkuCode          string
	StorageUnitID    string
	BatchNo          string
	ExpiryStatus     string
	AsOfDate         time.Time
	ExpiryWindowDays int
	CursorBatchID    string
	Limit            int
}

// InventoryBatchStore is deliberately separate from DispenseStore: batch
// snapshots must not be used by inventory/dispense mutation paths.
type InventoryBatchStore interface {
	ListInventoryBatches(ctx context.Context, query InventoryBatchQuery) ([]domain.InventoryBatch, error)
}
