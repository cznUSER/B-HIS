package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

var _ port.AccessStore = (*Store)(nil)
var _ port.AtomicAccessStore = (*Store)(nil)

func (s *Store) ListAccessModules(ctx context.Context) ([]domain.AccessModule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT module_id, module_code, module_name, description, status, hidden, department_functions, need_specify_department, second_permission_id, home_page_url
		FROM dbo.access_module
		ORDER BY module_code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.AccessModule, 0)
	for rows.Next() {
		var item domain.AccessModule
		var description, homepage sql.NullString
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &description, &item.Status, &item.Hidden, &item.DepartmentFunctions, &item.NeedSpecifyDepartment, &item.SecondPermissionID, &homepage); err != nil {
			return nil, err
		}
		item.Description = description.String
		item.HomePageURL = homepage.String
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) CreateAccessModule(ctx context.Context, item domain.AccessModule) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO dbo.access_module (module_id,module_code,module_name,description,status,hidden,department_functions,need_specify_department,second_permission_id,home_page_url) VALUES (@id,@code,@name,@description,@status,@hidden,@functions,@specify,@second,@homepage)`, sql.Named("id", item.ID), sql.Named("code", item.Code), sql.Named("name", item.Name), sql.Named("description", nullIfEmpty(item.Description)), sql.Named("status", item.Status), sql.Named("hidden", item.Hidden), sql.Named("functions", item.DepartmentFunctions), sql.Named("specify", item.NeedSpecifyDepartment), sql.Named("second", item.SecondPermissionID), sql.Named("homepage", nullIfEmpty(item.HomePageURL)))
	return classifyWriteError(err)
}

func (s *Store) UpdateAccessModule(ctx context.Context, item domain.AccessModule) error {
	res, err := s.db.ExecContext(ctx, `UPDATE dbo.access_module SET module_name=@name,description=@description,status=@status,hidden=@hidden,department_functions=@functions,need_specify_department=@specify,second_permission_id=@second,home_page_url=@homepage WHERE module_code=@code`, sql.Named("code", item.Code), sql.Named("name", item.Name), sql.Named("description", nullIfEmpty(item.Description)), sql.Named("status", item.Status), sql.Named("hidden", item.Hidden), sql.Named("functions", item.DepartmentFunctions), sql.Named("specify", item.NeedSpecifyDepartment), sql.Named("second", item.SecondPermissionID), sql.Named("homepage", nullIfEmpty(item.HomePageURL)))
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

func (s *Store) ListPermissionCategories(ctx context.Context) ([]domain.PermissionCategory, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT category_id, category_name, description, status
		FROM dbo.permission_category
		ORDER BY category_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.PermissionCategory, 0)
	for rows.Next() {
		var item domain.PermissionCategory
		var description sql.NullString
		if err := rows.Scan(&item.ID, &item.Name, &description, &item.Status); err != nil {
			return nil, err
		}
		item.Description = description.String
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) CreatePermissionCategory(ctx context.Context, item domain.PermissionCategory) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO dbo.permission_category(category_id,category_name,description,status) VALUES(@id,@name,@description,@status)`, sql.Named("id", item.ID), sql.Named("name", item.Name), sql.Named("description", nullIfEmpty(item.Description)), sql.Named("status", item.Status))
	return classifyWriteError(err)
}
func (s *Store) UpdatePermissionCategory(ctx context.Context, item domain.PermissionCategory) error {
	res, err := s.db.ExecContext(ctx, `UPDATE dbo.permission_category SET category_name=@name,description=@description,status=@status WHERE category_id=@id`, sql.Named("id", item.ID), sql.Named("name", item.Name), sql.Named("description", nullIfEmpty(item.Description)), sql.Named("status", item.Status))
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

func (s *Store) ListPermissionModes(ctx context.Context) ([]domain.PermissionMode, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT mode_code, mode_name, status
		FROM dbo.permission_mode
		ORDER BY mode_code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.PermissionMode, 0)
	for rows.Next() {
		var item domain.PermissionMode
		if err := rows.Scan(&item.Code, &item.Name, &item.Status); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) CreatePermissionMode(ctx context.Context, item domain.PermissionMode) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO dbo.permission_mode(mode_code,mode_name,status) VALUES(@code,@name,@status)`, sql.Named("code", item.Code), sql.Named("name", item.Name), sql.Named("status", item.Status))
	return classifyWriteError(err)
}
func (s *Store) UpdatePermissionMode(ctx context.Context, item domain.PermissionMode) error {
	res, err := s.db.ExecContext(ctx, `UPDATE dbo.permission_mode SET mode_name=@name,status=@status WHERE mode_code=@code`, sql.Named("code", item.Code), sql.Named("name", item.Name), sql.Named("status", item.Status))
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

func (s *Store) ListAccessPermissions(ctx context.Context, moduleCode, categoryID string) ([]domain.AccessPermission, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT permission_code, permission_name, description, module_code, category_id,
			mode_code, second_permission_code, actions, status, hidden, db_role_name, need_second_permission
		FROM dbo.access_permission
		WHERE (@module_code IS NULL OR module_code = @module_code)
		  AND (@category_id IS NULL OR category_id = @category_id)
		ORDER BY permission_code`,
		sql.Named("module_code", nullIfEmpty(strings.TrimSpace(moduleCode))),
		sql.Named("category_id", nullIfEmpty(strings.TrimSpace(categoryID))),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.AccessPermission, 0)
	for rows.Next() {
		item, err := scanAccessPermission(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (s *Store) CreateAccessPermission(ctx context.Context, permission domain.AccessPermission) error {
	return insertAccessPermission(ctx, s.db, permission)
}

func (s *Store) CreateAccessPermissionWithAudit(ctx context.Context, permission domain.AccessPermission, event domain.AuditEvent) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := insertAccessPermission(ctx, tx, permission); err != nil {
		return err
	}
	if err := appendAudit(ctx, tx, event); err != nil {
		return fmt.Errorf("%w: %v", port.ErrAuditUnavailable, err)
	}
	return tx.Commit()
}

type accessExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func insertAccessPermission(ctx context.Context, q accessExecer, permission domain.AccessPermission) error {
	_, err := q.ExecContext(ctx, `
		INSERT INTO dbo.access_permission (
			permission_code, permission_name, description, module_code, category_id,
			mode_code, second_permission_code, actions, status, hidden
			, db_role_name, need_second_permission
		) VALUES (
			@permission_code, @permission_name, @description, @module_code, @category_id,
			@mode_code, @second_permission_code, @actions, @status, @hidden, @db_role_name, @need_second_permission
		)`,
		sql.Named("permission_code", permission.Code),
		sql.Named("permission_name", permission.Name),
		sql.Named("description", nullIfEmpty(permission.Description)),
		sql.Named("module_code", permission.ModuleCode),
		sql.Named("category_id", nullIfEmpty(permission.CategoryID)),
		sql.Named("mode_code", nullIfEmpty(permission.ModeCode)),
		sql.Named("second_permission_code", nullIfEmpty(permission.SecondPermissionCode)),
		sql.Named("actions", strings.Join(permission.Actions, ",")),
		sql.Named("status", permission.Status),
		sql.Named("hidden", permission.Hidden),
		sql.Named("db_role_name", nullIfEmpty(permission.DBRoleName)), sql.Named("need_second_permission", permission.NeedSecondPermission),
	)
	return classifyWriteError(err)
}

func (s *Store) UpdateAccessPermission(ctx context.Context, permission domain.AccessPermission) error {
	return updateAccessPermission(ctx, s.db, permission)
}

func (s *Store) UpdateAccessPermissionWithAudit(ctx context.Context, permission domain.AccessPermission, event domain.AuditEvent) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := updateAccessPermission(ctx, tx, permission); err != nil {
		return err
	}
	if err := appendAudit(ctx, tx, event); err != nil {
		return fmt.Errorf("%w: %v", port.ErrAuditUnavailable, err)
	}
	return tx.Commit()
}

