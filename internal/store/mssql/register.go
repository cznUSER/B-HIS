package mssql

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

// SQL follows migrations/sqlserver/core/003_register. Do not add columns here.

const registerCols = `register_id, register_no, patient_id, department_id, doctor_id, register_sort,
	visit_date, status, register_fee, register_charged, registered_at, back_operator_id, back_operated_at`

func (s *Store) CreateRegister(ctx context.Context, r domain.Register) error {
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM dbo.register_bill
		WHERE is_deleted = 0 AND register_no = @register_no`,
		sql.Named("register_no", r.RegisterNo),
	).Scan(&n)
	if err != nil {
		return err
	}
	if n > 0 {
		return port.ErrDuplicateBusinessNo
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO dbo.register_bill (
			register_id, register_no, patient_id, department_id, doctor_id, register_sort,
			visit_date, status, register_fee, register_charged, registered_at,
			created_at, created_by, updated_at, updated_by, is_deleted
		) VALUES (
			@register_id, @register_no, @patient_id, @department_id, @doctor_id, @register_sort,
			@visit_date, @status, @register_fee, @register_charged, @registered_at,
			@created_at, @created_by, @updated_at, @updated_by, 0
		)`,
		sql.Named("register_id", r.RegisterID),
		sql.Named("register_no", r.RegisterNo),
		sql.Named("patient_id", r.PatientID),
		sql.Named("department_id", r.DepartmentID),
		sql.Named("doctor_id", r.DoctorID),
		sql.Named("register_sort", nullIfEmpty(r.RegisterSort)),
		sql.Named("visit_date", r.VisitDate.Format("2006-01-02")),
		sql.Named("status", r.Status),
		sql.Named("register_fee", centsToDecimal(r.RegisterFeeCents)),
		sql.Named("register_charged", r.RegisterCharged),
		sql.Named("registered_at", r.RegisteredAt.UTC()),
		sql.Named("created_at", r.CreatedAt.UTC()),
		sql.Named("created_by", nullIfEmpty(r.CreatedBy)),
		sql.Named("updated_at", r.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(r.UpdatedBy)),
	)
	if err != nil {
		return classifyWriteError(err)
	}
	return nil
}

// CreateRegisterWithIdempotency keeps the register bill and its idempotency
// record in one serializable transaction. Other register lifecycle operations
// intentionally remain separate scopes.
func (s *Store) CreateRegisterWithIdempotency(ctx context.Context, r domain.Register, scope, key, requestHash string, expiresAt time.Time) (domain.Register, bool, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return domain.Register{}, false, err
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
				return domain.Register{}, false, err
			}
		} else {
			if savedHash != requestHash {
				return domain.Register{}, false, port.ErrIdempotencyConflict
			}
			switch state {
			case "PENDING":
				return domain.Register{}, false, port.ErrIdempotencyInProgress
			case "COMPLETED":
				if !resourceID.Valid || resourceID.String == "" {
					return domain.Register{}, false, port.ErrNotFound
				}
				reg, getErr := scanRegister(tx.QueryRowContext(ctx, `SELECT `+registerCols+`
					FROM dbo.register_bill WHERE register_id=@register_id AND is_deleted=0`, sql.Named("register_id", resourceID.String)))
				if errors.Is(getErr, sql.ErrNoRows) {
					return domain.Register{}, false, port.ErrNotFound
				}
				if getErr != nil {
					return domain.Register{}, false, getErr
				}
				if err := tx.Commit(); err != nil {
					return domain.Register{}, false, err
				}
				return *reg, true, nil
			case "FAILED":
				if _, err := tx.ExecContext(ctx, `DELETE FROM dbo.idempotency_record WHERE scope=@scope AND idem_key=@idem_key`, sql.Named("scope", scope), sql.Named("idem_key", key)); err != nil {
					return domain.Register{}, false, err
				}
			default:
				return domain.Register{}, false, port.ErrIdempotencyInProgress
			}
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return domain.Register{}, false, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO dbo.idempotency_record(scope, idem_key, request_hash, state, resource_type, resource_id, created_at, expires_at)
		VALUES(@scope, @idem_key, @request_hash, 'PENDING', 'Register', NULL, SYSUTCDATETIME(), @expires_at)`,
		sql.Named("scope", scope), sql.Named("idem_key", key), sql.Named("request_hash", requestHash), sql.Named("expires_at", expiresAt.UTC())); err != nil {
		return domain.Register{}, false, err
	}
	if err := insertRegister(ctx, tx, r); err != nil {
		return domain.Register{}, false, classifyWriteError(err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE dbo.idempotency_record
		SET state='COMPLETED', resource_type='Register', resource_id=@resource_id, completed_at=SYSUTCDATETIME()
		WHERE scope=@scope AND idem_key=@idem_key`,
		sql.Named("resource_id", r.RegisterID), sql.Named("scope", scope), sql.Named("idem_key", key)); err != nil {
		return domain.Register{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Register{}, false, err
	}
	return r, false, nil
}

