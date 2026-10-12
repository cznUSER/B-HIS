package mssql

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

// Master-data SQL follows 003/004/006 plus 008_master (sort_no, is_controlled, dict_item).
// 008 DDL is not changed here. 2026-10-13

const departmentCols = `department_id, dept_code, dept_name, status, sort_no, created_at, created_by, updated_at, updated_by`

const doctorCols = `doctor_id, department_id, doctor_code, doctor_name, status, created_at, created_by, updated_at, updated_by`

const catalogCols = `catalog_item_id, item_no, item_name, item_type, spec, units, CONVERT(varchar(40), price), status, is_controlled, created_at, created_by, updated_at, updated_by`

func (s *Store) ListDepartments(ctx context.Context, q port.DepartmentQuery) ([]domain.Department, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+departmentCols+`
		FROM dbo.department
		WHERE is_deleted = 0
		  AND (@status IS NULL OR status = @status)
		  AND (@q IS NULL OR CHARINDEX(@q, dept_code) > 0 OR CHARINDEX(@q, dept_name) > 0)
		ORDER BY sort_no, dept_code`,
		sql.Named("status", nullIfEmpty(q.Status)),
		sql.Named("q", nullIfEmpty(q.Q)),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Department, 0)
	for rows.Next() {
		d, err := scanDepartment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (s *Store) GetDepartment(ctx context.Context, departmentID string) (*domain.Department, error) {
	d, err := scanDepartment(s.db.QueryRowContext(ctx, `
		SELECT `+departmentCols+`
		FROM dbo.department
		WHERE department_id = @id AND is_deleted = 0`, sql.Named("id", departmentID)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	return d, err
}

func (s *Store) CreateDepartment(ctx context.Context, d domain.Department) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO dbo.department (
			department_id, dept_code, dept_name, status, sort_no,
			created_at, created_by, updated_at, updated_by
		) VALUES (
			@department_id, @dept_code, @dept_name, @status, @sort_no,
			@created_at, @created_by, @updated_at, @updated_by
		)`, departmentArgs(d)...)
	return classifyWriteError(err)
}

func (s *Store) UpdateDepartment(ctx context.Context, d domain.Department) (*domain.Department, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE dbo.department
		SET dept_code = @dept_code, dept_name = @dept_name, status = @status, sort_no = @sort_no,
			updated_at = @updated_at, updated_by = @updated_by
		WHERE department_id = @department_id AND is_deleted = 0`,
		sql.Named("department_id", d.DepartmentID),
		sql.Named("dept_code", d.DeptCode),
		sql.Named("dept_name", d.DeptName),
		sql.Named("status", d.Status),
		sql.Named("sort_no", d.SortNo),
		sql.Named("updated_at", d.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(d.UpdatedBy)),
	)
	if err != nil {
		return nil, classifyWriteError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, port.ErrNotFound
	}
	return s.GetDepartment(ctx, d.DepartmentID)
}

func departmentArgs(d domain.Department) []any {
	return []any{
		sql.Named("department_id", d.DepartmentID),
		sql.Named("dept_code", d.DeptCode),
		sql.Named("dept_name", d.DeptName),
		sql.Named("status", d.Status),
		sql.Named("sort_no", d.SortNo),
		sql.Named("created_at", d.CreatedAt.UTC()),
		sql.Named("created_by", nullIfEmpty(d.CreatedBy)),
		sql.Named("updated_at", d.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(d.UpdatedBy)),
	}
}

func scanDepartment(sc scanner) (*domain.Department, error) {
	var d domain.Department
	var createdBy, updatedBy sql.NullString
	if err := sc.Scan(
		&d.DepartmentID, &d.DeptCode, &d.DeptName, &d.Status, &d.SortNo,
		&d.CreatedAt, &createdBy, &d.UpdatedAt, &updatedBy,
	); err != nil {
		return nil, err
	}
	d.CreatedBy = createdBy.String
	d.UpdatedBy = updatedBy.String
	return &d, nil
}

func (s *Store) ListDoctors(ctx context.Context, q port.DoctorQuery) ([]domain.Doctor, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+doctorCols+`
		FROM dbo.doctor
		WHERE is_deleted = 0
		  AND (@department_id IS NULL OR department_id = @department_id)
		  AND (@status IS NULL OR status = @status)
		  AND (@q IS NULL OR CHARINDEX(@q, doctor_code) > 0 OR CHARINDEX(@q, doctor_name) > 0)
		ORDER BY doctor_code`,
		sql.Named("department_id", nullIfEmpty(q.DepartmentID)),
		sql.Named("status", nullIfEmpty(q.Status)),
		sql.Named("q", nullIfEmpty(q.Q)),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Doctor, 0)
	for rows.Next() {
		d, err := scanDoctor(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (s *Store) GetDoctor(ctx context.Context, doctorID string) (*domain.Doctor, error) {
	d, err := scanDoctor(s.db.QueryRowContext(ctx, `
		SELECT `+doctorCols+`
		FROM dbo.doctor
		WHERE doctor_id = @id AND is_deleted = 0`, sql.Named("id", doctorID)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	return d, err
}

func (s *Store) CreateDoctor(ctx context.Context, d domain.Doctor) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO dbo.doctor (
			doctor_id, department_id, doctor_code, doctor_name, status,
			created_at, created_by, updated_at, updated_by
		) VALUES (
			@doctor_id, @department_id, @doctor_code, @doctor_name, @status,
			@created_at, @created_by, @updated_at, @updated_by
		)`, doctorArgs(d)...)
	return classifyWriteError(err)
}

func (s *Store) UpdateDoctor(ctx context.Context, d domain.Doctor) (*domain.Doctor, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE dbo.doctor
		SET department_id = @department_id, doctor_code = @doctor_code, doctor_name = @doctor_name,
			status = @status, updated_at = @updated_at, updated_by = @updated_by
		WHERE doctor_id = @doctor_id AND is_deleted = 0`,
		sql.Named("doctor_id", d.DoctorID),
		sql.Named("department_id", d.DepartmentID),
		sql.Named("doctor_code", d.DoctorCode),
		sql.Named("doctor_name", d.DoctorName),
		sql.Named("status", d.Status),
		sql.Named("updated_at", d.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(d.UpdatedBy)),
	)
	if err != nil {
		return nil, classifyWriteError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, port.ErrNotFound
	}
	return s.GetDoctor(ctx, d.DoctorID)
}

func doctorArgs(d domain.Doctor) []any {
	return []any{
		sql.Named("doctor_id", d.DoctorID),
		sql.Named("department_id", d.DepartmentID),
		sql.Named("doctor_code", d.DoctorCode),
		sql.Named("doctor_name", d.DoctorName),
		sql.Named("status", d.Status),
		sql.Named("created_at", d.CreatedAt.UTC()),
		sql.Named("created_by", nullIfEmpty(d.CreatedBy)),
		sql.Named("updated_at", d.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(d.UpdatedBy)),
	}
}

func scanDoctor(sc scanner) (*domain.Doctor, error) {
	var d domain.Doctor
	var createdBy, updatedBy sql.NullString
	if err := sc.Scan(
		&d.DoctorID, &d.DepartmentID, &d.DoctorCode, &d.DoctorName, &d.Status,
		&d.CreatedAt, &createdBy, &d.UpdatedAt, &updatedBy,
	); err != nil {
		return nil, err
	}
	d.CreatedBy = createdBy.String
	d.UpdatedBy = updatedBy.String
	return &d, nil
}

func (s *Store) ListCatalog(ctx context.Context, q port.CatalogQuery) ([]domain.CatalogItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+catalogCols+`
		FROM dbo.catalog_item
		WHERE is_deleted = 0
		  AND (@item_type IS NULL OR item_type = @item_type)
		  AND (@status IS NULL OR status = @status)
		  AND (@q IS NULL OR CHARINDEX(@q, item_no) > 0 OR CHARINDEX(@q, item_name) > 0)
		ORDER BY item_no`,
		sql.Named("item_type", nullIfEmpty(q.ItemType)),
		sql.Named("status", nullIfEmpty(q.Status)),
		sql.Named("q", nullIfEmpty(q.Q)),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.CatalogItem, 0)
	for rows.Next() {
		c, err := scanCatalog(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (s *Store) GetCatalog(ctx context.Context, catalogItemID string) (*domain.CatalogItem, error) {
	c, err := scanCatalog(s.db.QueryRowContext(ctx, `
		SELECT `+catalogCols+`
		FROM dbo.catalog_item
		WHERE catalog_item_id = @id AND is_deleted = 0`, sql.Named("id", catalogItemID)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	return c, err
}

func (s *Store) CreateCatalog(ctx context.Context, item domain.CatalogItem, skuID string) (*domain.CatalogItem, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO dbo.catalog_item (
			catalog_item_id, item_no, item_name, item_type, spec, units, price, status, is_controlled,
			created_at, created_by, updated_at, updated_by
		) VALUES (
			@catalog_item_id, @item_no, @item_name, @item_type, @spec, @units, @price, @status, @is_controlled,
			@created_at, @created_by, @updated_at, @updated_by
		)`, catalogArgs(item)...); err != nil {
		return nil, classifyWriteError(err)
	}
	if item.ItemType == domain.ItemTypeDrug {
		if skuID == "" {
			return nil, port.ErrNotDrugSKU
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO dbo.inventory_item (
				sku_id, item_no, item_name, qty_on_hand, units, status,
				created_at, created_by, updated_at, updated_by
			) VALUES (
				@sku_id, @item_no, @item_name, 0, @units, 'ACTIVE',
				@created_at, @created_by, @updated_at, @updated_by
			)`,
			sql.Named("sku_id", skuID),
			sql.Named("item_no", item.ItemNo),
			sql.Named("item_name", item.ItemName),
			sql.Named("units", item.Units),
			sql.Named("created_at", item.CreatedAt.UTC()),
			sql.Named("created_by", nullIfEmpty(item.CreatedBy)),
			sql.Named("updated_at", item.UpdatedAt.UTC()),
			sql.Named("updated_by", nullIfEmpty(item.UpdatedBy)),
		); err != nil {
			return nil, classifyWriteError(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	cp := item
	return &cp, nil
}

func (s *Store) UpdateCatalog(ctx context.Context, item domain.CatalogItem) (*domain.CatalogItem, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var oldNo, itemType string
	err = tx.QueryRowContext(ctx, `
		SELECT item_no, item_type FROM dbo.catalog_item WITH (UPDLOCK, ROWLOCK)
		WHERE catalog_item_id = @id AND is_deleted = 0`, sql.Named("id", item.CatalogItemID)).Scan(&oldNo, &itemType)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	item.ItemType = itemType
	if _, err := tx.ExecContext(ctx, `
		UPDATE dbo.catalog_item
		SET item_no = @item_no, item_name = @item_name, spec = @spec, units = @units,
			price = @price, status = @status, is_controlled = @is_controlled,
			updated_at = @updated_at, updated_by = @updated_by
		WHERE catalog_item_id = @catalog_item_id AND is_deleted = 0`,
		sql.Named("catalog_item_id", item.CatalogItemID),
		sql.Named("item_no", item.ItemNo),
		sql.Named("item_name", item.ItemName),
		sql.Named("spec", nullIfEmpty(item.Spec)),
		sql.Named("units", item.Units),
		sql.Named("price", centsToDecimal(item.PriceCents)),
		sql.Named("status", item.Status),
		sql.Named("is_controlled", controlledBit(item.Controlled)),
		sql.Named("updated_at", item.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(item.UpdatedBy)),
	); err != nil {
		return nil, classifyWriteError(err)
	}
	if itemType == domain.ItemTypeDrug {
		res, err := tx.ExecContext(ctx, `
			UPDATE dbo.inventory_item
			SET item_no = @item_no, item_name = @item_name, units = @units,
				updated_at = @updated_at, updated_by = @updated_by
			WHERE item_no = @old_no AND is_deleted = 0`,
			sql.Named("item_no", item.ItemNo),
			sql.Named("item_name", item.ItemName),
			sql.Named("units", item.Units),
			sql.Named("updated_at", item.UpdatedAt.UTC()),
			sql.Named("updated_by", nullIfEmpty(item.UpdatedBy)),
			sql.Named("old_no", oldNo),
		)
		if err != nil {
			return nil, classifyWriteError(err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}
		if n == 0 {
			skuID := "sku-" + item.CatalogItemID
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO dbo.inventory_item (
					sku_id, item_no, item_name, qty_on_hand, units, status,
					created_at, created_by, updated_at, updated_by
				) VALUES (
					@sku_id, @item_no, @item_name, 0, @units, 'ACTIVE',
					@updated_at, @updated_by, @updated_at, @updated_by
				)`,
				sql.Named("sku_id", skuID),
				sql.Named("item_no", item.ItemNo),
				sql.Named("item_name", item.ItemName),
				sql.Named("units", item.Units),
				sql.Named("updated_at", item.UpdatedAt.UTC()),
				sql.Named("updated_by", nullIfEmpty(item.UpdatedBy)),
			); err != nil {
				return nil, classifyWriteError(err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetCatalog(ctx, item.CatalogItemID)
}

func controlledBit(v bool) int {
	if v {
		return 1
	}
	return 0
}

func catalogArgs(item domain.CatalogItem) []any {
	return []any{
		sql.Named("catalog_item_id", item.CatalogItemID),
		sql.Named("item_no", item.ItemNo),
		sql.Named("item_name", item.ItemName),
		sql.Named("item_type", item.ItemType),
		sql.Named("spec", nullIfEmpty(item.Spec)),
		sql.Named("units", item.Units),
		sql.Named("price", centsToDecimal(item.PriceCents)),
		sql.Named("status", item.Status),
		sql.Named("is_controlled", controlledBit(item.Controlled)),
		sql.Named("created_at", item.CreatedAt.UTC()),
		sql.Named("created_by", nullIfEmpty(item.CreatedBy)),
		sql.Named("updated_at", item.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(item.UpdatedBy)),
	}
}

func scanCatalog(sc scanner) (*domain.CatalogItem, error) {
	var c domain.CatalogItem
	var spec, price, createdBy, updatedBy sql.NullString
	var controlled bool
	if err := sc.Scan(
		&c.CatalogItemID, &c.ItemNo, &c.ItemName, &c.ItemType, &spec, &c.Units, &price, &c.Status, &controlled,
		&c.CreatedAt, &createdBy, &c.UpdatedAt, &updatedBy,
	); err != nil {
		return nil, err
	}
	c.Spec = spec.String
	c.Controlled = controlled
	c.CreatedBy = createdBy.String
	c.UpdatedBy = updatedBy.String
	cents, err := decimalToCents(price.String)
	if err != nil {
		return nil, err
	}
	c.PriceCents = cents
	return &c, nil
}

func (s *Store) AdjustInventory(ctx context.Context, skuID string, deltaMilli int64, remark, operatorID, rowVersion, ledgerID string, at time.Time) (*domain.InventoryItem, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var qtyText, rv string
	err = tx.QueryRowContext(ctx, `
		SELECT CONVERT(varchar(40), qty_on_hand), CONVERT(varchar(30), CONVERT(bigint, row_version))
		FROM dbo.inventory_item WITH (UPDLOCK, ROWLOCK)
		WHERE sku_id = @sku_id AND is_deleted = 0`, sql.Named("sku_id", skuID)).Scan(&qtyText, &rv)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotDrugSKU
	}
	if err != nil {
		return nil, err
	}
	on, err := decimalToMilli(qtyText)
	if err != nil {
		return nil, err
	}
	if rowVersion != "" && rowVersion != rv {
		return nil, port.ErrConflict
	}
	if deltaMilli < 0 {
		need := -deltaMilli
		if deltaMilli == math.MinInt64 || on < need {
			return nil, port.ErrAdjustNegative
		}
	}
	if deltaMilli > 0 && on > math.MaxInt64-deltaMilli {
		return nil, port.ErrAdjustNegative
	}
	at = at.UTC()
	if _, err := tx.ExecContext(ctx, `
		UPDATE dbo.inventory_item
		SET qty_on_hand = qty_on_hand + @delta, updated_at = @updated_at, updated_by = @updated_by
		WHERE sku_id = @sku_id AND is_deleted = 0`,
		sql.Named("delta", milliToDecimal(deltaMilli)),
		sql.Named("updated_at", at),
		sql.Named("updated_by", nullIfEmpty(operatorID)),
		sql.Named("sku_id", skuID),
	); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO dbo.inventory_ledger (
			ledger_id, sku_id, qty_delta, biz_type, biz_id, remark, created_at, created_by
		) VALUES (
			@ledger_id, @sku_id, @qty_delta, 'ADJUST', @ledger_id, @remark, @created_at, @created_by
		)`,
		sql.Named("ledger_id", ledgerID),
		sql.Named("sku_id", skuID),
		sql.Named("qty_delta", milliToDecimal(deltaMilli)),
		sql.Named("remark", remark),
		sql.Named("created_at", at),
		sql.Named("created_by", nullIfEmpty(operatorID)),
	); err != nil {
		return nil, err
	}
	item, err := scanInventory(tx.QueryRowContext(ctx, `
		SELECT sku_id, item_no, item_name, CONVERT(varchar(40), qty_on_hand), units, status,
			CONVERT(varchar(30), CONVERT(bigint, row_version)), updated_at
		FROM dbo.inventory_item
		WHERE sku_id = @sku_id AND is_deleted = 0`, sql.Named("sku_id", skuID)))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return item, nil
}

func scanInventory(sc scanner) (*domain.InventoryItem, error) {
	var it domain.InventoryItem
	var qty sql.NullString
	if err := sc.Scan(&it.SkuID, &it.ItemNo, &it.ItemName, &qty, &it.Units, &it.Status, &it.RowVersion, &it.UpdatedAt); err != nil {
		return nil, err
	}
	milli, err := decimalToMilli(qty.String)
	if err != nil {
		return nil, err
	}
	it.QtyOnHandMilli = milli
	return &it, nil
}

func (s *Store) ListDictItems(ctx context.Context, dictType, status string) ([]domain.DictItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT dict_item_id, dict_type, code, name, status, sort_no, created_at, created_by, updated_at, updated_by
		FROM dbo.dict_item
		WHERE is_deleted = 0 AND dict_type = @dict_type
		  AND (@status IS NULL OR status = @status)
		ORDER BY sort_no, code`,
		sql.Named("dict_type", dictType),
		sql.Named("status", nullIfEmpty(status)),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.DictItem, 0)
	for rows.Next() {
		it, err := scanDict(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *it)
	}
	return out, rows.Err()
}

func (s *Store) CreateDictItem(ctx context.Context, item domain.DictItem) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO dbo.dict_item (
			dict_item_id, dict_type, code, name, status, sort_no, created_at, created_by, updated_at, updated_by
		) VALUES (
			@dict_item_id, @dict_type, @code, @name, @status, @sort_no, @created_at, @created_by, @updated_at, @updated_by
		)`, dictArgs(item)...)
	return classifyWriteError(err)
}

func (s *Store) UpdateDictItem(ctx context.Context, item domain.DictItem) (*domain.DictItem, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE dbo.dict_item
		SET name = @name, status = @status, sort_no = @sort_no, updated_at = @updated_at, updated_by = @updated_by
		WHERE dict_item_id = @dict_item_id AND dict_type = @dict_type AND is_deleted = 0`,
		sql.Named("dict_item_id", item.DictItemID),
		sql.Named("dict_type", item.DictType),
		sql.Named("name", item.Name),
		sql.Named("status", item.Status),
		sql.Named("sort_no", item.SortNo),
		sql.Named("updated_at", item.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(item.UpdatedBy)),
	)
	if err != nil {
		return nil, classifyWriteError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, port.ErrNotFound
	}
	got, err := scanDict(s.db.QueryRowContext(ctx, `
		SELECT dict_item_id, dict_type, code, name, status, sort_no, created_at, created_by, updated_at, updated_by
		FROM dbo.dict_item
		WHERE dict_item_id = @id AND is_deleted = 0`, sql.Named("id", item.DictItemID)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	return got, err
}

func dictArgs(item domain.DictItem) []any {
	return []any{
		sql.Named("dict_item_id", item.DictItemID),
		sql.Named("dict_type", item.DictType),
		sql.Named("code", item.Code),
		sql.Named("name", item.Name),
		sql.Named("status", item.Status),
		sql.Named("sort_no", item.SortNo),
		sql.Named("created_at", item.CreatedAt.UTC()),
		sql.Named("created_by", nullIfEmpty(item.CreatedBy)),
		sql.Named("updated_at", item.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(item.UpdatedBy)),
	}
}

func scanDict(sc scanner) (*domain.DictItem, error) {
	var it domain.DictItem
	var createdBy, updatedBy sql.NullString
	if err := sc.Scan(
		&it.DictItemID, &it.DictType, &it.Code, &it.Name, &it.Status, &it.SortNo,
		&it.CreatedAt, &createdBy, &it.UpdatedAt, &updatedBy,
	); err != nil {
		return nil, err
	}
	it.CreatedBy = createdBy.String
	it.UpdatedBy = updatedBy.String
	return &it, nil
}

func (s *Store) ListUsers(ctx context.Context, q port.UserQuery) ([]domain.User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id, username, display_name, status, created_at, updated_at
		FROM dbo.app_user
		WHERE is_deleted = 0
		  AND (@status IS NULL OR status = @status)
		  AND (@q IS NULL OR CHARINDEX(@q, username) > 0 OR CHARINDEX(@q, display_name) > 0)
		ORDER BY username`,
		sql.Named("status", nullIfEmpty(q.Status)),
		sql.Named("q", nullIfEmpty(q.Q)),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.User, 0)
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		roles, err := loadRoles(ctx, s.db, u.ID)
		if err != nil {
			return nil, err
		}
		u.Roles = roles
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) CreateUser(ctx context.Context, u domain.User) (*domain.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO dbo.app_user (
			user_id, username, display_name, password_hash, status, created_at, created_by, updated_at, updated_by
		) VALUES (
			@user_id, @username, @display_name, @password_hash, @status, @created_at, @created_by, @updated_at, @updated_by
		)`,
		sql.Named("user_id", u.ID),
		sql.Named("username", u.Username),
		sql.Named("display_name", u.DisplayName),
		sql.Named("password_hash", u.PasswordHash),
		sql.Named("status", u.Status),
		sql.Named("created_at", u.CreatedAt.UTC()),
		sql.Named("created_by", nullIfEmpty(u.CreatedBy)),
		sql.Named("updated_at", u.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(u.UpdatedBy)),
	); err != nil {
		return nil, classifyWriteError(err)
	}
	if err := insertRoles(ctx, tx, u.ID, u.Roles); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	u.PasswordHash = ""
	if u.Roles == nil {
		u.Roles = []domain.Role{}
	}
	return &u, nil
}

func (s *Store) UpdateUser(ctx context.Context, u domain.User, replaceRoles bool) (*domain.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, `
		UPDATE dbo.app_user
		SET display_name = @display_name, status = @status, updated_at = @updated_at, updated_by = @updated_by
		WHERE user_id = @user_id AND is_deleted = 0`,
		sql.Named("display_name", u.DisplayName),
		sql.Named("status", u.Status),
		sql.Named("updated_at", u.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(u.UpdatedBy)),
		sql.Named("user_id", u.ID),
	)
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, port.ErrNotFound
	}
	if replaceRoles {
		if _, err := tx.ExecContext(ctx, `DELETE FROM dbo.user_role WHERE user_id = @user_id`, sql.Named("user_id", u.ID)); err != nil {
			return nil, err
		}
		if err := insertRoles(ctx, tx, u.ID, u.Roles); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.adminUser(ctx, u.ID)
}

func (s *Store) ResetPassword(ctx context.Context, userID, passwordHash, operatorID string, at time.Time) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE dbo.app_user
		SET password_hash = @password_hash, updated_at = @updated_at, updated_by = @updated_by
		WHERE user_id = @user_id AND is_deleted = 0`,
		sql.Named("password_hash", passwordHash),
		sql.Named("updated_at", at.UTC()),
		sql.Named("updated_by", nullIfEmpty(operatorID)),
		sql.Named("user_id", userID),
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

func (s *Store) adminUser(ctx context.Context, userID string) (*domain.User, error) {
	var u domain.User
	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, username, display_name, status, created_at, updated_at
		FROM dbo.app_user WHERE user_id = @user_id AND is_deleted = 0`, sql.Named("user_id", userID)).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	roles, err := loadRoles(ctx, s.db, u.ID)
	if err != nil {
		return nil, err
	}
	u.Roles = roles
	return &u, nil
}

type roleQuery interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func loadRoles(ctx context.Context, q interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}, userID string) ([]domain.Role, error) {
	rows, err := q.QueryContext(ctx, `SELECT role_code FROM dbo.user_role WHERE user_id = @user_id ORDER BY role_code`, sql.Named("user_id", userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Role, 0)
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		out = append(out, domain.Role(code))
	}
	return out, rows.Err()
}

func insertRoles(ctx context.Context, q roleQuery, userID string, roles []domain.Role) error {
	for _, role := range roles {
		if _, err := q.ExecContext(ctx, `
			INSERT INTO dbo.user_role (user_id, role_code) VALUES (@user_id, @role_code)`,
			sql.Named("user_id", userID),
			sql.Named("role_code", string(role)),
		); err != nil {
			return err
		}
	}
	return nil
}
