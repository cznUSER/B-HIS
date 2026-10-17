package mssql

import (
	"context"
	"database/sql"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

var _ port.DispenseStore = (*Store)(nil)

func (s *Store) ListInventoryLedger(ctx context.Context, query port.InventoryLedgerQuery) ([]domain.InventoryLedgerEntry, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		;WITH ledger_balance AS (
			SELECT l.ledger_id, l.sku_id, i.item_no AS sku_code, l.biz_type, l.biz_id, l.remark,
				CONVERT(bigint, l.qty_delta * 1000) AS qty_delta_milli,
				CONVERT(bigint, (
					i.qty_on_hand - COALESCE(SUM(l.qty_delta) OVER (
						PARTITION BY l.sku_id
						ORDER BY l.created_at DESC, l.ledger_id DESC
						ROWS BETWEEN UNBOUNDED PRECEDING AND 1 PRECEDING
					), 0)
				) * 1000) AS qty_after_milli,
				l.created_at,
				COALESCE(NULLIF(u.display_name, N''), NULLIF(l.created_by, ''), N'') AS operator_name
			FROM dbo.inventory_ledger AS l
			JOIN dbo.inventory_item AS i ON i.sku_id = l.sku_id AND i.is_deleted = 0
			LEFT JOIN dbo.app_user AS u ON u.user_id = l.created_by AND u.is_deleted = 0
		)
		SELECT TOP (@limit) ledger_id, sku_id, sku_code, biz_type, biz_id, remark,
			qty_delta_milli, qty_after_milli, created_at, operator_name
		FROM ledger_balance
		WHERE (@sku_code IS NULL OR sku_code = @sku_code)
		  AND (@biz_type IS NULL OR biz_type = @biz_type)
		  AND (@has_from = 0 OR created_at >= @from_at)
		  AND (@has_to = 0 OR created_at < @to_at)
		  AND (@has_cursor = 0 OR created_at < @cursor_at OR (created_at = @cursor_at AND ledger_id < @cursor_ledger_id))
		ORDER BY created_at DESC, ledger_id DESC`,
		sql.Named("limit", limit),
		sql.Named("sku_code", nullIfEmpty(query.SkuCode)),
		sql.Named("biz_type", nullIfEmpty(query.BizType)),
		sql.Named("has_from", query.HasFrom),
		sql.Named("from_at", query.FromAt.UTC()),
		sql.Named("has_to", query.HasTo),
		sql.Named("to_at", query.ToAt.UTC()),
		sql.Named("has_cursor", query.HasCursor),
		sql.Named("cursor_at", query.CursorAt.UTC()),
		sql.Named("cursor_ledger_id", nullIfEmpty(query.CursorLedgerID)),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.InventoryLedgerEntry, 0)
	for rows.Next() {
		var item domain.InventoryLedgerEntry
		var bizID, remark sql.NullString
		if err := rows.Scan(
			&item.LedgerID, &item.SkuID, &item.SkuCode, &item.BizType, &bizID, &remark,
			&item.QtyDeltaMilli, &item.QtyAfterMilli, &item.CreatedAt, &item.OperatorName,
		); err != nil {
			return nil, err
		}
		item.BizID = bizID.String
		item.Remark = remark.String
		item.CreatedAt = item.CreatedAt.UTC()
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
