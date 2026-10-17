package mssql

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

func (s *Store) ListUserRoleScopes(ctx context.Context, userID string) ([]domain.UserRoleScope, error) {
	return loadUserRoleScopes(ctx, s.db, userID)
}

func loadUserRoleScopes(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, userID string) ([]domain.UserRoleScope, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT user_id, role_code, department_id, status
		FROM dbo.user_role_scope
		WHERE user_id = @user_id
		ORDER BY role_code, department_id`, sql.Named("user_id", userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.UserRoleScope, 0)
	for rows.Next() {
		var scope domain.UserRoleScope
		if err := rows.Scan(&scope.UserID, &scope.RoleCode, &scope.DepartmentID, &scope.Status); err != nil {
			return nil, err
		}
		out = append(out, scope)
	}
	return out, rows.Err()
}

func (s *Store) ReplaceUserRoleScopes(ctx context.Context, userID string, scopes []domain.UserRoleScope) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var userCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM dbo.app_user WHERE user_id = @user_id AND is_deleted = 0`, sql.Named("user_id", userID)).Scan(&userCount); err != nil {
		return err
	}
	if userCount == 0 {
		return port.ErrNotFound
	}
	if err := replaceUserRoleScopesTx(ctx, tx, userID, scopes); err != nil {
		return err
	}
	return tx.Commit()
}

func replaceUserRoleScopesTx(ctx context.Context, tx *sql.Tx, userID string, scopes []domain.UserRoleScope) error {
	normalized := make([]domain.UserRoleScope, 0, len(scopes))
	seen := make(map[string]bool, len(scopes))
	for _, scope := range scopes {
		scope.UserID = userID
		scope.RoleCode = strings.ToUpper(strings.TrimSpace(scope.RoleCode))
		scope.DepartmentID = strings.TrimSpace(scope.DepartmentID)
		scope.Status = strings.ToUpper(strings.TrimSpace(scope.Status))
		key := scope.RoleCode + "\x00" + scope.DepartmentID
		if scope.RoleCode == "" || scope.DepartmentID == "" || (scope.Status != domain.StatusActive && scope.Status != domain.StatusDisabled) || seen[key] {
			return port.ErrInvalidUserRoleScope
		}
		seen[key] = true
		var roleCount int
		if err := tx.QueryRowContext(ctx, `
			SELECT COUNT(1)
			FROM dbo.user_role ur
			INNER JOIN dbo.access_role ar ON ar.role_code = ur.role_code
			WHERE ur.user_id = @user_id AND ur.role_code = @role_code AND ar.status = 'ACTIVE'`,
			sql.Named("user_id", userID), sql.Named("role_code", scope.RoleCode)).Scan(&roleCount); err != nil {
			return err
		}
		var departmentStatus sql.NullString
		if err := tx.QueryRowContext(ctx, `
			SELECT status FROM dbo.department
			WHERE department_id = @department_id AND is_deleted = 0`,
			sql.Named("department_id", scope.DepartmentID)).Scan(&departmentStatus); err != nil {
			if err == sql.ErrNoRows {
				return port.ErrInvalidUserRoleScope
			}
			return err
		}
		if roleCount == 0 || !departmentStatus.Valid || scope.Status == domain.StatusActive && departmentStatus.String != domain.StatusActive {
			return port.ErrInvalidUserRoleScope
		}
		normalized = append(normalized, scope)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM dbo.user_role_scope WHERE user_id = @user_id`, sql.Named("user_id", userID)); err != nil {
		return err
	}
	for _, scope := range normalized {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO dbo.user_role_scope (user_id, role_code, department_id, status)
			VALUES (@user_id, @role_code, @department_id, @status)`,
			sql.Named("user_id", userID), sql.Named("role_code", scope.RoleCode),
			sql.Named("department_id", scope.DepartmentID), sql.Named("status", scope.Status)); err != nil {
			return classifyWriteError(err)
		}
	}
	return nil
}

func retainAssignedRoleScopesTx(ctx context.Context, tx *sql.Tx, userID string, roles []domain.Role) error {
	if len(roles) == 0 {
		if _, err := tx.ExecContext(ctx, `DELETE FROM dbo.user_role_scope WHERE user_id = @user_id`, sql.Named("user_id", userID)); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM dbo.user_role WHERE user_id = @user_id`, sql.Named("user_id", userID)); err != nil {
			return err
		}
		return nil
	}
	args := []any{sql.Named("user_id", userID)}
	placeholders := make([]string, 0, len(roles))
	for i, role := range roles {
		name := "role_" + strconv.Itoa(i)
		placeholders = append(placeholders, "@"+name)
		args = append(args, sql.Named(name, string(role)))
	}
	keep := strings.Join(placeholders, ",")
	if _, err := tx.ExecContext(ctx, `DELETE FROM dbo.user_role_scope WHERE user_id = @user_id AND role_code NOT IN (`+keep+")", args...); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM dbo.user_role WHERE user_id = @user_id AND role_code NOT IN (`+keep+")", args...); err != nil {
		return err
	}
	return insertMissingRoles(ctx, tx, userID, roles)
}

func insertMissingRoles(ctx context.Context, tx *sql.Tx, userID string, roles []domain.Role) error {
	for _, role := range roles {
		if _, err := tx.ExecContext(ctx, `
			IF NOT EXISTS (SELECT 1 FROM dbo.user_role WHERE user_id = @user_id AND role_code = @role_code)
			INSERT INTO dbo.user_role (user_id, role_code) VALUES (@user_id, @role_code)`,
			sql.Named("user_id", userID), sql.Named("role_code", string(role))); err != nil {
			return classifyWriteError(err)
		}
	}
	return nil
}
