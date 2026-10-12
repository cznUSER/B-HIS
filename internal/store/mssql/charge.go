package mssql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

// SQL follows migrations/sqlserver/core/005_charge. Do not add columns here.
// On-hand is dbo.inventory_item (006). Dispense totals are dbo.dispense_item (007). 2026-10-12

type dispenseRec struct {
	dispensed int64
	returned  int64
}

const chargeCols = `charge_id, square_no, order_id, register_id, patient_id,
	total_fee, square_fee, favourable_fee, status, successor_charge_id, abated_from_id,
	paid_at, created_at, created_by, updated_at, updated_by`

const chargeItemCols = `charge_item_id, charge_id, order_item_id, line_no, item_no, item_name,
	qty, unit_price, total_fee, can_back, refunded_qty`

func (s *Store) CreateCharge(ctx context.Context, c domain.Charge) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if c.OrderID != "" {
		var n int
		err = tx.QueryRowContext(ctx, `
			SELECT COUNT(1) FROM dbo.charge
			WHERE is_deleted = 0 AND order_id = @order_id`,
			sql.Named("order_id", c.OrderID),
		).Scan(&n)
		if err != nil {
			return err
		}
		if n > 0 {
			return port.ErrChargeExists
		}
	}
	if err := insertCharge(ctx, tx, c); err != nil {
		return classifyWriteError(err)
	}
	return tx.Commit()
}

func (s *Store) GetCharge(ctx context.Context, chargeID string) (*domain.Charge, error) {
	return loadCharge(ctx, s.db, chargeID)
}