func updateAccessPermission(ctx context.Context, q accessExecer, permission domain.AccessPermission) error {
	res, err := q.ExecContext(ctx, `
		UPDATE dbo.access_permission
		SET permission_name = @permission_name, description = @description, module_code = @module_code,
			category_id = @category_id, mode_code = @mode_code, second_permission_code = @second_permission_code,
			actions = @actions, status = @status, hidden = @hidden, db_role_name=@db_role_name, need_second_permission=@need_second_permission
		WHERE permission_code = @permission_code`,
		sql.Named("permission_code", permission.Code),
		sql.Named("permission_name", permission.Name),
		sql.Named("description", nullIfEmpty(permission.Description)),
		sql.Named("module_code", permission.ModuleCode),
		sql.Named("category_id", nullIfEmpty(permission.CategoryID)),
		sql.Named("mode_code", nullIfEmpty(permission.ModeCode)),
		sql.Named("second_permission_code", nullIfEmpty(permission.SecondPermissionCode)),
		sql.Named("actions", strings.Join(permission.Actions, ",")),
		sql.Named("status", permission.Status),
		sql.Named("hidden", permission.Hidden),
		sql.Named("db_role_name", nullIfEmpty(permission.DBRoleName)), sql.Named("need_second_permission", permission.NeedSecondPermission),
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

func (s *Store) ListAccessRoles(ctx context.Context) ([]domain.AccessRole, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT role_code, role_name, description, module_code, department_id, status, hidden, is_system, order_sort, stand_name
		FROM dbo.access_role
		ORDER BY role_code`)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AccessRole, 0)
	for rows.Next() {
		var item domain.AccessRole
		var description, moduleCode, departmentID, standName sql.NullString
		if err := rows.Scan(&item.Code, &item.Name, &description, &moduleCode, &departmentID, &item.Status, &item.Hidden, &item.System, &item.OrderSort, &standName); err != nil {
			_ = rows.Close()
			return nil, err
		}
		item.Description = description.String
		item.ModuleCode = moduleCode.String
		item.DepartmentID = departmentID.String
		item.StandName = standName.String
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range out {
		permissions, err := s.listRolePermissionCodes(ctx, out[i].Code)
		if err != nil {
			return nil, err
		}
		out[i].PermissionCodes = permissions
	}
	return out, nil
}

func (s *Store) CreateAccessRole(ctx context.Context, role domain.AccessRole) error {
	role = normalizeAccessRole(role)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := insertAccessRole(ctx, tx, role); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *Store) CreateAccessRoleWithAudit(ctx context.Context, role domain.AccessRole, event domain.AuditEvent) error {
	role = normalizeAccessRole(role)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := insertAccessRole(ctx, tx, role); err != nil {
		return err
	}
	if err := appendAudit(ctx, tx, event); err != nil {
		return fmt.Errorf("%w: %v", port.ErrAuditUnavailable, err)
	}
	return tx.Commit()
}

func insertAccessRole(ctx context.Context, tx roleQuery, role domain.AccessRole) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO dbo.access_role (
			role_code, role_name, description, module_code, department_id, status, hidden, is_system, order_sort, stand_name
		) VALUES (
			@role_code, @role_name, @description, @module_code, @department_id, @status, @hidden, @is_system, @order_sort, @stand_name
		)`,
		sql.Named("role_code", role.Code),
		sql.Named("role_name", role.Name),
		sql.Named("description", nullIfEmpty(role.Description)),
		sql.Named("module_code", nullIfEmpty(role.ModuleCode)),
		sql.Named("department_id", nullIfEmpty(role.DepartmentID)),
		sql.Named("status", role.Status),
		sql.Named("hidden", role.Hidden),
		sql.Named("is_system", role.System),
		sql.Named("order_sort", role.OrderSort), sql.Named("stand_name", nullIfEmpty(role.StandName)),
	); err != nil {
		return classifyWriteError(err)
	}
	if _, err := tx.ExecContext(ctx, `
		IF EXISTS (SELECT 1 FROM dbo.role WHERE role_code = @role_code)
			UPDATE dbo.role SET role_name = @role_name WHERE role_code = @role_code;
		ELSE
			INSERT INTO dbo.role (role_code, role_name) VALUES (@role_code, @role_name);`,
		sql.Named("role_code", role.Code), sql.Named("role_name", role.Name)); err != nil {
		return classifyWriteError(err)
	}
	if err := insertAccessRolePermissions(ctx, tx, role.Code, role.PermissionCodes); err != nil {
		return classifyWriteError(err)
	}
	return nil
}

func (s *Store) UpdateAccessRole(ctx context.Context, role domain.AccessRole) error {
	role = normalizeAccessRole(role)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := updateAccessRole(ctx, tx, role); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *Store) UpdateAccessRoleWithAudit(ctx context.Context, role domain.AccessRole, event domain.AuditEvent) error {
	role = normalizeAccessRole(role)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := updateAccessRole(ctx, tx, role); err != nil {
		return err
	}
	if err := appendAudit(ctx, tx, event); err != nil {
		return fmt.Errorf("%w: %v", port.ErrAuditUnavailable, err)
	}
	return tx.Commit()
}

func updateAccessRole(ctx context.Context, tx roleQuery, role domain.AccessRole) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE dbo.access_role
		SET role_name = @role_name, description = @description, module_code = @module_code,
			department_id = @department_id, status = @status, hidden = @hidden, is_system = @is_system, order_sort=@order_sort, stand_name=@stand_name
		WHERE role_code = @role_code`,
		sql.Named("role_code", role.Code),
		sql.Named("role_name", role.Name),
		sql.Named("description", nullIfEmpty(role.Description)),
		sql.Named("module_code", nullIfEmpty(role.ModuleCode)),
		sql.Named("department_id", nullIfEmpty(role.DepartmentID)),
		sql.Named("status", role.Status),
		sql.Named("hidden", role.Hidden),
		sql.Named("is_system", role.System),
		sql.Named("order_sort", role.OrderSort), sql.Named("stand_name", nullIfEmpty(role.StandName)),
	)
	if err != nil {
		return classifyWriteError(err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return port.ErrNotFound
	}
	if _, err := tx.ExecContext(ctx, `
		IF EXISTS (SELECT 1 FROM dbo.role WHERE role_code = @role_code)
			UPDATE dbo.role SET role_name = @role_name WHERE role_code = @role_code;
		ELSE
			INSERT INTO dbo.role (role_code, role_name) VALUES (@role_code, @role_name);`,
		sql.Named("role_code", role.Code), sql.Named("role_name", role.Name)); err != nil {
		return classifyWriteError(err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM dbo.access_role_permission WHERE role_code = @role_code`, sql.Named("role_code", role.Code)); err != nil {
		return err
	}
	if err := insertAccessRolePermissions(ctx, tx, role.Code, role.PermissionCodes); err != nil {
		return classifyWriteError(err)
	}
	return nil
}

