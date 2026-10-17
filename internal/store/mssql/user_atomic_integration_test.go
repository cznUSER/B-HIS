package mssql

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/google/uuid"
)

func TestUpdateUserWithAuditSerializesLastSystemAdminDemotion(t *testing.T) {
	dsn := os.Getenv("BHIS_DASHBOARD_TEST_DSN")
	if dsn == "" {
		t.Skip("set BHIS_DASHBOARD_TEST_DSN to the isolated BHIS_PHASE2_TEST_20261015 database")
	}
	db, err := Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	var databaseName string
	if err := db.QueryRowContext(ctx, `SELECT DB_NAME()`).Scan(&databaseName); err != nil {
		t.Fatal(err)
	}
	if databaseName != "BHIS_PHASE2_TEST_20261015" {
		t.Fatalf("refusing to write user atomic fixture to database %q", databaseName)
	}

	suffix := uuid.NewString()[:20]
	user1 := "atomic-u1-" + suffix
	user2 := "atomic-u2-" + suffix
	name1 := "atomic_n1_" + suffix
	name2 := "atomic_n2_" + suffix
	request1 := "atomic-demote-1-" + suffix
	request2 := "atomic-demote-2-" + suffix
	cleanup := func() {
		for _, statement := range []struct {
			query string
			args  []any
		}{
			{`DELETE FROM dbo.audit_log WHERE request_id IN (@request1,@request2)`, []any{sql.Named("request1", request1), sql.Named("request2", request2)}},
			{`DELETE FROM dbo.user_role WHERE user_id IN (@user1,@user2)`, []any{sql.Named("user1", user1), sql.Named("user2", user2)}},
			{`DELETE FROM dbo.app_user WHERE user_id IN (@user1,@user2)`, []any{sql.Named("user1", user1), sql.Named("user2", user2)}},
		} {
			if _, err := db.ExecContext(ctx, statement.query, statement.args...); err != nil {
				t.Errorf("cleanup atomic user fixture: %v", err)
			}
		}
	}
	cleanup()
	existingSystemStatuses := map[string]string{}
	activeRows, err := db.QueryContext(ctx, `
		SELECT DISTINCT u.user_id, u.status
		FROM dbo.app_user u INNER JOIN dbo.user_role ur ON ur.user_id = u.user_id
		WHERE u.is_deleted = 0 AND u.status = @status AND ur.role_code = @role_code`,
		sql.Named("status", domain.StatusActive), sql.Named("role_code", string(domain.RoleSystem)))
	if err != nil {
		t.Fatal(err)
	}
	for activeRows.Next() {
		var id, status string
		if err := activeRows.Scan(&id, &status); err != nil {
			activeRows.Close()
			t.Fatal(err)
		}
		existingSystemStatuses[id] = status
	}
	if err := activeRows.Err(); err != nil {
		activeRows.Close()
		t.Fatal(err)
	}
	activeRows.Close()
	for id := range existingSystemStatuses {
		if _, err := db.ExecContext(ctx, `UPDATE dbo.app_user SET status=@status WHERE user_id=@user_id`,
			sql.Named("status", domain.StatusDisabled), sql.Named("user_id", id)); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		for id, status := range existingSystemStatuses {
			if _, err := db.ExecContext(ctx, `UPDATE dbo.app_user SET status=@status WHERE user_id=@user_id`,
				sql.Named("status", status), sql.Named("user_id", id)); err != nil {
				t.Errorf("restore existing system user %s: %v", id, err)
			}
		}
	}()
	defer cleanup()
	for _, row := range []struct {
		id, username string
	}{
		{user1, name1}, {user2, name2},
	} {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO dbo.app_user(user_id,username,display_name,password_hash,status,created_at,updated_at)
			VALUES(@user_id,@username,N'Atomic User','not-a-login-hash','ACTIVE',SYSUTCDATETIME(),SYSUTCDATETIME())`,
			sql.Named("user_id", row.id), sql.Named("username", row.username)); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO dbo.user_role(user_id,role_code) VALUES(@user_id,@role_code)`,
			sql.Named("user_id", row.id), sql.Named("role_code", string(domain.RoleSystem))); err != nil {
			t.Fatal(err)
		}
	}

	store := New(db)
	rows := make([]*domain.User, 2)
	for i, id := range []string{user1, user2} {
		rows[i], err = store.FindByID(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		rows[i].Status = domain.StatusDisabled
		rows[i].UpdatedAt = time.Now().UTC()
		rows[i].UpdatedBy = user1
	}
	events := []domain.AuditEvent{
		{ID: uuid.NewString(), OccurredAt: time.Now().UTC(), ActorID: user1, ActorRoles: string(domain.RoleSystem), Action: domain.ActionUserDeactivate, ResourceType: "User", ResourceID: user1, Outcome: domain.OutcomeSuccess, RequestID: request1},
		{ID: uuid.NewString(), OccurredAt: time.Now().UTC(), ActorID: user1, ActorRoles: string(domain.RoleSystem), Action: domain.ActionUserDeactivate, ResourceType: "User", ResourceID: user2, Outcome: domain.OutcomeSuccess, RequestID: request2},
	}
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range rows {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, errs[index] = store.UpdateUserWithAudit(ctx, *rows[index], false, events[index])
		}(i)
	}
	wg.Wait()
	successes := 0
	lastAdmin := 0
	for _, callErr := range errs {
		if callErr == nil {
			successes++
		} else if errors.Is(callErr, port.ErrLastSystemAdmin) {
			lastAdmin++
		} else {
			t.Fatalf("unexpected atomic demotion error: %v", callErr)
		}
	}
	if successes != 1 || lastAdmin != 1 {
		t.Fatalf("atomic demotion results = successes %d last-admin %d, want 1/1", successes, lastAdmin)
	}
	count, err := store.CountSystemAdmins(ctx)
	if err != nil || count != 1 {
		t.Fatalf("active SYSTEM administrators = %d, err=%v, want 1", count, err)
	}
	var auditCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM dbo.audit_log WHERE request_id IN (@request1,@request2)`, sql.Named("request1", request1), sql.Named("request2", request2)).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 {
		t.Fatalf("successful demotion audits = %d, want 1", auditCount)
	}
}

func TestResetPasswordWithAuditRollsBackWithAuditFailure(t *testing.T) {
	dsn := os.Getenv("BHIS_DASHBOARD_TEST_DSN")
	if dsn == "" {
		t.Skip("set BHIS_DASHBOARD_TEST_DSN to the isolated BHIS_PHASE2_TEST_20261015 database")
	}
	db, err := Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	var databaseName string
	if err := db.QueryRowContext(ctx, `SELECT DB_NAME()`).Scan(&databaseName); err != nil {
		t.Fatal(err)
	}
	if databaseName != "BHIS_PHASE2_TEST_20261015" {
		t.Fatalf("refusing to write password reset fixture to database %q", databaseName)
	}
	suffix := uuid.NewString()[:20]
	userID := "atomic-rp-" + suffix
	username := "atomic_rp_" + suffix
	requestOK := "atomic-reset-ok-" + suffix
	requestFail := "atomic-reset-fail-" + suffix
	cleanup := func() {
		for _, statement := range []struct {
			query string
			args  []any
		}{
			{`DELETE FROM dbo.audit_log WHERE request_id IN (@ok,@fail)`, []any{sql.Named("ok", requestOK), sql.Named("fail", requestFail)}},
			{`DELETE FROM dbo.app_user WHERE user_id = @user_id`, []any{sql.Named("user_id", userID)}},
		} {
			if _, err := db.ExecContext(ctx, statement.query, statement.args...); err != nil {
				t.Errorf("cleanup reset fixture: %v", err)
			}
		}
	}
	cleanup()
	defer cleanup()
	if _, err := db.ExecContext(ctx, `
		INSERT INTO dbo.app_user(user_id,username,display_name,password_hash,status,auth_version,created_at,updated_at)
		VALUES(@user_id,@username,N'Atomic Reset','old-hash','ACTIVE',0,SYSUTCDATETIME(),SYSUTCDATETIME())`,
		sql.Named("user_id", userID), sql.Named("username", username)); err != nil {
		t.Fatal(err)
	}
	store := New(db)
	when := time.Now().UTC()
	if err := store.ResetPasswordWithAudit(ctx, userID, "new-hash", userID, domain.AuditEvent{
		ID: uuid.NewString(), OccurredAt: when, ActorID: userID, Action: domain.ActionUserResetPassword,
		ResourceType: "User", ResourceID: userID, Outcome: domain.OutcomeSuccess, RequestID: requestOK,
	}); err != nil {
		t.Fatal(err)
	}
	var hash string
	var mustChange bool
	var authVersion int
	if err := db.QueryRowContext(ctx, `SELECT password_hash, must_change_password, auth_version FROM dbo.app_user WHERE user_id=@user_id`, sql.Named("user_id", userID)).Scan(&hash, &mustChange, &authVersion); err != nil {
		t.Fatal(err)
	}
	if hash != "new-hash" || !mustChange || authVersion != 1 {
		t.Fatalf("successful reset state = hash %q must_change %t auth_version %d", hash, mustChange, authVersion)
	}
	var auditCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM dbo.audit_log WHERE request_id=@request_id`, sql.Named("request_id", requestOK)).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 {
		t.Fatalf("successful reset audit count = %d, want 1", auditCount)
	}
	err = store.ResetPasswordWithAudit(ctx, userID, "should-rollback", userID, domain.AuditEvent{
		ID: uuid.NewString(), OccurredAt: when.Add(time.Second), ActorID: userID, Action: strings.Repeat("x", 65),
		ResourceType: "User", ResourceID: userID, Outcome: domain.OutcomeSuccess, RequestID: requestFail,
	})
	if !errors.Is(err, port.ErrAuditUnavailable) {
		t.Fatalf("audit failure = %v, want ErrAuditUnavailable", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT password_hash, auth_version FROM dbo.app_user WHERE user_id=@user_id`, sql.Named("user_id", userID)).Scan(&hash, &authVersion); err != nil {
		t.Fatal(err)
	}
	if hash != "new-hash" || authVersion != 1 {
		t.Fatalf("failed reset changed user: hash %q auth_version %d", hash, authVersion)
	}
}
