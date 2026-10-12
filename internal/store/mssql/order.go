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

// SQL follows migrations/sqlserver/core/004_order. Do not add columns here. 2026-10-12

const orderCols = `order_id, order_no, register_id, patient_id, department_id, doctor_id,
	diagnose_name, recipe_sort, status, total_fee, charged, submitted_at, abated_at, abated_by, created_at`

const orderItemCols = `order_item_id, order_id, line_no, item_no, item_type, item_name, spec, units,
	amount, price, total_fee, usage_text, frequency_text, dosage, dosage_unit, proc_department_id`

type orderQuery interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func (s *Store) CreateOrder(ctx context.Context, o domain.Order) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var n int
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM dbo.order_sheet
		WHERE is_deleted = 0 AND order_no = @order_no`,
		sql.Named("order_no", o.OrderNo),
	).Scan(&n)
	if err != nil {
		return err
	}
	if n > 0 {
		return port.ErrDuplicateBusinessNo
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO dbo.order_sheet (
			order_id, order_no, register_id, patient_id, department_id, doctor_id,
			diagnose_name, recipe_sort, status, total_fee, charged, submitted_at,
			created_at, created_by, updated_at, updated_by, is_deleted
		) VALUES (
			@order_id, @order_no, @register_id, @patient_id, @department_id, @doctor_id,
			@diagnose_name, @recipe_sort, @status, @total_fee, @charged, @submitted_at,
			@created_at, @created_by, @updated_at, @updated_by, 0
		)`,
		sql.Named("order_id", o.OrderID),
		sql.Named("order_no", o.OrderNo),
		sql.Named("register_id", o.RegisterID),
		sql.Named("patient_id", o.PatientID),
		sql.Named("department_id", o.DepartmentID),
		sql.Named("doctor_id", o.DoctorID),
		sql.Named("diagnose_name", nullIfEmpty(o.DiagnoseName)),
		sql.Named("recipe_sort", nullIfEmpty(o.RecipeSort)),
		sql.Named("status", o.Status),
		sql.Named("total_fee", centsToDecimal(o.TotalFeeCents)),
		sql.Named("charged", o.Charged),
		sql.Named("submitted_at", nullTime(o.SubmittedAt)),
		sql.Named("created_at", o.CreatedAt.UTC()),
		sql.Named("created_by", nullIfEmpty(o.CreatedBy)),
		sql.Named("updated_at", o.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(o.UpdatedBy)),
	)
	if err != nil {
		return classifyWriteError(err)
	}
	if err := insertOrderItems(ctx, tx, o.Items, o.CreatedAt); err != nil {
		return classifyWriteError(err)
	}
	return tx.Commit()
}

func (s *Store) GetOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	return loadOrder(ctx, s.db, orderID)
}

