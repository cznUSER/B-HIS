package memory

import (
	"context"
	"fmt"
	"strings"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

var _ port.AtomicUserAccessBundleStore = (*Store)(nil)

// CreateUserAccessBundleWithAudit inserts a new identity and its complete
// permission-override set while holding the same pair of locks as updates.
func (s *Store) CreateUserAccessBundleWithAudit(_ context.Context, u domain.User, replaceRoleScopes bool, roleScopes []domain.UserRoleScope, replaceOverrides bool, overrides []domain.PermissionOverride, event domain.AuditEvent) (*domain.User, error) {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()

	if replaceRoleScopes {
		if err := s.validateUserRoleScopesLocked(u.ID, u.Roles, roleScopes); err != nil {
			return nil, err
		}
	}
	var normalizedOverrides []domain.PermissionOverride
	if replaceOverrides {
		var err error
		normalizedOverrides, err = normalizeUserAccessOverrides(u.ID, overrides)
		if err != nil {
			return nil, err
		}
	}
	created, err := s.createUserLocked(u, nil)
	if err != nil {
		return nil, err
	}
	if replaceRoleScopes {
		s.roleScopes[u.ID] = normalizeUserRoleScopes(u.ID, roleScopes)
	}
	if replaceOverrides {
		s.overrides[u.ID] = normalizedOverrides
	}
	s.appendAuditLocked(event)
	return created, nil
}

// UpdateUserAccessBundleWithAudit keeps the identity and access-control
// aggregates behind one lock. The access mutex is acquired first, matching
// the existing access-with-audit methods before taking the audit/user lock.
func (s *Store) UpdateUserAccessBundleWithAudit(_ context.Context, u domain.User, replaceRoles, replaceRoleScopes bool, roleScopes []domain.UserRoleScope, replaceOverrides bool, overrides []domain.PermissionOverride, event domain.AuditEvent) (*domain.User, error) {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()

	cur, ok := s.byID[u.ID]
	if !ok {
		return nil, port.ErrNotFound
	}
	if cur.Status == domain.StatusActive && cur.HasSystemRole() && (u.Status != domain.StatusActive || !u.HasSystemRole()) {
		count := 0
		for _, row := range s.byID {
			if row.Status == domain.StatusActive && row.HasSystemRole() {
				count++
			}
		}
		if count <= 1 {
			return nil, port.ErrLastSystemAdmin
		}
	}

	if replaceRoleScopes {
		if err := s.validateUserRoleScopesLocked(u.ID, u.Roles, roleScopes); err != nil {
			return nil, err
		}
	}
	var normalizedOverrides []domain.PermissionOverride
	if replaceOverrides {
		var err error
		normalizedOverrides, err = normalizeUserAccessOverrides(u.ID, overrides)
		if err != nil {
			return nil, err
		}
	}
	updated, err := s.updateUserLocked(u, replaceRoles)
	if err != nil {
		return nil, err
	}
	if replaceRoles {
		assigned := make(map[string]bool, len(u.Roles))
		for _, role := range u.Roles {
			assigned[string(role)] = true
		}
		kept := s.roleScopes[u.ID][:0]
		for _, scope := range s.roleScopes[u.ID] {
			if assigned[scope.RoleCode] {
				kept = append(kept, scope)
			}
		}
		s.roleScopes[u.ID] = kept
	}
	if replaceRoleScopes {
		s.roleScopes[u.ID] = normalizeUserRoleScopes(u.ID, roleScopes)
	}
	if replaceOverrides {
		s.overrides[u.ID] = normalizedOverrides
	}
	s.appendAuditLocked(event)
	return updated, nil
}

func normalizeUserRoleScopes(userID string, scopes []domain.UserRoleScope) []domain.UserRoleScope {
	items := make([]domain.UserRoleScope, 0, len(scopes))
	for _, scope := range scopes {
		scope.UserID = userID
		scope.RoleCode = strings.TrimSpace(scope.RoleCode)
		scope.DepartmentID = strings.TrimSpace(scope.DepartmentID)
		scope.Status = strings.ToUpper(strings.TrimSpace(scope.Status))
		items = append(items, scope)
	}
	return items
}

func (s *Store) validateUserRoleScopesLocked(userID string, roles []domain.Role, scopes []domain.UserRoleScope) error {
	assigned := make(map[string]bool, len(roles))
	for _, role := range roles {
		assigned[string(role)] = true
	}
	seen := make(map[string]bool, len(scopes))
	for _, scope := range normalizeUserRoleScopes(userID, scopes) {
		if scope.UserID != userID || scope.RoleCode == "" || scope.DepartmentID == "" || !assigned[scope.RoleCode] || (scope.Status != domain.StatusActive && scope.Status != domain.StatusDisabled) {
			return port.ErrInvalidUserRoleScope
		}
		role, ok := s.accessRoles[scope.RoleCode]
		if !ok || role.Status == domain.StatusDisabled || role.Status == domain.StatusInactive {
			return port.ErrInvalidUserRoleScope
		}
		key := scope.RoleCode + "\x00" + scope.DepartmentID
		if seen[key] {
			return port.ErrInvalidUserRoleScope
		}
		seen[key] = true
		dept, ok := s.departments[scope.DepartmentID]
		if !ok || dept.IsDeleted || (scope.Status == domain.StatusActive && dept.Status != domain.StatusActive) {
			return port.ErrInvalidUserRoleScope
		}
	}
	return nil
}

func normalizeUserAccessOverrides(userID string, overrides []domain.PermissionOverride) ([]domain.PermissionOverride, error) {
	normalized := make([]domain.PermissionOverride, 0, len(overrides))
	seen := make(map[string]struct{}, len(overrides))
	for _, item := range overrides {
		item.UserID = userID
		item.PermissionCode = strings.TrimSpace(item.PermissionCode)
		item.ModuleCode = strings.TrimSpace(item.ModuleCode)
		item.DepartmentID = strings.TrimSpace(item.DepartmentID)
		item.Effect = strings.ToUpper(strings.TrimSpace(item.Effect))
		key := strings.Join([]string{item.PermissionCode, item.ModuleCode, item.DepartmentID}, "\x00")
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("duplicate permission override")
		}
		seen[key] = struct{}{}
		normalized = append(normalized, item)
	}
	return normalized, nil
}
