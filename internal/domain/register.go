package domain

import "time"

// Register status values match OpenAPI 0.3.6 TRegisterStatus and 003_register.
const (
	RegisterStatusNormal     = "NORMAL"
	RegisterStatusUsed       = "USED"
	RegisterStatusBack       = "BACK"
	RegisterStatusDiagnosing = "DIAGNOSING"
	RegisterStatusDiagnosed  = "DIAGNOSED"
	RegisterStatusPaused     = "PAUSED"
)

// Department is dbo.department. IDs match 003_register_seed (dept-1 / dept-2).
type Department struct {
	DepartmentID string
	DeptCode     string
	DeptName     string
	Status       string
	SortNo       int
	CreatedAt    time.Time
	CreatedBy    string
	UpdatedAt    time.Time
	UpdatedBy    string
	IsDeleted    bool
}

// Doctor is dbo.doctor. IDs match 003_register_seed (doc-1 / doc-2).
type Doctor struct {
	DoctorID     string
	DepartmentID string
	DoctorCode   string
	DoctorName   string
	Status       string
	CreatedAt    time.Time
	CreatedBy    string
	UpdatedAt    time.Time
	UpdatedBy    string
	IsDeleted    bool
}

// Register is dbo.register_bill. RegisterFeeCents is yuan × 100.
// Encounter ≡ RegisterBill; encounterId ≡ RegisterID. 2026-10-12
type Register struct {
	RegisterID       string
	RegisterNo       string
	PatientID        string
	DepartmentID     string
	DoctorID         string
	RegisterSort     string
	VisitDate        time.Time
	Status           string
	RegisterFeeCents int64
	RegisterCharged  bool
	RegisteredAt     time.Time
	BackOperatorID   string
	BackOperatedAt   time.Time
	CreatedAt        time.Time
	CreatedBy        string
	UpdatedAt        time.Time
	UpdatedBy        string
	IsDeleted        bool
}