func (s *Store) ListUserPermissionOverrides(ctx context.Context, userID string) ([]domain.PermissionOverride, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id, permission_code, module_code, department_id, effect
		FROM dbo.user_permission_override
		WHERE user_id = @user_id
		ORDER BY permission_code, module_code, department_id`, sql.Named("user_id", userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.PermissionOverride, 0)
	for rows.Next() {
		var item domain.PermissionOverride
		var moduleCode, departmentID sql.NullString
		if err := rows.Scan(&item.UserID, &item.PermissionCode, &moduleCode, &departmentID, &item.Effect); err != nil {
			return nil, err
		}
		item.ModuleCode = moduleCode.String
		item.DepartmentID = departmentID.String
		item.Effect = strings.ToUpper(strings.TrimSpace(item.Effect))
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) ReplaceUserPermissionOverrides(ctx context.Context, userID string, overrides []domain.PermissionOverride) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := replaceUserPermissionOverridesTx(ctx, tx, userID, overrides); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ReplaceUserPermissionOverridesWithAudit(ctx context.Context, userID string, overrides []domain.PermissionOverride, event domain.AuditEvent) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := replaceUserPermissionOverridesTx(ctx, tx, userID, overrides); err != nil {
		return err
	}
	if err := appendAudit(ctx, tx, event); err != nil {
		return fmt.Errorf("%w: %v", port.ErrAuditUnavailable, err)
	}
	return tx.Commit()
}

func replaceUserPermissionOverridesTx(ctx context.Context, tx accessExecer, userID string, overrides []domain.PermissionOverride) error {
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("user id is required")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM dbo.user_permission_override WHERE user_id = @user_id`, sql.Named("user_id", userID)); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(overrides))
	for _, item := range overrides {
		item.UserID = userID
		item.PermissionCode = strings.TrimSpace(item.PermissionCode)
		item.ModuleCode = strings.TrimSpace(item.ModuleCode)
		item.DepartmentID = strings.TrimSpace(item.DepartmentID)
		item.Effect = strings.ToUpper(strings.TrimSpace(item.Effect))
		if item.PermissionCode == "" {
			return fmt.Errorf("permission code is required")
		}
		if item.Effect != domain.PermissionAllow && item.Effect != domain.PermissionDeny {
			return fmt.Errorf("invalid permission override effect %q", item.Effect)
		}
		key := strings.Join([]string{item.PermissionCode, item.ModuleCode, item.DepartmentID}, "\x00")
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate permission override")
		}
		seen[key] = struct{}{}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO dbo.user_permission_override (
				user_id, permission_code, module_code, department_id, effect
			) VALUES (
				@user_id, @permission_code, @module_code, @department_id, @effect
			)`,
			sql.Named("user_id", item.UserID),
			sql.Named("permission_code", item.PermissionCode),
			sql.Named("module_code", item.ModuleCode),
			sql.Named("department_id", item.DepartmentID),
			sql.Named("effect", item.Effect),
		); err != nil {
			return classifyWriteError(err)
		}
	}
	return nil
}

func (s *Store) CountSystemAdmins(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM dbo.app_user AS u
		INNER JOIN dbo.user_role AS ur ON ur.user_id = u.user_id
		WHERE u.is_deleted = 0 AND u.status = @status AND ur.role_code = @role_code`,
		sql.Named("status", domain.StatusActive),
		sql.Named("role_code", string(domain.RoleSystem)),
	).Scan(&count)
	return count, err
}