func (s *Store) SearchOrders(ctx context.Context, q port.OrderQuery) ([]domain.Order, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT TOP (@limit) `+orderCols+`
		FROM dbo.order_sheet
		WHERE is_deleted = 0
		  AND (@register_id IS NULL OR register_id = @register_id)
		  AND (@patient_id IS NULL OR patient_id = @patient_id)
		  AND (@order_no IS NULL OR order_no = @order_no)
		  AND (@status IS NULL OR status = @status)
		ORDER BY created_at, order_id`,
		sql.Named("limit", limit),
		sql.Named("register_id", nullIfEmpty(q.RegisterID)),
		sql.Named("patient_id", nullIfEmpty(q.PatientID)),
		sql.Named("order_no", nullIfEmpty(q.OrderNo)),
		sql.Named("status", nullIfEmpty(q.Status)),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Order, 0)
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		items, err := listOrderItems(ctx, s.db, out[i].OrderID)
		if err != nil {
			return nil, err
		}
		out[i].Items = items
	}
	return out, nil
}

func (s *Store) UpdateOrderDraft(ctx context.Context, o domain.Order, replaceItems bool) (*domain.Order, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	err = tx.QueryRowContext(ctx, `
		SELECT status FROM dbo.order_sheet WITH (UPDLOCK, ROWLOCK)
		WHERE order_id = @order_id AND is_deleted = 0`,
		sql.Named("order_id", o.OrderID),
	).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if status != domain.OrderStatusSave {
		return nil, port.ErrOrderNotEditable
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE dbo.order_sheet
		SET patient_id = @patient_id,
		    department_id = @department_id,
		    doctor_id = @doctor_id,
		    diagnose_name = @diagnose_name,
		    recipe_sort = @recipe_sort,
		    total_fee = @total_fee,
		    updated_at = @updated_at,
		    updated_by = @updated_by
		WHERE order_id = @order_id AND is_deleted = 0 AND status = 'SAVE'`,
		sql.Named("patient_id", o.PatientID),
		sql.Named("department_id", o.DepartmentID),
		sql.Named("doctor_id", o.DoctorID),
		sql.Named("diagnose_name", nullIfEmpty(o.DiagnoseName)),
		sql.Named("recipe_sort", nullIfEmpty(o.RecipeSort)),
		sql.Named("total_fee", centsToDecimal(o.TotalFeeCents)),
		sql.Named("updated_at", o.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(o.UpdatedBy)),
		sql.Named("order_id", o.OrderID),
	)
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n != 1 {
		return nil, port.ErrOrderNotEditable
	}
	if replaceItems {
		if _, err := tx.ExecContext(ctx, `DELETE FROM dbo.order_item WHERE order_id = @order_id`, sql.Named("order_id", o.OrderID)); err != nil {
			return nil, err
		}
		if err := insertOrderItems(ctx, tx, o.Items, o.UpdatedAt); err != nil {
			return nil, classifyWriteError(err)
		}
	}
	loaded, err := loadOrder(ctx, tx, o.OrderID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return loaded, nil
}

