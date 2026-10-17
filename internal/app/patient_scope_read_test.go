package app

import (
	"context"
	"errors"
	"testing"

	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/store/memory"
)

func TestPatientReadsAcceptScopedOutpatientGrant(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	svc := &Service{Users: store, Master: store, Patients: store}
	user := domain.User{ID: "patient-scope-read", Username: "patient-scope-read", DisplayName: "范围挂号员", Roles: []domain.Role{domain.RoleRegistrar}, Status: domain.StatusActive}
	store.Add(user)
	if err := store.ReplaceUserRoleScopes(ctx, user.ID, []domain.UserRoleScope{{UserID: user.ID, RoleCode: string(domain.RoleRegistrar), DepartmentID: "dept-1", Status: domain.StatusActive}}); err != nil {
		t.Fatal(err)
	}
	patient, err := svc.CreatePatient(ctx, systemClaims(), validInput(), "patient-scope-seed", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	claims := &auth.Claims{}
	claims.Subject = user.ID
	if _, err := svc.SearchPatients(ctx, claims, PatientSearch{Name: patient.Name}); err != nil {
		t.Fatalf("scoped patient search: %v", err)
	}
	if _, err := svc.GetPatient(ctx, claims, patient.PatientID); err != nil {
		t.Fatalf("scoped patient read: %v", err)
	}
}

func TestPatientReadsRejectDisabledScopeAndExplicitDeny(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		status string
		deny   bool
	}{
		{name: "disabled scope", status: domain.StatusDisabled},
		{name: "explicit deny", status: domain.StatusActive, deny: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := memory.New()
			svc := &Service{Users: store, Master: store, Patients: store}
			user := domain.User{ID: "patient-scope-denied-" + tc.name, Username: "patient-scope-denied-" + tc.name, DisplayName: tc.name, Roles: []domain.Role{domain.RoleRegistrar}, Status: domain.StatusActive}
			store.Add(user)
			if err := store.ReplaceUserRoleScopes(ctx, user.ID, []domain.UserRoleScope{{UserID: user.ID, RoleCode: string(domain.RoleRegistrar), DepartmentID: "dept-1", Status: tc.status}}); err != nil {
				t.Fatal(err)
			}
			if tc.deny {
				if err := store.ReplaceUserPermissionOverrides(ctx, user.ID, []domain.PermissionOverride{
					{UserID: user.ID, PermissionCode: domain.PermPatientCreate, ModuleCode: "outpatient", Effect: domain.PermissionDeny},
					{UserID: user.ID, PermissionCode: domain.PermEncounterCreate, ModuleCode: "outpatient", Effect: domain.PermissionDeny},
					{UserID: user.ID, PermissionCode: domain.PermEncounterCheckIn, ModuleCode: "outpatient", Effect: domain.PermissionDeny},
					{UserID: user.ID, PermissionCode: domain.PermEncounterCancel, ModuleCode: "outpatient", Effect: domain.PermissionDeny},
				}); err != nil {
					t.Fatal(err)
				}
			}
			claims := &auth.Claims{}
			claims.Subject = user.ID
			_, err := svc.SearchPatients(ctx, claims, PatientSearch{Name: "任意"})
			var appErr *Error
			if !errors.As(err, &appErr) || appErr.Status != 403 {
				t.Fatalf("patient read error = %v, want 403", err)
			}
		})
	}
}
