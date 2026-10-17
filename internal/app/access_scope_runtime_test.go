package app

import (
	"context"
	"errors"
	"testing"

	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/store/memory"
	"github.com/golang-jwt/jwt/v5"
)

func TestCreateRegisterEnforcesRoleDepartmentAndUserDenyScopes(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	svc := &Service{Users: store, Master: store, Audit: store, Patients: store, Registers: store}

	patient, err := svc.CreatePatient(ctx, systemClaims(), validInput(), "req-scope-patient", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}

	if err := store.CreateAccessRole(ctx, domain.AccessRole{
		Code: "DEPT1_REG", Name: "内科挂号", ModuleCode: "outpatient", DepartmentID: "dept-1",
		PermissionCodes: []string{domain.PermEncounterCreate}, Status: domain.StatusActive,
	}); err != nil {
		t.Fatal(err)
	}
	deptUser := addScopeTestUser(store, "scope-dept-user", "DEPT1_REG")
	assertRegisterDenied(t, expectRegisterPermission(svc, claimsForUser(deptUser.ID), patient.PatientID, "dept-2"))
	if err := expectRegisterPermission(svc, claimsForUser(deptUser.ID), patient.PatientID, "dept-1"); err != nil {
		t.Fatalf("department-scoped role should authorize its department: %v", err)
	}

	if err := store.CreateAccessRole(ctx, domain.AccessRole{
		Code: "REG_ALL", Name: "挂号", ModuleCode: "outpatient",
		PermissionCodes: []string{domain.PermEncounterCreate}, Status: domain.StatusActive,
	}); err != nil {
		t.Fatal(err)
	}
	denyUser := addScopeTestUser(store, "scope-deny-user", "REG_ALL")
	if err := store.ReplaceUserPermissionOverrides(ctx, denyUser.ID, []domain.PermissionOverride{{
		PermissionCode: domain.PermEncounterCreate, ModuleCode: "outpatient", DepartmentID: "dept-2", Effect: domain.PermissionDeny,
	}}); err != nil {
		t.Fatal(err)
	}
	assertRegisterDenied(t, expectRegisterPermission(svc, claimsForUser(denyUser.ID), patient.PatientID, "dept-2"))
	if err := expectRegisterPermission(svc, claimsForUser(denyUser.ID), patient.PatientID, "dept-1"); err != nil {
		t.Fatalf("department-scoped DENY must not block other departments: %v", err)
	}
}

func TestSuperAndDeveloperBypassPermissionScopes(t *testing.T) {
	for _, user := range []domain.User{
		{Super: true, AccessControlLoaded: true},
		{Developer: true, AccessControlLoaded: true},
	} {
		if !hasPermScope(&user, domain.PermChargeRefund, "outpatient", "dept-2") {
			t.Fatalf("protected account flags should bypass department permission checks: %#v", user)
		}
	}
}