func (s *Store) SearchCharges(ctx context.Context, q port.ChargeQuery) ([]domain.Charge, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	// created_at is UTC. The list day is Asia/Shanghai (UTC+8, no DST), same as memory.
	rows, err := s.db.QueryContext(ctx, `SELECT TOP (@limit) `+chargeCols+`
		FROM dbo.charge
		WHERE is_deleted = 0
		  AND (@patient_id IS NULL OR patient_id = @patient_id)
		  AND (@register_id IS NULL OR register_id = @register_id)
		  AND (@order_id IS NULL OR order_id = @order_id)
		  AND (@from_date IS NULL OR CONVERT(char(10), DATEADD(hour, 8, created_at), 23) >= @from_date)
		  AND (@to_date IS NULL OR CONVERT(char(10), DATEADD(hour, 8, created_at), 23) <= @to_date)
		ORDER BY created_at, charge_id`,
		sql.Named("limit", limit),
		sql.Named("patient_id", nullIfEmpty(q.PatientID)),
		sql.Named("register_id", nullIfEmpty(q.RegisterID)),
		sql.Named("order_id", nullIfEmpty(q.OrderID)),
		sql.Named("from_date", nullIfEmpty(q.FromDate)),
		sql.Named("to_date", nullIfEmpty(q.ToDate)),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Charge, 0)
	for rows.Next() {
		c, err := scanCharge(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		items, err := listChargeItems(ctx, s.db, out[i].ChargeID)
		if err != nil {
			return nil, err
		}
		out[i].Items = items
	}
	return out, nil
}

func (s *Store) PayCharge(ctx context.Context, chargeID string, payment domain.Payment) (*domain.Charge, *domain.Payment, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var status, orderID string
	var paid sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT status, order_id, paid_at
		FROM dbo.charge WITH (UPDLOCK, ROWLOCK)
		WHERE charge_id = @charge_id AND is_deleted = 0`,
		sql.Named("charge_id", chargeID),
	).Scan(&status, &orderID, &paid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, port.ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	if status == domain.ChargeStatusAbate {
		return nil, nil, port.ErrChargeAlreadyAbated
	}
	if paid.Valid {
		return nil, nil, port.ErrChargeAlreadyPaid
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE dbo.charge
		SET status = @status, paid_at = @paid_at, updated_at = @updated_at, updated_by = @updated_by
		WHERE charge_id = @charge_id AND is_deleted = 0`,
		sql.Named("status", domain.ChargeStatusNormal),
		sql.Named("paid_at", payment.PaidAt.UTC()),
		sql.Named("updated_at", payment.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(payment.UpdatedBy)),
		sql.Named("charge_id", chargeID),
	); err != nil {
		return nil, nil, err
	}
	if err := insertPayment(ctx, tx, payment); err != nil {
		return nil, nil, classifyWriteError(err)
	}
	if orderID != "" {
		res, err := tx.ExecContext(ctx, `
			UPDATE dbo.order_sheet
			SET charged = 1, updated_at = @updated_at, updated_by = @updated_by
			WHERE order_id = @order_id AND is_deleted = 0`,
			sql.Named("updated_at", payment.UpdatedAt.UTC()),
			sql.Named("updated_by", nullIfEmpty(payment.UpdatedBy)),
			sql.Named("order_id", orderID),
		)
		if err != nil {
			return nil, nil, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return nil, nil, err
		}
		if n != 1 {
			return nil, nil, port.ErrNotFound
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	loaded, err := loadCharge(ctx, s.db, chargeID)
	if err != nil {
		return nil, nil, err
	}
	pay := payment
	return loaded, &pay, nil
}

func (s *Store) RefundCharge(ctx context.Context, old domain.Charge, newCharge *domain.Charge, payment domain.Payment) (*domain.Charge, *domain.Charge, *domain.Payment, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	var paid sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT status, paid_at
		FROM dbo.charge WITH (UPDLOCK, ROWLOCK)
		WHERE charge_id = @charge_id AND is_deleted = 0`,
		sql.Named("charge_id", old.ChargeID),
	).Scan(&status, &paid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil, port.ErrNotFound
	}
	if err != nil {
		return nil, nil, nil, err
	}
	if status == domain.ChargeStatusAbate {
		return nil, nil, nil, port.ErrChargeAlreadyAbated
	}
	if !paid.Valid {
		return nil, nil, nil, port.ErrChargeNotPaid
	}
	// Insert the successor first so fk_charge_successor can point at it.
	if newCharge != nil {
		if err := insertCharge(ctx, tx, *newCharge); err != nil {
			return nil, nil, nil, classifyWriteError(err)
		}
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE dbo.charge
		SET status = @status,
		    successor_charge_id = @successor_charge_id,
		    updated_at = @updated_at,
		    updated_by = @updated_by
		WHERE charge_id = @charge_id AND is_deleted = 0`,
		sql.Named("status", domain.ChargeStatusAbate),
		sql.Named("successor_charge_id", nullIfEmpty(old.SuccessorChargeID)),
		sql.Named("updated_at", old.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(old.UpdatedBy)),
		sql.Named("charge_id", old.ChargeID),
	); err != nil {
		return nil, nil, nil, err
	}
	for _, it := range old.Items {
		if _, err := tx.ExecContext(ctx, `
			UPDATE dbo.charge_item
			SET refunded_qty = @refunded_qty, updated_at = @updated_at
			WHERE charge_item_id = @charge_item_id AND charge_id = @charge_id`,
			sql.Named("refunded_qty", milliToDecimal(it.RefundedQtyMilli)),
			sql.Named("updated_at", old.UpdatedAt.UTC()),
			sql.Named("charge_item_id", it.ChargeItemID),
			sql.Named("charge_id", old.ChargeID),
		); err != nil {
			return nil, nil, nil, err
		}
	}
	if err := insertPayment(ctx, tx, payment); err != nil {
		return nil, nil, nil, classifyWriteError(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, nil, err
	}
	loaded, err := loadCharge(ctx, s.db, old.ChargeID)
	if err != nil {
		return nil, nil, nil, err
	}
	var loadedNew *domain.Charge
	if newCharge != nil {
		loadedNew, err = loadCharge(ctx, s.db, newCharge.ChargeID)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	pay := payment
	return loaded, loadedNew, &pay, nil
}

func (s *Store) DrugOnHandMilli(ctx context.Context, itemNo string) (int64, error) {
	var qty sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT CONVERT(varchar(40), qty_on_hand)
		FROM dbo.inventory_item
		WHERE item_no = @item_no AND is_deleted = 0 AND status = 'ACTIVE'`,
		sql.Named("item_no", itemNo),
	).Scan(&qty)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return decimalToMilli(qty.String)
}

// SetDrugOnHandMilli writes inventory_item.qty_on_hand so charge and dispense share one row. 2026-10-12
func (s *Store) SetDrugOnHandMilli(itemNo string, milli int64) error {
	if itemNo == "" || milli < 0 {
		return fmt.Errorf("invalid on-hand")
	}
	res, err := s.db.ExecContext(context.Background(), `
		UPDATE dbo.inventory_item
		SET qty_on_hand = @qty, status = 'ACTIVE', updated_at = SYSUTCDATETIME()
		WHERE item_no = @item_no AND is_deleted = 0`,
		sql.Named("qty", milliToDecimal(milli)),
		sql.Named("item_no", itemNo),
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return port.ErrNotFound
	}
	return nil
}

func (s *Store) DispenseLedger(ctx context.Context, chargeItemID string) (int64, int64, bool, error) {
	disp, ret, n, err := s.sumDispenseItems(ctx, "charge_item_id", chargeItemID)
	if err != nil {
		return 0, 0, false, err
	}
	if n > 0 {
		return disp, ret, true, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.dispenseStub[chargeItemID]
	if !ok {
		return 0, 0, false, nil
	}
	return rec.dispensed, rec.returned, true, nil
}

// SetDispenseMilli records a charge-item stub used when no dispense_item row exists yet. 2026-10-12
func (s *Store) SetDispenseMilli(ctx context.Context, chargeItemID string, dispensedMilli, returnedMilli int64) error {
	if chargeItemID == "" || dispensedMilli < 0 || returnedMilli < 0 || returnedMilli > dispensedMilli {
		return fmt.Errorf("invalid dispense qty")
	}
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM dbo.charge_item WHERE charge_item_id = @id`,
		sql.Named("id", chargeItemID),
	).Scan(&n)
	if err != nil {
		return err
	}
	if n == 0 {
		return port.ErrNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dispenseStub == nil {
		s.dispenseStub = map[string]dispenseRec{}
	}
	s.dispenseStub[chargeItemID] = dispenseRec{dispensed: dispensedMilli, returned: returnedMilli}
	return nil
}

// SetChargeItemCanBack updates can_back. Used to reach ITEM_NOT_CAN_BACK. 2026-10-12
func (s *Store) SetChargeItemCanBack(ctx context.Context, chargeItemID string, canBack bool) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE dbo.charge_item SET can_back = @can_back, updated_at = SYSUTCDATETIME()
		WHERE charge_item_id = @id`,
		sql.Named("can_back", canBack),
		sql.Named("id", chargeItemID),
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return port.ErrNotFound
	}
	return nil
}

func insertCharge(ctx context.Context, tx *sql.Tx, c domain.Charge) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO dbo.charge (
			charge_id, square_no, order_id, register_id, patient_id,
			total_fee, square_fee, favourable_fee, status,
			successor_charge_id, abated_from_id, paid_at,
			created_at, created_by, updated_at, updated_by, is_deleted
		) VALUES (
			@charge_id, @square_no, @order_id, @register_id, @patient_id,
			@total_fee, @square_fee, @favourable_fee, @status,
			@successor_charge_id, @abated_from_id, @paid_at,
			@created_at, @created_by, @updated_at, @updated_by, 0
		)`,
		sql.Named("charge_id", c.ChargeID),
		sql.Named("square_no", c.SquareNo),
		sql.Named("order_id", nullIfEmpty(c.OrderID)),
		sql.Named("register_id", c.RegisterID),
		sql.Named("patient_id", c.PatientID),
		sql.Named("total_fee", centsToDecimal(c.TotalFeeCents)),
		sql.Named("square_fee", centsToDecimal(c.SquareFeeCents)),
		sql.Named("favourable_fee", centsToDecimal(c.FavourableFeeCents)),
		sql.Named("status", c.Status),
		sql.Named("successor_charge_id", nullIfEmpty(c.SuccessorChargeID)),
		sql.Named("abated_from_id", nullIfEmpty(c.AbatedFromID)),
		sql.Named("paid_at", nullTime(c.PaidAt)),
		sql.Named("created_at", c.CreatedAt.UTC()),
		sql.Named("created_by", nullIfEmpty(c.CreatedBy)),
		sql.Named("updated_at", c.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(c.UpdatedBy)),
	)
	if err != nil {
		return err
	}
	return insertChargeItems(ctx, tx, c.Items)
}

func insertChargeItems(ctx context.Context, tx *sql.Tx, items []domain.ChargeItem) error {
	for _, it := range items {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO dbo.charge_item (
				charge_item_id, charge_id, order_item_id, line_no, item_no, item_name,
				qty, unit_price, total_fee, can_back, refunded_qty, created_at, updated_at
			) VALUES (
				@charge_item_id, @charge_id, @order_item_id, @line_no, @item_no, @item_name,
				@qty, @unit_price, @total_fee, @can_back, @refunded_qty, SYSUTCDATETIME(), SYSUTCDATETIME()
			)`,
			sql.Named("charge_item_id", it.ChargeItemID),
			sql.Named("charge_id", it.ChargeID),
			sql.Named("order_item_id", nullIfEmpty(it.OrderItemID)),
			sql.Named("line_no", it.LineNo),
			sql.Named("item_no", it.ItemNo),
			sql.Named("item_name", it.ItemName),
			sql.Named("qty", milliToDecimal(it.QtyMilli)),
			sql.Named("unit_price", centsToDecimal(it.UnitPriceCents)),
			sql.Named("total_fee", centsToDecimal(it.TotalFeeCents)),
			sql.Named("can_back", it.CanBack),
			sql.Named("refunded_qty", milliToDecimal(it.RefundedQtyMilli)),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertPayment(ctx context.Context, tx *sql.Tx, p domain.Payment) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO dbo.payment (
			payment_id, payment_no, charge_id, direction, pay_type, amount, status,
			paid_at, operator_id, created_at, created_by, updated_at, updated_by
		) VALUES (
			@payment_id, @payment_no, @charge_id, @direction, @pay_type, @amount, @status,
			@paid_at, @operator_id, @created_at, @created_by, @updated_at, @updated_by
		)`,
		sql.Named("payment_id", p.PaymentID),
		sql.Named("payment_no", p.PaymentNo),
		sql.Named("charge_id", p.ChargeID),
		sql.Named("direction", p.Direction),
		sql.Named("pay_type", p.PayType),
		sql.Named("amount", centsToDecimal(p.AmountCents)),
		sql.Named("status", p.Status),
		sql.Named("paid_at", nullTime(p.PaidAt)),
		sql.Named("operator_id", nullIfEmpty(p.OperatorID)),
		sql.Named("created_at", p.CreatedAt.UTC()),
		sql.Named("created_by", nullIfEmpty(p.CreatedBy)),
		sql.Named("updated_at", p.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(p.UpdatedBy)),
	)
	return err
}

func loadCharge(ctx context.Context, q orderQuery, chargeID string) (*domain.Charge, error) {
	c, err := scanCharge(q.QueryRowContext(ctx, `SELECT `+chargeCols+`
		FROM dbo.charge WHERE charge_id = @charge_id AND is_deleted = 0`,
		sql.Named("charge_id", chargeID)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	items, err := listChargeItems(ctx, q, chargeID)
	if err != nil {
		return nil, err
	}
	c.Items = items
	return c, nil
}

func listChargeItems(ctx context.Context, q orderQuery, chargeID string) ([]domain.ChargeItem, error) {
	rows, err := q.QueryContext(ctx, `SELECT `+chargeItemCols+`
		FROM dbo.charge_item WHERE charge_id = @charge_id ORDER BY line_no, charge_item_id`,
		sql.Named("charge_id", chargeID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.ChargeItem, 0)
	for rows.Next() {
		it, err := scanChargeItem(rows)
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

func scanCharge(sc scanner) (*domain.Charge, error) {
	var c domain.Charge
	var orderID, successor, abatedFrom, createdBy, updatedBy sql.NullString
	var total, square, fav sql.NullString
	var paid sql.NullTime
	var created, updated sql.NullTime
	if err := sc.Scan(
		&c.ChargeID, &c.SquareNo, &orderID, &c.RegisterID, &c.PatientID,
		&total, &square, &fav, &c.Status, &successor, &abatedFrom,
		&paid, &created, &createdBy, &updated, &updatedBy,
	); err != nil {
		return nil, err
	}
	c.OrderID = orderID.String
	c.SuccessorChargeID = successor.String
	c.AbatedFromID = abatedFrom.String
	c.CreatedBy = createdBy.String
	c.UpdatedBy = updatedBy.String
	var err error
	if c.TotalFeeCents, err = decimalToCents(total.String); err != nil {
		return nil, err
	}
	if c.SquareFeeCents, err = decimalToCents(square.String); err != nil {
		return nil, err
	}
	if c.FavourableFeeCents, err = decimalToCents(fav.String); err != nil {
		return nil, err
	}
	if paid.Valid {
		c.PaidAt = paid.Time
	}
	if created.Valid {
		c.CreatedAt = created.Time
	}
	if updated.Valid {
		c.UpdatedAt = updated.Time
	}
	if c.Items == nil {
		c.Items = []domain.ChargeItem{}
	}
	return &c, nil
}

func scanChargeItem(sc scanner) (*domain.ChargeItem, error) {
	var it domain.ChargeItem
	var orderItemID sql.NullString
	var qty, price, fee, refunded sql.NullString
	if err := sc.Scan(
		&it.ChargeItemID, &it.ChargeID, &orderItemID, &it.LineNo, &it.ItemNo, &it.ItemName,
		&qty, &price, &fee, &it.CanBack, &refunded,
	); err != nil {
		return nil, err
	}
	it.OrderItemID = orderItemID.String
	var err error
	if it.QtyMilli, err = decimalToMilli(qty.String); err != nil {
		return nil, err
	}
	if it.UnitPriceCents, err = decimalToCents(price.String); err != nil {
		return nil, err
	}
	if it.TotalFeeCents, err = decimalToCents(fee.String); err != nil {
		return nil, err
	}
	if it.RefundedQtyMilli, err = decimalToMilli(refunded.String); err != nil {
		return nil, err
	}
	return &it, nil
}
