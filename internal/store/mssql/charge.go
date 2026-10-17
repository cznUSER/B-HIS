package mssql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

// SQL follows migrations/sqlserver/core/005_charge. Do not add columns here.
// On-hand is dbo.inventory_item (006). Dispense totals are dbo.dispense_item (007). 2026-10-12

type dispenseRec struct {
	dispensed int64
	returned  int64
}

func isRegisterFeeCharge(c domain.Charge) bool {
	return len(c.Items) == 1 && c.Items[0].ItemNo == "REGISTER-FEE"
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
			SELECT COUNT(1) FROM dbo.charge WITH (UPDLOCK, HOLDLOCK)
			WHERE is_deleted = 0 AND order_id = @order_id`,
			sql.Named("order_id", c.OrderID),
		).Scan(&n)
		if err != nil {
			return err
		}
		if n > 0 {
			return port.ErrChargeExists
		}
	} else if isRegisterFeeCharge(c) {
		var n int
		if err := tx.QueryRowContext(ctx, `
			SELECT COUNT(1) FROM dbo.charge AS ch WITH (UPDLOCK, HOLDLOCK)
			JOIN dbo.charge_item AS ci ON ci.charge_id = ch.charge_id
			WHERE ch.is_deleted = 0 AND ch.register_id = @register_id
			  AND ch.order_id IS NULL AND ch.status <> 'ABATE' AND ci.item_no = 'REGISTER-FEE'`,
			sql.Named("register_id", c.RegisterID)).Scan(&n); err != nil {
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

// CreateChargeWithIdempotency keeps the idempotency row, charge header, and
// all charge items in one serializable transaction. The order check is inside
// the same transaction so a replay is resolved before the one-charge rule.
func (s *Store) CreateChargeWithIdempotency(ctx context.Context, c domain.Charge, scope, key, requestHash string, expiresAt time.Time) (domain.Charge, bool, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return domain.Charge{}, false, err
	}
	defer func() { _ = tx.Rollback() }()

	var savedHash, state string
	var resourceID sql.NullString
	var savedExpires time.Time
	err = tx.QueryRowContext(ctx, `
		SELECT request_hash, state, resource_id, expires_at
		FROM dbo.idempotency_record WITH (UPDLOCK, HOLDLOCK)
		WHERE scope = @scope AND idem_key = @idem_key`,
		sql.Named("scope", scope), sql.Named("idem_key", key)).Scan(&savedHash, &state, &resourceID, &savedExpires)
	if err == nil {
		if !savedExpires.After(time.Now().UTC()) {
			if _, err := tx.ExecContext(ctx, `DELETE FROM dbo.idempotency_record WHERE scope=@scope AND idem_key=@idem_key`, sql.Named("scope", scope), sql.Named("idem_key", key)); err != nil {
				return domain.Charge{}, false, err
			}
		} else {
			if savedHash != requestHash {
				return domain.Charge{}, false, port.ErrIdempotencyConflict
			}
			switch state {
			case "PENDING":
				return domain.Charge{}, false, port.ErrIdempotencyInProgress
			case "COMPLETED":
				if !resourceID.Valid || resourceID.String == "" {
					return domain.Charge{}, false, port.ErrNotFound
				}
				existing, getErr := loadCharge(ctx, tx, resourceID.String)
				if errors.Is(getErr, port.ErrNotFound) {
					return domain.Charge{}, false, port.ErrNotFound
				}
				if getErr != nil {
					return domain.Charge{}, false, getErr
				}
				if err := tx.Commit(); err != nil {
					return domain.Charge{}, false, err
				}
				return *existing, true, nil
			case "FAILED":
				if _, err := tx.ExecContext(ctx, `DELETE FROM dbo.idempotency_record WHERE scope=@scope AND idem_key=@idem_key`, sql.Named("scope", scope), sql.Named("idem_key", key)); err != nil {
					return domain.Charge{}, false, err
				}
			default:
				return domain.Charge{}, false, port.ErrIdempotencyInProgress
			}
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return domain.Charge{}, false, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO dbo.idempotency_record(scope, idem_key, request_hash, state, resource_type, resource_id, created_at, expires_at)
		VALUES(@scope, @idem_key, @request_hash, 'PENDING', 'Charge', NULL, SYSUTCDATETIME(), @expires_at)`,
		sql.Named("scope", scope), sql.Named("idem_key", key), sql.Named("request_hash", requestHash), sql.Named("expires_at", expiresAt.UTC())); err != nil {
		return domain.Charge{}, false, err
	}
	if c.OrderID != "" {
		var n int
		if err := tx.QueryRowContext(ctx, `
			SELECT COUNT(1) FROM dbo.charge WITH (UPDLOCK, HOLDLOCK)
			WHERE is_deleted = 0 AND order_id = @order_id`,
			sql.Named("order_id", c.OrderID)).Scan(&n); err != nil {
			return domain.Charge{}, false, err
		}
		if n > 0 {
			return domain.Charge{}, false, port.ErrChargeExists
		}
	} else if isRegisterFeeCharge(c) {
		var n int
		if err := tx.QueryRowContext(ctx, `
			SELECT COUNT(1) FROM dbo.charge AS ch WITH (UPDLOCK, HOLDLOCK)
			JOIN dbo.charge_item AS ci ON ci.charge_id = ch.charge_id
			WHERE ch.is_deleted = 0 AND ch.register_id = @register_id
			  AND ch.order_id IS NULL AND ch.status <> 'ABATE' AND ci.item_no = 'REGISTER-FEE'`,
			sql.Named("register_id", c.RegisterID)).Scan(&n); err != nil {
			return domain.Charge{}, false, err
		}
		if n > 0 {
			return domain.Charge{}, false, port.ErrChargeExists
		}
	}
	if err := insertCharge(ctx, tx, c); err != nil {
		return domain.Charge{}, false, classifyWriteError(err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE dbo.idempotency_record
		SET state='COMPLETED', resource_type='Charge', resource_id=@resource_id, completed_at=SYSUTCDATETIME()
		WHERE scope=@scope AND idem_key=@idem_key`,
		sql.Named("resource_id", c.ChargeID), sql.Named("scope", scope), sql.Named("idem_key", key)); err != nil {
		return domain.Charge{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Charge{}, false, err
	}
	if c.Items == nil {
		c.Items = []domain.ChargeItem{}
	}
	return c, false, nil
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

	var status, orderID, registerID string
	var paid sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT status, COALESCE(order_id, ''), register_id, paid_at
		FROM dbo.charge WITH (UPDLOCK, ROWLOCK)
		WHERE charge_id = @charge_id AND is_deleted = 0`,
		sql.Named("charge_id", chargeID),
	).Scan(&status, &orderID, &registerID, &paid)
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
	} else {
		res, err := tx.ExecContext(ctx, `UPDATE dbo.register_bill SET register_charged=1, updated_at=@updated_at, updated_by=@updated_by WHERE register_id=@register_id AND is_deleted=0`, sql.Named("updated_at", payment.UpdatedAt.UTC()), sql.Named("updated_by", nullIfEmpty(payment.UpdatedBy)), sql.Named("register_id", registerID))
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

// LookupPayIdempotency returns a completed payment response before the caller
// performs mutable charge-state validation. This lets a replay succeed even
// though the charge is now NORMAL/paid. The lookup itself is read-only apart
// from expiry cleanup.
func (s *Store) LookupPayIdempotency(ctx context.Context, scope, key, requestHash string) (*domain.Charge, *domain.Payment, bool, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, nil, false, err
	}
	defer func() { _ = tx.Rollback() }()
	resourceID, state, found, err := readChargeIdempotency(ctx, tx, scope, key, requestHash)
	if err != nil || !found {
		if err == nil {
			err = tx.Commit()
		}
		return nil, nil, false, err
	}
	pay, err := loadPayment(ctx, tx, resourceID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, false, port.ErrNotFound
	}
	if err != nil {
		return nil, nil, false, err
	}
	if state != "COMPLETED" || pay.Direction != domain.PaymentDirectionCharge {
		return nil, nil, false, port.ErrNotFound
	}
	cur, err := loadCharge(ctx, tx, pay.ChargeID)
	if errors.Is(err, port.ErrNotFound) {
		return nil, nil, false, port.ErrNotFound
	}
	if err != nil {
		return nil, nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, false, err
	}
	return cur, pay, true, nil
}

