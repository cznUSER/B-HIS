package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/cznUSER/B-HIS/internal/store/seed"
)

func TestUserAccessBundleCreateAndUpdateAtomicallyInMemory(t *testing.T) {
	s := New()
	ctx := context.Background()
	createdID := "bundle-memory-user"
	created, err := s.CreateUserAccessBundleWithAudit(ctx, domain.User{
		ID: createdID, Username: "bundle-memory-user", DisplayName: "批量用户", PasswordHash: "hash",
		Roles: []domain.Role{domain.RoleRegistrar}, Status: domain.StatusActive,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}, false, nil, true, []domain.PermissionOverride{{PermissionCode: "charge:refund", Effect: "deny"}}, domain.AuditEvent{
		ID: "bundle-memory-create-audit", ActorID: seed.DemoUserID, Action: "user.create", Outcome: domain.OutcomeSuccess,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created == nil || created.ID != createdID {
		t.Fatalf("created user = %#v", created)
	}
	overrides, err := s.ListUserPermissionOverrides(ctx, createdID)
	if err != nil {
		t.Fatal(err)
	}
	if len(overrides) != 1 || overrides[0].UserID != createdID || overrides[0].Effect != domain.PermissionDeny {
		t.Fatalf("created overrides = %#v", overrides)
	}

	next := *created
	next.DisplayName = "批量用户已更新"
	next.Roles = []domain.Role{domain.RoleDoctor}
	next.UpdatedAt = time.Now().UTC()
	updated, err := s.UpdateUserAccessBundleWithAudit(ctx, next, true, false, nil, true, []domain.PermissionOverride{{PermissionCode: "charge:refund", Effect: domain.PermissionAllow}}, domain.AuditEvent{
		ID: "bundle-memory-update-audit", ActorID: seed.DemoUserID, Action: "user.update", Outcome: domain.OutcomeSuccess,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.DisplayName != next.DisplayName || len(updated.Roles) != 1 || updated.Roles[0] != domain.RoleDoctor {
		t.Fatalf("updated user = %#v", updated)
	}
	overrides, err = s.ListUserPermissionOverrides(ctx, createdID)
	if err != nil {
		t.Fatal(err)
	}
	if len(overrides) != 1 || overrides[0].Effect != domain.PermissionAllow {
		t.Fatalf("updated overrides = %#v", overrides)
	}
	audits := s.Audits()
	seen := map[string]bool{}
	for _, item := range audits {
		seen[item.ID] = true
	}
	if !seen["bundle-memory-create-audit"] || !seen["bundle-memory-update-audit"] {
		t.Fatalf("bundle audit rows = %#v", audits)
	}
}

func TestAccessStoreAtomicMutationsAppendSuccessAudit(t *testing.T) {
	s := New()
	ctx := context.Background()
	role := domain.AccessRole{Code: "ATOMIC_ROLE", Name: "Atomic Role", ModuleCode: "master", Status: domain.StatusActive, PermissionCodes: []string{"inventory:read"}}
	if err := s.CreateAccessRoleWithAudit(ctx, role, domain.AuditEvent{Action: "role.create", Outcome: domain.OutcomeSuccess, ResourceID: role.Code}); err != nil {
		t.Fatal(err)
	}
	permission := domain.AccessPermission{Code: "atomic:permission", Name: "Atomic Permission", ModuleCode: "master", Status: domain.StatusActive, Actions: []string{"S"}}
	if err := s.CreateAccessPermissionWithAudit(ctx, permission, domain.AuditEvent{Action: "permission.create", Outcome: domain.OutcomeSuccess, ResourceID: permission.Code}); err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceUserPermissionOverridesWithAudit(ctx, "demo-user", []domain.PermissionOverride{{PermissionCode: permission.Code, Effect: domain.PermissionAllow}}, domain.AuditEvent{Action: "user.permission_override", Outcome: domain.OutcomeSuccess, ResourceID: "demo-user"}); err != nil {
		t.Fatal(err)
	}
	audits := s.Audits()
	seen := map[string]bool{}
	for _, audit := range audits {
		if audit.Outcome == domain.OutcomeSuccess {
			seen[audit.Action] = true
		}
	}
	for _, action := range []string{"role.create", "permission.create", "user.permission_override"} {
		if !seen[action] {
			t.Fatalf("missing atomic success audit %q in %#v", action, audits)
		}
	}
}

func TestAccessStoreSeedsHerpRolesAndPermissions(t *testing.T) {
	s := New()
	roles, err := s.ListAccessRoles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) < 5 {
		t.Fatalf("expected HERP-compatible seed roles, got %d", len(roles))
	}
	permissions, err := s.ListAccessPermissions(context.Background(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(permissions) == 0 {
		t.Fatal("expected seeded permissions")
	}
}

func TestAccessStoreUserOverrideRoundTrip(t *testing.T) {
	s := New()
	want := []domain.PermissionOverride{{UserID: "demo-user", PermissionCode: "charge:refund", Effect: domain.PermissionDeny}}
	if err := s.ReplaceUserPermissionOverrides(context.Background(), "demo-user", want); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListUserPermissionOverrides(context.Background(), "demo-user")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Effect != domain.PermissionDeny {
		t.Fatalf("unexpected override round trip: %#v", got)
	}
}

func TestUserAccessBundleRejectsDuplicatePermissionOverride(t *testing.T) {
	s := New()
	ctx := context.Background()
	userID := "duplicate-bundle-memory-user"
	_, err := s.CreateUserAccessBundleWithAudit(ctx, domain.User{
		ID: userID, Username: userID, DisplayName: "重复覆盖用户", PasswordHash: "hash",
		Roles: []domain.Role{domain.RoleRegistrar}, Status: domain.StatusActive,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}, false, nil, true, []domain.PermissionOverride{
		{PermissionCode: "charge:refund", Effect: domain.PermissionDeny},
		{PermissionCode: "charge:refund", Effect: domain.PermissionAllow},
	}, domain.AuditEvent{ID: "duplicate-bundle-memory-audit", ActorID: seed.DemoUserID, Action: "user.create", Outcome: domain.OutcomeSuccess})
	if err == nil || err.Error() != "duplicate permission override" {
		t.Fatalf("duplicate bundle error = %v, want duplicate permission override", err)
	}
	if _, findErr := s.FindByID(ctx, userID); !errors.Is(findErr, port.ErrNotFound) {
		t.Fatalf("duplicate bundle created user: %v", findErr)
	}
}
