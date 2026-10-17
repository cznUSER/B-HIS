package domain

// Permission codes are a Phase1 skeleton (权限与审计 §3).
// The login API returns roles only; enforcement lands with outpatient use cases.
const (
	PermPatientCreate    = "patient:create"
	PermEncounterCreate  = "encounter:create"
	PermEncounterCheckIn = "encounter:check-in"
	PermEncounterCancel  = "encounter:cancel"
	// PermDiagnoseWrite grants access to the outpatient diagnosis workspace and
	// its SAVE/COMMIT/ABATE lifecycle. It is intentionally separate from
	// encounter check-in so registration staff cannot author clinical entries.
	PermDiagnoseWrite    = "diagnose:write"
	PermOrderSubmit      = "order:submit"
	PermOrderRecipeCheck = "order:recipe-check"
	PermChargePay        = "charge:pay"
	PermChargeRefund     = "charge:refund"
	PermDispenseCreate   = "dispense:create"
	PermDispenseReturn   = "dispense:return"
	PermInventoryRead    = "inventory:read"
	PermInventoryAdjust  = "inventory:adjust"
	PermAuditRead        = "audit:read"
	PermAdminUser        = "admin:user"
	PermAdminRole        = "admin:role"
	PermAdminPermission  = "admin:permission"
	PermAdminAll         = "admin:*"
)

// RolePermissions documents the role × permission skeleton. Login does not enforce it.
var RolePermissions = map[Role][]string{
	RoleRegistrar: {
		PermPatientCreate, PermEncounterCreate, PermEncounterCheckIn, PermEncounterCancel,
	},
	RoleDoctor: {
		PermPatientCreate, PermEncounterCheckIn, PermDiagnoseWrite, PermOrderSubmit,
	},
	RoleCashier: {
		PermPatientCreate, PermChargePay, PermChargeRefund,
	},
	RolePharmacist: {
		PermOrderRecipeCheck, PermDispenseCreate, PermDispenseReturn, PermInventoryRead, PermInventoryBatchRead,
	},
	RoleSystem: {PermAdminAll},
}
