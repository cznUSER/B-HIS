package mssql

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

// SQL follows migrations/sqlserver/core/006_inventory and 007_dispense. Do not add columns. 2026-10-12

const dispenseCols = `dispense_id, dispense_no, order_id, charge_id, register_id, patient_id,
	status, dispensed_at, operator_id, dept_id, return_reason, created_at, created_by, updated_at, updated_by`

const dispenseItemCols = `dispense_item_id, dispense_id, order_item_id, charge_item_id, item_no, item_name, units,
	CONVERT(varchar(40), qty_charged), CONVERT(varchar(40), qty_dispensed), CONVERT(varchar(40), qty_returned), line_status,
	created_at, updated_at`

const sumChargeItemSQL = `
	SELECT COUNT(1),
		CONVERT(varchar(40), COALESCE(SUM(qty_dispensed), 0)),
		CONVERT(varchar(40), COALESCE(SUM(qty_returned), 0))
	FROM dbo.dispense_item WHERE charge_item_id = @id`

const sumOrderItemSQL = `
	SELECT COUNT(1),
		CONVERT(varchar(40), COALESCE(SUM(qty_dispensed), 0)),
		CONVERT(varchar(40), COALESCE(SUM(qty_returned), 0))
	FROM dbo.dispense_item WHERE order_item_id = @id`

// CommitDispense is one transaction: check on-hand and available qty, decrement, ledger DISPENSE
// (negative), insert the slip, and set the order to PROC. A refusal rolls back. 2026-10-12
func (s *Store) CommitDispense(ctx context.Context, d domain.Dispense) (*domain.Dispense, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var orderStatus string
	err = tx.QueryRowContext(ctx, `
		SELECT status FROM dbo.order_sheet WITH (UPDLOCK, ROWLOCK)
		WHERE order_id = @order_id AND is_deleted = 0`,
		sql.Named("order_id", d.OrderID),
	).Scan(&orderStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if orderStatus != domain.OrderStatusCommit && orderStatus != domain.OrderStatusProc {
		return nil, port.ErrIllegalStateTransition
	}
	var chargeStatus, chargeOrder string
	var paid sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT status, order_id, paid_at FROM dbo.charge WITH (UPDLOCK, ROWLOCK)
		WHERE charge_id = @charge_id AND is_deleted = 0`,
		sql.Named("charge_id", d.ChargeID),
	).Scan(&chargeStatus, &chargeOrder, &paid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotCharged
	}
	if err != nil {
		return nil, err
	}
	if chargeStatus != domain.ChargeStatusNormal || !paid.Valid || chargeOrder != d.OrderID {
		return nil, port.ErrNotCharged
	}
	if len(d.Items) == 0 {
		return nil, port.ErrDispenseExceeds
	}
	pending := map[string]int64{}
	needs := map[string]int64{}
	for i := range d.Items {
		it := &d.Items[i]
		var orderItemID, itemNo, itemName sql.NullString
		var qtyS sql.NullString
		err = tx.QueryRowContext(ctx, `
			SELECT order_item_id, item_no, item_name, CONVERT(varchar(40), qty)
			FROM dbo.charge_item
			WHERE charge_item_id = @charge_item_id AND charge_id = @charge_id`,
			sql.Named("charge_item_id", it.ChargeItemID),
			sql.Named("charge_id", d.ChargeID),
		).Scan(&orderItemID, &itemNo, &itemName, &qtyS)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, port.ErrIllegalStateTransition
		}
		if err != nil {
			return nil, err
		}
		charged, err := decimalToMilli(qtyS.String)
		if err != nil {
			return nil, err
		}
		if it.OrderItemID != "" && orderItemID.String != "" && it.OrderItemID != orderItemID.String {
			return nil, port.ErrIllegalStateTransition
		}
		kind, err := orderItemType(ctx, tx, orderItemID.String)
		if err != nil {
			return nil, err
		}
		if kind != domain.ItemTypeDrug {
			return nil, port.ErrIllegalStateTransition
		}
		disp, ret, _, err := sumDispense(ctx, tx, sumChargeItemSQL, it.ChargeItemID)
		if err != nil {
			return nil, err
		}
		net := disp - ret + pending[it.ChargeItemID]
		if net >= charged {
			return nil, port.ErrAlreadyDispensed
		}
		if it.QtyDispensedMilli <= 0 || net+it.QtyDispensedMilli > charged {
			return nil, port.ErrDispenseExceeds
		}
		pending[it.ChargeItemID] += it.QtyDispensedMilli
		it.OrderItemID = orderItemID.String
		it.ItemNo = itemNo.String
		if it.ItemName == "" {
			it.ItemName = itemName.String
		}
		it.QtyChargedMilli = charged
		it.QtyReturnedMilli = 0
		it.LineStatus = domain.LineStatus(charged, net+it.QtyDispensedMilli)
		needs[it.ItemNo] += it.QtyDispensedMilli
	}
	skus, err := lockAndCheckStock(ctx, tx, needs, false)
	if err != nil {
		return nil, err
	}
	now := d.UpdatedAt
	if now.IsZero() {
		now = d.DispensedAt
	}
	if err := applyStock(ctx, tx, skus, needs, false, now, d.UpdatedBy); err != nil {
		return nil, err
	}
	for _, it := range d.Items {
		if err := insertLedger(ctx, tx, domain.InventoryLedger{
			LedgerID: it.DispenseItemID + ":dispense", SkuID: skus[it.ItemNo],
			QtyDeltaMilli: -it.QtyDispensedMilli, BizType: domain.LedgerBizDispense, BizID: d.DispenseID,
			Remark: it.DispenseItemID, CreatedAt: now, CreatedBy: d.CreatedBy,
		}); err != nil {
			return nil, err
		}
	}
	d.Status = domain.DispenseStatusDone
	if err := insertDispense(ctx, tx, d); err != nil {
		return nil, classifyWriteError(err)
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE dbo.order_sheet
		SET status = @status, updated_at = @updated_at, updated_by = @updated_by
		WHERE order_id = @order_id AND is_deleted = 0 AND status IN ('COMMIT', 'PROC')`,
		sql.Named("status", domain.OrderStatusProc),
		sql.Named("updated_at", now.UTC()),
		sql.Named("updated_by", nullIfEmpty(d.UpdatedBy)),
		sql.Named("order_id", d.OrderID),
	)
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, port.ErrIllegalStateTransition
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	out := d
	return &out, nil
}

