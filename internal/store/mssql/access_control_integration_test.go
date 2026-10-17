package mssql

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/google/uuid"
)

func TestAccessRoleAndUserOverridesPersistAtomically(t *testing.T) {
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
		t.Fatalf("refusing to write access-control fixture to database %q", databaseName)
	}

	suffix := newTestSuffix()
	roleCode := "T" + suffix[:16]
	permissionCode := "P" + suffix[:16]
	userID := "U" + suffix[:20]
	username := "ituser_" + suffix[:20]
	cleanup := func() {
		statements := []struct {
			query string
			args  []any
		}{
			{`DELETE FROM dbo.audit_log WHERE request_id LIKE @prefix`, []any{sql.Named("prefix", "atomic-%"+suffix)}},
			{`DELETE FROM dbo.user_permission_override WHERE user_id=@user_id`, []any{sql.Named("user_id", userID)}},
			{`DELETE FROM dbo.access_role_permission WHERE role_code=@role_code`, []any{sql.Named("role_code", roleCode)}},
			{`DELETE FROM dbo.access_role WHERE role_code=@role_code`, []any{sql.Named("role_code", roleCode)}},
			{`DELETE FROM dbo.access_role_permission WHERE role_code=@role_code`, []any{sql.Named("role_code", roleCode+"2")}},
			{`DELETE FROM dbo.access_role WHERE role_code=@role_code`, []any{sql.Named("role_code", roleCode+"2")}},
			{`DELETE FROM dbo.access_permission WHERE permission_code=@permission_code`, []any{sql.Named("permission_code", permissionCode)}},
			{`DELETE FROM dbo.role WHERE role_code=@role_code`, []any{sql.Named("role_code", roleCode)}},
			{`DELETE FROM dbo.app_user WHERE user_id=@user_id`, []any{sql.Named("user_id", userID)}},
		}
		for _, statement := range statements {
			if _, err := db.ExecContext(ctx, statement.query, statement.args...); err != nil {
				t.Errorf("cleanup access-control fixture: %v", err)
			}
		}
	}
	cleanup()
	defer cleanup()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO dbo.app_user(user_id,username,display_name,password_hash,status,created_at)
		VALUES(@user_id,@username,N'Access Control Test','not-a-login-hash','ACTIVE',SYSUTCDATETIME())`,
		sql.Named("user_id", userID), sql.Named("username", username)); err != nil {
		t.Fatal(err)
	}

	store := New(db)
	role := domain.AccessRole{Code: roleCode, Name: "Access Test", ModuleCode: "master", Status: "ACTIVE", PermissionCodes: []string{"inventory:read"}}
	if err := store.CreateAccessRole(ctx, role); err != nil {
		t.Fatal(err)
	}
	role.PermissionCodes = []string{"audit:read"}
	if err := store.UpdateAccessRole(ctx, role); err != nil {
		t.Fatal(err)
	}
	roles, err := store.ListAccessRoles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var savedRole *domain.AccessRole
	for i := range roles {
		if roles[i].Code == roleCode {
			savedRole = &roles[i]
			break
		}
	}
	if savedRole == nil || !reflect.DeepEqual(savedRole.PermissionCodes, []string{"audit:read"}) {
		t.Fatalf("saved role permissions = %+v, want [audit:read]", savedRole)
	}

	wantOverride := domain.PermissionOverride{UserID: userID, PermissionCode: "inventory:read", ModuleCode: "master", DepartmentID: "dept-test", Effect: domain.PermissionDeny}
	if err := store.ReplaceUserPermissionOverrides(ctx, userID, []domain.PermissionOverride{wantOverride}); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceUserPermissionOverrides(ctx, userID, []domain.PermissionOverride{
		{PermissionCode: "inventory:read", ModuleCode: "master", DepartmentID: "dept-test", Effect: domain.PermissionDeny},
		{PermissionCode: "inventory:read", ModuleCode: "master", DepartmentID: "dept-test", Effect: domain.PermissionAllow},
	}); err == nil || !strings.Contains(err.Error(), "duplicate permission override") {
		t.Fatalf("duplicate permission override error = %v, want explicit duplicate error", err)
	}
	if err := store.ReplaceUserPermissionOverrides(ctx, userID, []domain.PermissionOverride{{PermissionCode: "inventory:read", Effect: "INVALID"}}); err == nil {
		t.Fatal("ReplaceUserPermissionOverrides accepted an invalid effect")
	}
	gotOverrides, err := store.ListUserPermissionOverrides(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotOverrides, []domain.PermissionOverride{wantOverride}) {
		t.Fatalf("overrides after failed replacement = %+v, want original row %+v", gotOverrides, wantOverride)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO dbo.user_permission_override(user_id,permission_code,module_code,department_id,effect)
		VALUES(@user_id,'inventory:read','master','invalid-effect','INVALID')`, sql.Named("user_id", userID)); err == nil {
		t.Fatal("database accepted an invalid permission override effect")
	}

	when := time.Now().UTC()
	badAudit := func(requestID string) domain.AuditEvent {
		return domain.AuditEvent{ID: uuid.NewString(), OccurredAt: when, ActorID: userID, Action: strings.Repeat("x", 65), ResourceType: "Access", RequestID: requestID, Outcome: domain.OutcomeSuccess}
	}
	assertAudit := func(requestID string) {
		t.Helper()
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM dbo.audit_log WHERE request_id=@request_id`, sql.Named("request_id", requestID)).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("audit count for %q = %d, want 1", requestID, count)
		}
	}
	role2 := domain.AccessRole{Code: roleCode + "2", Name: "Atomic Role", ModuleCode: "master", Status: domain.StatusActive, PermissionCodes: []string{"inventory:read"}}
	roleCreateRequest := "atomic-role-create-" + suffix
	if err := store.CreateAccessRoleWithAudit(ctx, role2, domain.AuditEvent{ID: uuid.NewString(), OccurredAt: when, ActorID: userID, Action: "role.create", ResourceType: "AccessRole", ResourceID: role2.Code, RequestID: roleCreateRequest, Outcome: domain.OutcomeSuccess}); err != nil {
		t.Fatalf("atomic role create = %v", err)
	}
	assertAudit(roleCreateRequest)
	role2.Name = "Atomic Role Changed"
	if err := store.UpdateAccessRoleWithAudit(ctx, role2, badAudit("atomic-role-update-fail-"+suffix)); !errors.Is(err, port.ErrAuditUnavailable) {
		t.Fatalf("atomic role audit failure = %v, want ErrAuditUnavailable", err)
	}
	roles, err = store.ListAccessRoles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range roles {
		if row.Code == role2.Code && row.Name != "Atomic Role" {
			t.Fatalf("failed atomic role update changed name to %q", row.Name)
		}
	}

	permission := domain.AccessPermission{Code: permissionCode, Name: "Atomic Permission", ModuleCode: "master", Status: domain.StatusActive, Actions: []string{"S"}}
	permissionCreateRequest := "atomic-permission-create-" + suffix
	if err := store.CreateAccessPermissionWithAudit(ctx, permission, domain.AuditEvent{ID: uuid.NewString(), OccurredAt: when, ActorID: userID, Action: "permission.create", ResourceType: "AccessPermission", ResourceID: permissionCode, RequestID: permissionCreateRequest, Outcome: domain.OutcomeSuccess}); err != nil {
		t.Fatalf("atomic permission create = %v", err)
	}
	assertAudit(permissionCreateRequest)
	permission.Name = "Atomic Permission Changed"
	if err := store.UpdateAccessPermissionWithAudit(ctx, permission, badAudit("atomic-permission-update-fail-"+suffix)); !errors.Is(err, port.ErrAuditUnavailable) {
		t.Fatalf("atomic permission audit failure = %v, want ErrAuditUnavailable", err)
	}
	permissions, err := store.ListAccessPermissions(ctx, "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range permissions {
		if row.Code == permissionCode && row.Name != "Atomic Permission" {
			t.Fatalf("failed atomic permission update changed name to %q", row.Name)
		}
	}

	atomicOverride := []domain.PermissionOverride{{UserID: userID, PermissionCode: "inventory:read", Effect: domain.PermissionAllow}}
	overrideRequest := "atomic-override-ok-" + suffix
	if err := store.ReplaceUserPermissionOverridesWithAudit(ctx, userID, atomicOverride, domain.AuditEvent{ID: uuid.NewString(), OccurredAt: when, ActorID: userID, Action: "user.permission_override", ResourceType: "User", ResourceID: userID, RequestID: overrideRequest, Outcome: domain.OutcomeSuccess}); err != nil {
		t.Fatalf("atomic override replace = %v", err)
	}
	assertAudit(overrideRequest)
	if err := store.ReplaceUserPermissionOverridesWithAudit(ctx, userID, []domain.PermissionOverride{{PermissionCode: "inventory:read", Effect: domain.PermissionDeny}}, badAudit("atomic-override-fail-"+suffix)); !errors.Is(err, port.ErrAuditUnavailable) {
		t.Fatalf("atomic override audit failure = %v, want ErrAuditUnavailable", err)
	}
	gotOverrides, err = store.ListUserPermissionOverrides(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotOverrides, atomicOverride) {
		t.Fatalf("failed atomic override changed rows = %+v, want %+v", gotOverrides, atomicOverride)
	}
}

func newTestSuffix() string {
	return uuid.NewString()[:26]
}
