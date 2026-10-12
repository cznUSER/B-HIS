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
	// ErrNotCharged is a dispense without a paid NORMAL charge.
	ErrNotCharged = errors.New("not charged")
	// ErrInsufficientStock is a dispense or charge whose drug qty exceeds qty_on_hand.
	ErrInsufficientStock = errors.New("insufficient stock")
	// ErrAlreadyDispensed is a dispense request against a line that is already FULL.
	ErrAlreadyDispensed = errors.New("already dispensed")
	// ErrDispenseExceeds is a dispense qty outside 0 < qty ≤ available.
	ErrDispenseExceeds = errors.New("dispense exceeds available")
	// ErrReturnExceeds is a return qty outside 0 < qty ≤ dispensed−returned.
	ErrReturnExceeds = errors.New("return exceeds dispense")
	// ErrNotDispensed is a return of a line that was not dispensed or is unknown.
	ErrNotDispensed = errors.New("not dispensed")
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
// DrugOnHandMilli reads inventory_item.qty_on_hand (006). Charge never decrements it.
// DispenseLedger is net dispensed for one charge item (real slips, else the test stub). 2026-10-12
type ChargeStore interface {
	CreateCharge(ctx context.Context, c domain.Charge) error
	GetCharge(ctx context.Context, chargeID string) (*domain.Charge, error)
	SearchCharges(ctx context.Context, q ChargeQuery) ([]domain.Charge, error)
	// PayCharge sets paid_at, inserts a CHARGE payment, and marks the order charged.
	PayCharge(ctx context.Context, chargeID string, payment domain.Payment) (*domain.Charge, *domain.Payment, error)
	// RefundCharge abates old, optionally inserts the remainder charge, and inserts a REFUND payment.
	// newCharge nil means a full refund (no zero-amount square).
	RefundCharge(ctx context.Context, old domain.Charge, newCharge *domain.Charge, payment domain.Payment) (*domain.Charge, *domain.Charge, *domain.Payment, error)
	// DrugOnHandMilli is inventory_item.qty_on_hand for item_no (quantity × 1000).
	// Unknown or inactive items are 0. TREAT is not in the table.
	DrugOnHandMilli(ctx context.Context, itemNo string) (int64, error)
	// DispenseLedger is dispensed and returned quantity for one charge item (× 1000).
	// Real dispense_item rows win. ok is false when nothing has been recorded.
	DispenseLedger(ctx context.Context, chargeItemID string) (dispensedMilli, returnedMilli int64, ok bool, err error)
}

// DispenseQuery filters are AND-ed. Empty strings are ignored.
// FromDate and ToDate are inclusive Asia/Shanghai days (YYYY-MM-DD) on dispensed_at.
type DispenseQuery struct {
	PatientID  string
	RegisterID string
	ChargeID   string
	OrderID    string
	FromDate   string
	ToDate     string
	Limit      int
}

// ReturnLine is one return inside a single return transaction.
type ReturnLine struct {
	DispenseItemID string
	ReturnMilli    int64
}

// DispenseStore is dispense slips plus the quantity ledger they mutate.
// CommitDispense and CommitReturn each run as one transaction: stock, ledger, slip.
// 2026-10-12
type DispenseStore interface {
	CommitDispense(ctx context.Context, d domain.Dispense) (*domain.Dispense, error)
	CommitReturn(ctx context.Context, dispenseID, reason, operatorID string, at time.Time, lines []ReturnLine) (*domain.Dispense, error)
	GetDispense(ctx context.Context, dispenseID string) (*domain.Dispense, error)
	SearchDispenses(ctx context.Context, q DispenseQuery) ([]domain.Dispense, error)
	ListInventory(ctx context.Context, itemNo string) ([]domain.InventoryItem, error)
	QtyOnHandMilli(ctx context.Context, itemNo string) (int64, error)
	OrderHasDispense(ctx context.Context, orderID string) (bool, error)
	NetByChargeItem(ctx context.Context, chargeItemID string) (dispensedMilli, returnedMilli int64, ok bool, err error)
	NetByOrderItem(ctx context.Context, orderItemID string) (dispensedMilli, returnedMilli int64, ok bool, err error)
}
