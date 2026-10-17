package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

func TestUserAccessBundlePersistsProfileAndOverridesTogether(t *testing.T) {
	ctx := context.Background()
	svc, store := newAccessSvc(t)
	overrides := []PermissionOverrideInput{{PermissionCode: "charge:refund", Effect: domain.PermissionDeny}}
	created, err := svc.CreateUser(ctx, systemClaims(), CreateUserInput{
		Username: "bundle-user", DisplayName: "整屏用户", Roles: []string{"REGISTRAR"},
		PermissionOverrides: &overrides,
	}, "req-bundle-create", "127.0.0.1")
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	if created.DisplayName != "整屏用户" {
		t.Fatalf("unexpected created user: %#v", created)
	}
	rows, err := store.ListUserPermissionOverrides(ctx, created.UserID)
	if err != nil || len(rows) != 1 || rows[0].Effect != domain.PermissionDeny {
		t.Fatalf("created overrides: err=%v rows=%+v", err, rows)
	}

	updatedOverrides := []PermissionOverrideInput{{PermissionCode: "charge:refund", Effect: domain.PermissionAllow}}
	updated, err := svc.PatchUser(ctx, systemClaims(), created.UserID, PatchUserInput{
		DisplayName:         stringPtr("整屏用户已更新"),
		PermissionOverrides: &updatedOverrides,
	}, "req-bundle-update", "127.0.0.1")
	if err != nil {
		t.Fatalf("update bundle: %v", err)
	}
	if updated.DisplayName != "整屏用户已更新" {
		t.Fatalf("unexpected updated user: %#v", updated)
	}
	rows, err = store.ListUserPermissionOverrides(ctx, created.UserID)
	if err != nil || len(rows) != 1 || rows[0].Effect != domain.PermissionAllow {
		t.Fatalf("updated overrides: err=%v rows=%+v", err, rows)
	}
}

func TestPermissionOverrideValidationRejectsDuplicateTarget(t *testing.T) {
	ctx := context.Background()
	svc, store := newAccessSvc(t)
	duplicates := []PermissionOverrideInput{
		{PermissionCode: "charge:refund", Effect: domain.PermissionDeny},
		{PermissionCode: "charge:refund", Effect: domain.PermissionAllow},
	}
	_, err := svc.CreateUser(ctx, systemClaims(), CreateUserInput{
		Username: "duplicate-override-user", DisplayName: "重复覆盖用户", Roles: []string{string(domain.RoleRegistrar)},
		PermissionOverrides: &duplicates,
	}, "req-duplicate-override", "127.0.0.1")
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Status != 400 || appErr.Code != CodeValidation {
		t.Fatalf("duplicate override error = %v, want 400 VALIDATION_ERROR", err)
	}
	fields, ok := appErr.Details["fields"].([]string)
	if !ok || len(fields) != 1 || fields[0] != "overrides[1].permissionCode" {
		t.Fatalf("duplicate override fields = %#v", appErr.Details["fields"])
	}
	if _, findErr := store.FindByUsername(ctx, "duplicate-override-user"); !errors.Is(findErr, port.ErrNotFound) {
		t.Fatalf("duplicate override user was persisted: %v", findErr)
	}
}

