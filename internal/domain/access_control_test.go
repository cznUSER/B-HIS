package domain

import "testing"

func TestEvaluateAccessPermissionUsesRoleUnionAndExplicitDeny(t *testing.T) {
	roles := []AccessRole{{Code: "REGISTRAR", PermissionCodes: []string{"patient:create", "encounter:create"}}}
	overrides := []PermissionOverride{{PermissionCode: "encounter:create", Effect: PermissionDeny}}
	if !EvaluateAccessPermission(roles, overrides, "patient:create", "outpatient", "") {
		t.Fatal("role permission should allow patient:create")
	}
	if EvaluateAccessPermission(roles, overrides, "encounter:create", "outpatient", "") {
		t.Fatal("explicit deny must override an inherited role permission")
	}
}

func TestEvaluateAccessPermissionRequiresMatchingScope(t *testing.T) {
	roles := []AccessRole{{
		Code:             "DOCTOR",
		PermissionCodes:  []string{"order:submit"},
		ModuleCode:       "outpatient",
		DepartmentID:     "dept-a",
	}}
	if EvaluateAccessPermission(roles, nil, "order:submit", "outpatient", "dept-b") {
		t.Fatal("permission must not cross a role department scope")
	}
	if !EvaluateAccessPermission(roles, nil, "order:submit", "outpatient", "dept-a") {
		t.Fatal("matching module and department should allow permission")
	}
}

func TestProtectLastSystemAdministrator(t *testing.T) {
	actor := User{ID: "system-1", Roles: []Role{RoleSystem}, Status: StatusActive}
	target := User{ID: "system-1", Roles: []Role{RoleSystem}, Status: StatusActive}
	if err := ValidateAdminChange(actor, target, []Role{RoleRegistrar}, StatusDisabled, 1); err == nil {
		t.Fatal("last system administrator must not be removable or disabled")
	}
	if err := ValidateAdminChange(actor, target, []Role{RoleSystem}, StatusActive, 2); err != nil {
		t.Fatalf("a second system administrator should permit editing: %v", err)
	}
}
