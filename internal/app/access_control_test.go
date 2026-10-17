package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/cznUSER/B-HIS/internal/store/memory"
)

func newAccessSvc(t *testing.T) (*Service, *memory.Store) {
	t.Helper()
	store := memory.New()
	return &Service{
		Users: store, Master: store, Audit: store,
		Now: func() time.Time { return time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC) },
	}, store
}

func TestAccessControlServiceRoleAndOverrideLifecycle(t *testing.T) {
	svc, _ := newAccessSvc(t)
	ctx := context.Background()
	claims := systemClaims()

	roles, err := svc.ListAccessRoles(ctx, claims)
	if err != nil || len(roles.Items) < 5 {
		t.Fatalf("seed roles: %v %#v", err, roles)
	}
	created, err := svc.CreateAccessRole(ctx, claims, CreateAccessRoleInput{
		Code: "NURSE", Name: "护士", PermissionCodes: []string{"patient:create"},
	}, "req-role", "127.0.0.1")
	if err != nil || created.Code != "NURSE" {
		t.Fatalf("create role: %v %#v", err, created)
	}
	updated, err := svc.UpdateAccessRole(ctx, claims, "NURSE", PatchAccessRoleInput{
		PermissionCodes: &[]string{"patient:create", "encounter:create"},
	}, "req-role", "127.0.0.1")
	if err != nil || len(updated.PermissionCodes) != 2 {
		t.Fatalf("update role: %v %#v", err, updated)
	}

	userID := claims.Subject
	input := ReplacePermissionOverridesInput{Overrides: []PermissionOverrideInput{{PermissionCode: "charge:refund", Effect: domain.PermissionDeny}}}
	list, err := svc.ReplaceUserPermissionOverrides(ctx, claims, userID, input, "req-override", "127.0.0.1")
	if err != nil || len(list.Items) != 1 || list.Items[0].Effect != domain.PermissionDeny {
		t.Fatalf("replace override: %v %#v", err, list)
	}
	list, err = svc.ListUserPermissionOverrides(ctx, claims, userID)
	if err != nil || len(list.Items) != 1 {
		t.Fatalf("list override: %v %#v", err, list)
	}
}

func TestUserAdminCannotGrantAdminWildcardOverride(t *testing.T) {
	ctx := context.Background()
	for _, target := range []string{"actor_admin:user", "target_admin:user"} {
		t.Run(target, func(t *testing.T) {
			svc, store, claims, otherID := newAccessBoundaryActor(t, domain.PermAdminUser)
			targetID := otherID
			if target == claims.Subject {
				targetID = claims.Subject
			}
			_, err := svc.ReplaceUserPermissionOverrides(ctx, claims, targetID, ReplacePermissionOverridesInput{
				Overrides: []PermissionOverrideInput{{PermissionCode: domain.PermAdminAll, Effect: domain.PermissionAllow}},
			}, "req-wildcard-grant", "127.0.0.1")
			var appErr *Error
			if !errors.As(err, &appErr) || appErr.Status != 403 || appErr.Code != CodeForbidden {
				t.Fatalf("admin:user must not grant admin:* ALLOW, got %v", err)
			}
			rows, listErr := store.ListUserPermissionOverrides(ctx, targetID)
			if listErr != nil || len(rows) != 0 {
				t.Fatalf("wildcard override was persisted: err=%v rows=%+v", listErr, rows)
			}
			audits := store.Audits()
			if len(audits) == 0 {
				t.Fatal("failed wildcard grant must be audited")
			}
			audit := audits[len(audits)-1]
			if audit.Action != "user.permission_override" || audit.Outcome != domain.OutcomeFail || audit.ErrorCode != CodeForbidden {
				t.Fatalf("unexpected failure audit: %+v", audit)
			}
		})
	}
}

func TestUserAdminCannotAllowAnyAdminPermissionOverride(t *testing.T) {
	ctx := context.Background()
	svc, store, claims, _ := newAccessBoundaryActor(t, domain.PermAdminUser)
	_, err := svc.ReplaceUserPermissionOverrides(ctx, claims, claims.Subject, ReplacePermissionOverridesInput{
		Overrides: []PermissionOverrideInput{{PermissionCode: domain.PermAdminRole, Effect: domain.PermissionAllow}},
	}, "req-admin-role-self-grant", "127.0.0.1")
	assertForbiddenAudit(t, err, store.Audits(), "user.permission_override")
	rows, listErr := store.ListUserPermissionOverrides(ctx, claims.Subject)
	if listErr != nil || len(rows) != 0 {
		t.Fatalf("admin permission self-grant was persisted: err=%v rows=%+v", listErr, rows)
	}
}

func TestUserAdminKeepsOrdinaryPermissionOverrides(t *testing.T) {
	ctx := context.Background()
	svc, _, claims, targetID := newAccessBoundaryActor(t, domain.PermAdminUser)
	result, err := svc.ReplaceUserPermissionOverrides(ctx, claims, targetID, ReplacePermissionOverridesInput{Overrides: []PermissionOverrideInput{
		{PermissionCode: domain.PermChargeRefund, Effect: domain.PermissionAllow},
		{PermissionCode: domain.PermPatientCreate, Effect: domain.PermissionDeny},
	}}, "req-ordinary-overrides", "127.0.0.1")
	if err != nil || len(result.Items) != 2 {
		t.Fatalf("ordinary ALLOW/DENY overrides should remain available: err=%v result=%+v", err, result)
	}
}