// CommitReturn credits qty_on_hand, writes a positive RETURN ledger, and updates the slip. 2026-10-12
func (s *Store) CommitReturn(ctx context.Context, dispenseID, reason, operatorID string, at time.Time, lines []port.ReturnLine) (*domain.Dispense, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	cur, err := loadDispense(ctx, tx, dispenseID, true)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, port.ErrNotDispensed
	}
	index := map[string]int{}
	for i, it := range cur.Items {
		index[it.DispenseItemID] = i
	}
	seen := map[string]bool{}
	needs := map[string]int64{}
	type step struct {
		idx   int
		milli int64
	}
	plan := make([]step, 0, len(lines))
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
		needs[it.ItemNo] += ln.ReturnMilli
		plan = append(plan, step{idx: idx, milli: ln.ReturnMilli})
	}
	skus, err := lockAndCheckStock(ctx, tx, needs, true)
	if err != nil {
		return nil, err
	}
	now := at
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if err := applyStock(ctx, tx, skus, needs, true, now, operatorID); err != nil {
		return nil, err
	}
	for i, step := range plan {
		it := &cur.Items[step.idx]
		disp, ret, _, err := sumDispense(ctx, tx, sumChargeItemSQL, it.ChargeItemID)
		if err != nil {
			return nil, err
		}
		// Previous lines in this transaction are already updated, so ret includes them.
		// This row still has its old qty_returned.
		newNet := disp - ret - step.milli
		it.QtyReturnedMilli += step.milli
		it.LineStatus = domain.LineStatus(it.QtyChargedMilli, newNet)
		it.UpdatedAt = now
		if _, err := tx.ExecContext(ctx, `
			UPDATE dbo.dispense_item
			SET qty_returned = @qty_returned, line_status = @line_status, updated_at = @updated_at
			WHERE dispense_item_id = @dispense_item_id`,
			sql.Named("qty_returned", milliToDecimal(it.QtyReturnedMilli)),
			sql.Named("line_status", it.LineStatus),
			sql.Named("updated_at", now.UTC()),
			sql.Named("dispense_item_id", it.DispenseItemID),
		); err != nil {
			return nil, err
		}
		if err := insertLedger(ctx, tx, domain.InventoryLedger{
			LedgerID: it.DispenseItemID + ":return:" + itoa(i) + ":" + itoa(int(now.UnixNano()%1000000)),
			SkuID:    skus[it.ItemNo], QtyDeltaMilli: step.milli, BizType: domain.LedgerBizReturn,
			BizID: cur.DispenseID, Remark: it.DispenseItemID, CreatedAt: now, CreatedBy: operatorID,
		}); err != nil {
			return nil, err
		}
	}
	cur.Status = domain.HeaderStatus(cur.Items)
	cur.ReturnReason = reason
	cur.UpdatedAt = now
	cur.UpdatedBy = operatorID
	if _, err := tx.ExecContext(ctx, `
		UPDATE dbo.dispense
		SET status = @status, return_reason = @return_reason, updated_at = @updated_at, updated_by = @updated_by
		WHERE dispense_id = @dispense_id AND is_deleted = 0`,
		sql.Named("status", cur.Status),
		sql.Named("return_reason", reason),
		sql.Named("updated_at", now.UTC()),
		sql.Named("updated_by", nullIfEmpty(operatorID)),
		sql.Named("dispense_id", cur.DispenseID),
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return cur, nil
}