func insertRegister(ctx context.Context, execer sqlExecer, r domain.Register) error {
	_, err := execer.ExecContext(ctx, `
		INSERT INTO dbo.register_bill (
			register_id, register_no, patient_id, department_id, doctor_id, register_sort,
			visit_date, status, register_fee, register_charged, registered_at,
			created_at, created_by, updated_at, updated_by, is_deleted
		) VALUES (
			@register_id, @register_no, @patient_id, @department_id, @doctor_id, @register_sort,
			@visit_date, @status, @register_fee, @register_charged, @registered_at,
			@created_at, @created_by, @updated_at, @updated_by, 0
		)`,
		sql.Named("register_id", r.RegisterID),
		sql.Named("register_no", r.RegisterNo),
		sql.Named("patient_id", r.PatientID),
		sql.Named("department_id", r.DepartmentID),
		sql.Named("doctor_id", r.DoctorID),
		sql.Named("register_sort", nullIfEmpty(r.RegisterSort)),
		sql.Named("visit_date", r.VisitDate.Format("2006-01-02")),
		sql.Named("status", r.Status),
		sql.Named("register_fee", centsToDecimal(r.RegisterFeeCents)),
		sql.Named("register_charged", r.RegisterCharged),
		sql.Named("registered_at", r.RegisteredAt.UTC()),
		sql.Named("created_at", r.CreatedAt.UTC()),
		sql.Named("created_by", nullIfEmpty(r.CreatedBy)),
		sql.Named("updated_at", r.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(r.UpdatedBy)),
	)
	return err
}

