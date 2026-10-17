package mssql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

var _ port.AtomicUserAccessBundleStore = (*Store)(nil)

// CreateUserAccessBundleWithAudit commits a new user, its roles, complete
// permission overrides, and the success audit in one SQL transaction.
func (s *Store) CreateUserAccessBundleWithAudit(ctx context.Context, u domain.User, replaceRoleScopes bool, roleScopes []domain.UserRoleScope, replaceOverrides bool, overrides []domain.PermissionOverride, event domain.AuditEvent) (*domain.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	args := []any{
		sql.Named("user_id", u.ID),
		sql.Named("username", u.Username),
		sql.Named("display_name", u.DisplayName),
		sql.Named("password_hash", u.PasswordHash),
		sql.Named("status", u.Status),
	}
	args = append(args, userLifecycleArgs(u)...)
	args = append(args,
		sql.Named("created_at", u.CreatedAt.UTC()),
		sql.Named("created_by", nullIfEmpty(u.CreatedBy)),
		sql.Named("updated_at", u.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(u.UpdatedBy)),
	)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO dbo.app_user (
			user_id, username, display_name, password_hash, status,
			description, card_no, input_code, service_star, expand_interface1, expand_interface2, input_code_compare_mode,
			must_change_password, cannot_modify_password, account_locked, account_stopped,
			hidden, is_system, is_super, is_developer, last_login_at, last_login_ip, lock_until, password_changed_at,
			failed_login_count, auth_version, created_at, created_by, updated_at, updated_by
		) VALUES (
			@user_id, @username, @display_name, @password_hash, @status,
			@description, @card_no, @input_code, @service_star, @expand_interface1, @expand_interface2, @input_code_compare_mode,
			@must_change_password, @cannot_modify_password, @account_locked, @account_stopped,
			@hidden, @is_system, @is_super, @is_developer, @last_login_at, @last_login_ip, @lock_until, @password_changed_at,
			@failed_login_count, @auth_version, @created_at, @created_by, @updated_at, @updated_by
		)`, args...); err != nil {
		return nil, classifyWriteError(err)
	}
	if err := insertRoles(ctx, tx, u.ID, u.Roles); err != nil {
		return nil, err
	}
	if replaceRoleScopes {
		if err := replaceUserRoleScopesTx(ctx, tx, u.ID, roleScopes); err != nil {
			return nil, err
		}
	}
	if replaceOverrides {
		if err := replaceUserPermissionOverridesTx(ctx, tx, u.ID, overrides); err != nil {
			return nil, err
		}
	}
	if err := appendAudit(ctx, tx, event); err != nil {
		return nil, fmt.Errorf("%w: %v", port.ErrAuditUnavailable, err)
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

// UpdateUserAccessBundleWithAudit commits profile/role changes, a complete
// override replacement, and the success audit in one SQL transaction. The
// same application lock as UpdateUserWithAudit serializes system-admin
// membership checks across concurrent requests.
func (s *Store) UpdateUserAccessBundleWithAudit(ctx context.Context, u domain.User, replaceRoles, replaceRoleScopes bool, roleScopes []domain.UserRoleScope, replaceOverrides bool, overrides []domain.PermissionOverride, event domain.AuditEvent) (*domain.User, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var lockResult int
	err = tx.QueryRowContext(ctx, `
		DECLARE @result int;
		EXEC @result = sp_getapplock
			@Resource = N'BHIS:user-system-admins',
			@LockMode = 'Exclusive',
			@LockOwner = 'Transaction',
			@LockTimeout = 5000;
		SELECT @result;`).Scan(&lockResult)
	if err != nil {
		return nil, err
	}
	if lockResult < 0 {
		return nil, port.ErrUserAdminBusy
	}

	var currentStatus string
	var currentSystem int
	err = tx.QueryRowContext(ctx, `
		SELECT u.status,
		       CASE WHEN EXISTS (SELECT 1 FROM dbo.user_role ur WHERE ur.user_id = u.user_id AND ur.role_code = @system_role) THEN 1 ELSE 0 END
		FROM dbo.app_user u WITH (UPDLOCK, HOLDLOCK)
		WHERE u.user_id = @user_id AND u.is_deleted = 0`,
		sql.Named("system_role", string(domain.RoleSystem)),
		sql.Named("user_id", u.ID),
	).Scan(&currentStatus, &currentSystem)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	removesSystemMembership := currentStatus == domain.StatusActive && currentSystem == 1 &&
		(u.Status != domain.StatusActive || !u.HasSystemRole())
	if removesSystemMembership {
		var count int
		if err := tx.QueryRowContext(ctx, `
			SELECT COUNT(1)
			FROM dbo.app_user AS u WITH (UPDLOCK, HOLDLOCK)
			INNER JOIN dbo.user_role AS ur WITH (UPDLOCK, HOLDLOCK) ON ur.user_id = u.user_id
			WHERE u.is_deleted = 0 AND u.status = @status AND ur.role_code = @role_code`,
			sql.Named("status", domain.StatusActive),
			sql.Named("role_code", string(domain.RoleSystem)),
		).Scan(&count); err != nil {
			return nil, err
		} else if count <= 1 {
			return nil, port.ErrLastSystemAdmin
		}
	}

	args := []any{sql.Named("display_name", u.DisplayName), sql.Named("status", u.Status), sql.Named("password_hash", u.PasswordHash)}
	args = append(args, userLifecycleArgs(u)...)
	args = append(args, sql.Named("updated_at", u.UpdatedAt.UTC()), sql.Named("updated_by", nullIfEmpty(u.UpdatedBy)), sql.Named("user_id", u.ID))
	res, err := tx.ExecContext(ctx, `
		UPDATE dbo.app_user
		SET display_name = @display_name, status = @status, password_hash = @password_hash, description = @description, card_no = @card_no,
			input_code = @input_code, service_star=@service_star, expand_interface1=@expand_interface1,
			expand_interface2=@expand_interface2, input_code_compare_mode=@input_code_compare_mode,
			must_change_password = @must_change_password,
			cannot_modify_password = @cannot_modify_password, account_locked = @account_locked,
			account_stopped = @account_stopped, hidden = @hidden, is_system = @is_system,
			is_super = @is_super, is_developer = @is_developer, last_login_at = @last_login_at,
			last_login_ip = @last_login_ip, lock_until = @lock_until, password_changed_at = @password_changed_at,
			failed_login_count = @failed_login_count, auth_version = @auth_version,
			updated_at = @updated_at, updated_by = @updated_by
		WHERE user_id = @user_id AND is_deleted = 0`, args...)
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
		if err := retainAssignedRoleScopesTx(ctx, tx, u.ID, u.Roles); err != nil {
			return nil, err
		}
	}
	if replaceRoleScopes {
		if err := replaceUserRoleScopesTx(ctx, tx, u.ID, roleScopes); err != nil {
			return nil, err
		}
	}
	if replaceOverrides {
		if err := replaceUserPermissionOverridesTx(ctx, tx, u.ID, overrides); err != nil {
			return nil, err
		}
	}
	if err := appendAudit(ctx, tx, event); err != nil {
		return nil, fmt.Errorf("%w: %v", port.ErrAuditUnavailable, err)
	}
	saved, err := scanAdminUser(tx.QueryRowContext(ctx, `
		SELECT `+userAdminColumns+`
		FROM dbo.app_user WHERE user_id = @user_id AND is_deleted = 0`, sql.Named("user_id", u.ID)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	roles, err := loadRoles(ctx, tx, u.ID)
	if err != nil {
		return nil, err
	}
	saved.Roles = roles
	saved.RoleScopes, err = loadUserRoleScopes(ctx, tx, u.ID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return saved, nil
}
