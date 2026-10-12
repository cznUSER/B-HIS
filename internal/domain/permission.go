package domain

// Permission codes are a Phase1 skeleton (权限与审计 §3).
// The login API returns roles only; enforcement lands with outpatient use cases.
const (
	PermPatientCreate    = "patient:create"
	PermEncounterCreate  = "encounter:create"
	PermEncounterCancel  = "encounter:cancel"
	PermOrderSubmit      = "order:submit"
	PermOrderRecipeCheck = "order:recipe-check"
	PermChargePay        = "charge:pay"
	PermChargeRefund     = "charge:refund"
	PermDispenseCreate   = "dispense:create"
	PermDispenseReturn   = "dispense:return"
	PermInventoryRead    = "inventory:read"
	PermAuditRead        = "audit:read"
	PermAdminAll         = "admin:*"
)

// RolePermissions documents the role × permission skeleton. Login does not enforce it.
var RolePermissions = map[Role][]string{
	RoleRegistrar: {
		PermPatientCreate, PermEncounterCreate, PermEncounterCancel,
	},
	RoleDoctor: {
		PermPatientCreate, PermOrderSubmit,
	},
	RoleCashier: {
		PermPatientCreate, PermChargePay, PermChargeRefund,
	},
	RolePharmacist: {
		PermOrderRecipeCheck, PermDispenseCreate, PermDispenseReturn, PermInventoryRead,
	},
	RoleSystem: {PermAdminAll},
}