// PayChargeWithIdempotency atomically records the idempotency key, payment,
// charge state, and order charged flag. A completed key points at payment_id,
// which is sufficient to reconstruct the payment response after a restart.
func (s *Store) PayChargeWithIdempotency(ctx context.Context, chargeID string, payment domain.Payment, scope, key, requestHash string, expiresAt time.Time) (*domain.Charge, *domain.Payment, bool, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, nil, false, err
	}
	defer func() { _ = tx.Rollback() }()
	resourceID, state, found, err := readChargeIdempotency(ctx, tx, scope, key, requestHash)
	if err != nil {
		return nil, nil, false, err
	}
	if found {
		if state == "PENDING" {
			return nil, nil, false, port.ErrIdempotencyInProgress
		}
		if state == "COMPLETED" {
			pay, getErr := loadPayment(ctx, tx, resourceID)
			if errors.Is(getErr, sql.ErrNoRows) {
				return nil, nil, false, port.ErrNotFound
			}
			if getErr != nil {
				return nil, nil, false, getErr
			}
			cur, getErr := loadCharge(ctx, tx, pay.ChargeID)
			if errors.Is(getErr, port.ErrNotFound) {
				return nil, nil, false, port.ErrNotFound
			}
			if getErr != nil {
				return nil, nil, false, getErr
			}
			if err := tx.Commit(); err != nil {
				return nil, nil, false, err
			}
			return cur, pay, true, nil
		}
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO dbo.idempotency_record(scope, idem_key, request_hash, state, resource_type, resource_id, created_at, expires_at)
		VALUES(@scope, @idem_key, @request_hash, 'PENDING', 'Payment', NULL, SYSUTCDATETIME(), @expires_at)`,
		sql.Named("scope", scope), sql.Named("idem_key", key), sql.Named("request_hash", requestHash), sql.Named("expires_at", expiresAt.UTC())); err != nil {
		return nil, nil, false, err
	}
	var status, orderID, registerID string
	var paid sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT status, COALESCE(order_id, ''), register_id, paid_at FROM dbo.charge WITH (UPDLOCK, ROWLOCK)
		WHERE charge_id=@charge_id AND is_deleted=0`, sql.Named("charge_id", chargeID)).Scan(&status, &orderID, &registerID, &paid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, false, port.ErrNotFound
	}
	if err != nil {
		return nil, nil, false, err
	}
	if status == domain.ChargeStatusAbate {
		return nil, nil, false, port.ErrChargeAlreadyAbated
	}
	if paid.Valid {
		return nil, nil, false, port.ErrChargeAlreadyPaid
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE dbo.charge SET status=@status, paid_at=@paid_at, updated_at=@updated_at, updated_by=@updated_by
		WHERE charge_id=@charge_id AND is_deleted=0`,
		sql.Named("status", domain.ChargeStatusNormal), sql.Named("paid_at", payment.PaidAt.UTC()), sql.Named("updated_at", payment.UpdatedAt.UTC()), sql.Named("updated_by", nullIfEmpty(payment.UpdatedBy)), sql.Named("charge_id", chargeID)); err != nil {
		return nil, nil, false, err
	}
	if err := insertPayment(ctx, tx, payment); err != nil {
		return nil, nil, false, classifyWriteError(err)
	}
	if orderID != "" {
		res, err := tx.ExecContext(ctx, `UPDATE dbo.order_sheet SET charged=1, updated_at=@updated_at, updated_by=@updated_by WHERE order_id=@order_id AND is_deleted=0`, sql.Named("updated_at", payment.UpdatedAt.UTC()), sql.Named("updated_by", nullIfEmpty(payment.UpdatedBy)), sql.Named("order_id", orderID))
		if err != nil {
			return nil, nil, false, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return nil, nil, false, err
		}
		if n != 1 {
			return nil, nil, false, port.ErrNotFound
		}
	} else {
		res, err := tx.ExecContext(ctx, `UPDATE dbo.register_bill SET register_charged=1, updated_at=@updated_at, updated_by=@updated_by WHERE register_id=@register_id AND is_deleted=0`, sql.Named("updated_at", payment.UpdatedAt.UTC()), sql.Named("updated_by", nullIfEmpty(payment.UpdatedBy)), sql.Named("register_id", registerID))
		if err != nil {
			return nil, nil, false, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return nil, nil, false, err
		}
		if n != 1 {
			return nil, nil, false, port.ErrNotFound
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE dbo.idempotency_record SET state='COMPLETED', resource_type='Payment', resource_id=@resource_id, completed_at=SYSUTCDATETIME() WHERE scope=@scope AND idem_key=@idem_key`, sql.Named("resource_id", payment.PaymentID), sql.Named("scope", scope), sql.Named("idem_key", key)); err != nil {
		return nil, nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, false, err
	}
	cur, err := loadCharge(ctx, s.db, chargeID)
	if err != nil {
		return nil, nil, false, err
	}
	p := payment
	return cur, &p, false, nil
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
	if old.OrderID == "" && newCharge == nil {
		res, err := tx.ExecContext(ctx, `UPDATE dbo.register_bill SET register_charged=0, updated_at=@updated_at, updated_by=@updated_by WHERE register_id=@register_id AND is_deleted=0`, sql.Named("updated_at", payment.UpdatedAt.UTC()), sql.Named("updated_by", nullIfEmpty(payment.UpdatedBy)), sql.Named("register_id", old.RegisterID))
		if err != nil {
			return nil, nil, nil, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return nil, nil, nil, err
		}
		if n != 1 {
			return nil, nil, nil, port.ErrNotFound
		}
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

// LookupRefundIdempotency rebuilds the composite refund response from the
// refund payment, abated charge, and successor charge persisted in SQL.
func (s *Store) LookupRefundIdempotency(ctx context.Context, scope, key, requestHash string) (*domain.Charge, *domain.Charge, *domain.Payment, bool, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, nil, nil, false, err
	}
	defer func() { _ = tx.Rollback() }()
	resourceID, state, found, err := readChargeIdempotency(ctx, tx, scope, key, requestHash)
	if err != nil || !found {
		if err == nil {
			err = tx.Commit()
		}
		return nil, nil, nil, false, err
	}
	if state != "COMPLETED" {
		return nil, nil, nil, false, port.ErrNotFound
	}
	pay, err := loadPayment(ctx, tx, resourceID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil, false, port.ErrNotFound
	}
	if err != nil || pay.Direction != domain.PaymentDirectionRefund {
		return nil, nil, nil, false, err
	}
	old, err := loadCharge(ctx, tx, pay.ChargeID)
	if err != nil {
		return nil, nil, nil, false, err
	}
	var next *domain.Charge
	if old.SuccessorChargeID != "" {
		next, err = loadCharge(ctx, tx, old.SuccessorChargeID)
		if err != nil {
			return nil, nil, nil, false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, nil, false, err
	}
	return old, next, pay, true, nil
}

// RefundChargeWithIdempotency atomically persists the refund payment, old
// charge abatement, optional successor, and idempotency mapping. The mapping
// points at payment_id; successor_charge_id supplies the rest of the response.
func (s *Store) RefundChargeWithIdempotency(ctx context.Context, old domain.Charge, newCharge *domain.Charge, payment domain.Payment, scope, key, requestHash string, expiresAt time.Time) (*domain.Charge, *domain.Charge, *domain.Payment, bool, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, nil, nil, false, err
	}
	defer func() { _ = tx.Rollback() }()
	resourceID, state, found, err := readChargeIdempotency(ctx, tx, scope, key, requestHash)
	if err != nil {
		return nil, nil, nil, false, err
	}
	if found {
		if state == "PENDING" {
			return nil, nil, nil, false, port.ErrIdempotencyInProgress
		}
		if state == "COMPLETED" {
			pay, getErr := loadPayment(ctx, tx, resourceID)
			if errors.Is(getErr, sql.ErrNoRows) {
				return nil, nil, nil, false, port.ErrNotFound
			}
			if getErr != nil {
				return nil, nil, nil, false, getErr
			}
			oldSaved, getErr := loadCharge(ctx, tx, pay.ChargeID)
			if getErr != nil {
				return nil, nil, nil, false, getErr
			}
			var next *domain.Charge
			if oldSaved.SuccessorChargeID != "" {
				next, getErr = loadCharge(ctx, tx, oldSaved.SuccessorChargeID)
				if getErr != nil {
					return nil, nil, nil, false, getErr
				}
			}
			if err := tx.Commit(); err != nil {
				return nil, nil, nil, false, err
			}
			return oldSaved, next, pay, true, nil
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO dbo.idempotency_record(scope, idem_key, request_hash, state, resource_type, resource_id, created_at, expires_at) VALUES(@scope, @idem_key, @request_hash, 'PENDING', 'Payment', NULL, SYSUTCDATETIME(), @expires_at)`, sql.Named("scope", scope), sql.Named("idem_key", key), sql.Named("request_hash", requestHash), sql.Named("expires_at", expiresAt.UTC())); err != nil {
		return nil, nil, nil, false, err
	}
	var status string
	var paid sql.NullTime
	if err := tx.QueryRowContext(ctx, `SELECT status, paid_at FROM dbo.charge WITH (UPDLOCK, ROWLOCK) WHERE charge_id=@charge_id AND is_deleted=0`, sql.Named("charge_id", old.ChargeID)).Scan(&status, &paid); errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil, false, port.ErrNotFound
	} else if err != nil {
		return nil, nil, nil, false, err
	}
	if status == domain.ChargeStatusAbate {
		return nil, nil, nil, false, port.ErrChargeAlreadyAbated
	}
	if !paid.Valid {
		return nil, nil, nil, false, port.ErrChargeNotPaid
	}
	if newCharge != nil {
		if err := insertCharge(ctx, tx, *newCharge); err != nil {
			return nil, nil, nil, false, classifyWriteError(err)
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE dbo.charge SET status=@status, successor_charge_id=@successor_charge_id, updated_at=@updated_at, updated_by=@updated_by WHERE charge_id=@charge_id AND is_deleted=0`, sql.Named("status", domain.ChargeStatusAbate), sql.Named("successor_charge_id", nullIfEmpty(old.SuccessorChargeID)), sql.Named("updated_at", old.UpdatedAt.UTC()), sql.Named("updated_by", nullIfEmpty(old.UpdatedBy)), sql.Named("charge_id", old.ChargeID)); err != nil {
		return nil, nil, nil, false, err
	}
	for _, it := range old.Items {
		if _, err := tx.ExecContext(ctx, `UPDATE dbo.charge_item SET refunded_qty=@refunded_qty, updated_at=@updated_at WHERE charge_item_id=@charge_item_id AND charge_id=@charge_id`, sql.Named("refunded_qty", milliToDecimal(it.RefundedQtyMilli)), sql.Named("updated_at", old.UpdatedAt.UTC()), sql.Named("charge_item_id", it.ChargeItemID), sql.Named("charge_id", old.ChargeID)); err != nil {
			return nil, nil, nil, false, err
		}
	}
	if err := insertPayment(ctx, tx, payment); err != nil {
		return nil, nil, nil, false, classifyWriteError(err)
	}
	if old.OrderID == "" && newCharge == nil {
		res, err := tx.ExecContext(ctx, `UPDATE dbo.register_bill SET register_charged=0, updated_at=@updated_at, updated_by=@updated_by WHERE register_id=@register_id AND is_deleted=0`, sql.Named("updated_at", payment.UpdatedAt.UTC()), sql.Named("updated_by", nullIfEmpty(payment.UpdatedBy)), sql.Named("register_id", old.RegisterID))
		if err != nil {
			return nil, nil, nil, false, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return nil, nil, nil, false, err
		}
		if n != 1 {
			return nil, nil, nil, false, port.ErrNotFound
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE dbo.idempotency_record SET state='COMPLETED', resource_type='Payment', resource_id=@resource_id, completed_at=SYSUTCDATETIME() WHERE scope=@scope AND idem_key=@idem_key`, sql.Named("resource_id", payment.PaymentID), sql.Named("scope", scope), sql.Named("idem_key", key)); err != nil {
		return nil, nil, nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, nil, false, err
	}
	loadedOld, err := loadCharge(ctx, s.db, old.ChargeID)
	if err != nil {
		return nil, nil, nil, false, err
	}
	var loadedNew *domain.Charge
	if newCharge != nil {
		loadedNew, err = loadCharge(ctx, s.db, newCharge.ChargeID)
		if err != nil {
			return nil, nil, nil, false, err
		}
	}
	p := payment
	return loadedOld, loadedNew, &p, false, nil
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

func readChargeIdempotency(ctx context.Context, tx *sql.Tx, scope, key, requestHash string) (resourceID, state string, found bool, err error) {
	var savedHash string
	var savedResource sql.NullString
	var expiresAt time.Time
	err = tx.QueryRowContext(ctx, `SELECT request_hash, state, resource_id, expires_at FROM dbo.idempotency_record WITH (UPDLOCK, HOLDLOCK) WHERE scope=@scope AND idem_key=@idem_key`, sql.Named("scope", scope), sql.Named("idem_key", key)).Scan(&savedHash, &state, &savedResource, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}
	if !expiresAt.After(time.Now().UTC()) {
		if _, err := tx.ExecContext(ctx, `DELETE FROM dbo.idempotency_record WHERE scope=@scope AND idem_key=@idem_key`, sql.Named("scope", scope), sql.Named("idem_key", key)); err != nil {
			return "", "", false, err
		}
		return "", "", false, nil
	}
	if savedHash != requestHash {
		return "", "", false, port.ErrIdempotencyConflict
	}
	if state == "FAILED" {
		if _, err := tx.ExecContext(ctx, `DELETE FROM dbo.idempotency_record WHERE scope=@scope AND idem_key=@idem_key`, sql.Named("scope", scope), sql.Named("idem_key", key)); err != nil {
			return "", "", false, err
		}
		return "", "", false, nil
	}
	if state != "PENDING" && state != "COMPLETED" {
		return "", "", false, port.ErrIdempotencyInProgress
	}
	return savedResource.String, state, true, nil
}

func loadPayment(ctx context.Context, q orderQuery, paymentID string) (*domain.Payment, error) {
	var p domain.Payment
	var amount sql.NullString
	var paid, created, updated sql.NullTime
	var operator, createdBy, updatedBy sql.NullString
	err := q.QueryRowContext(ctx, `SELECT payment_id, payment_no, charge_id, direction, pay_type, amount, status, paid_at, operator_id, created_at, created_by, updated_at, updated_by FROM dbo.payment WHERE payment_id=@payment_id`, sql.Named("payment_id", paymentID)).Scan(
		&p.PaymentID, &p.PaymentNo, &p.ChargeID, &p.Direction, &p.PayType, &amount, &p.Status, &paid, &operator, &created, &createdBy, &updated, &updatedBy,
	)
	if err != nil {
		return nil, err
	}
	var convErr error
	p.AmountCents, convErr = decimalToCents(amount.String)
	if convErr != nil {
		return nil, convErr
	}
	p.OperatorID = operator.String
	p.CreatedBy = createdBy.String
	p.UpdatedBy = updatedBy.String
	if paid.Valid {
		p.PaidAt = paid.Time
	}
	if created.Valid {
		p.CreatedAt = created.Time
	}
	if updated.Valid {
		p.UpdatedAt = updated.Time
	}
	return &p, nil
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
