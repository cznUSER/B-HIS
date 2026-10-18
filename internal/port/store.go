// Package port defines store interfaces. app and domain must not import a SQL driver.
package port

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
)

// StoreSet is the application-facing collection opened by a database driver.
// Keeping this contract in port lets SQL Server and the reserved Dameng
// adapter plug into the same composition root without leaking driver details.
type StoreSet struct {
	Users       UserStore
	Revocations RevocationStore
	Audit       AuditStore
	Patients    PatientStore
	Registers   RegisterStore
	Orders      OrderStore
	Charges     ChargeStore
	Dispenses   DispenseStore
	Closer      io.Closer
}

// DatabaseDriver is the boundary for a persistent backend. The application
// layer depends only on StoreSet; individual drivers own connection details.
type DatabaseDriver interface {
	Name() string
	Open(dsn string) (StoreSet, error)
}

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
	// ErrRegisterNotComplete is a completion when status is not DIAGNOSING.
	ErrRegisterNotComplete = errors.New("register not complete")
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
	// ErrDuplicateDeptCode is a live department.dept_code clash (ux_department_dept_code).
	ErrDuplicateDeptCode = errors.New("duplicate dept code")
	// ErrDuplicateDoctorCode is a live doctor.doctor_code clash.
	ErrDuplicateDoctorCode = errors.New("duplicate doctor code")
	// ErrDuplicateItemNo is a live catalog item_no or inventory item_no clash.
	ErrDuplicateItemNo = errors.New("duplicate item no")
	// ErrDuplicateUsername is a live app_user.username clash.
	ErrDuplicateUsername         = errors.New("duplicate username")
	ErrDuplicateAccessPermission = errors.New("duplicate access permission")
	ErrDuplicateAccessCatalog    = errors.New("duplicate access catalog entry")
	ErrInvalidUserRoleScope      = errors.New("invalid user role scope")
	ErrDuplicateUserRoleScope    = errors.New("duplicate user role scope")
	// ErrDuplicateDictCode is a live dict_item (dict_type, code) clash.
	ErrDuplicateDictCode = errors.New("duplicate dict code")
	// ErrAdjustNegative is an inventory adjust whose result would be below zero.
	ErrAdjustNegative = errors.New("adjust result negative")
	// ErrNotDrugSKU is an adjust of a missing inventory row (TREAT has none).
	ErrNotDrugSKU = errors.New("not drug sku")
	// ErrConflict is an optimistic row_version mismatch.
	ErrConflict = errors.New("conflict")
	// ErrLastSystemAdmin protects the final active SYSTEM administrator.
	ErrLastSystemAdmin = errors.New("last system administrator")
	// ErrAuditUnavailable means a successful mutation could not be durably audited.
	ErrAuditUnavailable = errors.New("audit unavailable")
	// ErrIdempotencyConflict means a key was reused with a different request hash.
	ErrIdempotencyConflict = errors.New("idempotency key conflict")
	// ErrIdempotencyInProgress means another request currently owns the key.
	ErrIdempotencyInProgress = errors.New("idempotency key in progress")
	// ErrUserAdminBusy means the serialized administrator mutation could not acquire its lock.
	ErrUserAdminBusy = errors.New("user administration busy")
	// ErrDictTypeReadonly is a write to a system enum that is not a maintainable code table.
	ErrDictTypeReadonly = errors.New("dict type readonly")
	// ErrDiagnoseNotCommittable is a diagnosis transition outside SAVE -> COMMIT.
	ErrDiagnoseNotCommittable = errors.New("diagnosis not committable")
	// ErrDiagnoseNotAbatable is a diagnosis transition outside SAVE/COMMIT -> ABATE.
	ErrDiagnoseNotAbatable = errors.New("diagnosis not abatable")
)