func TestInactiveAccessModuleRevokesPermissionsUntilReenabled(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	svc := &Service{Users: store, Master: store, Audit: store, Patients: store, Registers: store}
	if err := store.CreateAccessRole(ctx, domain.AccessRole{
		Code: "OUTPATIENT_ACCESS", Name: "门诊访问", ModuleCode: "outpatient",
		PermissionCodes: []string{domain.PermEncounterCreate}, Status: domain.StatusActive,
	}); err != nil {
		t.Fatal(err)
	}
	user := addScopeTestUser(store, "module-state-user", "OUTPATIENT_ACCESS")
	claims := claimsForUser(user.ID)

	input := validInput()
	patient, err := svc.CreatePatient(ctx, systemClaims(), input, "req-module-patient-1", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	status := domain.StatusInactive
	if _, err := svc.UpdateAccessModule(ctx, systemClaims(), "outpatient", PatchAccessModuleInput{Status: &status}, "req-module-off", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	assertRegisterDenied(t, expectRegisterPermission(svc, claims, patient.PatientID, "dept-1"))

	status = domain.StatusActive
	if _, err := svc.UpdateAccessModule(ctx, systemClaims(), "outpatient", PatchAccessModuleInput{Status: &status}, "req-module-on", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	input.IDNumber = "110101199001159999"
	input.Phone = "13800000001"
	patient, err = svc.CreatePatient(ctx, systemClaims(), input, "req-module-patient-2", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if err := expectRegisterPermission(svc, claims, patient.PatientID, "dept-1"); err != nil {
		t.Fatalf("permission should return after module is reenabled: %v", err)
	}
}

func addScopeTestUser(store *memory.Store, id, role string) domain.User {
	user := domain.User{ID: id, Username: id, DisplayName: id, Roles: []domain.Role{domain.Role(role)}, Status: domain.StatusActive}
	store.Add(user)
	return user
}

func claimsForUser(id string) *auth.Claims {
	return &auth.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: id}}
}

func expectRegisterPermission(svc *Service, claims *auth.Claims, patientID, departmentID string) error {
	_, err := svc.CreateRegister(context.Background(), claims, CreateRegisterInput{
		PatientID: patientID, DepartmentID: departmentID,
		DoctorID:  map[string]string{"dept-1": "doc-1", "dept-2": "doc-2"}[departmentID],
		VisitDate: "2026-10-14",
	}, "req-scope-register", "127.0.0.1")
	return err
}

func assertRegisterDenied(t *testing.T, err error) {
	t.Helper()
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Status != 403 {
		t.Fatalf("expected permission denial (403), got %v", err)
	}
}

func TestEffectivePermissionsUseExplicitDepartmentsAndRecheckDepartmentStatus(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	svc := &Service{Users: store, Master: store, Audit: store}
	if err := store.CreateAccessRole(ctx, domain.AccessRole{
		Code: "MULTI_DEPT", Name: "Multi department", ModuleCode: "outpatient", DepartmentID: "dept-1",
		PermissionCodes: []string{domain.PermEncounterCreate}, Status: domain.StatusActive,
	}); err != nil {
		t.Fatal(err)
	}
	user := domain.User{ID: "multi-dept-user", Username: "multi-dept-user", Roles: []domain.Role{"MULTI_DEPT"}, Status: domain.StatusActive}
	store.Add(user)
	if err := store.ReplaceUserRoleScopes(ctx, user.ID, []domain.UserRoleScope{
		{UserID: user.ID, RoleCode: "MULTI_DEPT", DepartmentID: "dept-1", Status: domain.StatusActive},
		{UserID: user.ID, RoleCode: "MULTI_DEPT", DepartmentID: "dept-2", Status: domain.StatusActive},
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.loadEffectivePermissions(ctx, &user); err != nil {
		t.Fatal(err)
	}
	for _, departmentID := range []string{"dept-1", "dept-2"} {
		if !hasPermScope(&user, domain.PermEncounterCreate, "outpatient", departmentID) {
			t.Fatalf("explicit scope should allow %s: %+v", departmentID, user.EffectivePermissions)
		}
	}
	if hasPermScope(&user, domain.PermEncounterCreate, "outpatient", "dept-3") {
		t.Fatal("explicit scopes must deny an unassigned department")
	}

	department, err := store.GetDepartment(ctx, "dept-2")
	if err != nil {
		t.Fatal(err)
	}
	department.Status = domain.StatusInactive
	if _, err := store.UpdateDepartment(ctx, *department); err != nil {
		t.Fatal(err)
	}
	if err := svc.loadEffectivePermissions(ctx, &user); err != nil {
		t.Fatal(err)
	}
	if hasPermScope(&user, domain.PermEncounterCreate, "outpatient", "dept-2") || !hasPermScope(&user, domain.PermEncounterCreate, "outpatient", "dept-1") {
		t.Fatalf("department status must be rechecked per resolution: %+v", user.EffectivePermissions)
	}
}

func TestUserDTOUsesEffectiveRoleScopeDepartments(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	svc := &Service{Users: store, Master: store}
	if err := store.CreateAccessRole(ctx, domain.AccessRole{
		Code: "MULTI_DEPT", Name: "Multi department", ModuleCode: "outpatient", DepartmentID: "dept-1",
		PermissionCodes: []string{domain.PermEncounterCreate}, Status: domain.StatusActive,
	}); err != nil {
		t.Fatal(err)
	}
	user := domain.User{ID: "dto-multi-dept-user", Username: "dto-multi-dept-user", Roles: []domain.Role{"MULTI_DEPT"}, Status: domain.StatusActive}
	store.Add(user)
	if err := store.ReplaceUserRoleScopes(ctx, user.ID, []domain.UserRoleScope{
		{UserID: user.ID, RoleCode: "MULTI_DEPT", DepartmentID: "dept-1", Status: domain.StatusActive},
		{UserID: user.ID, RoleCode: "MULTI_DEPT", DepartmentID: "dept-2", Status: domain.StatusActive},
	}); err != nil {
		t.Fatal(err)
	}

	dto := svc.userDTO(ctx, &user)
	if !dto.AccessControlLoaded {
		t.Fatal("effective access snapshot should be marked loaded")
	}
	if len(dto.Permissions) != 1 || dto.Permissions[0] != domain.PermEncounterCreate {
		t.Fatalf("effective permissions = %v", dto.Permissions)
	}
	wantScopes := []string{"outpatient:dept-1", "outpatient:dept-2"}
	if len(dto.Scopes) != len(wantScopes) || dto.Scopes[0] != wantScopes[0] || dto.Scopes[1] != wantScopes[1] {
		t.Fatalf("effective scopes = %v, want %v", dto.Scopes, wantScopes)
	}
}