func (s *Store) CommitOrder(ctx context.Context, o domain.Order) (*domain.Order, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	err = tx.QueryRowContext(ctx, `
		SELECT status FROM dbo.order_sheet WITH (UPDLOCK, ROWLOCK)
		WHERE order_id = @order_id AND is_deleted = 0`,
		sql.Named("order_id", o.OrderID),
	).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if status != domain.OrderStatusSave {
		return nil, port.ErrOrderNotCommittable
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE dbo.order_sheet
		SET status = 'COMMIT',
		    total_fee = @total_fee,
		    submitted_at = @submitted_at,
		    updated_at = @updated_at,
		    updated_by = @updated_by
		WHERE order_id = @order_id AND is_deleted = 0 AND status = 'SAVE'`,
		sql.Named("total_fee", centsToDecimal(o.TotalFeeCents)),
		sql.Named("submitted_at", o.SubmittedAt.UTC()),
		sql.Named("updated_at", o.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(o.UpdatedBy)),
		sql.Named("order_id", o.OrderID),
	)
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n != 1 {
		return nil, port.ErrOrderNotCommittable
	}
	for _, it := range o.Items {
		if _, err := tx.ExecContext(ctx, `
			UPDATE dbo.order_item
			SET total_fee = @total_fee, updated_at = @updated_at
			WHERE order_item_id = @order_item_id AND order_id = @order_id`,
			sql.Named("total_fee", centsToDecimal(it.TotalFeeCents)),
			sql.Named("updated_at", o.UpdatedAt.UTC()),
			sql.Named("order_item_id", it.OrderItemID),
			sql.Named("order_id", o.OrderID),
		); err != nil {
			return nil, err
		}
	}
	loaded, err := loadOrder(ctx, tx, o.OrderID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return loaded, nil
}

func (s *Store) AbateOrder(ctx context.Context, orderID, operatorID string, at time.Time) (*domain.Order, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	var charged bool
	err = tx.QueryRowContext(ctx, `
		SELECT status, charged FROM dbo.order_sheet WITH (UPDLOCK, ROWLOCK)
		WHERE order_id = @order_id AND is_deleted = 0`,
		sql.Named("order_id", orderID),
	).Scan(&status, &charged)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if charged {
		return nil, port.ErrOrderChargedNotAbatable
	}
	if status != domain.OrderStatusSave && status != domain.OrderStatusCommit {
		return nil, port.ErrIllegalStateTransition
	}
	at = at.UTC()
	res, err := tx.ExecContext(ctx, `
		UPDATE dbo.order_sheet
		SET status = 'ABATE',
		    abated_at = @abated_at,
		    abated_by = @abated_by,
		    updated_at = @updated_at,
		    updated_by = @updated_by
		WHERE order_id = @order_id AND is_deleted = 0 AND charged = 0
		  AND status IN ('SAVE', 'COMMIT')`,
		sql.Named("abated_at", at),
		sql.Named("abated_by", operatorID),
		sql.Named("updated_at", at),
		sql.Named("updated_by", nullIfEmpty(operatorID)),
		sql.Named("order_id", orderID),
	)
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n != 1 {
		return nil, port.ErrIllegalStateTransition
	}
	loaded, err := loadOrder(ctx, tx, orderID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return loaded, nil
}

func (s *Store) CatalogByItemNo(ctx context.Context, itemNo string) (*domain.CatalogItem, error) {
	var c domain.CatalogItem
	var spec sql.NullString
	var price sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT catalog_item_id, item_no, item_name, item_type, spec, units, price, status
		FROM dbo.catalog_item
		WHERE item_no = @item_no AND is_deleted = 0 AND status = 'ACTIVE'`,
		sql.Named("item_no", itemNo),
	).Scan(&c.CatalogItemID, &c.ItemNo, &c.ItemName, &c.ItemType, &spec, &c.Units, &price, &c.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.Spec = spec.String
	cents, err := decimalToCents(price.String)
	if err != nil {
		return nil, err
	}
	c.PriceCents = cents
	return &c, nil
}

func insertOrderItems(ctx context.Context, tx *sql.Tx, items []domain.OrderItem, now time.Time) error {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	for _, it := range items {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO dbo.order_item (
				order_item_id, order_id, line_no, item_no, item_type, item_name, spec, units,
				amount, price, total_fee, usage_text, frequency_text, dosage, dosage_unit, proc_department_id,
				created_at, updated_at
			) VALUES (
				@order_item_id, @order_id, @line_no, @item_no, @item_type, @item_name, @spec, @units,
				@amount, @price, @total_fee, @usage_text, @frequency_text, @dosage, @dosage_unit, @proc_department_id,
				@created_at, @updated_at
			)`,
			sql.Named("order_item_id", it.OrderItemID),
			sql.Named("order_id", it.OrderID),
			sql.Named("line_no", it.LineNo),
			sql.Named("item_no", it.ItemNo),
			sql.Named("item_type", it.ItemType),
			sql.Named("item_name", it.ItemName),
			sql.Named("spec", nullIfEmpty(it.Spec)),
			sql.Named("units", it.Units),
			sql.Named("amount", milliToDecimal(it.AmountMilli)),
			sql.Named("price", centsToDecimal(it.PriceCents)),
			sql.Named("total_fee", centsToDecimal(it.TotalFeeCents)),
			sql.Named("usage_text", nullIfEmpty(it.UsageText)),
			sql.Named("frequency_text", nullIfEmpty(it.FrequencyText)),
			sql.Named("dosage", nullIfEmpty(it.Dosage)),
			sql.Named("dosage_unit", nullIfEmpty(it.DosageUnit)),
			sql.Named("proc_department_id", nullIfEmpty(it.ProcDepartmentID)),
			sql.Named("created_at", now),
			sql.Named("updated_at", now),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func loadOrder(ctx context.Context, q orderQuery, orderID string) (*domain.Order, error) {
	o, err := scanOrder(q.QueryRowContext(ctx, `SELECT `+orderCols+`
		FROM dbo.order_sheet WHERE order_id = @order_id AND is_deleted = 0`,
		sql.Named("order_id", orderID)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	items, err := listOrderItems(ctx, q, orderID)
	if err != nil {
		return nil, err
	}
	o.Items = items
	return o, nil
}

func listOrderItems(ctx context.Context, q orderQuery, orderID string) ([]domain.OrderItem, error) {
	rows, err := q.QueryContext(ctx, `SELECT `+orderItemCols+`
		FROM dbo.order_item WHERE order_id = @order_id ORDER BY line_no, order_item_id`,
		sql.Named("order_id", orderID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.OrderItem, 0)
	for rows.Next() {
		it, err := scanOrderItem(rows)
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

func scanOrder(sc scanner) (*domain.Order, error) {
	var o domain.Order
	var diagnose, sort, fee, abatedBy sql.NullString
	var charged bool
	var submitted, abated sql.NullTime
	var created time.Time
	if err := sc.Scan(
		&o.OrderID, &o.OrderNo, &o.RegisterID, &o.PatientID, &o.DepartmentID, &o.DoctorID,
		&diagnose, &sort, &o.Status, &fee, &charged, &submitted, &abated, &abatedBy, &created,
	); err != nil {
		return nil, err
	}
	o.DiagnoseName = diagnose.String
	o.RecipeSort = sort.String
	cents, err := decimalToCents(fee.String)
	if err != nil {
		return nil, err
	}
	o.TotalFeeCents = cents
	o.Charged = charged
	if submitted.Valid {
		o.SubmittedAt = submitted.Time
	}
	if abated.Valid {
		o.AbatedAt = abated.Time
	}
	o.AbatedBy = abatedBy.String
	o.CreatedAt = created
	if o.Items == nil {
		o.Items = []domain.OrderItem{}
	}
	return &o, nil
}

func scanOrderItem(sc scanner) (*domain.OrderItem, error) {
	var it domain.OrderItem
	var spec, amount, price, fee sql.NullString
	var usage, freq, dosage, dosageUnit, proc sql.NullString
	if err := sc.Scan(
		&it.OrderItemID, &it.OrderID, &it.LineNo, &it.ItemNo, &it.ItemType, &it.ItemName, &spec, &it.Units,
		&amount, &price, &fee, &usage, &freq, &dosage, &dosageUnit, &proc,
	); err != nil {
		return nil, err
	}
	it.Spec = spec.String
	milli, err := decimalToMilli(amount.String)
	if err != nil {
		return nil, err
	}
	it.AmountMilli = milli
	pc, err := decimalToCents(price.String)
	if err != nil {
		return nil, err
	}
	it.PriceCents = pc
	tc, err := decimalToCents(fee.String)
	if err != nil {
		return nil, err
	}
	it.TotalFeeCents = tc
	it.UsageText = usage.String
	it.FrequencyText = freq.String
	it.Dosage = dosage.String
	it.DosageUnit = dosageUnit.String
	it.ProcDepartmentID = proc.String
	return &it, nil
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC()
}

func milliToDecimal(m int64) string {
	sign := ""
	if m < 0 {
		sign = "-"
		m = -m
	}
	return sign + strconv.FormatInt(m/1000, 10) + "." + pad3(m%1000)
}

func pad3(n int64) string {
	s := strconv.FormatInt(n, 10)
	for len(s) < 3 {
		s = "0" + s
	}
	return s
}

func decimalToMilli(s string) (int64, error) {
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
	for len(frac) < 3 {
		frac += "0"
	}
	extra := ""
	if len(frac) > 3 {
		extra = frac[3:]
		frac = frac[:3]
	}
	f, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, err
	}
	if extra != "" && extra[0] >= '5' {
		f++
	}
	if f >= 1000 {
		w++
		f -= 1000
	}
	v := w*1000 + f
	if neg {
		v = -v
	}
	return v, nil
}