func (s *Store) GetRegister(ctx context.Context, registerID string) (*domain.Register, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+registerCols+`
		FROM dbo.register_bill WHERE register_id = @register_id AND is_deleted = 0`,
		sql.Named("register_id", registerID))
	r, err := scanRegister(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Store) SearchRegisters(ctx context.Context, q port.RegisterQuery) ([]domain.Register, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	var visit any
	if strings.TrimSpace(q.VisitDate) != "" {
		visit = q.VisitDate
	}
	rows, err := s.db.QueryContext(ctx, `SELECT TOP (@limit) `+registerCols+`
		FROM dbo.register_bill
		WHERE is_deleted = 0
		  AND (@patient_id IS NULL OR patient_id = @patient_id)
		  AND (@register_no IS NULL OR register_no = @register_no)
		  AND (@visit_date IS NULL OR visit_date = @visit_date)
		ORDER BY registered_at, register_id`,
		sql.Named("limit", limit),
		sql.Named("patient_id", nullIfEmpty(q.PatientID)),
		sql.Named("register_no", nullIfEmpty(q.RegisterNo)),
		sql.Named("visit_date", visit),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Register, 0)
	for rows.Next() {
		r, err := scanRegister(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// CancelRegister locks the row, then writes BACK only from NORMAL.
func (s *Store) CancelRegister(ctx context.Context, registerID, operatorID string, at time.Time) (*domain.Register, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	err = tx.QueryRowContext(ctx, `
		SELECT status FROM dbo.register_bill WITH (UPDLOCK, ROWLOCK)
		WHERE register_id = @register_id AND is_deleted = 0`,
		sql.Named("register_id", registerID),
	).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	switch status {
	case domain.RegisterStatusNormal:
	case domain.RegisterStatusBack:
		return nil, port.ErrRegisterAlreadyBack
	default:
		return nil, port.ErrRegisterNotCancellable
	}
	at = at.UTC()
	res, err := tx.ExecContext(ctx, `
		UPDATE dbo.register_bill
		SET status = 'BACK',
		    back_operator_id = @back_operator_id,
		    back_operated_at = @back_operated_at,
		    updated_at = @updated_at,
		    updated_by = @updated_by
		WHERE register_id = @register_id AND is_deleted = 0 AND status = 'NORMAL'`,
		sql.Named("back_operator_id", operatorID),
		sql.Named("back_operated_at", at),
		sql.Named("updated_at", at),
		sql.Named("updated_by", nullIfEmpty(operatorID)),
		sql.Named("register_id", registerID),
	)
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n != 1 {
		return nil, port.ErrRegisterNotCancellable
	}
	reg, err := scanRegister(tx.QueryRowContext(ctx, `SELECT `+registerCols+`
		FROM dbo.register_bill WHERE register_id = @register_id AND is_deleted = 0`,
		sql.Named("register_id", registerID)))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return reg, nil
}

// CheckInRegister locks the row, then writes DIAGNOSING only from NORMAL or USED.
// Back columns are left as they are. 2026-10-12
func (s *Store) CheckInRegister(ctx context.Context, registerID, operatorID string, at time.Time) (*domain.Register, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	err = tx.QueryRowContext(ctx, `
		SELECT status FROM dbo.register_bill WITH (UPDLOCK, ROWLOCK)
		WHERE register_id = @register_id AND is_deleted = 0`,
		sql.Named("register_id", registerID),
	).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	switch status {
	case domain.RegisterStatusNormal, domain.RegisterStatusUsed:
	default:
		return nil, port.ErrRegisterNotCheckIn
	}
	at = at.UTC()
	res, err := tx.ExecContext(ctx, `
		UPDATE dbo.register_bill
		SET status = 'DIAGNOSING',
		    updated_at = @updated_at,
		    updated_by = @updated_by
		WHERE register_id = @register_id AND is_deleted = 0 AND status IN ('NORMAL', 'USED')`,
		sql.Named("updated_at", at),
		sql.Named("updated_by", nullIfEmpty(operatorID)),
		sql.Named("register_id", registerID),
	)
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n != 1 {
		return nil, port.ErrRegisterNotCheckIn
	}
	reg, err := scanRegister(tx.QueryRowContext(ctx, `SELECT `+registerCols+`
		FROM dbo.register_bill WHERE register_id = @register_id AND is_deleted = 0`,
		sql.Named("register_id", registerID)))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return reg, nil
}

// CompleteRegister locks the row, then writes DIAGNOSED only from DIAGNOSING.
func (s *Store) CompleteRegister(ctx context.Context, registerID, operatorID string, at time.Time) (*domain.Register, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	err = tx.QueryRowContext(ctx, `
		SELECT status FROM dbo.register_bill WITH (UPDLOCK, ROWLOCK)
		WHERE register_id = @register_id AND is_deleted = 0`,
		sql.Named("register_id", registerID),
	).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if status != domain.RegisterStatusDiagnosing {
		return nil, port.ErrRegisterNotComplete
	}
	at = at.UTC()
	res, err := tx.ExecContext(ctx, `
		UPDATE dbo.register_bill
		SET status = 'DIAGNOSED',
		    updated_at = @updated_at,
		    updated_by = @updated_by
		WHERE register_id = @register_id AND is_deleted = 0 AND status = 'DIAGNOSING'`,
		sql.Named("updated_at", at),
		sql.Named("updated_by", nullIfEmpty(operatorID)),
		sql.Named("register_id", registerID),
	)
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n != 1 {
		return nil, port.ErrRegisterNotComplete
	}
	reg, err := scanRegister(tx.QueryRowContext(ctx, `SELECT `+registerCols+`
		FROM dbo.register_bill WHERE register_id = @register_id AND is_deleted = 0`,
		sql.Named("register_id", registerID)))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return reg, nil
}

func (s *Store) DepartmentExists(ctx context.Context, departmentID string) (bool, error) {
	return s.masterExists(ctx, `SELECT COUNT(1) FROM dbo.department WHERE department_id = @id AND is_deleted = 0`, departmentID)
}

func (s *Store) DoctorExists(ctx context.Context, doctorID string) (bool, error) {
	return s.masterExists(ctx, `SELECT COUNT(1) FROM dbo.doctor WHERE doctor_id = @id AND is_deleted = 0`, doctorID)
}

func (s *Store) masterExists(ctx context.Context, q, id string) (bool, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, q, sql.Named("id", id)).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func scanRegister(sc scanner) (*domain.Register, error) {
	var r domain.Register
	var sort, fee, backOp sql.NullString
	var charged bool
	var visit, registered time.Time
	var backAt sql.NullTime
	if err := sc.Scan(
		&r.RegisterID, &r.RegisterNo, &r.PatientID, &r.DepartmentID, &r.DoctorID, &sort,
		&visit, &r.Status, &fee, &charged, &registered, &backOp, &backAt,
	); err != nil {
		return nil, err
	}
	r.RegisterSort = sort.String
	r.VisitDate = visit
	cents, err := decimalToCents(fee.String)
	if err != nil {
		return nil, err
	}
	r.RegisterFeeCents = cents
	r.RegisterCharged = charged
	r.RegisteredAt = registered
	r.BackOperatorID = backOp.String
	if backAt.Valid {
		r.BackOperatedAt = backAt.Time
	}
	return &r, nil
}

func centsToDecimal(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return sign + strconv.FormatInt(cents/100, 10) + "." + pad2(cents%100)
}

func pad2(n int64) string {
	if n < 10 {
		return "0" + strconv.FormatInt(n, 10)
	}
	return strconv.FormatInt(n, 10)
}

func decimalToCents(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = strings.TrimPrefix(s, "-")
	}
	whole, frac, _ := strings.Cut(s, ".")
	if whole == "" {
		whole = "0"
	}
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, err
	}
	frac = (frac + "00")[:2]
	f, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, err
	}
	cents := w*100 + f
	if neg {
		cents = -cents
	}
	return cents, nil
}
