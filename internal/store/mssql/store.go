// Package mssql is the Phase1 SQL Server store. SQL stays in this package.
package mssql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/microsoft/go-mssqldb"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

// Store implements identity ports on SQL Server.
type Store struct {
	db *sql.DB
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

func New(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	return s.find(ctx, `SELECT user_id, username, display_name, password_hash, status
		FROM dbo.app_user WHERE username = @username AND is_deleted = 0`, sql.Named("username", username))
}

func (s *Store) FindByID(ctx context.Context, id string) (*domain.User, error) {
	return s.find(ctx, `SELECT user_id, username, display_name, password_hash, status
		FROM dbo.app_user WHERE user_id = @user_id AND is_deleted = 0`, sql.Named("user_id", id))
}

func (s *Store) find(ctx context.Context, q string, arg sql.NamedArg) (*domain.User, error) {
	var u domain.User
	err := s.db.QueryRowContext(ctx, q, arg).Scan(&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash, &u.Status)
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
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO dbo.audit_log (
			audit_id, occurred_at, actor_id, actor_roles, action,
			resource_type, resource_id, outcome, error_code, request_id, client_ip
		) VALUES (
			@audit_id, @occurred_at, @actor_id, @actor_roles, @action,
			@resource_type, @resource_id, @outcome, @error_code, @request_id, @client_ip
		)`,
		sql.Named("audit_id", e.ID),
		sql.Named("occurred_at", e.OccurredAt.UTC()),
		sql.Named("actor_id", nullIfEmpty(e.ActorID)),
		sql.Named("actor_roles", nullIfEmpty(e.ActorRoles)),
		sql.Named("action", e.Action),
		sql.Named("resource_type", nullIfEmpty(e.ResourceType)),
		sql.Named("resource_id", nullIfEmpty(e.ResourceID)),
		sql.Named("outcome", e.Outcome),
		sql.Named("error_code", nullIfEmpty(e.ErrorCode)),
		sql.Named("request_id", nullIfEmpty(e.RequestID)),
		sql.Named("client_ip", nullIfEmpty(e.ClientIP)),
	)
	return err
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
