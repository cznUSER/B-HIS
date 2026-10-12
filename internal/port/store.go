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
	// ErrOrderNotEditable is a header/item change when status is not SAVE.
	ErrOrderNotEditable = errors.New("order not editable")
	// ErrOrderNotCommittable is a commit when status is not SAVE.
	ErrOrderNotCommittable = errors.New("order not committable")
	// ErrOrderChargedNotAbatable is an abate of a charged order.
	ErrOrderChargedNotAbatable = errors.New("order charged not abatable")
	// ErrIllegalStateTransition is a refused status change (PROC/ABATE abate, etc.).
	ErrIllegalStateTransition = errors.New("illegal state transition")
	// ErrChargeExists is a second Charge for an order that already has one (default 1:1, no split).
	ErrChargeExists = errors.New("charge exists for order")
	// ErrChargeAlreadyPaid is a second pay of a charge that already has paid_at.
	ErrChargeAlreadyPaid = errors.New("charge already paid")
	// ErrChargeAlreadyAbated is a refund of an ABATE charge.
	ErrChargeAlreadyAbated = errors.New("charge already abated")
	// ErrChargeNotPaid is a refund before pay.
	ErrChargeNotPaid = errors.New("charge not paid")
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

// AuditStore appends identity events, patient.create, register.create / register.check-in / register.cancel,
// and order.create / order.commit / order.abate. Implementations must not persist passwords.
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

// OrderQuery is an AND of the provided filters. Empty strings are ignored.
// RegisterID, PatientID, OrderNo, and Status are exact.
type OrderQuery struct {
	RegisterID string
	PatientID  string
	OrderNo    string
	Status     string
	Limit      int
}

// OrderStore is the order sheet, its items, and the catalog lookup.
// memory and sqlserver both implement it. 2026-10-12
type OrderStore interface {
	CreateOrder(ctx context.Context, o domain.Order) error
	GetOrder(ctx context.Context, orderID string) (*domain.Order, error)
	SearchOrders(ctx context.Context, q OrderQuery) ([]domain.Order, error)
	// UpdateOrderDraft writes header fields. When replaceItems is true the item rows are replaced.
	// Status must be SAVE; otherwise ErrOrderNotEditable.
	UpdateOrderDraft(ctx context.Context, o domain.Order, replaceItems bool) (*domain.Order, error)
	// CommitOrder persists SAVE→COMMIT, submittedAt, and the recomputed fees.
	// Status must still be SAVE; otherwise ErrOrderNotCommittable.
	CommitOrder(ctx context.Context, o domain.Order) (*domain.Order, error)
	// AbateOrder moves SAVE/COMMIT with charged=false to ABATE.
	// charged=true returns ErrOrderChargedNotAbatable. Other statuses return ErrIllegalStateTransition.
	AbateOrder(ctx context.Context, orderID, operatorID string, at time.Time) (*domain.Order, error)
	CatalogByItemNo(ctx context.Context, itemNo string) (*domain.CatalogItem, error)
}

// ChargeQuery is an AND of the provided filters. Empty strings are ignored.
// FromDate and ToDate are inclusive Asia/Shanghai calendar days (YYYY-MM-DD)
// matched against charge.created_at, not the register visit date. 2026-10-12
type ChargeQuery struct {
	PatientID  string
	RegisterID string
	OrderID    string
	FromDate   string
	ToDate     string
	Limit      int
}

// ChargeStore is the square, its items, and its payments.
// memory and sqlserver both implement it. Pay and refund are one transaction each.
// Drug on-hand and dispense/return quantities are a charge-side stub (not 005 columns):
// default drug on-hand is domain.DefaultDrugOnHandMilli so the demo catalog can pay,
// and tests can lower it to reach INSUFFICIENT_STOCK. 2026-10-12
type ChargeStore interface {
	CreateCharge(ctx context.Context, c domain.Charge) error
	GetCharge(ctx context.Context, chargeID string) (*domain.Charge, error)
	SearchCharges(ctx context.Context, q ChargeQuery) ([]domain.Charge, error)
	// PayCharge sets paid_at, inserts a CHARGE payment, and marks the order charged.
	PayCharge(ctx context.Context, chargeID string, payment domain.Payment) (*domain.Charge, *domain.Payment, error)
	// RefundCharge abates old, optionally inserts the remainder charge, and inserts a REFUND payment.
	// newCharge nil means a full refund (no zero-amount square).
	RefundCharge(ctx context.Context, old domain.Charge, newCharge *domain.Charge, payment domain.Payment) (*domain.Charge, *domain.Charge, *domain.Payment, error)
	// DrugOnHandMilli is the stub on-hand for a drug item_no (quantity × 1000).
	// Unknown item numbers use domain.DefaultDrugOnHandMilli.
	DrugOnHandMilli(ctx context.Context, itemNo string) (int64, error)
	// DispenseLedger is the stub dispensed/returned quantity for one charge item (× 1000).
	// ok is false when nobody has recorded an explicit ledger row.
	DispenseLedger(ctx context.Context, chargeItemID string) (dispensedMilli, returnedMilli int64, ok bool, err error)
}
