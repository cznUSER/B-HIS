package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

func TestReplaceUserRoleScopesValidatesMembershipDepartmentAndStatus(t *testing.T) {
	ctx := context.Background()
	s := New()
	if err := s.CreateAccessRole(ctx, domain.AccessRole{Code: "CLINIC", Name: "Clinic", ModuleCode: "outpatient", Status: domain.StatusActive}); err != nil {
		t.Fatal(err)
	}
	s.Add(domain.User{ID: "scope-user", Username: "scope-user", Roles: []domain.Role{"CLINIC"}, Status: domain.StatusActive})
	valid := []domain.UserRoleScope{{UserID: "scope-user", RoleCode: "CLINIC", DepartmentID: "dept-1", Status: domain.StatusActive}}
	if err := s.ReplaceUserRoleScopes(ctx, "scope-user", valid); err != nil {
		t.Fatalf("valid scope replacement: %v", err)
	}
	got, err := s.ListUserRoleScopes(ctx, "scope-user")
	if err != nil || len(got) != 1 || got[0] != valid[0] {
		t.Fatalf("scope round trip = %+v, %v", got, err)
	}

	tests := []struct {
		name string
		rows []domain.UserRoleScope
	}{
		{name: "unassigned role", rows: []domain.UserRoleScope{{UserID: "scope-user", RoleCode: "OTHER", DepartmentID: "dept-1", Status: domain.StatusActive}}},
		{name: "duplicate department", rows: []domain.UserRoleScope{valid[0], valid[0]}},
		{name: "unknown department", rows: []domain.UserRoleScope{{UserID: "scope-user", RoleCode: "CLINIC", DepartmentID: "missing", Status: domain.StatusActive}}},
		{name: "invalid status", rows: []domain.UserRoleScope{{UserID: "scope-user", RoleCode: "CLINIC", DepartmentID: "dept-1", Status: "UNKNOWN"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := s.ReplaceUserRoleScopes(ctx, "scope-user", test.rows); !errors.Is(err, port.ErrInvalidUserRoleScope) {
				t.Fatalf("replace error = %v, want ErrInvalidUserRoleScope", err)
			}
		})
	}

	dept, err := s.GetDepartment(ctx, "dept-2")
	if err != nil {
		t.Fatal(err)
	}
	dept.Status = domain.StatusInactive
	if _, err := s.UpdateDepartment(ctx, *dept); err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceUserRoleScopes(ctx, "scope-user", []domain.UserRoleScope{{UserID: "scope-user", RoleCode: "CLINIC", DepartmentID: "dept-2", Status: domain.StatusActive}}); !errors.Is(err, port.ErrInvalidUserRoleScope) {
		t.Fatalf("active scope to inactive department = %v, want ErrInvalidUserRoleScope", err)
	}
	if err := s.ReplaceUserRoleScopes(ctx, "scope-user", []domain.UserRoleScope{{UserID: "scope-user", RoleCode: "CLINIC", DepartmentID: "dept-2", Status: domain.StatusDisabled}}); err != nil {
		t.Fatalf("disabled scope to inactive department should be storable: %v", err)
	}
	if err := s.ReplaceUserRoleScopes(ctx, "scope-user", nil); err != nil {
		t.Fatalf("clear scopes: %v", err)
	}
	got, err = s.ListUserRoleScopes(ctx, "scope-user")
	if err != nil || len(got) != 0 {
		t.Fatalf("cleared scopes = %+v, %v", got, err)
	}
}

func TestUserAccessBundlePresenceAndRoleRemovalSemantics(t *testing.T) {
	ctx := context.Background()
	s := New()
	for _, role := range []domain.AccessRole{
		{Code: "KEEP", Name: "Keep", ModuleCode: "outpatient", Status: domain.StatusActive},
		{Code: "DROP", Name: "Drop", ModuleCode: "outpatient", Status: domain.StatusActive},
	} {
		if err := s.CreateAccessRole(ctx, role); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC()
	user, err := s.CreateUserAccessBundleWithAudit(ctx, domain.User{
		ID: "bundle-scope-user", Username: "bundle-scope-user", DisplayName: "Bundle Scope", PasswordHash: "hash",
		Roles: []domain.Role{"KEEP", "DROP"}, Status: domain.StatusActive, CreatedAt: now, UpdatedAt: now,
	}, true,
		[]domain.UserRoleScope{
			{UserID: "bundle-scope-user", RoleCode: "KEEP", DepartmentID: "dept-1", Status: domain.StatusActive},
			{UserID: "bundle-scope-user", RoleCode: "DROP", DepartmentID: "dept-2", Status: domain.StatusDisabled},
		}, true, []domain.PermissionOverride{{PermissionCode: "charge:refund", Effect: domain.PermissionDeny}},
		domain.AuditEvent{ID: "bundle-scope-create-audit", Outcome: domain.OutcomeSuccess})
	if err != nil || user == nil {
		t.Fatalf("create bundle: user=%+v err=%v", user, err)
	}

	// Role-only replacement keeps the retained role's scope, removes the dropped role's scope,
	// and leaves permission overrides intact.
	user.Roles = []domain.Role{"KEEP"}
	user.DisplayName = "Role Updated"
	if _, err := s.UpdateUserAccessBundleWithAudit(ctx, *user, true, false, nil, false, nil,
		domain.AuditEvent{ID: "bundle-scope-roles-audit", Outcome: domain.OutcomeSuccess}); err != nil {
		t.Fatal(err)
	}
	scopes, err := s.ListUserRoleScopes(ctx, user.ID)
	if err != nil || len(scopes) != 1 || scopes[0].RoleCode != "KEEP" {
		t.Fatalf("scopes after role removal = %+v, %v", scopes, err)
	}
	overrides, err := s.ListUserPermissionOverrides(ctx, user.ID)
	if err != nil || len(overrides) != 1 || overrides[0].Effect != domain.PermissionDeny {
		t.Fatalf("overrides after role-only update = %+v, %v", overrides, err)
	}

	// Scope-only replacement keeps roles and overrides.
	user.DisplayName = "Scope Updated"
	if _, err := s.UpdateUserAccessBundleWithAudit(ctx, *user, false, true, []domain.UserRoleScope{{
		UserID: user.ID, RoleCode: "KEEP", DepartmentID: "dept-2", Status: domain.StatusActive,
	}}, false, nil, domain.AuditEvent{ID: "bundle-scope-only-audit", Outcome: domain.OutcomeSuccess}); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.FindByID(ctx, user.ID)
	if err != nil || len(loaded.Roles) != 1 || loaded.Roles[0] != "KEEP" {
		t.Fatalf("roles after scope-only update = %+v, %v", loaded, err)
	}
	overrides, err = s.ListUserPermissionOverrides(ctx, user.ID)
	if err != nil || len(overrides) != 1 || overrides[0].Effect != domain.PermissionDeny {
		t.Fatalf("overrides after scope-only update = %+v, %v", overrides, err)
	}

	// Present empty scopes clear the explicit replacement.
	if _, err := s.UpdateUserAccessBundleWithAudit(ctx, *user, false, true, []domain.UserRoleScope{}, false, nil,
		domain.AuditEvent{ID: "bundle-scope-clear-audit", Outcome: domain.OutcomeSuccess}); err != nil {
		t.Fatal(err)
	}
	scopes, err = s.ListUserRoleScopes(ctx, user.ID)
	if err != nil || len(scopes) != 0 {
		t.Fatalf("scopes after explicit clear = %+v, %v", scopes, err)
	}
}
