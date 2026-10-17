package app

import (
	"testing"

	"github.com/cznUSER/B-HIS/internal/domain"
)

func TestInventoryReadAcceptsScopedGrantAcrossGlobalSnapshot(t *testing.T) {
	user := &domain.User{
		AccessControlLoaded: true,
		EffectivePermissions: []domain.PermissionGrant{{
			PermissionCode: domain.PermInventoryRead,
			ModuleCode:     "master",
			DepartmentID:   "dept-pharmacy",
		}},
	}
	if !hasPermAcrossScopes(user, domain.PermInventoryRead, "master") {
		t.Fatal("a scoped inventory read grant should authorize the unfiltered inventory snapshot")
	}
}

func TestInventoryReadRejectsMissingDisabledAndDeniedGrants(t *testing.T) {
	tests := []struct {
		name string
		user *domain.User
	}{
		{
			name: "missing grant",
			user: &domain.User{AccessControlLoaded: true},
		},
		{
			name: "disabled role has no effective grant",
			user: &domain.User{AccessControlLoaded: true, Roles: []domain.Role{"DISABLED_PHARMACIST"}},
		},
		{
			name: "explicit deny",
			user: &domain.User{
				AccessControlLoaded: true,
				EffectivePermissions: []domain.PermissionGrant{{
					PermissionCode: domain.PermInventoryRead,
					ModuleCode:     "master",
					DepartmentID:   "dept-pharmacy",
				}},
				DeniedPermissions: []domain.PermissionGrant{{
					PermissionCode: domain.PermInventoryRead,
					ModuleCode:     "master",
					DepartmentID:   "dept-pharmacy",
				}},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if hasPermAcrossScopes(tc.user, domain.PermInventoryRead, "master") {
				t.Fatalf("inventory read should be denied for %s", tc.name)
			}
		})
	}
}

func TestInventoryReadKeepsLegacyRoleFallback(t *testing.T) {
	user := &domain.User{Roles: []domain.Role{domain.RolePharmacist}}
	if !hasPermAcrossScopes(user, domain.PermInventoryRead, "master") {
		t.Fatal("legacy pharmacist role should keep inventory read access before the access snapshot is loaded")
	}
}
