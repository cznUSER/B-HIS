package domain

import "testing"

func TestExpandUserRoleGrantsUsesExplicitScopesAndPreservesLegacyFallback(t *testing.T) {
	roles := []AccessRole{{
		Code: "CLINIC", ModuleCode: "outpatient", DepartmentID: "legacy-dept",
		PermissionCodes: []string{"order:submit"}, Status: StatusActive,
	}}
	activeDepartments := map[string]bool{"legacy-dept": true, "dept-1": true, "dept-2": true}

	t.Run("explicit active rows replace legacy and allow both departments", func(t *testing.T) {
		grants := ExpandUserRoleGrants("user-1", roles, []UserRoleScope{
			{UserID: "user-1", RoleCode: "CLINIC", DepartmentID: "dept-1", Status: StatusActive},
			{UserID: "user-1", RoleCode: "CLINIC", DepartmentID: "dept-2", Status: StatusActive},
		}, activeDepartments)
		if !hasPermissionGrant(grants, "order:submit", "outpatient", "dept-1") || !hasPermissionGrant(grants, "order:submit", "outpatient", "dept-2") {
			t.Fatalf("explicit departments should be granted: %+v", grants)
		}
		if hasPermissionGrant(grants, "order:submit", "outpatient", "legacy-dept") || hasPermissionGrant(grants, "order:submit", "outpatient", "dept-3") {
			t.Fatalf("explicit scopes must replace legacy and exclude other departments: %+v", grants)
		}
	})

	t.Run("disabled explicit row suppresses legacy fallback", func(t *testing.T) {
		grants := ExpandUserRoleGrants("user-1", roles, []UserRoleScope{
			{UserID: "user-1", RoleCode: "CLINIC", DepartmentID: "dept-1", Status: StatusDisabled},
		}, activeDepartments)
		if len(grants) != 0 {
			t.Fatalf("disabled explicit row must suppress legacy fallback: %+v", grants)
		}
	})

	t.Run("no explicit rows preserve legacy single department", func(t *testing.T) {
		grants := ExpandUserRoleGrants("user-1", roles, nil, activeDepartments)
		if !hasPermissionGrant(grants, "order:submit", "outpatient", "legacy-dept") || hasPermissionGrant(grants, "order:submit", "outpatient", "dept-1") {
			t.Fatalf("legacy department behavior changed: %+v", grants)
		}
	})

	t.Run("inactive departments produce no scoped grant", func(t *testing.T) {
		grants := ExpandUserRoleGrants("user-1", roles, []UserRoleScope{
			{UserID: "user-1", RoleCode: "CLINIC", DepartmentID: "dept-1", Status: StatusActive},
		}, map[string]bool{"dept-1": false})
		if len(grants) != 0 {
			t.Fatalf("inactive department must not produce a grant: %+v", grants)
		}
	})
}

func hasPermissionGrant(grants []PermissionGrant, code, module, department string) bool {
	for _, grant := range grants {
		if grant.PermissionCode == code && grant.ModuleCode == module && grant.DepartmentID == department {
			return true
		}
	}
	return false
}