func TestUserAdminCannotAssignSystemRole(t *testing.T) {
	ctx := context.Background()
	t.Run("create", func(t *testing.T) {
		svc, store, claims, _ := newAccessBoundaryActor(t, domain.PermAdminUser)
		_, err := svc.CreateUser(ctx, claims, CreateUserInput{Username: "blocked-system-create", DisplayName: "普通账号", Roles: []string{string(domain.RoleSystem)}}, "req-create-system", "127.0.0.1")
		assertForbiddenAudit(t, err, store.Audits(), domain.ActionUserCreate)
		users, listErr := store.ListUsers(ctx, port.UserQuery{Q: "blocked-system-create"})
		if listErr != nil || len(users) != 0 {
			t.Fatalf("SYSTEM role user was created: err=%v users=%+v", listErr, users)
		}
	})
	t.Run("patch", func(t *testing.T) {
		svc, store, claims, targetID := newAccessBoundaryActor(t, domain.PermAdminUser)
		_, err := svc.PatchUser(ctx, claims, targetID, PatchUserInput{Roles: []string{string(domain.RoleSystem)}}, "req-patch-system", "127.0.0.1")
		assertForbiddenAudit(t, err, store.Audits(), domain.ActionUserUpdate)
		unchanged, findErr := store.FindByID(ctx, targetID)
		if findErr != nil || unchanged.HasSystemRole() {
			t.Fatalf("SYSTEM role was assigned: err=%v user=%+v", findErr, unchanged)
		}
	})
}

func TestSystemAdministratorCanAssignSystemRoleAndWildcardOverride(t *testing.T) {
	ctx := context.Background()
	svc, store := newAccessSvc(t)
	claims := systemClaims()
	created, err := svc.CreateUser(ctx, claims, CreateUserInput{Username: "system-created", DisplayName: "系统账号", Roles: []string{string(domain.RoleSystem)}}, "req-create-system", "127.0.0.1")
	if err != nil || len(created.Roles) != 1 || created.Roles[0] != string(domain.RoleSystem) {
		t.Fatalf("SYSTEM administrator should still assign SYSTEM role: err=%v user=%+v", err, created)
	}
	target := domain.User{ID: "wildcard-target", Username: "wildcard-target", DisplayName: "目标", Status: domain.StatusActive}
	store.Add(target)
	result, err := svc.ReplaceUserPermissionOverrides(ctx, claims, target.ID, ReplacePermissionOverridesInput{Overrides: []PermissionOverrideInput{
		{PermissionCode: domain.PermAdminAll, Effect: domain.PermissionAllow},
	}}, "req-system-wildcard", "127.0.0.1")
	if err != nil || len(result.Items) != 1 || result.Items[0].PermissionCode != domain.PermAdminAll {
		t.Fatalf("SYSTEM administrator should retain wildcard override operation: err=%v overrides=%+v", err, result)
	}
}

func TestNonSystemAdminCannotCreateOrUpdateAdminPermissionRole(t *testing.T) {
	ctx := context.Background()
	t.Run("create", func(t *testing.T) {
		svc, store, claims, _ := newAccessBoundaryActor(t, domain.PermAdminRole)
		_, err := svc.CreateAccessRole(ctx, claims, CreateAccessRoleInput{Code: "ADMIN_GRANT", Name: "高权限角色", PermissionCodes: []string{domain.PermAdminAll}}, "req-create-admin-role", "127.0.0.1")
		assertForbiddenAudit(t, err, store.Audits(), "role.create")
		roles, listErr := store.ListAccessRoles(ctx)
		if listErr != nil {
			t.Fatal(listErr)
		}
		for _, role := range roles {
			if role.Code == "ADMIN_GRANT" {
				t.Fatal("admin permission role was created")
			}
		}
	})
	t.Run("update assigned role", func(t *testing.T) {
		svc, store := newAccessSvc(t)
		role := domain.AccessRole{Code: "SELF_ADMIN_ROLE", Name: "用户管理员", PermissionCodes: []string{domain.PermAdminUser, domain.PermAdminRole}, Status: domain.StatusActive}
		if err := store.CreateAccessRole(ctx, role); err != nil {
			t.Fatal(err)
		}
		actor := domain.User{ID: "self-role-actor", Username: "self-role-actor", Roles: []domain.Role{domain.Role(role.Code)}, Status: domain.StatusActive}
		store.Add(actor)
		claims := &auth.Claims{Username: actor.Username}
		claims.Subject = actor.ID
		permissions := []string{domain.PermAdminUser, domain.PermAdminRole, domain.PermAdminAll}
		_, err := svc.UpdateAccessRole(ctx, claims, role.Code, PatchAccessRoleInput{PermissionCodes: &permissions}, "req-update-admin-role", "127.0.0.1")
		assertForbiddenAudit(t, err, store.Audits(), "role.update")
		rows, listErr := store.ListAccessRoles(ctx)
		if listErr != nil {
			t.Fatal(listErr)
		}
		for _, row := range rows {
			if row.Code == role.Code && hasString(row.PermissionCodes, domain.PermAdminAll) {
				t.Fatal("assigned role gained admin wildcard permission")
			}
		}
	})
}

