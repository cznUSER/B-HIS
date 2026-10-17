package mssql

import (
	"context"
	"database/sql"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

var _ port.InventoryBatchStore = (*Store)(nil)

// ListInventoryBatches reads only dbo.inventory_batch, an isolated snapshot
// table. It never joins or updates inventory_item/ledger/dispense tables.
func (s *Store) ListInventoryBatches(ctx context.Context, query port.InventoryBatchQuery) ([]domain.InventoryBatch, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT batch_id, sku_id, item_no, item_name, storage_unit_id, batch_no,
			product_no, supplier_id, product_date, expire_date,
			CONVERT(varchar(40), qty_on_hand), units, source_ref, row_version
		FROM dbo.inventory_batch
		WHERE is_deleted = 0
		  AND (@sku_code IS NULL OR item_no = @sku_code)
		  AND (@storage_unit_id IS NULL OR storage_unit_id = @storage_unit_id)
		  AND (@batch_no IS NULL OR batch_no = @batch_no)
		ORDER BY batch_id`,
		sql.Named("sku_code", nullIfEmpty(query.SkuCode)),
		sql.Named("storage_unit_id", nullIfEmpty(query.StorageUnitID)),
		sql.Named("batch_no", nullIfEmpty(query.BatchNo)),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.InventoryBatch, 0)
	for rows.Next() {
		var row domain.InventoryBatch
		var storage, batch, product, supplier, source, version sql.NullString
		var productDate, expireDate sql.NullTime
		var qty sql.NullString
		if err := rows.Scan(&row.BatchID, &row.SkuID, &row.ItemNo, &row.ItemName, &storage, &batch, &product, &supplier, &productDate, &expireDate, &qty, &row.Units, &source, &version); err != nil {
			return nil, err
		}
		row.StorageUnitID, row.BatchNo, row.ProductNo, row.SupplierID, row.SourceRef, row.RowVersion = storage.String, batch.String, product.String, supplier.String, source.String, version.String
		if productDate.Valid {
			v := productDate.Time.UTC()
			row.ProductDate = &v
		}
		if expireDate.Valid {
			v := expireDate.Time.UTC()
			row.ExpireDate = &v
		}
		row.QtyOnHandMilli, err = decimalToMilli(qty.String)
		if err != nil {
			return nil, err
		}
		row.ExpiryStatus = domain.ClassifyExpiry(row.ExpireDate, query.AsOfDate, query.ExpiryWindowDays)
		if query.ExpiryStatus != "" && row.ExpiryStatus != query.ExpiryStatus {
			continue
		}
		if query.CursorBatchID != "" && row.BatchID <= query.CursorBatchID {
			continue
		}
		out = append(out, row)
		if query.Limit > 0 && len(out) >= query.Limit {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
