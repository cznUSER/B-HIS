// Package port defines store interfaces. app and domain must not import a SQL driver.
package port

import (
	"context"
	"errors"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
)

var (
	// ErrNotFound is returned when a record does not exist (or is soft-deleted).
	ErrNotFound = errors.New("not found")
	// ErrDuplicateIDNumber is a non-empty id_number clash (002_patient ux_patient_id_number).
	ErrDuplicateIDNumber = errors.New("duplicate id number")
	// ErrDuplicateCardNo is a non-empty card_no clash (ux_patient_card_no).
	ErrDuplicateCardNo = errors.New("duplicate card no")
	// ErrDuplicateBusinessNo is a hosp_pk_no or register_no clash. Callers may retry with a new number.
	ErrDuplicateBusinessNo = errors.New("duplicate business number")
	// ErrRegisterAlreadyBack is a second cancel of a BACK register (AC-04 / R-04).
	ErrRegisterAlreadyBack = errors.New("register already back")
	// ErrRegisterNotCancellable is a cancel when status is not NORMAL (AC-03).
	ErrRegisterNotCancellable = errors.New("register not cancellable")
	// ErrRegisterNotCheckIn is a check-in when status is not NORMAL or USED.
	// 2026-10-12
	ErrRegisterNotCheckIn = errors.New("register not check-in")
)

// UserStore loads identity records. Phase1 default is SQL Server; memory is the local demo.
type UserStore interface {
	FindByUsername(ctx context.Context, username string) (*domain.User, error)
	FindByID(ctx context.Context, id string) (*domain.User, error)
}

// RevocationStore is the JWT jti blacklist. Entries live at least until token exp.
type RevocationStore interface {
	Revoke(ctx context.Context, jti, userID string, expiresAt time.Time) error
	IsRevoked(ctx context.Context, jti string) (bool, error)
}

// AuditStore appends identity events, patient.create, and register.create / register.check-in / register.cancel.
// Implementations must not persist passwords.
type AuditStore interface {
	Append(ctx context.Context, event domain.AuditEvent) error
}

// PatientQuery is an AND of the provided filters. Empty strings are ignored by the store.
// Name and Address are substring matches; CardNo, IDNumber, and HospPkNo are exact.
type PatientQuery struct {
	CardNo   string
	IDNumber string
	Name     string
	HospPkNo string
	Address  string
	Limit    int
}

// PatientStore is the MPI port. memory and sqlserver both implement it.
type PatientStore interface {
	Create(ctx context.Context, p domain.Patient) error
	Get(ctx context.Context, patientID string) (*domain.Patient, error)
	Search(ctx context.Context, q PatientQuery) ([]domain.Patient, error)
}

// RegisterQuery is an AND of the provided filters. Empty strings are ignored.
// PatientID, RegisterNo, and VisitDate (YYYY-MM-DD) are exact.
type RegisterQuery struct {
	PatientID  string
	RegisterNo string
	VisitDate  string
	Limit      int
}

// RegisterStore is the register bill plus department/doctor master data.
// Method names differ from PatientStore because memory and sqlserver implement both.
type RegisterStore interface {
	CreateRegister(ctx context.Context, r domain.Register) error
	GetRegister(ctx context.Context, registerID string) (*domain.Register, error)
	SearchRegisters(ctx context.Context, q RegisterQuery) ([]domain.Register, error)
	// CancelRegister moves NORMAL → BACK and writes the back operator columns.
	// BACK returns ErrRegisterAlreadyBack. Any other status returns ErrRegisterNotCancellable.
	// Neither refusal changes the row.
	CancelRegister(ctx context.Context, registerID, operatorID string, at time.Time) (*domain.Register, error)
	// CheckInRegister moves NORMAL or USED → DIAGNOSING and writes updated_by.
	// Any other status returns ErrRegisterNotCheckIn and leaves the row unchanged.
	// 2026-10-12
	CheckInRegister(ctx context.Context, registerID, operatorID string, at time.Time) (*domain.Register, error)
	DepartmentExists(ctx context.Context, departmentID string) (bool, error)
	DoctorExists(ctx context.Context, doctorID string) (bool, error)
}