func (s *Store) GetDispense(ctx context.Context, dispenseID string) (*domain.Dispense, error) {
	return loadDispense(ctx, s.db, dispenseID, false)
}

func (s *Store) SearchDispenses(ctx context.Context, q port.DispenseQuery) ([]domain.Dispense, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT TOP (@limit) `+dispenseCols+`
		FROM dbo.dispense
		WHERE is_deleted = 0
		  AND (@patient_id IS NULL OR patient_id = @patient_id)
		  AND (@register_id IS NULL OR register_id = @register_id)
		  AND (@charge_id IS NULL OR charge_id = @charge_id)
		  AND (@order_id IS NULL OR order_id = @order_id)
		  AND (@from_date IS NULL OR CONVERT(char(10), DATEADD(hour, 8, dispensed_at), 23) >= @from_date)
		  AND (@to_date IS NULL OR CONVERT(char(10), DATEADD(hour, 8, dispensed_at), 23) <= @to_date)
		ORDER BY dispensed_at, dispense_id`,
		sql.Named("limit", limit),
		sql.Named("patient_id", nullIfEmpty(q.PatientID)),
		sql.Named("register_id", nullIfEmpty(q.RegisterID)),
		sql.Named("charge_id", nullIfEmpty(q.ChargeID)),
		sql.Named("order_id", nullIfEmpty(q.OrderID)),
		sql.Named("from_date", nullIfEmpty(q.FromDate)),
		sql.Named("to_date", nullIfEmpty(q.ToDate)),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Dispense, 0)
	for rows.Next() {
		d, err := scanDispense(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		items, err := listDispenseItems(ctx, s.db, out[i].DispenseID)
		if err != nil {
			return nil, err
		}
		out[i].Items = items
	}
	return out, nil
}

func (s *Store) ListInventory(ctx context.Context, itemNo string) ([]domain.InventoryItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT sku_id, item_no, item_name, CONVERT(varchar(40), qty_on_hand), units, status,
			CONVERT(varchar(30), CONVERT(bigint, row_version)), updated_at
		FROM dbo.inventory_item
		WHERE is_deleted = 0 AND (@item_no IS NULL OR item_no = @item_no)
		ORDER BY item_no`,
		sql.Named("item_no", nullIfEmpty(itemNo)),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.InventoryItem, 0)
	for rows.Next() {
		var it domain.InventoryItem
		var qty sql.NullString
		if err := rows.Scan(&it.SkuID, &it.ItemNo, &it.ItemName, &qty, &it.Units, &it.Status, &it.RowVersion, &it.UpdatedAt); err != nil {
			return nil, err
		}
		milli, err := decimalToMilli(qty.String)
		if err != nil {
			return nil, err
		}
		it.QtyOnHandMilli = milli
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) QtyOnHandMilli(ctx context.Context, itemNo string) (int64, error) {
	return s.DrugOnHandMilli(ctx, itemNo)
}