// UserStore loads identity records. Production wiring uses SQL Server; the
// memory implementation is reserved for isolated unit tests.
type UserStore interface {
	FindByUsername(ctx context.Context, username string) (*domain.User, error)
	FindByID(ctx context.Context, id string) (*domain.User, error)
	RecordLoginFailure(ctx context.Context, id string, at time.Time, threshold int, lockDuration time.Duration) (bool, error)
	RecordSuccessfulLogin(ctx context.Context, id, clientIP string, at time.Time) error
	ChangePassword(ctx context.Context, id, passwordHash, operatorID string, at time.Time) error
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
// Name and Address are substring matches; PatientNo, CardNo, IDNumber, and HospPkNo are exact.
type PatientQuery struct {
	PatientNo string
	CardNo    string
	IDNumber  string
	Name      string
	HospPkNo  string
	Address   string
	Limit     int
	Offset    int
}

// PatientStore is the MPI port. Persistent drivers implement it; memory is
// kept only as a test double.
type PatientStore interface {
	Create(ctx context.Context, p domain.Patient) error
	Get(ctx context.Context, patientID string) (*domain.Patient, error)
	Search(ctx context.Context, q PatientQuery) ([]domain.Patient, error)
}

// AtomicPatientIdempotencyStore creates a patient and its idempotency record in
// one store transaction. Replayed is true when an already completed key is
// returned; the method must never create a second patient for that key.
type AtomicPatientIdempotencyStore interface {
	CreateWithIdempotency(ctx context.Context, p domain.Patient, scope, key, requestHash string, expiresAt time.Time) (patient domain.Patient, replayed bool, err error)
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
// Method names differ from PatientStore because persistent drivers implement
// both register and master-data operations.
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
	// CompleteRegister moves DIAGNOSING → DIAGNOSED and writes updated_by.
	// Any other status returns ErrRegisterNotComplete and leaves the row unchanged.
	CompleteRegister(ctx context.Context, registerID, operatorID string, at time.Time) (*domain.Register, error)
	DepartmentExists(ctx context.Context, departmentID string) (bool, error)
	DoctorExists(ctx context.Context, doctorID string) (bool, error)
	// GetDepartment and GetDoctor return inactive rows. Missing or soft-deleted rows are ErrNotFound.
	// Callers that open a new register or order must reject status != ACTIVE. 2026-10-13
	GetDepartment(ctx context.Context, departmentID string) (*domain.Department, error)
	GetDoctor(ctx context.Context, doctorID string) (*domain.Doctor, error)
}

// AtomicRegisterIdempotencyStore creates a register bill and its idempotency
// record in one store transaction. Replayed is true when an already completed
// key is returned; the method must never create a second register for a key.
type AtomicRegisterIdempotencyStore interface {
	CreateRegisterWithIdempotency(ctx context.Context, r domain.Register, scope, key, requestHash string, expiresAt time.Time) (register domain.Register, replayed bool, err error)
}

// DiagnoseQuery filters diagnoses by their owning register.
type DiagnoseQuery struct {
	RegisterID string
	Limit      int
}

// DiagnoseStore persists the optional diagnosis lifecycle.
type DiagnoseStore interface {
	CreateDiagnose(ctx context.Context, d domain.Diagnose) error
	GetDiagnose(ctx context.Context, diagnoseID string) (*domain.Diagnose, error)
	ListDiagnoses(ctx context.Context, q DiagnoseQuery) ([]domain.Diagnose, error)
	CommitDiagnose(ctx context.Context, diagnoseID, operatorID string, at time.Time) (*domain.Diagnose, error)
	AbateDiagnose(ctx context.Context, diagnoseID, operatorID string, at time.Time) (*domain.Diagnose, error)
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
// Persistent drivers implement it. 2026-10-12
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

// AtomicOrderIdempotencyStore creates an order sheet, all of its item rows, and
// the idempotency record in one store transaction. Replayed is true when an
// already completed key is returned; the method must never create a second
// order for that key.
type AtomicOrderIdempotencyStore interface {
	CreateOrderWithIdempotency(ctx context.Context, o domain.Order, scope, key, requestHash string, expiresAt time.Time) (order domain.Order, replayed bool, err error)
}

// AtomicChargeIdempotencyStore creates a charge, all of its item rows, and the
// idempotency record in one store transaction. Replayed is true when an
// already completed key is returned; the method must never create a second
// charge for that key.
type AtomicChargeIdempotencyStore interface {
	CreateChargeWithIdempotency(ctx context.Context, c domain.Charge, scope, key, requestHash string, expiresAt time.Time) (charge domain.Charge, replayed bool, err error)
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
// Persistent drivers implement it. Pay and refund are one transaction each.
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

// AtomicChargePaymentIdempotencyStore atomically persists a cash payment and
// the charge/order state it mutates. It is separate from ChargeStore so legacy
// adapters can continue to support the non-idempotent method.
type AtomicChargePaymentIdempotencyStore interface {
	LookupPayIdempotency(ctx context.Context, scope, key, requestHash string) (*domain.Charge, *domain.Payment, bool, error)
	LookupRefundIdempotency(ctx context.Context, scope, key, requestHash string) (*domain.Charge, *domain.Charge, *domain.Payment, bool, error)
	PayChargeWithIdempotency(ctx context.Context, chargeID string, payment domain.Payment, scope, key, requestHash string, expiresAt time.Time) (*domain.Charge, *domain.Payment, bool, error)
	RefundChargeWithIdempotency(ctx context.Context, old domain.Charge, newCharge *domain.Charge, payment domain.Payment, scope, key, requestHash string, expiresAt time.Time) (*domain.Charge, *domain.Charge, *domain.Payment, bool, error)
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

// InventoryLedgerQuery uses a half-open time range in UTC. Cursor order is newest first.
type InventoryLedgerQuery struct {
	SkuCode        string
	BizType        string
	FromAt         time.Time
	ToAt           time.Time
	HasFrom        bool
	HasTo          bool
	CursorAt       time.Time
	CursorLedgerID string
	HasCursor      bool
	Limit          int
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
	ListInventoryLedger(ctx context.Context, query InventoryLedgerQuery) ([]domain.InventoryLedgerEntry, error)
	QtyOnHandMilli(ctx context.Context, itemNo string) (int64, error)
	OrderHasDispense(ctx context.Context, orderID string) (bool, error)
	NetByChargeItem(ctx context.Context, chargeItemID string) (dispensedMilli, returnedMilli int64, ok bool, err error)
	NetByOrderItem(ctx context.Context, orderItemID string) (dispensedMilli, returnedMilli int64, ok bool, err error)
}

// AtomicDispenseIdempotencyStore keeps the idempotency record in the same
// transaction as the dispense slip/items, stock quantity, and inventory ledger.
// Replays return the original dispense resource; a changed request hash is a
// conflict and a pending key is reported as in progress.
type AtomicDispenseIdempotencyStore interface {
	LookupDispenseIdempotency(ctx context.Context, scope, key, requestHash string) (*domain.Dispense, bool, error)
	CreateDispenseWithIdempotency(ctx context.Context, d domain.Dispense, scope, key, requestHash string, expiresAt time.Time) (domain.Dispense, bool, error)
	ReturnDispenseWithIdempotency(ctx context.Context, dispenseID, reason, operatorID string, at time.Time, lines []ReturnLine, scope, key, requestHash string, expiresAt time.Time) (domain.Dispense, bool, error)
}
