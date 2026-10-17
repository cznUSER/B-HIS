package domain

import "errors"

const (
	PermissionAllow = "ALLOW"
	PermissionDeny  = "DENY"
)

// AccessModule mirrors HERP acc_modules without coupling business handlers to storage IDs.
type AccessModule struct {
	ID                    string
	Code                  string
	Name                  string
	Description           string
	Status                string
	Hidden                bool
	DepartmentFunctions   int64
	NeedSpecifyDepartment bool
	SecondPermissionID    int
	HomePageURL           string
}

type PermissionCategory struct {
	ID          string
	Name        string
	Description string
	Status      string
}

type PermissionMode struct {
	Code   string
	Name   string
	Status string
}

type AccessPermission struct {
	Code                 string
	Name                 string
	Description          string
	ModuleCode           string
	CategoryID           string
	ModeCode             string
	SecondPermissionCode string
	Actions              []string
	Status               string
	Hidden               bool
	DBRoleName           string
	NeedSecondPermission bool
}

type AccessRole struct {
	Code            string
	Name            string
	Description     string
	ModuleCode      string
	DepartmentID    string
	PermissionCodes []string
	Status          string
	Hidden          bool
	System          bool
	OrderSort       int
	StandName       string
}

// UserRoleScope is an explicit department grant for one assigned user-role pair.
// ModuleCode is inherited from AccessRole and is intentionally not stored here.
type UserRoleScope struct {
	UserID       string
	RoleCode     string
	DepartmentID string
	Status       string
}

type PermissionOverride struct {
	UserID         string
	PermissionCode string
	ModuleCode     string
	DepartmentID   string
	Effect         string
}

type PermissionGrant struct {
	PermissionCode string
	ModuleCode     string
	DepartmentID   string
}

// ExpandUserRoleGrants resolves role permissions against explicit scopes. Any
// row for a user-role pair suppresses its legacy department fallback, including
// DISABLED rows. Departments are supplied from the current active department
// snapshot so a disabled or deleted department cannot produce a grant.
func ExpandUserRoleGrants(userID string, roles []AccessRole, scopes []UserRoleScope, activeDepartments map[string]bool) []PermissionGrant {
	explicit := make(map[string][]UserRoleScope)
	for _, scope := range scopes {
		if scope.UserID == userID {
			explicit[scope.RoleCode] = append(explicit[scope.RoleCode], scope)
		}
	}
	grants := make(map[PermissionGrant]bool)
	for _, role := range roles {
		if role.Status != StatusActive {
			continue
		}
		roleScopes, hasExplicit := explicit[role.Code]
		departments := make([]string, 0, len(roleScopes))
		if hasExplicit {
			for _, scope := range roleScopes {
				if scope.Status == StatusActive && activeDepartments[scope.DepartmentID] {
					departments = append(departments, scope.DepartmentID)
				}
			}
		} else if role.DepartmentID == "" || activeDepartments[role.DepartmentID] {
			departments = append(departments, role.DepartmentID)
		}
		for _, departmentID := range departments {
			for _, code := range role.PermissionCodes {
				grants[PermissionGrant{PermissionCode: code, ModuleCode: role.ModuleCode, DepartmentID: departmentID}] = true
			}
		}
	}
	out := make([]PermissionGrant, 0, len(grants))
	for grant := range grants {
		out = append(out, grant)
	}
	return out
}

func EvaluateAccessPermission(roles []AccessRole, overrides []PermissionOverride, code, moduleCode, departmentID string) bool {
	allowed := false
	for _, role := range roles {
		if !scopeMatches(role.ModuleCode, role.DepartmentID, moduleCode, departmentID) {
			continue
		}
		for _, permission := range role.PermissionCodes {
			if permission == code {
				allowed = true
			}
		}
	}
	for _, override := range overrides {
		if override.PermissionCode != code || !scopeMatches(override.ModuleCode, override.DepartmentID, moduleCode, departmentID) {
			continue
		}
		if override.Effect == PermissionDeny {
			return false
		}
		if override.Effect == PermissionAllow {
			allowed = true
		}
	}
	return allowed
}

func scopeMatches(roleModule, roleDepartment, moduleCode, departmentID string) bool {
	return (roleModule == "" || roleModule == moduleCode) && (roleDepartment == "" || roleDepartment == departmentID)
}

func ValidateAdminChange(actor, target User, nextRoles []Role, nextStatus string, systemAdminCount int) error {
	if actor.ID != target.ID {
		return nil
	}
	if target.HasSystemRole() && (nextStatus != StatusActive || !hasSystemRole(nextRoles)) && systemAdminCount <= 1 {
		return errors.New("last system administrator cannot be disabled or lose SYSTEM role")
	}
	return nil
}

func (u User) HasSystemRole() bool {
	for _, role := range u.Roles {
		if role == RoleSystem {
			return true
		}
	}
	return false
}

func hasSystemRole(roles []Role) bool {
	for _, role := range roles {
		if role == RoleSystem {
			return true
		}
	}
	return false
}
