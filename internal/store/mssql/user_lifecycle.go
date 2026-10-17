package mssql

import (
	"context"
	"database/sql"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

var _ port.UserStore = (*Store)(nil)

func (s *Store) RecordLoginFailure(ctx context.Context, userID string, at time.Time, threshold int, lockDuration time.Duration) (bool, error) {
	if threshold < 1 {
		threshold = 3
	}
	if lockDuration <= 0 {
		lockDuration = 15 * time.Minute
	}
	at = at.UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	var failed int
	var locked bool
	var lockUntil sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT failed_login_count, account_locked, lock_until
		FROM dbo.app_user WITH (UPDLOCK, ROWLOCK)
		WHERE user_id = @user_id AND is_deleted = 0`, sql.Named("user_id", userID)).
		Scan(&failed, &locked, &lockUntil)
	if err == sql.ErrNoRows {
		return false, port.ErrNotFound
	}
	if err != nil {
		return false, err
	}
	if locked && (!lockUntil.Valid || lockUntil.Time.After(at)) {
		return true, nil
	}
	if locked && !lockUntil.Time.After(at) {
		failed = 0
	}
	failed++
	nextLocked := failed >= threshold
	var nextUntil any
	if nextLocked {
		nextUntil = at.Add(lockDuration)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE dbo.app_user
		SET failed_login_count = @failed, account_locked = @locked, lock_until = @lock_until
		WHERE user_id = @user_id AND is_deleted = 0`,
		sql.Named("failed", failed), sql.Named("locked", nextLocked), sql.Named("lock_until", nextUntil), sql.Named("user_id", userID)); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return nextLocked, nil
}

func (s *Store) RecordSuccessfulLogin(ctx context.Context, userID, clientIP string, at time.Time) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE dbo.app_user
		SET last_login_at = @last_login_at, last_login_ip = @last_login_ip,
			failed_login_count = 0, account_locked = 0, lock_until = NULL
		WHERE user_id = @user_id AND is_deleted = 0`,
		sql.Named("last_login_at", at.UTC()), sql.Named("last_login_ip", nullIfEmpty(clientIP)), sql.Named("user_id", userID))
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

func (s *Store) ChangePassword(ctx context.Context, userID, passwordHash, operatorID string, at time.Time) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE dbo.app_user
		SET password_hash = @password_hash, must_change_password = 0, password_changed_at = @changed_at,
			auth_version = auth_version + 1, updated_at = @changed_at, updated_by = @updated_by
		WHERE user_id = @user_id AND is_deleted = 0`,
		sql.Named("password_hash", passwordHash), sql.Named("changed_at", at.UTC()),
		sql.Named("updated_by", nullIfEmpty(operatorID)), sql.Named("user_id", userID))
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

func currentTimeValue(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC()
}

func userLifecycleArgs(u domain.User) []any {
	return []any{
		sql.Named("description", nullIfEmpty(u.Description)),
		sql.Named("card_no", nullIfEmpty(u.CardNo)),
		sql.Named("input_code", nullIfEmpty(u.InputCode)),
		sql.Named("service_star", nullableInt(u.ServiceStar)),
		sql.Named("expand_interface1", nullableInt32(u.ExpandInterface1)),
		sql.Named("expand_interface2", nullableInt32(u.ExpandInterface2)),
		sql.Named("input_code_compare_mode", nullableInt32(u.InputCodeCompareMode)),
		sql.Named("must_change_password", u.MustChangePassword),
		sql.Named("cannot_modify_password", u.CannotModifyPassword),
		sql.Named("account_locked", u.AccountLocked),
		sql.Named("account_stopped", u.AccountStopped),
		sql.Named("hidden", u.Hidden),
		sql.Named("is_system", u.System),
		sql.Named("is_super", u.Super),
		sql.Named("is_developer", u.Developer),
		sql.Named("last_login_at", currentTimeValue(u.LastLoginAt)),
		sql.Named("last_login_ip", nullIfEmpty(u.LastLoginIP)),
		sql.Named("lock_until", currentTimeValue(u.LockUntil)),
		sql.Named("password_changed_at", currentTimeValue(u.PasswordChangedAt)),
		sql.Named("failed_login_count", u.FailedLoginCount),
		sql.Named("auth_version", u.AuthVersion),
	}
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}
func nullableInt32(value *int32) any {
	if value == nil {
		return nil
	}
	return *value
}
