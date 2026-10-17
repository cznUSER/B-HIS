// Package mssql is the Phase1 SQL Server store. SQL stays in this package.
package mssql

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	_ "github.com/microsoft/go-mssqldb"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

// Store implements identity ports on SQL Server.
// dispenseStub is only the pre-dispense refund fixture. On-hand and slips live in 006/007. 2026-10-12
type Store struct {
	db           *sql.DB
	mu           sync.Mutex
	dispenseStub map[string]dispenseRec
}

func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func New(db *sql.DB) *Store {
	return &Store{
		db:           db,
		dispenseStub: make(map[string]dispenseRec),
	}
}

func (s *Store) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	return s.find(ctx, `SELECT `+userColumns+`
		FROM dbo.app_user WHERE username = @username AND is_deleted = 0`, sql.Named("username", username))
}

func (s *Store) FindByID(ctx context.Context, id string) (*domain.User, error) {
	return s.find(ctx, `SELECT `+userColumns+`
		FROM dbo.app_user WHERE user_id = @user_id AND is_deleted = 0`, sql.Named("user_id", id))
}

func (s *Store) find(ctx context.Context, q string, arg sql.NamedArg) (*domain.User, error) {
	u, err := scanUser(s.db.QueryRowContext(ctx, q, arg))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT role_code FROM dbo.user_role WHERE user_id = @user_id ORDER BY role_code`, sql.Named("user_id", u.ID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		u.Roles = append(u.Roles, domain.Role(code))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if u.Roles == nil {
		u.Roles = []domain.Role{}
	}
	return u, nil
}

const userColumns = `user_id, username, display_name, description, card_no, input_code, password_hash, status,
	service_star, expand_interface1, expand_interface2, input_code_compare_mode,
	must_change_password, cannot_modify_password, account_locked, account_stopped, hidden, is_system, is_super, is_developer,
	last_login_at, last_login_ip, lock_until, password_changed_at, failed_login_count, auth_version, created_at, created_by, updated_at, updated_by`

const userAdminColumns = `user_id, username, display_name, description, card_no, input_code, status,
	service_star, expand_interface1, expand_interface2, input_code_compare_mode,
	must_change_password, cannot_modify_password, account_locked, account_stopped, hidden, is_system, is_super, is_developer,
	last_login_at, last_login_ip, lock_until, password_changed_at, failed_login_count, auth_version, created_at, created_by, updated_at, updated_by`

func scanUser(sc scanner) (*domain.User, error) {
	var u domain.User
	var description, cardNo, inputCode, lastLoginIP, createdBy, updatedBy sql.NullString
	var lastLoginAt, lockUntil, passwordChangedAt sql.NullTime
	var serviceStar, expand1, expand2, inputMode sql.NullInt64
	err := sc.Scan(
		&u.ID, &u.Username, &u.DisplayName, &description, &cardNo, &inputCode, &u.PasswordHash, &u.Status,
		&serviceStar, &expand1, &expand2, &inputMode,
		&u.MustChangePassword, &u.CannotModifyPassword, &u.AccountLocked, &u.AccountStopped, &u.Hidden, &u.System, &u.Super, &u.Developer,
		&lastLoginAt, &lastLoginIP, &lockUntil, &passwordChangedAt, &u.FailedLoginCount, &u.AuthVersion,
		&u.CreatedAt, &createdBy, &u.UpdatedAt, &updatedBy,
	)
	if err != nil {
		return nil, err
	}
	if serviceStar.Valid {
		value := int(serviceStar.Int64)
		u.ServiceStar = &value
	}
	if expand1.Valid {
		value := int32(expand1.Int64)
		u.ExpandInterface1 = &value
	}
	if expand2.Valid {
		value := int32(expand2.Int64)
		u.ExpandInterface2 = &value
	}
	if inputMode.Valid {
		value := int32(inputMode.Int64)
		u.InputCodeCompareMode = &value
	}
	u.Description = description.String
	u.CardNo = cardNo.String
	u.InputCode = inputCode.String
	u.LastLoginIP = lastLoginIP.String
	u.CreatedBy = createdBy.String
	u.UpdatedBy = updatedBy.String
	if lastLoginAt.Valid {
		u.LastLoginAt = lastLoginAt.Time.UTC()
	}
	if lockUntil.Valid {
		u.LockUntil = lockUntil.Time.UTC()
	}
	if passwordChangedAt.Valid {
		u.PasswordChangedAt = passwordChangedAt.Time.UTC()
	}
	return &u, nil
}

func scanAdminUser(sc scanner) (*domain.User, error) {
	var u domain.User
	var description, cardNo, inputCode, lastLoginIP, createdBy, updatedBy sql.NullString
	var lastLoginAt, lockUntil, passwordChangedAt sql.NullTime
	var serviceStar, expand1, expand2, inputMode sql.NullInt64
	err := sc.Scan(
		&u.ID, &u.Username, &u.DisplayName, &description, &cardNo, &inputCode, &u.Status,
		&serviceStar, &expand1, &expand2, &inputMode,
		&u.MustChangePassword, &u.CannotModifyPassword, &u.AccountLocked, &u.AccountStopped, &u.Hidden, &u.System, &u.Super, &u.Developer,
		&lastLoginAt, &lastLoginIP, &lockUntil, &passwordChangedAt, &u.FailedLoginCount, &u.AuthVersion,
		&u.CreatedAt, &createdBy, &u.UpdatedAt, &updatedBy,
	)
	if err != nil {
		return nil, err
	}
	if serviceStar.Valid {
		value := int(serviceStar.Int64)
		u.ServiceStar = &value
	}
	if expand1.Valid {
		value := int32(expand1.Int64)
		u.ExpandInterface1 = &value
	}
	if expand2.Valid {
		value := int32(expand2.Int64)
		u.ExpandInterface2 = &value
	}
	if inputMode.Valid {
		value := int32(inputMode.Int64)
		u.InputCodeCompareMode = &value
	}
	u.Description = description.String
	u.CardNo = cardNo.String
	u.InputCode = inputCode.String
	u.LastLoginIP = lastLoginIP.String
	u.CreatedBy = createdBy.String
	u.UpdatedBy = updatedBy.String
	if lastLoginAt.Valid {
		u.LastLoginAt = lastLoginAt.Time.UTC()
	}
	if lockUntil.Valid {
		u.LockUntil = lockUntil.Time.UTC()
	}
	if passwordChangedAt.Valid {
		u.PasswordChangedAt = passwordChangedAt.Time.UTC()
	}
	return &u, nil
}

func (s *Store) Revoke(ctx context.Context, jti, userID string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO dbo.token_revocation (jti, user_id, expires_at)
		SELECT @jti, @user_id, @expires_at
		WHERE NOT EXISTS (SELECT 1 FROM dbo.token_revocation WHERE jti = @jti)`,
		sql.Named("jti", jti),
		sql.Named("user_id", userID),
		sql.Named("expires_at", expiresAt.UTC()),
	)
	return err
}