func TestUserAdminCannotAssignExistingAdminPermissionRole(t *testing.T) {
	ctx := context.Background()
	t.Run("create user", func(t *testing.T) {
		svc, store, claims, _ := newAccessBoundaryActor(t, domain.PermAdminUser)
		if err := store.CreateAccessRole(ctx, domain.AccessRole{Code: "EXISTING_ADMIN_ROLE", Name: "高权限自定义角色", PermissionCodes: []string{domain.PermAdminPermission}, Status: domain.StatusActive}); err != nil {
			t.Fatal(err)
		}
		_, err := svc.CreateUser(ctx, claims, CreateUserInput{Username: "blocked-high-role", DisplayName: "受限账号", Roles: []string{"EXISTING_ADMIN_ROLE"}}, "req-create-high-role", "127.0.0.1")
		assertForbiddenAudit(t, err, store.Audits(), domain.ActionUserCreate)
		users, listErr := store.ListUsers(ctx, port.UserQuery{Q: "blocked-high-role"})
		if listErr != nil || len(users) != 0 {
			t.Fatalf("high permission role user was created: err=%v users=%+v", listErr, users)
		}
	})
	t.Run("update user", func(t *testing.T) {
		svc, store, claims, targetID := newAccessBoundaryActor(t, domain.PermAdminUser)
		if err := store.CreateAccessRole(ctx, domain.AccessRole{Code: "EXISTING_ADMIN_ROLE", Name: "高权限自定义角色", PermissionCodes: []string{domain.PermAdminPermission}, Status: domain.StatusActive}); err != nil {
			t.Fatal(err)
		}
		_, err := svc.PatchUser(ctx, claims, targetID, PatchUserInput{Roles: []string{"EXISTING_ADMIN_ROLE"}}, "req-patch-high-role", "127.0.0.1")
		assertForbiddenAudit(t, err, store.Audits(), domain.ActionUserUpdate)
		unchanged, findErr := store.FindByID(ctx, targetID)
		if findErr != nil || hasString(unchanged.RoleCodes(), "EXISTING_ADMIN_ROLE") {
			t.Fatalf("high permission role was assigned: err=%v user=%+v", findErr, unchanged)
		}
	})
}

func assertForbiddenAudit(t *testing.T, err error, audits []domain.AuditEvent, action string) {
	t.Helper()
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Status != 403 || appErr.Code != CodeForbidden {
		t.Fatalf("expected stable forbidden error, got %v", err)
	}
	if len(audits) == 0 {
		t.Fatal("forbidden authorization change must be audited")
	}
	audit := audits[len(audits)-1]
	if audit.Action != action || audit.Outcome != domain.OutcomeFail || audit.ErrorCode != CodeForbidden {
		t.Fatalf("unexpected failed authorization audit: %+v", audit)
	}
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestAccessControlServiceRequiresAdmin(t *testing.T) {
	svc, store := newAccessSvc(t)
	claims := systemClaims()
	claims.Roles = []string{string(domain.RoleDoctor)}
	store.Add(domain.User{ID: "doctor-access-test", Username: "doctor-access-test", Roles: []domain.Role{domain.RoleDoctor}, Status: domain.StatusActive})
	claims.Subject = "doctor-access-test"
	if _, err := svc.ListAccessModules(context.Background(), claims); err == nil {
		t.Fatal("doctor must not list access modules")
	}
}

func TestRoleProtectedFlagsRequireDeveloper(t *testing.T) {
	ctx := context.Background()
	svc, store := newAccessSvc(t)
	role := domain.AccessRole{Code: "PROTECTED_ROLE", Name: "受保护角色", Status: domain.StatusActive, Hidden: true, System: true}
	if err := store.CreateAccessRole(ctx, role); err != nil {
		t.Fatal(err)
	}
	actor := domain.User{ID: "role-admin", Username: "role-admin", DisplayName: "角色管理员", Status: domain.StatusActive, Roles: []domain.Role{"ROLE_ADMIN"}}
	if err := store.CreateAccessRole(ctx, domain.AccessRole{Code: "ROLE_ADMIN", Name: "角色管理员", Status: domain.StatusActive, PermissionCodes: []string{domain.PermAdminRole}}); err != nil {
		t.Fatal(err)
	}
	store.Add(actor)
	claims := &auth.Claims{RegisteredClaims: jwtSubject(actor.ID), Roles: []string{"ROLE_ADMIN"}}
	if _, err := svc.ListAccessRoles(ctx, claims); err != nil {
		t.Fatal(err)
	}
	name := "不应修改"
	if _, err := svc.UpdateAccessRole(ctx, claims, role.Code, PatchAccessRoleInput{Name: &name}, "req", "127.0.0.1"); err == nil {
		t.Fatal("non-developer must not edit protected role")
	}
	hidden := true
	if _, err := svc.CreateAccessRole(ctx, claims, CreateAccessRoleInput{Code: "HIDDEN_NEW", Name: "隐藏角色", Hidden: &hidden}, "req", "127.0.0.1"); err == nil {
		t.Fatal("non-developer must not create hidden role")
	}
	developer := domain.User{ID: "role-developer", Username: "role-developer", DisplayName: "开发者", Status: domain.StatusActive, Developer: true}
	store.Add(developer)
	devClaims := &auth.Claims{RegisteredClaims: jwtSubject(developer.ID)}
	if _, err := svc.UpdateAccessRole(ctx, devClaims, role.Code, PatchAccessRoleInput{Name: &name}, "req", "127.0.0.1"); err != nil {
		t.Fatalf("developer should edit protected role: %v", err)
	}
}

func TestAccessPermissionCreateAndSoftDisable(t *testing.T) {
	svc, store := newAccessSvc(t)
	ctx := context.Background()
	claims := systemClaims()
	created, err := svc.CreateAccessPermission(ctx, claims, CreateAccessPermissionInput{
		Code: "patient:archive", Name: "患者归档", ModuleCode: "outpatient", CategoryID: "patient",
		ModeCode: "password", Actions: []string{"N", "M"}, Status: domain.StatusActive,
	}, "req-perm", "127.0.0.1")
	if err != nil || created.Code != "patient:archive" || len(created.Actions) != 2 {
		t.Fatalf("create permission: %v %#v", err, created)
	}
	status := domain.StatusInactive
	updated, err := svc.UpdateAccessPermission(ctx, claims, "patient:archive", PatchAccessPermissionInput{Status: &status}, "req-perm", "127.0.0.1")
	if err != nil || updated.Status != domain.StatusInactive {
		t.Fatalf("soft-disable permission: %v %#v", err, updated)
	}
	rows, err := store.ListAccessPermissions(ctx, "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.Code == "patient:archive" && row.Status == domain.StatusInactive {
			return
		}
	}
	t.Fatal("updated permission was not persisted")
}

