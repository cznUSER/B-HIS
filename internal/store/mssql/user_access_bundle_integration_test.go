package mssql

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/cznUSER/B-HIS/internal/store/seed"
	"github.com/google/uuid"
)

func TestUserAccessBundleCreateUpdateAndAuditRollback(t *testing.T) {
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
		t.Fatalf("refusing to write user access bundle fixture to database %q", databaseName)
	}

	suffix := uuid.NewString()[:20]
	userID := "bundle-u-" + suffix
	username := "bundle_u_" + suffix
	failedID := "bundle-f-" + suffix
	failedName := "bundle_f_" + suffix
	requestPrefix := "bundle-" + suffix
	cleanup := func() {
		for _, statement := range []struct {
			query string
			args  []any
		}{
			{`DELETE FROM dbo.audit_log WHERE request_id LIKE @prefix`, []any{sql.Named("prefix", requestPrefix+"%")}},
			{`DELETE FROM dbo.user_role_scope WHERE user_id IN (@user_id,@failed_id)`, []any{sql.Named("user_id", userID), sql.Named("failed_id", failedID)}},
			{`DELETE FROM dbo.user_permission_override WHERE user_id IN (@user_id,@failed_id)`, []any{sql.Named("user_id", userID), sql.Named("failed_id", failedID)}},
			{`DELETE FROM dbo.user_role WHERE user_id IN (@user_id,@failed_id)`, []any{sql.Named("user_id", userID), sql.Named("failed_id", failedID)}},
			{`DELETE FROM dbo.app_user WHERE user_id IN (@user_id,@failed_id)`, []any{sql.Named("user_id", userID), sql.Named("failed_id", failedID)}},
		} {
			if _, err := db.ExecContext(ctx, statement.query, statement.args...); err != nil {
				t.Errorf("cleanup user access bundle fixture: %v", err)
			}
		}
	}
	cleanup()
	defer cleanup()

	store := New(db)
	now := time.Now().UTC()
	created, err := store.CreateUserAccessBundleWithAudit(ctx, domain.User{
		ID: userID, Username: username, DisplayName: "Bundle User", PasswordHash: "not-a-login-hash",
		Roles: []domain.Role{domain.RoleRegistrar}, Status: domain.StatusActive,
		CreatedAt: now, UpdatedAt: now,
	}, true, []domain.UserRoleScope{
		{UserID: userID, RoleCode: string(domain.RoleRegistrar), DepartmentID: "dept-1", Status: domain.StatusActive},
		{UserID: userID, RoleCode: string(domain.RoleRegistrar), DepartmentID: "dept-2", Status: domain.StatusDisabled},
	}, true, []domain.PermissionOverride{{PermissionCode: "charge:refund", Effect: domain.PermissionDeny}}, domain.AuditEvent{
		ID: uuid.NewString(), OccurredAt: now, ActorID: seed.DemoUserID, ActorRoles: string(domain.RoleSystem),
		Action: domain.ActionUserCreate, ResourceType: "User", ResourceID: userID, Outcome: domain.OutcomeSuccess,
		RequestID: requestPrefix + "create",
	})
	if err != nil {
		t.Fatalf("bundle create = %v", err)
	}
	if created == nil || created.ID != userID || created.PasswordHash != "" || len(created.Roles) != 1 || created.Roles[0] != domain.RoleRegistrar {
		t.Fatalf("bundle create result = %#v", created)
	}
	assertBundleAuditCount(t, ctx, db, requestPrefix+"create", 1)
	assertBundleOverride(t, ctx, store, userID, domain.PermissionDeny)
	assertBundleScopes(t, ctx, store, userID, []domain.UserRoleScope{
		{UserID: userID, RoleCode: string(domain.RoleRegistrar), DepartmentID: "dept-1", Status: domain.StatusActive},
		{UserID: userID, RoleCode: string(domain.RoleRegistrar), DepartmentID: "dept-2", Status: domain.StatusDisabled},
	})

	updatedInput, err := store.FindByID(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	updatedInput.DisplayName = "Bundle User Updated"
	updatedInput.Roles = []domain.Role{domain.RoleDoctor}
	updatedInput.UpdatedAt = now.Add(time.Minute)
	updated, err := store.UpdateUserAccessBundleWithAudit(ctx, *updatedInput, true, true, []domain.UserRoleScope{{
		UserID: userID, RoleCode: string(domain.RoleDoctor), DepartmentID: "dept-2", Status: domain.StatusActive,
	}}, true, []domain.PermissionOverride{{PermissionCode: "charge:refund", Effect: domain.PermissionAllow}}, domain.AuditEvent{
		ID: uuid.NewString(), OccurredAt: now.Add(time.Minute), ActorID: seed.DemoUserID, ActorRoles: string(domain.RoleSystem),
		Action: domain.ActionUserUpdate, ResourceType: "User", ResourceID: userID, Outcome: domain.OutcomeSuccess,
		RequestID: requestPrefix + "update",
	})
	if err != nil {
		t.Fatalf("bundle update = %v", err)
	}
	if updated.DisplayName != "Bundle User Updated" || len(updated.Roles) != 1 || updated.Roles[0] != domain.RoleDoctor {
		t.Fatalf("bundle update result = %#v", updated)
	}
	assertBundleAuditCount(t, ctx, db, requestPrefix+"update", 1)
	assertBundleOverride(t, ctx, store, userID, domain.PermissionAllow)
	assertBundleScopes(t, ctx, store, userID, []domain.UserRoleScope{{
		UserID: userID, RoleCode: string(domain.RoleDoctor), DepartmentID: "dept-2", Status: domain.StatusActive,
	}})

	// Scope-only update must preserve role assignments and overrides when those fields are omitted.
	updatedInput, err = store.FindByID(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateUserAccessBundleWithAudit(ctx, *updatedInput, false, true, []domain.UserRoleScope{{
		UserID: userID, RoleCode: string(domain.RoleDoctor), DepartmentID: "dept-1", Status: domain.StatusActive,
	}}, false, nil, domain.AuditEvent{
		ID: uuid.NewString(), OccurredAt: now.Add(90 * time.Second), ActorID: seed.DemoUserID, ActorRoles: string(domain.RoleSystem),
		Action: domain.ActionUserUpdate, ResourceType: "User", ResourceID: userID, Outcome: domain.OutcomeSuccess,
		RequestID: requestPrefix + "scope-only",
	}); err != nil {
		t.Fatalf("scope-only update = %v", err)
	}
	assertBundleOverride(t, ctx, store, userID, domain.PermissionAllow)
	assertBundleScopes(t, ctx, store, userID, []domain.UserRoleScope{{
		UserID: userID, RoleCode: string(domain.RoleDoctor), DepartmentID: "dept-1", Status: domain.StatusActive,
	}})

	failedCreateEvent := domain.AuditEvent{
		ID: uuid.NewString(), OccurredAt: now, ActorID: seed.DemoUserID, ActorRoles: string(domain.RoleSystem),
		Action: strings.Repeat("x", 65), ResourceType: "User", ResourceID: failedID, Outcome: domain.OutcomeSuccess,
		RequestID: requestPrefix + "create-fail",
	}
	if _, err := store.CreateUserAccessBundleWithAudit(ctx, domain.User{
		ID: failedID, Username: failedName, DisplayName: "Should Roll Back", PasswordHash: "hash",
		Roles: []domain.Role{domain.RoleRegistrar}, Status: domain.StatusActive, CreatedAt: now, UpdatedAt: now,
	}, true, []domain.UserRoleScope{{UserID: failedID, RoleCode: string(domain.RoleRegistrar), DepartmentID: "dept-1", Status: domain.StatusActive}}, true,
		[]domain.PermissionOverride{{PermissionCode: "charge:refund", Effect: domain.PermissionDeny}}, failedCreateEvent); !errors.Is(err, port.ErrAuditUnavailable) {
		t.Fatalf("failed bundle create = %v, want ErrAuditUnavailable", err)
	}
	if _, err := store.FindByID(ctx, failedID); !errors.Is(err, port.ErrNotFound) {
		t.Fatalf("failed bundle create user lookup = %v, want ErrNotFound", err)
	}

	beforeFailedUpdate, err := store.FindByID(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	beforeOverrides, err := store.ListUserPermissionOverrides(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	beforeScopes, err := store.ListUserRoleScopes(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	beforeFailedUpdate.DisplayName = "Must Roll Back"
	beforeFailedUpdate.Roles = []domain.Role{domain.RoleRegistrar}
	beforeFailedUpdate.UpdatedAt = now.Add(2 * time.Minute)
	failedUpdateEvent := domain.AuditEvent{
		ID: uuid.NewString(), OccurredAt: now.Add(2 * time.Minute), ActorID: seed.DemoUserID, ActorRoles: string(domain.RoleSystem),
		Action: strings.Repeat("x", 65), ResourceType: "User", ResourceID: userID, Outcome: domain.OutcomeSuccess,
		RequestID: requestPrefix + "update-fail",
	}
	if _, err := store.UpdateUserAccessBundleWithAudit(ctx, *beforeFailedUpdate, true, true, []domain.UserRoleScope{{
		UserID: userID, RoleCode: string(domain.RoleRegistrar), DepartmentID: "dept-1", Status: domain.StatusActive,
	}}, true, []domain.PermissionOverride{{PermissionCode: "charge:refund", Effect: domain.PermissionDeny}}, failedUpdateEvent); !errors.Is(err, port.ErrAuditUnavailable) {
		t.Fatalf("failed bundle update = %v, want ErrAuditUnavailable", err)
	}
	afterFailedUpdate, err := store.FindByID(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if afterFailedUpdate.DisplayName != "Bundle User Updated" || len(afterFailedUpdate.Roles) != 1 || afterFailedUpdate.Roles[0] != domain.RoleDoctor {
		t.Fatalf("failed bundle update changed user = %#v", afterFailedUpdate)
	}
	afterOverrides, err := store.ListUserPermissionOverrides(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(afterOverrides) != len(beforeOverrides) || afterOverrides[0].Effect != beforeOverrides[0].Effect {
		t.Fatalf("failed bundle update changed overrides = %#v, before %#v", afterOverrides, beforeOverrides)
	}
	afterScopes, err := store.ListUserRoleScopes(ctx, userID)
	if err != nil || len(afterScopes) != len(beforeScopes) || afterScopes[0] != beforeScopes[0] {
		t.Fatalf("failed bundle update changed scopes = %#v, before %#v, err=%v", afterScopes, beforeScopes, err)
	}
}

func assertBundleAuditCount(t *testing.T, ctx context.Context, db *sql.DB, requestID string, want int) {
	t.Helper()
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM dbo.audit_log WHERE request_id=@request_id`, sql.Named("request_id", requestID)).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("audit count for %q = %d, want %d", requestID, count, want)
	}
}

func assertBundleOverride(t *testing.T, ctx context.Context, store *Store, userID, wantEffect string) {
	t.Helper()
	rows, err := store.ListUserPermissionOverrides(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].UserID != userID || rows[0].Effect != wantEffect {
		t.Fatalf("overrides for %s = %#v, want effect %s", userID, rows, wantEffect)
	}
}

func assertBundleScopes(t *testing.T, ctx context.Context, store *Store, userID string, want []domain.UserRoleScope) {
	t.Helper()
	got, err := store.ListUserRoleScopes(ctx, userID)
	if err != nil || len(got) != len(want) {
		t.Fatalf("scopes for %s = %+v, want %+v, err=%v", userID, got, want, err)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("scopes for %s = %+v, want %+v", userID, got, want)
		}
	}
}