func (s *Store) IsRevoked(ctx context.Context, jti string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM dbo.token_revocation
		WHERE jti = @jti AND expires_at > SYSUTCDATETIME()`,
		sql.Named("jti", jti),
	).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Store) Append(ctx context.Context, e domain.AuditEvent) error {
	return appendAudit(ctx, s.db, e)
}

type auditInserter interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func appendAudit(ctx context.Context, q auditInserter, e domain.AuditEvent) error {
	if e.ModuleID == "" {
		e.ModuleID = domain.AuditModuleID(e.Action)
	}
	if e.Description == "" {
		e.Description = domain.AuditDescription(e.Action)
	}
	_, err := q.ExecContext(ctx, `
		INSERT INTO dbo.audit_log (
			audit_id, occurred_at, actor_id, actor_username, actor_roles, module_id, department_id, action,
			description, resource_type, resource_id, outcome, error_code, request_id, client_ip, client_mac, build_version
		) VALUES (
			@audit_id, @occurred_at, @actor_id,
			COALESCE(@actor_username, (SELECT TOP 1 username FROM dbo.app_user WHERE user_id = @actor_id)),
			@actor_roles, @module_id, @department_id, @action,
			@description, @resource_type, @resource_id, @outcome, @error_code, @request_id, @client_ip, @client_mac, @build_version
		)`,
		sql.Named("audit_id", e.ID),
		sql.Named("occurred_at", e.OccurredAt.UTC()),
		sql.Named("actor_id", nullIfEmpty(e.ActorID)),
		sql.Named("actor_username", nullIfEmpty(e.ActorUsername)),
		sql.Named("actor_roles", nullIfEmpty(e.ActorRoles)),
		sql.Named("module_id", nullIfEmpty(e.ModuleID)),
		sql.Named("department_id", nullIfEmpty(e.DepartmentID)),
		sql.Named("action", e.Action),
		sql.Named("description", nullIfEmpty(e.Description)),
		sql.Named("resource_type", nullIfEmpty(e.ResourceType)),
		sql.Named("resource_id", nullIfEmpty(e.ResourceID)),
		sql.Named("outcome", e.Outcome),
		sql.Named("error_code", nullIfEmpty(e.ErrorCode)),
		sql.Named("request_id", nullIfEmpty(e.RequestID)),
		sql.Named("client_ip", nullIfEmpty(e.ClientIP)),
		sql.Named("client_mac", nullIfEmpty(e.ClientMAC)),
		sql.Named("build_version", nullIfEmpty(e.BuildVersion)),
	)
	return err
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
