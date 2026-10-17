package domain

import "time"

// Diagnose is the optional structured diagnosis attached to a register.
// RegisterID is the encounter identity in the Phase2 contract.
type Diagnose struct {
	DiagnoseID string
	RegisterID string
	Status     string
	Content    string
	CreatedAt  time.Time
	CreatedBy  string
	UpdatedAt  time.Time
	UpdatedBy  string
	IsDeleted  bool
}

const (
	DiagnoseStatusNew    = "NEW"
	DiagnoseStatusSave   = "SAVE"
	DiagnoseStatusCommit = "COMMIT"
	DiagnoseStatusAbate  = "ABATE"
)
