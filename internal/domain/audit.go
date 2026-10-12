package domain

import "time"

const (
	ActionLoginSuccess         = "LOGIN_SUCCESS"
	ActionLoginFailure         = "LOGIN_FAILURE"
	ActionLogout               = "LOGOUT"
	ActionPatientCreate        = "patient.create"
	ActionRegisterCreate       = "register.create"
	ActionRegisterCheckIn      = "register.check-in"
	ActionRegisterCancel       = "register.cancel"
	ActionOrderCreate          = "order.create"
	ActionOrderCommit          = "order.commit"
	ActionOrderAbate           = "order.abate"
	ActionChargeCreate         = "charge.create"
	ActionChargePay            = "charge.pay"
	ActionChargeRefund         = "charge.refund"
	ActionDispenseCreate       = "dispense.create"
	ActionDispenseReturn       = "dispense.return"
	ActionDepartmentCreate     = "department.create"
	ActionDepartmentUpdate     = "department.update"
	ActionDepartmentDeactivate = "department.deactivate"
	ActionDoctorCreate         = "doctor.create"
	ActionDoctorUpdate         = "doctor.update"
	ActionCatalogCreate        = "catalog.create"
	ActionCatalogUpdate        = "catalog.update"
	ActionInventoryAdjust      = "inventory.adjust"
	ActionDictCreate           = "dict.create"
	ActionDictUpdate           = "dict.update"
	ActionUserCreate           = "user.create"
	ActionUserUpdate           = "user.update"
	ActionUserDeactivate       = "user.deactivate"
	ActionUserResetPassword    = "user.reset_password"

	OutcomeSuccess = "SUCCESS"
	OutcomeFail    = "FAIL"
)

// AuditEvent is an append-only identity audit row (LOGIN_SUCCESS / LOGIN_FAILURE / LOGOUT).
type AuditEvent struct {
	ID           string
	OccurredAt   time.Time
	ActorID      string
	ActorRoles   string
	Action       string
	ResourceType string
	ResourceID   string
	Outcome      string
	ErrorCode    string
	RequestID    string
	ClientIP     string
	DurationMS   int64
}