func (s *Store) OrderHasDispense(ctx context.Context, orderID string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM dbo.dispense WHERE order_id = @order_id AND is_deleted = 0`,
		sql.Named("order_id", orderID),
	).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Store) NetByChargeItem(ctx context.Context, chargeItemID string) (int64, int64, bool, error) {
	return s.DispenseLedger(ctx, chargeItemID)
}

func (s *Store) NetByOrderItem(ctx context.Context, orderItemID string) (int64, int64, bool, error) {
	if orderItemID == "" {
		return 0, 0, false, nil
	}
	disp, ret, n, err := sumDispense(ctx, s.db, sumOrderItemSQL, orderItemID)
	if err != nil {
		return 0, 0, false, err
	}
	if n == 0 {
		return 0, 0, false, nil
	}
	return disp, ret, true, nil
}

func (s *Store) sumDispenseItems(ctx context.Context, column, id string) (int64, int64, int, error) {
	switch column {
	case "charge_item_id":
		return sumDispense(ctx, s.db, sumChargeItemSQL, id)
	case "order_item_id":
		return sumDispense(ctx, s.db, sumOrderItemSQL, id)
	default:
		return 0, 0, 0, errors.New("unknown dispense sum column")
	}
}

type rowQuery interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func sumDispense(ctx context.Context, q rowQuery, query, id string) (int64, int64, int, error) {
	var n int
	var dispS, retS sql.NullString
	err := q.QueryRowContext(ctx, query, sql.Named("id", id)).Scan(&n, &dispS, &retS)
	if err != nil {
		return 0, 0, 0, err
	}
	disp, err := decimalToMilli(dispS.String)
	if err != nil {
		return 0, 0, 0, err
	}
	ret, err := decimalToMilli(retS.String)
	if err != nil {
		return 0, 0, 0, err
	}
	return disp, ret, n, nil
}

func orderItemType(ctx context.Context, q rowQuery, orderItemID string) (string, error) {
	if orderItemID == "" {
		return "", port.ErrIllegalStateTransition
	}
	var kind string
	err := q.QueryRowContext(ctx, `
		SELECT item_type FROM dbo.order_item WHERE order_item_id = @order_item_id`,
		sql.Named("order_item_id", orderItemID),
	).Scan(&kind)
	if errors.Is(err, sql.ErrNoRows) {
		return "", port.ErrIllegalStateTransition
	}
	if err != nil {
		return "", err
	}
	return kind, nil
}

// lockAndCheckStock locks inventory rows in item_no order.
// credit==false requires on-hand ≥ need. credit==true only requires the row to exist.
func lockAndCheckStock(ctx context.Context, tx *sql.Tx, needs map[string]int64, credit bool) (map[string]string, error) {
	keys := make([]string, 0, len(needs))
	for itemNo := range needs {
		keys = append(keys, itemNo)
	}
	sort.Strings(keys)
	skus := map[string]string{}
	for _, itemNo := range keys {
		var sku, qtyS string
		err := tx.QueryRowContext(ctx, `
			SELECT sku_id, CONVERT(varchar(40), qty_on_hand)
			FROM dbo.inventory_item WITH (UPDLOCK, ROWLOCK)
			WHERE item_no = @item_no AND is_deleted = 0 AND status = 'ACTIVE'`,
			sql.Named("item_no", itemNo),
		).Scan(&sku, &qtyS)
		if errors.Is(err, sql.ErrNoRows) {
			if credit {
				return nil, port.ErrNotFound
			}
			return nil, port.ErrInsufficientStock
		}
		if err != nil {
			return nil, err
		}
		on, err := decimalToMilli(qtyS)
		if err != nil {
			return nil, err
		}
		if !credit && needs[itemNo] > on {
			return nil, port.ErrInsufficientStock
		}
		skus[itemNo] = sku
	}
	return skus, nil
}

func applyStock(ctx context.Context, tx *sql.Tx, skus map[string]string, needs map[string]int64, credit bool, at time.Time, by string) error {
	for itemNo, qty := range needs {
		op := "qty_on_hand - @qty"
		pred := "AND qty_on_hand >= @qty"
		if credit {
			op = "qty_on_hand + @qty"
			pred = ""
		}
		res, err := tx.ExecContext(ctx, `
			UPDATE dbo.inventory_item
			SET qty_on_hand = `+op+`, updated_at = @updated_at, updated_by = @updated_by
			WHERE sku_id = @sku_id AND is_deleted = 0 `+pred,
			sql.Named("qty", milliToDecimal(qty)),
			sql.Named("updated_at", at.UTC()),
			sql.Named("updated_by", nullIfEmpty(by)),
			sql.Named("sku_id", skus[itemNo]),
		)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return port.ErrInsufficientStock
		}
	}
	return nil
}

func insertLedger(ctx context.Context, tx *sql.Tx, row domain.InventoryLedger) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO dbo.inventory_ledger (
			ledger_id, sku_id, qty_delta, biz_type, biz_id, remark, created_at, created_by
		) VALUES (
			@ledger_id, @sku_id, @qty_delta, @biz_type, @biz_id, @remark, @created_at, @created_by
		)`,
		sql.Named("ledger_id", row.LedgerID),
		sql.Named("sku_id", row.SkuID),
		sql.Named("qty_delta", milliToDecimal(row.QtyDeltaMilli)),
		sql.Named("biz_type", row.BizType),
		sql.Named("biz_id", nullIfEmpty(row.BizID)),
		sql.Named("remark", nullIfEmpty(row.Remark)),
		sql.Named("created_at", row.CreatedAt.UTC()),
		sql.Named("created_by", nullIfEmpty(row.CreatedBy)),
	)
	return err
}