func TestPermissionHiddenFlagRequiresDeveloper(t *testing.T) {
	svc, store := newAccessSvc(t)
	ctx := context.Background()
	if err := store.CreateAccessRole(ctx, domain.AccessRole{Code: "PERM_ADMIN", Name: "权限管理员", Status: domain.StatusActive, PermissionCodes: []string{domain.PermAdminPermission}}); err != nil {
		t.Fatal(err)
	}
	store.Add(domain.User{ID: "permission-admin", Username: "permission-admin", DisplayName: "权限管理员", Status: domain.StatusActive, Roles: []domain.Role{"PERM_ADMIN"}})
	claims := &auth.Claims{RegisteredClaims: jwtSubject("permission-admin"), Roles: []string{"PERM_ADMIN"}}
	hidden := true
	if _, err := svc.CreateAccessPermission(ctx, claims, CreateAccessPermissionInput{Code: "hidden:permission", Name: "隐藏权限", ModuleCode: "outpatient", CategoryID: "patient", ModeCode: "password", Hidden: true}, "req", "127.0.0.1"); err == nil {
		t.Fatal("non-developer must not create hidden permission")
	}
	if err := store.CreateAccessPermission(ctx, domain.AccessPermission{Code: "visible:permission", Name: "可见权限", ModuleCode: "outpatient", CategoryID: "patient", ModeCode: "password", Status: domain.StatusActive}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateAccessPermission(ctx, claims, "visible:permission", PatchAccessPermissionInput{Hidden: &hidden}, "req", "127.0.0.1"); err == nil {
		t.Fatal("non-developer must not hide permission")
	}
}

func TestCreateUserAcceptsCustomAccessRole(t *testing.T) {
	svc, _ := newAccessSvc(t)
	ctx := context.Background()
	claims := systemClaims()
	if _, err := svc.CreateAccessRole(ctx, claims, CreateAccessRoleInput{
		Code: "NURSE", Name: "护士", PermissionCodes: []string{"patient:create"},
	}, "req-role", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	created, err := svc.CreateUser(ctx, claims, CreateUserInput{
		Username: "nurse-custom", DisplayName: "护士", Roles: []string{"nurse"},
	}, "req-user", "127.0.0.1")
	if err != nil || len(created.Roles) != 1 || created.Roles[0] != "NURSE" {
		t.Fatalf("create custom role user: %v %#v", err, created)
	}
}

func TestUserAdminPasswordCanBeSetWithoutReturningHash(t *testing.T) {
	svc, store := newAccessSvc(t)
	ctx := context.Background()
	claims := systemClaims()
	created, err := svc.CreateUser(ctx, claims, CreateUserInput{Username: "password-user", DisplayName: "密码用户", Password: "Stronger!9"}, "req-user", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	row, err := store.FindByID(ctx, created.UserID)
	if err != nil || !auth.CheckPassword(row.PasswordHash, "Stronger!9") {
		t.Fatalf("custom password was not hashed: %v", err)
	}
	newPassword := "Another!9"
	updated, err := svc.PatchUser(ctx, claims, created.UserID, PatchUserInput{Password: &newPassword}, "req-user", "127.0.0.1")
	if err != nil {
		t.Fatalf("password patch failed: %v %#v", err, updated)
	}
	row, err = store.FindByID(ctx, created.UserID)
	if err != nil || !auth.CheckPassword(row.PasswordHash, newPassword) {
		t.Fatalf("patched password was not hashed: %v", err)
	}
}

func TestPatchUserCannotRemoveLastSystemAdministrator(t *testing.T) {
	svc, store := newAccessSvc(t)
	claims := systemClaims()
	admin, err := store.FindByID(context.Background(), claims.Subject)
	if err != nil {
		t.Fatal(err)
	}
	if !admin.HasSystemRole() {
		t.Fatal("fixture must be the system administrator")
	}
	status := domain.StatusDisabled
	_, err = svc.PatchUser(context.Background(), claims, admin.ID, PatchUserInput{Status: &status}, "req-admin-lockout", "127.0.0.1")
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Code != CodeLastSystemAdmin {
		t.Fatalf("expected LAST_SYSTEM_ADMIN, got %v", err)
	}
}

func TestPatchUserConcurrentSystemAdminDemotionKeepsOneAdministrator(t *testing.T) {
	svc, store := newAccessSvc(t)
	ctx := context.Background()
	first, err := store.FindByID(ctx, systemClaims().Subject)
	if err != nil {
		t.Fatal(err)
	}
	second := domain.User{ID: "system-concurrent-2", Username: "system-concurrent-2", DisplayName: "并发管理员二", Status: domain.StatusActive, Roles: []domain.Role{domain.RoleSystem}}
	store.Add(second)
	claims := systemClaims()
	claims.Subject = first.ID
	status := domain.StatusDisabled
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, userID := range []string{first.ID, second.ID} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			_, err := svc.PatchUser(ctx, claims, id, PatchUserInput{Status: &status}, "req-concurrent-"+id, "127.0.0.1")
			errs <- err
		}(userID)
	}
	wg.Wait()
	close(errs)
	successes := 0
	lastAdmin := 0
	for err := range errs {
		if err == nil {
			successes++
			continue
		}
		var appErr *Error
		if errors.As(err, &appErr) && appErr.Code == CodeLastSystemAdmin {
			lastAdmin++
			continue
		}
		t.Fatalf("unexpected concurrent demotion error: %v", err)
	}
	if successes != 1 || lastAdmin != 1 {
		t.Fatalf("concurrent demotion results = successes %d last-admin %d, want 1/1", successes, lastAdmin)
	}
	count, err := store.CountSystemAdmins(ctx)
	if err != nil || count != 1 {
		t.Fatalf("active SYSTEM administrators = %d, err=%v, want 1", count, err)
	}
	audits := store.Audits()
	seenSuccess := 0
	for _, event := range audits {
		if event.Action == domain.ActionUserDeactivate && event.Outcome == domain.OutcomeSuccess && strings.HasPrefix(event.RequestID, "req-concurrent-") {
			seenSuccess++
		}
	}
	if seenSuccess != 1 {
		t.Fatalf("successful demotion audits = %d, want 1", seenSuccess)
	}
}

type failingAtomicMaster struct{ port.MasterStore }

func (f failingAtomicMaster) UpdateUserWithAudit(context.Context, domain.User, bool, domain.AuditEvent) (*domain.User, error) {
	return nil, port.ErrAuditUnavailable
}

type failingCreateAtomicMaster struct {
	port.MasterStore
	called *bool
}

func (f failingCreateAtomicMaster) CreateUserWithAudit(context.Context, domain.User, domain.AuditEvent) (*domain.User, error) {
	*f.called = true
	return nil, port.ErrAuditUnavailable
}

func TestCreateUserAuditFailureDoesNotMutateUser(t *testing.T) {
	svc, store := newAccessSvc(t)
	called := false
	svc.Master = failingCreateAtomicMaster{MasterStore: store, called: &called}
	claims := systemClaims()
	username := "atomic-create-audit-failure"
	if _, err := svc.CreateUser(context.Background(), claims, CreateUserInput{Username: username, DisplayName: "审计失败创建"}, "req-create-audit-failure", "127.0.0.1"); err == nil {
		t.Fatal("create should fail when audit transaction is unavailable")
	}
	if !called {
		t.Fatal("atomic create capability was not used")
	}
	users, err := store.ListUsers(context.Background(), port.UserQuery{Q: username})
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 0 {
		t.Fatalf("user changed despite audit failure: %+v", users)
	}
}

func TestPatchUserAuditFailureDoesNotReportSuccess(t *testing.T) {
	svc, store := newAccessSvc(t)
	svc.Master = failingAtomicMaster{MasterStore: store}
	claims := systemClaims()
	before, err := store.FindByID(context.Background(), claims.Subject)
	if err != nil {
		t.Fatal(err)
	}
	display := "审计故障后不应提交"
	_, err = svc.PatchUser(context.Background(), claims, claims.Subject, PatchUserInput{DisplayName: &display}, "req-audit-failure", "127.0.0.1")
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Code != CodeAuditUnavailable || appErr.Status != 503 {
		t.Fatalf("audit failure = %v, want USER_AUDIT_UNAVAILABLE/503", err)
	}
	after, err := store.FindByID(context.Background(), claims.Subject)
	if err != nil || after.DisplayName != before.DisplayName {
		t.Fatalf("user changed despite audit failure: before=%q after=%q err=%v", before.DisplayName, after.DisplayName, err)
	}
}

type failingResetAtomicMaster struct {
	port.MasterStore
	called *bool
}

func (f failingResetAtomicMaster) ResetPasswordWithAudit(context.Context, string, string, string, domain.AuditEvent) error {
	*f.called = true
	return port.ErrAuditUnavailable
}

func TestResetUserPasswordAuditFailureDoesNotMutateUser(t *testing.T) {
	svc, store := newAccessSvc(t)
	called := false
	svc.Master = failingResetAtomicMaster{MasterStore: store, called: &called}
	claims := systemClaims()
	before, err := store.FindByID(context.Background(), claims.Subject)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ResetUserPassword(context.Background(), claims, claims.Subject, "req-reset-audit-failure", "127.0.0.1"); err == nil {
		t.Fatal("reset should fail when audit transaction is unavailable")
	}
	if !called {
		t.Fatal("atomic reset capability was not used")
	}
	after, err := store.FindByID(context.Background(), claims.Subject)
	if err != nil {
		t.Fatal(err)
	}
	if after.PasswordHash != before.PasswordHash || after.AuthVersion != before.AuthVersion {
		t.Fatalf("user changed despite audit failure: before=%+v after=%+v", before, after)
	}
}

func TestSystemRoleCannotPromoteItselfToDeveloper(t *testing.T) {
	svc, store := newAccessSvc(t)
	admin := domain.User{ID: "system-only", Username: "system-only", DisplayName: "系统角色", Status: domain.StatusActive, Roles: []domain.Role{domain.RoleSystem}}
	store.Add(admin)
	claims := systemClaims()
	claims.Subject = admin.ID
	developer := true
	_, err := svc.PatchUser(context.Background(), claims, admin.ID, PatchUserInput{Developer: &developer}, "req-self-promote", "127.0.0.1")
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Status != 403 {
		t.Fatalf("SYSTEM role must not grant developer bypass: %v", err)
	}
	unchanged, err := store.FindByID(context.Background(), admin.ID)
	if err != nil || unchanged.Developer {
		t.Fatalf("system account was promoted: err=%v user=%+v", err, unchanged)
	}
}

func TestSuperCannotChangeDeveloperFlag(t *testing.T) {
	svc, store := newAccessSvc(t)
	actor := domain.User{ID: "super-actor", Username: "super-actor", DisplayName: "超级", Status: domain.StatusActive, Super: true}
	store.Add(actor)
	claims := systemClaims()
	claims.Subject = actor.ID
	developer := true
	_, err := svc.PatchUser(context.Background(), claims, actor.ID, PatchUserInput{Developer: &developer}, "req-super-promote", "127.0.0.1")
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Status != 403 {
		t.Fatalf("super must not modify developer flag: %v", err)
	}
	unchanged, err := store.FindByID(context.Background(), actor.ID)
	if err != nil || unchanged.Developer {
		t.Fatalf("super account was promoted: err=%v user=%+v", err, unchanged)
	}
}

func TestUserListHidesHiddenUsersUnlessActorIsDeveloper(t *testing.T) {
	svc, store := newAccessSvc(t)
	store.Add(domain.User{ID: "hidden-user", Username: "hidden-user", DisplayName: "隐藏账号", Status: domain.StatusActive, Hidden: true})
	admin := domain.User{ID: "system-only", Username: "system-only", DisplayName: "系统角色", Status: domain.StatusActive, Roles: []domain.Role{domain.RoleSystem}}
	store.Add(admin)

	regularClaims := systemClaims()
	regularClaims.Subject = admin.ID
	regular, err := svc.ListUsers(context.Background(), regularClaims, "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range regular.Items {
		if row.UserID == "hidden-user" {
			t.Fatal("hidden account must not be visible to SYSTEM role")
		}
	}

	developer := domain.User{ID: "developer-actor", Username: "developer-actor", DisplayName: "开发者", Status: domain.StatusActive, Developer: true}
	store.Add(developer)
	claims := systemClaims()
	claims.Subject = developer.ID
	visible, err := svc.ListUsers(context.Background(), claims, "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range visible.Items {
		if row.UserID == "hidden-user" {
			return
		}
	}
	t.Fatal("developer must be able to see hidden accounts")
}

func TestAccessCatalogCreateUpdateAndModuleInt64RoundTrip(t *testing.T) {
	svc, store := newAccessSvc(t)
	ctx, claims := context.Background(), systemClaims()
	module, err := svc.CreateAccessModule(ctx, claims, CreateAccessModuleInput{
		Code: "clinical-meta", Name: "临床配置", DepartmentFunctions: "9007199254740993", HomePageURL: "/clinical",
		NeedSpecifyDepartment: true, SecondPermissionID: 2,
	}, "req-module", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if module.DepartmentFunctions != "9007199254740993" {
		t.Fatalf("module bigint must round-trip exactly, got %q", module.DepartmentFunctions)
	}
	home := "/clinical/home"
	module, err = svc.UpdateAccessModule(ctx, claims, "clinical-meta", PatchAccessModuleInput{HomePageURL: &home}, "req-module", "127.0.0.1")
	if err != nil || module.HomePageURL != home {
		t.Fatalf("patch module: %v %#v", err, module)
	}

	category, err := svc.CreatePermissionCategory(ctx, claims, CreatePermissionCategoryInput{ID: "clinical", Name: "临床"}, "req-category", "127.0.0.1")
	if err != nil || category.ID != "clinical" {
		t.Fatalf("create category: %v %#v", err, category)
	}
	disabled := domain.StatusInactive
	category, err = svc.UpdatePermissionCategory(ctx, claims, "clinical", PatchPermissionCategoryInput{Status: &disabled}, "req-category", "127.0.0.1")
	if err != nil || category.Status != domain.StatusInactive {
		t.Fatalf("disable category: %v %#v", err, category)
	}

	mode, err := svc.CreatePermissionMode(ctx, claims, CreatePermissionModeInput{Code: "card-pin", Name: "卡+PIN"}, "req-mode", "127.0.0.1")
	if err != nil || mode.Status != domain.StatusActive {
		t.Fatalf("create mode: %v %#v", err, mode)
	}
	mode, err = svc.UpdatePermissionMode(ctx, claims, "card-pin", PatchPermissionModeInput{Status: &disabled}, "req-mode", "127.0.0.1")
	if err != nil || mode.Status != domain.StatusInactive {
		t.Fatalf("disable mode: %v %#v", err, mode)
	}
	modules, err := store.ListAccessModules(ctx)
	if err != nil || len(modules) == 0 {
		t.Fatalf("catalog persistence missing: %v", err)
	}
}

func TestAccessRoleAndPermissionLegacyMetadataRoundTrip(t *testing.T) {
	svc, store := newAccessSvc(t)
	ctx, claims := context.Background(), systemClaims()
	permission, err := svc.CreateAccessPermission(ctx, claims, CreateAccessPermissionInput{
		Code: "patient:legacy", Name: "遗留权限", ModuleCode: "outpatient", CategoryID: "patient", ModeCode: "password",
		DBRoleName: "HERP_PATIENT", NeedSecondPermission: true,
	}, "req-permission", "127.0.0.1")
	if err != nil || permission.DBRoleName != "HERP_PATIENT" || !permission.NeedSecondPermission {
		t.Fatalf("permission metadata: %v %#v", err, permission)
	}
	role, err := svc.CreateAccessRole(ctx, claims, CreateAccessRoleInput{Code: "WARD", Name: "病区角色", PermissionCodes: []string{permission.Code}, OrderSort: 31, StandName: "WARD_STANDARD"}, "req-role", "127.0.0.1")
	if err != nil || role.OrderSort != 31 || role.StandName != "WARD_STANDARD" {
		t.Fatalf("role metadata: %v %#v", err, role)
	}
	newOrder := 42
	newStandard := "WARD_STANDARD_V2"
	role, err = svc.UpdateAccessRole(ctx, claims, "WARD", PatchAccessRoleInput{OrderSort: &newOrder, StandName: &newStandard}, "req-role", "127.0.0.1")
	if err != nil || role.OrderSort != newOrder || role.StandName != newStandard {
		t.Fatalf("patch role metadata: %v %#v", err, role)
	}
	dbRole := "HERP_PATIENT_V2"
	permission, err = svc.UpdateAccessPermission(ctx, claims, "patient:legacy", PatchAccessPermissionInput{DBRoleName: &dbRole}, "req-permission", "127.0.0.1")
	if err != nil || permission.DBRoleName != dbRole {
		t.Fatalf("patch permission metadata: %v %#v", err, permission)
	}
	permissions, err := store.ListAccessPermissions(ctx, "", "")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range permissions {
		if row.Code == permission.Code {
			found = row.DBRoleName == dbRole && row.NeedSecondPermission
		}
	}
	if !found {
		t.Fatal("permission metadata not retained")
	}
}

func TestUserLegacyFieldsCreateAndExplicitNullPatch(t *testing.T) {
	svc, store := newAccessSvc(t)
	ctx, claims := context.Background(), systemClaims()
	star, expand := 4, int32(17)
	user, err := svc.CreateUser(ctx, claims, CreateUserInput{Username: "legacy-user", DisplayName: "字段用户", ServiceStar: &star, ExpandInterface1: &expand}, "req-user", "127.0.0.1")
	if err != nil || user.ServiceStar == nil || *user.ServiceStar != star || user.ExpandInterface1 == nil || *user.ExpandInterface1 != expand || user.InputCodeCompareMode == nil || *user.InputCodeCompareMode != 2 {
		t.Fatalf("user legacy fields create: %v %#v", err, user)
	}
	if _, err := store.FindByID(ctx, user.UserID); err != nil {
		t.Fatal(err)
	}
	var patch PatchUserInput
	if err := json.Unmarshal([]byte(`{"serviceStar":null,"inputCodeCompareMode":null}`), &patch); err != nil {
		t.Fatal(err)
	}
	updated, err := svc.PatchUser(ctx, claims, user.UserID, patch, "req-user", "127.0.0.1")
	if err != nil || updated.ServiceStar != nil || updated.InputCodeCompareMode != nil {
		t.Fatalf("explicit null must clear nullable legacy fields: %v %#v", err, updated)
	}
	invalid := 6
	if _, err := svc.CreateUser(ctx, claims, CreateUserInput{Username: "invalid-star", DisplayName: "invalid", ServiceStar: &invalid}, "req-user", "127.0.0.1"); err == nil {
		t.Fatal("serviceStar outside 1..5 must be rejected")
	}
}

type accessBoundaryOperation struct {
	name string
	run  func(*Service, *auth.Claims, string) error
}

func TestAccessControlManagementPermissionBoundaries(t *testing.T) {
	ctx := context.Background()
	operations := []accessBoundaryOperation{
		{name: "role list", run: func(s *Service, c *auth.Claims, targetID string) error {
			_, err := s.ListAccessRoles(ctx, c)
			return err
		}},
		{name: "role create", run: func(s *Service, c *auth.Claims, _ string) error {
			_, err := s.CreateAccessRole(ctx, c, CreateAccessRoleInput{Code: "BOUNDARY_NEW", Name: "边界角色", PermissionCodes: []string{"patient:create"}}, "req", "127.0.0.1")
			return err
		}},
		{name: "role update", run: func(s *Service, c *auth.Claims, _ string) error {
			name := "边界角色更新"
			_, err := s.UpdateAccessRole(ctx, c, "BOUNDARY_TARGET", PatchAccessRoleInput{Name: &name}, "req", "127.0.0.1")
			return err
		}},
		{name: "module list", run: func(s *Service, c *auth.Claims, _ string) error {
			_, err := s.ListAccessModules(ctx, c)
			return err
		}},
		{name: "module create", run: func(s *Service, c *auth.Claims, _ string) error {
			_, err := s.CreateAccessModule(ctx, c, CreateAccessModuleInput{Code: "boundary-module", Name: "边界模块"}, "req", "127.0.0.1")
			return err
		}},
		{name: "module update", run: func(s *Service, c *auth.Claims, _ string) error {
			name := "门诊更新"
			_, err := s.UpdateAccessModule(ctx, c, "outpatient", PatchAccessModuleInput{Name: &name}, "req", "127.0.0.1")
			return err
		}},
		{name: "category list", run: func(s *Service, c *auth.Claims, _ string) error {
			_, err := s.ListPermissionCategories(ctx, c)
			return err
		}},
		{name: "category create", run: func(s *Service, c *auth.Claims, _ string) error {
			_, err := s.CreatePermissionCategory(ctx, c, CreatePermissionCategoryInput{ID: "boundary", Name: "边界分类"}, "req", "127.0.0.1")
			return err
		}},
		{name: "category update", run: func(s *Service, c *auth.Claims, _ string) error {
			name := "患者更新"
			_, err := s.UpdatePermissionCategory(ctx, c, "patient", PatchPermissionCategoryInput{Name: &name}, "req", "127.0.0.1")
			return err
		}},
		{name: "mode list", run: func(s *Service, c *auth.Claims, _ string) error {
			_, err := s.ListPermissionModes(ctx, c)
			return err
		}},
		{name: "mode create", run: func(s *Service, c *auth.Claims, _ string) error {
			_, err := s.CreatePermissionMode(ctx, c, CreatePermissionModeInput{Code: "boundary", Name: "边界认证"}, "req", "127.0.0.1")
			return err
		}},
		{name: "mode update", run: func(s *Service, c *auth.Claims, _ string) error {
			name := "密码更新"
			_, err := s.UpdatePermissionMode(ctx, c, "password", PatchPermissionModeInput{Name: &name}, "req", "127.0.0.1")
			return err
		}},
		{name: "permission list", run: func(s *Service, c *auth.Claims, _ string) error {
			_, err := s.ListAccessPermissions(ctx, c, "", "")
			return err
		}},
		{name: "permission create", run: func(s *Service, c *auth.Claims, _ string) error {
			_, err := s.CreateAccessPermission(ctx, c, CreateAccessPermissionInput{Code: "patient:boundary", Name: "边界权限", ModuleCode: "outpatient", CategoryID: "patient", ModeCode: "password"}, "req", "127.0.0.1")
			return err
		}},
		{name: "permission update", run: func(s *Service, c *auth.Claims, _ string) error {
			name := "患者创建更新"
			_, err := s.UpdateAccessPermission(ctx, c, "patient:create", PatchAccessPermissionInput{Name: &name}, "req", "127.0.0.1")
			return err
		}},
		{name: "override list", run: func(s *Service, c *auth.Claims, targetID string) error {
			_, err := s.ListUserPermissionOverrides(ctx, c, targetID)
			return err
		}},
		{name: "override replace", run: func(s *Service, c *auth.Claims, targetID string) error {
			_, err := s.ReplaceUserPermissionOverrides(ctx, c, targetID, ReplacePermissionOverridesInput{Overrides: []PermissionOverrideInput{{PermissionCode: "charge:refund", Effect: domain.PermissionDeny}}}, "req", "127.0.0.1")
			return err
		}},
	}

	cases := []struct {
		permission string
		allowed    map[string]bool
	}{
		{permission: "admin:user", allowed: map[string]bool{
			"role list": true, "module list": true, "category list": true, "mode list": true, "permission list": true,
			"override list": true, "override replace": true,
		}},
		{permission: "admin:role", allowed: map[string]bool{
			"role list": true, "role create": true, "role update": true,
			"module list": true, "permission list": true,
		}},
		{permission: "admin:permission", allowed: map[string]bool{
			"module list": true, "module create": true, "module update": true,
			"category list": true, "category create": true, "category update": true,
			"mode list": true, "mode create": true, "mode update": true,
			"permission list": true, "permission create": true, "permission update": true,
		}},
	}

	for _, tc := range cases {
		t.Run(tc.permission, func(t *testing.T) {
			for _, operation := range operations {
				t.Run(operation.name, func(t *testing.T) {
					svc, store, claims, targetID := newAccessBoundaryActor(t, tc.permission)
					if err := store.CreateAccessRole(ctx, domain.AccessRole{Code: "BOUNDARY_TARGET", Name: "目标角色", PermissionCodes: []string{"patient:create"}, Status: domain.StatusActive}); err != nil {
						t.Fatal(err)
					}
					err := operation.run(svc, claims, targetID)
					if tc.allowed[operation.name] {
						if err != nil {
							t.Fatalf("%s should be allowed with %s: %v", operation.name, tc.permission, err)
						}
						return
					}
					var appErr *Error
					if !errors.As(err, &appErr) || appErr.Status != 403 {
						t.Fatalf("%s should be forbidden with %s, got %v", operation.name, tc.permission, err)
					}
				})
			}
		})
	}
}

func TestAccessControlAdminBypassesManagementPermissionBoundaries(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name       string
		permission string
		super      bool
		developer  bool
	}{
		{name: "admin wildcard", permission: "admin:*"},
		{name: "super", super: true},
		{name: "developer", developer: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, store := newAccessSvc(t)
			user := domain.User{ID: "bypass-actor", Username: "bypass-actor", Status: domain.StatusActive, Super: tc.super, Developer: tc.developer}
			if tc.permission != "" {
				if err := store.CreateAccessRole(ctx, domain.AccessRole{Code: "BYPASS_ROLE", Name: "绕过角色", PermissionCodes: []string{tc.permission}, Status: domain.StatusActive}); err != nil {
					t.Fatal(err)
				}
				user.Roles = []domain.Role{"BYPASS_ROLE"}
			}
			store.Add(user)
			store.Add(domain.User{ID: "bypass-target", Username: "bypass-target", Status: domain.StatusActive})
			claims := &auth.Claims{Username: user.Username}
			claims.Subject = user.ID
			if _, err := svc.CreateAccessRole(ctx, claims, CreateAccessRoleInput{Code: "BYPASS_NEW", Name: "绕过角色", PermissionCodes: []string{"patient:create"}}, "req", "127.0.0.1"); err != nil {
				t.Fatalf("role write should be allowed: %v", err)
			}
			if _, err := svc.CreateAccessModule(ctx, claims, CreateAccessModuleInput{Code: "bypass-module", Name: "绕过模块"}, "req", "127.0.0.1"); err != nil {
				t.Fatalf("catalog write should be allowed: %v", err)
			}
			if _, err := svc.ReplaceUserPermissionOverrides(ctx, claims, "bypass-target", ReplacePermissionOverridesInput{}, "req", "127.0.0.1"); err != nil {
				t.Fatalf("user override write should be allowed: %v", err)
			}
		})
	}
}

func newAccessBoundaryActor(t *testing.T, permission string) (*Service, *memory.Store, *auth.Claims, string) {
	t.Helper()
	ctx := context.Background()
	svc, store := newAccessSvc(t)
	roleCode := "BOUNDARY_" + permission
	userID := "actor_" + permission
	if err := store.CreateAccessRole(ctx, domain.AccessRole{Code: roleCode, Name: roleCode, PermissionCodes: []string{permission}, Status: domain.StatusActive}); err != nil {
		t.Fatal(err)
	}
	store.Add(domain.User{ID: userID, Username: userID, Roles: []domain.Role{domain.Role(roleCode)}, Status: domain.StatusActive})
	store.Add(domain.User{ID: "target_" + permission, Username: "target_" + permission, Status: domain.StatusActive})
	claims := &auth.Claims{Username: userID, Roles: []string{roleCode}}
	claims.Subject = userID
	return svc, store, claims, "target_" + permission
}
