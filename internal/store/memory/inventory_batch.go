package memory

import (
	"context"
	"sort"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

var _ port.InventoryBatchStore = (*Store)(nil)

// AddInventoryBatch is a test/fixture helper. There is intentionally no HTTP
// mutation path for isolated batch snapshots.
func (s *Store) AddInventoryBatch(row domain.InventoryBatch) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inventoryBatches[row.BatchID] = row
}

func (s *Store) ListInventoryBatches(_ context.Context, query port.InventoryBatchQuery) ([]domain.InventoryBatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := make([]domain.InventoryBatch, 0, len(s.inventoryBatches))
	for _, row := range s.inventoryBatches {
		if query.CursorBatchID != "" && row.BatchID <= query.CursorBatchID {
			continue
		}
		if query.SkuCode != "" && row.ItemNo != query.SkuCode {
			continue
		}
		if query.StorageUnitID != "" && row.StorageUnitID != query.StorageUnitID {
			continue
		}
		if query.BatchNo != "" && row.BatchNo != query.BatchNo {
			continue
		}
		row.ExpiryStatus = domain.ClassifyExpiry(row.ExpireDate, query.AsOfDate, query.ExpiryWindowDays)
		if query.ExpiryStatus != "" && row.ExpiryStatus != query.ExpiryStatus {
			continue
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].BatchID < rows[j].BatchID })
	if query.Limit > 0 && len(rows) > query.Limit {
		rows = rows[:query.Limit]
	}
	return rows, nil
}