func TestUserRoleScopesAreReturnedAndPresenceControlsReplacement(t *testing.T) {
	ctx := context.Background()
	svc, store := newAccessSvc(t)
	scopes := []UserRoleScopeInput{{RoleCode: string(domain.RoleRegistrar), DepartmentID: "dept-1", Status: domain.StatusActive}}
	created, err := svc.CreateUser(ctx, systemClaims(), CreateUserInput{
		Username: "scope-presence-user", DisplayName: "范围用户", Roles: []string{string(domain.RoleRegistrar)}, RoleScopes: &scopes,
	}, "req-scope-presence-create", "127.0.0.1")
	if err != nil {
		t.Fatalf("create scoped user: %v", err)
	}
	if len(created.RoleScopes) != 1 || created.RoleScopes[0].DepartmentID != "dept-1" {
		t.Fatalf("create DTO scopes = %+v", created.RoleScopes)
	}

	if _, err := svc.PatchUser(ctx, systemClaims(), created.UserID, PatchUserInput{DisplayName: stringPtr("范围用户更新")}, "req-scope-presence-omit", "127.0.0.1"); err != nil {
		t.Fatalf("patch omitting roleScopes: %v", err)
	}
	rows, err := store.ListUserRoleScopes(ctx, created.UserID)
	if err != nil || len(rows) != 1 || rows[0].DepartmentID != "dept-1" {
		t.Fatalf("omitted roleScopes changed rows = %+v, err=%v", rows, err)
	}

	emptyScopes := []UserRoleScopeInput{}
	updated, err := svc.PatchUser(ctx, systemClaims(), created.UserID, PatchUserInput{RoleScopes: &emptyScopes}, "req-scope-presence-clear", "127.0.0.1")
	if err != nil {
		t.Fatalf("patch explicit empty roleScopes: %v", err)
	}
	rows, err = store.ListUserRoleScopes(ctx, created.UserID)
	if err != nil || len(rows) != 0 || len(updated.RoleScopes) != 0 {
		t.Fatalf("explicit empty roleScopes did not clear: rows=%+v dto=%+v err=%v", rows, updated.RoleScopes, err)
	}
}

func TestUserRoleScopeValidationUsesIndexedFields(t *testing.T) {
	ctx := context.Background()
	svc, _ := newAccessSvc(t)
	duplicate := []UserRoleScopeInput{
		{RoleCode: string(domain.RoleRegistrar), DepartmentID: "dept-1", Status: domain.StatusActive},
		{RoleCode: string(domain.RoleRegistrar), DepartmentID: "dept-1", Status: domain.StatusActive},
	}
	_, err := svc.CreateUser(ctx, systemClaims(), CreateUserInput{
		Username: "scope-invalid-user", DisplayName: "范围用户", Roles: []string{string(domain.RoleRegistrar)}, RoleScopes: &duplicate,
	}, "req-scope-invalid", "127.0.0.1")
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Status != 400 || appErr.Code != CodeValidation {
		t.Fatalf("duplicate scope error = %v, want 400 VALIDATION_ERROR", err)
	}
	fields, ok := appErr.Details["fields"].([]string)
	if !ok || len(fields) != 1 || fields[0] != "roleScopes[1].departmentId" {
		t.Fatalf("duplicate scope fields = %#v", appErr.Details["fields"])
	}
}

func TestCreateUserEmptyScopesDoNotDiscardOtherValidationErrors(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name  string
		input CreateUserInput
	}{
		{
			name: "unassigned role",
			input: CreateUserInput{
				Username: "scope-bad-role", DisplayName: "Invalid role", Roles: []string{"NO_SUCH"},
				RoleScopes: &[]UserRoleScopeInput{},
			},
		},
		{
			name: "invalid profile",
			input: CreateUserInput{
				Username: "scope-bad-profile", DisplayName: "Invalid profile", Roles: []string{string(domain.RoleRegistrar)},
				Description: strings.Repeat("x", 600), RoleScopes: &[]UserRoleScopeInput{},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc, store := newAccessSvc(t)
			_, err := svc.CreateUser(ctx, systemClaims(), test.input, "req-scope-presence-invalid", "127.0.0.1")
			var appErr *Error
			if !errors.As(err, &appErr) || appErr.Status != 400 || appErr.Code != CodeValidation {
				t.Fatalf("create error = %v, want 400 VALIDATION_ERROR", err)
			}
			if _, findErr := store.FindByUsername(ctx, test.input.Username); !errors.Is(findErr, port.ErrNotFound) {
				t.Fatalf("invalid user was persisted: lookup error = %v", findErr)
			}
		})
	}
}

func stringPtr(value string) *string { return &value }