func insertDispense(ctx context.Context, tx *sql.Tx, d domain.Dispense) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO dbo.dispense (
			dispense_id, dispense_no, order_id, charge_id, register_id, patient_id,
			status, dispensed_at, operator_id, dept_id, return_reason,
			created_at, created_by, updated_at, updated_by, is_deleted
		) VALUES (
			@dispense_id, @dispense_no, @order_id, @charge_id, @register_id, @patient_id,
			@status, @dispensed_at, @operator_id, @dept_id, @return_reason,
			@created_at, @created_by, @updated_at, @updated_by, 0
		)`,
		sql.Named("dispense_id", d.DispenseID),
		sql.Named("dispense_no", d.DispenseNo),
		sql.Named("order_id", d.OrderID),
		sql.Named("charge_id", nullIfEmpty(d.ChargeID)),
		sql.Named("register_id", d.RegisterID),
		sql.Named("patient_id", d.PatientID),
		sql.Named("status", d.Status),
		sql.Named("dispensed_at", d.DispensedAt.UTC()),
		sql.Named("operator_id", nullIfEmpty(d.OperatorID)),
		sql.Named("dept_id", nullIfEmpty(d.DeptID)),
		sql.Named("return_reason", nullIfEmpty(d.ReturnReason)),
		sql.Named("created_at", d.CreatedAt.UTC()),
		sql.Named("created_by", nullIfEmpty(d.CreatedBy)),
		sql.Named("updated_at", d.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(d.UpdatedBy)),
	)
	if err != nil {
		return err
	}
	for _, it := range d.Items {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO dbo.dispense_item (
				dispense_item_id, dispense_id, order_item_id, charge_item_id, item_no, item_name, units,
				qty_charged, qty_dispensed, qty_returned, line_status, created_at, updated_at
			) VALUES (
				@dispense_item_id, @dispense_id, @order_item_id, @charge_item_id, @item_no, @item_name, @units,
				@qty_charged, @qty_dispensed, @qty_returned, @line_status, @created_at, @updated_at
			)`,
			sql.Named("dispense_item_id", it.DispenseItemID),
			sql.Named("dispense_id", d.DispenseID),
			sql.Named("order_item_id", nullIfEmpty(it.OrderItemID)),
			sql.Named("charge_item_id", nullIfEmpty(it.ChargeItemID)),
			sql.Named("item_no", it.ItemNo),
			sql.Named("item_name", it.ItemName),
			sql.Named("units", nullIfEmpty(it.Units)),
			sql.Named("qty_charged", milliToDecimal(it.QtyChargedMilli)),
			sql.Named("qty_dispensed", milliToDecimal(it.QtyDispensedMilli)),
			sql.Named("qty_returned", milliToDecimal(it.QtyReturnedMilli)),
			sql.Named("line_status", it.LineStatus),
			sql.Named("created_at", it.CreatedAt.UTC()),
			sql.Named("updated_at", it.UpdatedAt.UTC()),
		); err != nil {
			return err
		}
	}
	return nil
}

