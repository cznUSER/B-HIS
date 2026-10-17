package app

import (
	"context"
	"sort"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

// loadEffectivePermissions refreshes RBAC on each authenticated request so role
// edits and per-user overrides take effect without waiting for token expiry.
func (s *Service) loadEffectivePermissions(ctx context.Context, user *domain.User) error {
	if user == nil || s.Master == nil {
		return nil
	}
	store, ok := s.Master.(port.AccessStore)
	if !ok || store == nil {
		return nil
	}
	roles, err := store.ListAccessRoles(ctx)
	if err != nil {
		return err
	}
	modules, err := store.ListAccessModules(ctx)
	if err != nil {
		return err
	}
	permissions, err := store.ListAccessPermissions(ctx, "", "")
	if err != nil {
		return err
	}
	overrides, err := store.ListUserPermissionOverrides(ctx, user.ID)
	if err != nil {
		return err
	}
	roleScopes, err := store.ListUserRoleScopes(ctx, user.ID)
	if err != nil {
		return err
	}
	departments, err := s.Master.ListDepartments(ctx, port.DepartmentQuery{Status: domain.StatusActive})
	if err != nil {
		return err
	}

	assigned := make(map[string]bool, len(user.Roles))
	for _, role := range user.Roles {
		assigned[string(role)] = true
	}
	activeModules := make(map[string]bool, len(modules))
	for _, module := range modules {
		if module.Status == domain.StatusActive {
			activeModules[module.Code] = true
		}
	}
	activePermissions := make(map[string]bool, len(permissions))
	for _, permission := range permissions {
		if permission.Status == domain.StatusActive && activeModules[permission.ModuleCode] {
			activePermissions[permission.Code] = true
		}
	}
	activeDepartments := make(map[string]bool, len(departments))
	for _, department := range departments {
		activeDepartments[department.DepartmentID] = true
	}
	grants := make(map[domain.PermissionGrant]bool)
	assignedRoles := make([]domain.AccessRole, 0, len(user.Roles))
	for _, role := range roles {
		if assigned[role.Code] {
			assignedRoles = append(assignedRoles, role)
		}
	}
	for _, grant := range domain.ExpandUserRoleGrants(user.ID, assignedRoles, roleScopes, activeDepartments) {
		if grant.PermissionCode == domain.PermAdminAll || activePermissions[grant.PermissionCode] {
			grants[grant] = true
		}
	}
	denied := make(map[domain.PermissionGrant]bool)
	allowed := make(map[domain.PermissionGrant]bool)
	for _, override := range overrides {
		grant := domain.PermissionGrant{PermissionCode: override.PermissionCode, ModuleCode: override.ModuleCode, DepartmentID: override.DepartmentID}
		switch override.Effect {
		case domain.PermissionDeny:
			denied[grant] = true
		case domain.PermissionAllow:
			allowed[grant] = true
		}
	}
	for grant := range allowed {
		if !denied[grant] && activePermissions[grant.PermissionCode] {
			grants[grant] = true
		}
	}

	user.AccessControlLoaded = true
	user.EffectivePermissions = user.EffectivePermissions[:0]
	for grant := range grants {
		user.EffectivePermissions = append(user.EffectivePermissions, grant)
	}
	user.DeniedPermissions = user.DeniedPermissions[:0]
	for grant := range denied {
		user.DeniedPermissions = append(user.DeniedPermissions, grant)
	}
	sort.Slice(user.EffectivePermissions, func(i, j int) bool {
		return grantKey(user.EffectivePermissions[i]) < grantKey(user.EffectivePermissions[j])
	})
	sort.Slice(user.DeniedPermissions, func(i, j int) bool { return grantKey(user.DeniedPermissions[i]) < grantKey(user.DeniedPermissions[j]) })
	return nil
}

func grantKey(grant domain.PermissionGrant) string {
	return grant.PermissionCode + "\x00" + grant.ModuleCode + "\x00" + grant.DepartmentID
}
