package port

import (
	"context"

	"github.com/cznUSER/B-HIS/internal/domain"
)

type AccessStore interface {
	ListAccessModules(ctx context.Context) ([]domain.AccessModule, error)
	CreateAccessModule(ctx context.Context, item domain.AccessModule) error
	UpdateAccessModule(ctx context.Context, item domain.AccessModule) error
	ListPermissionCategories(ctx context.Context) ([]domain.PermissionCategory, error)
	CreatePermissionCategory(ctx context.Context, item domain.PermissionCategory) error
	UpdatePermissionCategory(ctx context.Context, item domain.PermissionCategory) error
	ListPermissionModes(ctx context.Context) ([]domain.PermissionMode, error)
	CreatePermissionMode(ctx context.Context, item domain.PermissionMode) error
	UpdatePermissionMode(ctx context.Context, item domain.PermissionMode) error
	ListAccessPermissions(ctx context.Context, moduleCode, categoryID string) ([]domain.AccessPermission, error)
	CreateAccessPermission(ctx context.Context, permission domain.AccessPermission) error
	UpdateAccessPermission(ctx context.Context, permission domain.AccessPermission) error
	ListAccessRoles(ctx context.Context) ([]domain.AccessRole, error)
	CreateAccessRole(ctx context.Context, role domain.AccessRole) error
	UpdateAccessRole(ctx context.Context, role domain.AccessRole) error
	ListUserPermissionOverrides(ctx context.Context, userID string) ([]domain.PermissionOverride, error)
	ReplaceUserPermissionOverrides(ctx context.Context, userID string, overrides []domain.PermissionOverride) error
	ListUserRoleScopes(ctx context.Context, userID string) ([]domain.UserRoleScope, error)
	ReplaceUserRoleScopes(ctx context.Context, userID string, scopes []domain.UserRoleScope) error
	CountSystemAdmins(ctx context.Context) (int, error)
}

// AtomicAccessStore combines an access-control mutation with its successful
// audit row. Implementations that can commit both records in one transaction
// expose this optional capability; callers must retain the AccessStore fallback
// for memory-compatible and legacy adapters.
type AtomicAccessStore interface {
	AccessStore
	CreateAccessRoleWithAudit(ctx context.Context, role domain.AccessRole, event domain.AuditEvent) error
	UpdateAccessRoleWithAudit(ctx context.Context, role domain.AccessRole, event domain.AuditEvent) error
	CreateAccessPermissionWithAudit(ctx context.Context, permission domain.AccessPermission, event domain.AuditEvent) error
	UpdateAccessPermissionWithAudit(ctx context.Context, permission domain.AccessPermission, event domain.AuditEvent) error
	ReplaceUserPermissionOverridesWithAudit(ctx context.Context, userID string, overrides []domain.PermissionOverride, event domain.AuditEvent) error
}