func loadDispense(ctx context.Context, q rowQuery, dispenseID string, lock bool) (*domain.Dispense, error) {
	query := `SELECT ` + dispenseCols + ` FROM dbo.dispense WHERE dispense_id = @dispense_id AND is_deleted = 0`
	if lock {
		query = `SELECT ` + dispenseCols + ` FROM dbo.dispense WITH (UPDLOCK, ROWLOCK) WHERE dispense_id = @dispense_id AND is_deleted = 0`
	}
	d, err := scanDispense(q.QueryRowContext(ctx, query, sql.Named("dispense_id", dispenseID)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	items, err := listDispenseItems(ctx, q, dispenseID)
	if err != nil {
		return nil, err
	}
	d.Items = items
	return d, nil
}

func listDispenseItems(ctx context.Context, q rowQuery, dispenseID string) ([]domain.DispenseItem, error) {
	rows, err := q.QueryContext(ctx, `SELECT `+dispenseItemCols+`
		FROM dbo.dispense_item WHERE dispense_id = @dispense_id ORDER BY dispense_item_id`,
		sql.Named("dispense_id", dispenseID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.DispenseItem, 0)
	for rows.Next() {
		it, err := scanDispenseItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanDispense(sc scanner) (*domain.Dispense, error) {
	var d domain.Dispense
	var chargeID, operatorID, deptID, reason, createdBy, updatedBy sql.NullString
	var dispensed, created, updated sql.NullTime
	if err := sc.Scan(
		&d.DispenseID, &d.DispenseNo, &d.OrderID, &chargeID, &d.RegisterID, &d.PatientID,
		&d.Status, &dispensed, &operatorID, &deptID, &reason, &created, &createdBy, &updated, &updatedBy,
	); err != nil {
		return nil, err
	}
	d.ChargeID = chargeID.String
	d.OperatorID = operatorID.String
	d.DeptID = deptID.String
	d.ReturnReason = reason.String
	d.CreatedBy = createdBy.String
	d.UpdatedBy = updatedBy.String
	if dispensed.Valid {
		d.DispensedAt = dispensed.Time
	}
	if created.Valid {
		d.CreatedAt = created.Time
	}
	if updated.Valid {
		d.UpdatedAt = updated.Time
	}
	return &d, nil
}

func scanDispenseItem(sc scanner) (*domain.DispenseItem, error) {
	var it domain.DispenseItem
	var orderItemID, chargeItemID, units sql.NullString
	var chargedS, dispS, retS sql.NullString
	var created, updated sql.NullTime
	if err := sc.Scan(
		&it.DispenseItemID, &it.DispenseID, &orderItemID, &chargeItemID, &it.ItemNo, &it.ItemName, &units,
		&chargedS, &dispS, &retS, &it.LineStatus, &created, &updated,
	); err != nil {
		return nil, err
	}
	it.OrderItemID = orderItemID.String
	it.ChargeItemID = chargeItemID.String
	it.Units = units.String
	var err error
	if it.QtyChargedMilli, err = decimalToMilli(chargedS.String); err != nil {
		return nil, err
	}
	if it.QtyDispensedMilli, err = decimalToMilli(dispS.String); err != nil {
		return nil, err
	}
	if it.QtyReturnedMilli, err = decimalToMilli(retS.String); err != nil {
		return nil, err
	}
	if created.Valid {
		it.CreatedAt = created.Time
	}
	if updated.Valid {
		it.UpdatedAt = updated.Time
	}
	return &it, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [16]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
