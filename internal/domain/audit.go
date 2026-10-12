package domain

import "time"

const (
	ActionLoginSuccess   = "LOGIN_SUCCESS"
	ActionLoginFailure   = "LOGIN_FAILURE"
	ActionLogout         = "LOGOUT"
	ActionPatientCreate  = "patient.create"
	ActionRegisterCreate = "register.create"
	ActionRegisterCancel = "register.cancel"

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