func (s *Store) listRolePermissionCodes(ctx context.Context, roleCode string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT permission_code
		FROM dbo.access_role_permission
		WHERE role_code = @role_code
		ORDER BY permission_code`, sql.Named("role_code", roleCode))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		out = append(out, code)
	}
	return out, rows.Err()
}

func insertAccessRolePermissions(ctx context.Context, q roleQuery, roleCode string, permissions []string) error {
	seen := make(map[string]struct{}, len(permissions))
	for _, code := range permissions {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		if _, err := q.ExecContext(ctx, `
			INSERT INTO dbo.access_role_permission (role_code, permission_code)
			VALUES (@role_code, @permission_code)`,
			sql.Named("role_code", roleCode),
			sql.Named("permission_code", code),
		); err != nil {
			return err
		}
	}
	return nil
}

func scanAccessPermission(sc scanner) (*domain.AccessPermission, error) {
	var item domain.AccessPermission
	var description, categoryID, modeCode, secondCode, actions, dbRoleName sql.NullString
	if err := sc.Scan(
		&item.Code, &item.Name, &description, &item.ModuleCode, &categoryID,
		&modeCode, &secondCode, &actions, &item.Status, &item.Hidden, &dbRoleName, &item.NeedSecondPermission,
	); err != nil {
		return nil, err
	}
	item.Description = description.String
	item.CategoryID = categoryID.String
	item.ModeCode = modeCode.String
	item.SecondPermissionCode = secondCode.String
	item.DBRoleName = dbRoleName.String
	item.Actions = splitAccessActions(actions.String)
	return &item, nil
}

func splitAccessActions(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func normalizeAccessRole(role domain.AccessRole) domain.AccessRole {
	role.Code = strings.TrimSpace(role.Code)
	role.Name = strings.TrimSpace(role.Name)
	role.Description = strings.TrimSpace(role.Description)
	role.ModuleCode = strings.TrimSpace(role.ModuleCode)
	role.DepartmentID = strings.TrimSpace(role.DepartmentID)
	role.Status = strings.ToUpper(strings.TrimSpace(role.Status))
	if role.Status == "" {
		role.Status = domain.StatusActive
	}
	return role
}
