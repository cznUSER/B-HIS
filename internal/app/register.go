package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

const (
	CodePatientNotFound          = "PATIENT_NOT_FOUND"
	CodeDepartmentNotFound       = "DEPARTMENT_NOT_FOUND"
	CodeDoctorNotFound           = "DOCTOR_NOT_FOUND"
	CodeDoctorDepartmentMismatch = "DOCTOR_DEPARTMENT_MISMATCH"
	CodeRegisterNotCancellable   = "REGISTER_NOT_CANCELLABLE"
	CodeRegisterBackNotAllowed   = "REGISTER_BACK_NOT_ALLOWED"
	CodeRegisterExpired          = "REGISTER_EXPIRED"
	CodeIllegalStateTransition   = "ILLEGAL_STATE_TRANSITION"

	msgPatientNotFound                = "患者不存在"
	msgDepartmentNotFound             = "科室不存在"
	msgDoctorNotFound                 = "医生不存在"
	msgDoctorDepartmentMismatch       = "医生不属于所选科室"
	msgRegisterNotCancellable         = "当前状态不可退号"
	msgRegisterActivityNotCancellable = "已有开单或收费记录，不能退号"
	msgRegisterBackNotAllowed         = "已退号，不能再次退号"
	msgRegisterExpired                = "挂号已超过有效期"
	msgIllegalStateTransition         = "当前状态不允许该操作"

	idemScopeRegisterCreate = "register.create"
	registerSearchLimit     = 100
	registerIdempotencyTTL  = 24 * time.Hour
	maxRegisterFee          = 9999999999999999.99
)

// CreateRegisterInput is OpenAPI CreateRegisterRequest.
// RegisterFee nil means 0. IdempotencyKey comes from the header, not the body.
type CreateRegisterInput struct {
	PatientID        string   `json:"patientId"`
	DepartmentID     string   `json:"departmentId"`
	DoctorID         string   `json:"doctorId"`
	VisitDate        string   `json:"visitDate"`
	VisitType        string   `json:"visitType"`
	AppointmentStart string   `json:"appointmentStart"`
	AppointmentEnd   string   `json:"appointmentEnd"`
	AppointmentNote  string   `json:"appointmentNote"`
	RegisterSort     *string  `json:"registerSort"`
	RegisterFee      *float64 `json:"registerFee"`
	IdempotencyKey   string   `json:"-"`
}

// RegisterDTO is OpenAPI RegisterDto. Nullable fields are JSON null when empty.
type RegisterDTO struct {
	RegisterID       string  `json:"registerId"`
	RegisterNo       string  `json:"registerNo"`
	PatientID        string  `json:"patientId"`
	PatientName      string  `json:"patientName,omitempty"`
	DepartmentID     string  `json:"departmentId"`
	DoctorID         string  `json:"doctorId"`
	RegisterSort     *string `json:"registerSort"`
	VisitDate        string  `json:"visitDate"`
	VisitType        string  `json:"visitType"`
	AppointmentStart *string `json:"appointmentStart"`
	AppointmentEnd   *string `json:"appointmentEnd"`
	AppointmentNote  *string `json:"appointmentNote"`
	Status           string  `json:"status"`
	RegisterFee      float64 `json:"registerFee"`
	RegisterCharged  bool    `json:"registerCharged"`
	RegisteredAt     string  `json:"registeredAt"`
	BackOperatorID   *string `json:"backOperatorId"`
	BackOperatedAt   *string `json:"backOperatedAt"`
}

// RegisterListDTO is the GET /registers body.
type RegisterListDTO struct {
	Items []RegisterDTO `json:"items"`
}

// RegisterSearch filters are AND-ed and exact. An empty query returns no rows,
// same as patient search.
type RegisterSearch struct {
	PatientID  string
	RegisterNo string
	VisitDate  string
}

type registerCanon struct {
	PatientID    string `json:"patientId"`
	DepartmentID string `json:"departmentId"`
	DoctorID     string `json:"doctorId"`
	VisitDate    string `json:"visitDate"`
	RegisterSort string `json:"registerSort"`
	FeeCents     int64  `json:"feeCents"`
}

// CreateRegister opens a bill at status NORMAL.
// Required: patientId, departmentId, doctorId, visitDate. registerSort is optional.
// registerFee defaults to 0. registerCharged defaults to false.
// Missing masters are 404 PATIENT_NOT_FOUND / DEPARTMENT_NOT_FOUND / DOCTOR_NOT_FOUND.
// POST requires encounter:create or admin:* (REGISTRAR, SYSTEM).
// Idempotency-Key is persisted when the register store provides the atomic
// register.create contract; memory and SQL implementations use one lock or
// transaction for the idempotency row and register bill.
func (s *Service) CreateRegister(ctx context.Context, claims *auth.Claims, in CreateRegisterInput, requestID, ip string) (*RegisterDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	norm, fields := normalizeRegister(in)
	if in.IdempotencyKey != "" && !validIdempotencyKey(in.IdempotencyKey) {
		fields = append(fields, "idempotencyKey")
	}
	if len(fields) > 0 {
		return nil, &Error{Status: 400, Code: CodeValidation, Message: msgValidation, Details: map[string]any{"fields": fields}}
	}
	if !hasPermScope(user, domain.PermEncounterCreate, "outpatient", norm.DepartmentID) {
		e := forbidden()
		s.auditRegister(ctx, start, user, domain.ActionRegisterCreate, "", domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	hash := hashRegister(norm)
	if in.IdempotencyKey != "" {
		if dto, err := s.lookupRegisterIdem(in.IdempotencyKey, hash); err != nil || dto != nil {
			return dto, err
		}
	}
	if s.Patients == nil || s.Registers == nil {
		return nil, errors.New("register store is not configured")
	}
	if err := s.requireRegisterMasters(ctx, norm.PatientID, norm.DepartmentID, norm.DoctorID); err != nil {
		var ae *Error
		if errors.As(err, &ae) {
			s.auditRegister(ctx, start, user, domain.ActionRegisterCreate, "", domain.OutcomeFail, ae.Code, requestID, ip)
		}
		return nil, err
	}

	now := s.now().UTC()
	visit, _ := parseVisitDate(norm.VisitDate)
	var created domain.Register
	var createErr error
	var replayed bool
	atomicIdemStore, atomicIdem := s.Registers.(port.AtomicRegisterIdempotencyStore)
	for attempt := 0; attempt < 5; attempt++ {
		id, err := newUUID()
		if err != nil {
			return nil, err
		}
		no, err := newBusinessNo("R", now)
		if err != nil {
			return nil, err
		}
		reg := domain.Register{
			RegisterID:       id,
			RegisterNo:       no,
			PatientID:        norm.PatientID,
			DepartmentID:     norm.DepartmentID,
			DoctorID:         norm.DoctorID,
			RegisterSort:     norm.RegisterSort,
			VisitDate:        visit,
			Status:           domain.RegisterStatusNormal,
			RegisterFeeCents: norm.FeeCents,
			RegisterCharged:  false,
			RegisteredAt:     now,
			CreatedAt:        now,
			CreatedBy:        user.ID,
			UpdatedAt:        now,
			UpdatedBy:        user.ID,
		}
		if atomicIdem && in.IdempotencyKey != "" {
			created, replayed, createErr = atomicIdemStore.CreateRegisterWithIdempotency(ctx, reg, idemScopeRegisterCreate, in.IdempotencyKey, hash, time.Now().UTC().Add(registerIdempotencyTTL))
		} else {
			createErr = s.Registers.CreateRegister(ctx, reg)
			if createErr == nil {
				created = reg
			}
		}
		if createErr == nil {
			break
		}
		if errors.Is(createErr, port.ErrDuplicateBusinessNo) {
			continue
		}
		break
	}
	if createErr != nil {
		switch {
		case errors.Is(createErr, port.ErrIdempotencyConflict):
			e := &Error{Status: 409, Code: CodeIdempotencyConflict, Message: msgIdempotencyConflict}
			s.auditRegister(ctx, start, user, domain.ActionRegisterCreate, "", domain.OutcomeFail, e.Code, requestID, ip)
			return nil, e
		case errors.Is(createErr, port.ErrIdempotencyInProgress):
			e := &Error{Status: 409, Code: CodeIdempotencyInProgress, Message: msgIdempotencyInProgress}
			s.auditRegister(ctx, start, user, domain.ActionRegisterCreate, "", domain.OutcomeFail, e.Code, requestID, ip)
			return nil, e
		default:
			return nil, createErr
		}
	}
	dto := toRegisterDTO(created)
	if replayed {
		return &dto, nil
	}
	if in.IdempotencyKey != "" && !atomicIdem {
		stored, err := s.saveRegisterIdem(in.IdempotencyKey, hash, dto)
		if err != nil {
			return nil, err
		}
		if stored != nil {
			dto = *stored
		}
	}
	s.auditRegister(ctx, start, user, domain.ActionRegisterCreate, dto.RegisterID, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

// SearchRegisters returns an AND of the given filters. With no filters the list is empty.
func (s *Service) SearchRegisters(ctx context.Context, claims *auth.Claims, q RegisterSearch) (*RegisterListDTO, error) {
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !hasAnyPermModule(user, "outpatient", domain.PermEncounterCreate, domain.PermEncounterCheckIn, domain.PermEncounterCancel) {
		return nil, forbidden()
	}
	q.PatientID = strings.TrimSpace(q.PatientID)
	q.RegisterNo = strings.TrimSpace(q.RegisterNo)
	q.VisitDate = strings.TrimSpace(q.VisitDate)
	empty := RegisterListDTO{Items: []RegisterDTO{}}
	if q.PatientID == "" && q.RegisterNo == "" && q.VisitDate == "" {
		return &empty, nil
	}
	if q.VisitDate != "" {
		if _, ok := parseVisitDate(q.VisitDate); !ok {
			return nil, &Error{Status: 400, Code: CodeValidation, Message: msgValidation, Details: map[string]any{"fields": []string{"visitDate"}}}
		}
	}
	if s.Registers == nil {
		return nil, errors.New("register store is not configured")
	}
	rows, err := s.Registers.SearchRegisters(ctx, port.RegisterQuery{
		PatientID: q.PatientID, RegisterNo: q.RegisterNo, VisitDate: q.VisitDate, Limit: registerSearchLimit,
	})
	if err != nil {
		return nil, err
	}
	items := make([]RegisterDTO, 0, len(rows))
	patientNames := make(map[string]string)
	for _, r := range rows {
		if !hasAnyPermScope(user, "outpatient", r.DepartmentID, domain.PermEncounterCreate, domain.PermEncounterCheckIn, domain.PermEncounterCancel) {
			continue
		}
		dto := toRegisterDTO(r)
		if name, ok := patientNames[r.PatientID]; ok {
			dto.PatientName = name
		} else if s.Patients != nil {
			patient, patientErr := s.Patients.Get(ctx, r.PatientID)
			if patientErr != nil && !errors.Is(patientErr, port.ErrNotFound) {
				return nil, patientErr
			}
			if patient != nil {
				dto.PatientName = patient.Name
			}
			patientNames[r.PatientID] = dto.PatientName
		}
		items = append(items, dto)
	}
	return &RegisterListDTO{Items: items}, nil
}

// GetRegister returns one bill or 404 NOT_FOUND.
func (s *Service) GetRegister(ctx context.Context, claims *auth.Claims, registerID string) (*RegisterDTO, error) {
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	registerID = strings.TrimSpace(registerID)
	if registerID == "" || s.Registers == nil {
		if s.Registers == nil && registerID != "" {
			return nil, errors.New("register store is not configured")
		}
		return nil, notFound()
	}
	r, err := s.Registers.GetRegister(ctx, registerID)
	if errors.Is(err, port.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, err
	}
	if !hasAnyPermScope(user, "outpatient", r.DepartmentID, domain.PermEncounterCreate, domain.PermEncounterCheckIn, domain.PermEncounterCancel) {
		return nil, forbidden()
	}
	dto := toRegisterDTO(*r)
	return &dto, nil
}

// CancelRegister moves NORMAL to BACK and records the current user.
// BACK again is 409 REGISTER_BACK_NOT_ALLOWED. Any other status is 409 REGISTER_NOT_CANCELLABLE.
// Refusals do not change status or the back columns.
func (s *Service) CancelRegister(ctx context.Context, claims *auth.Claims, registerID, requestID, ip string) (*RegisterDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	registerID = strings.TrimSpace(registerID)
	if registerID == "" {
		return nil, notFound()
	}
	if s.Registers == nil {
		return nil, errors.New("register store is not configured")
	}
	reg, err := s.Registers.GetRegister(ctx, registerID)
	if errors.Is(err, port.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, err
	}
	if !hasPermScope(user, domain.PermEncounterCancel, "outpatient", reg.DepartmentID) {
		e := forbidden()
		s.auditRegister(ctx, start, user, domain.ActionRegisterCancel, registerID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	if active, err := s.registerHasActivity(ctx, registerID); err != nil {
		return nil, err
	} else if active {
		e := &Error{Status: 409, Code: CodeRegisterNotCancellable, Message: msgRegisterActivityNotCancellable}
		s.auditRegister(ctx, start, user, domain.ActionRegisterCancel, registerID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	updated, err := s.Registers.CancelRegister(ctx, registerID, user.ID, s.now().UTC())
	if err != nil {
		switch {
		case errors.Is(err, port.ErrNotFound):
			return nil, notFound()
		case errors.Is(err, port.ErrRegisterAlreadyBack):
			e := &Error{Status: 409, Code: CodeRegisterBackNotAllowed, Message: msgRegisterBackNotAllowed}
			s.auditRegister(ctx, start, user, domain.ActionRegisterCancel, registerID, domain.OutcomeFail, e.Code, requestID, ip)
			return nil, e
		case errors.Is(err, port.ErrRegisterNotCancellable):
			e := &Error{Status: 409, Code: CodeRegisterNotCancellable, Message: msgRegisterNotCancellable}
			s.auditRegister(ctx, start, user, domain.ActionRegisterCancel, registerID, domain.OutcomeFail, e.Code, requestID, ip)
			return nil, e
		default:
			return nil, err
		}
	}
	dto := toRegisterDTO(*updated)
	s.auditRegister(ctx, start, user, domain.ActionRegisterCancel, dto.RegisterID, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

// registerHasActivity prevents a NORMAL visit from being withdrawn after a
// standalone registration-fee charge has entered the workflow. Existing
// submitted-order退号 behavior remains permissive until the order is charged.
func (s *Service) registerHasActivity(ctx context.Context, registerID string) (bool, error) {
	if s.Charges != nil {
		rows, err := s.Charges.SearchCharges(ctx, port.ChargeQuery{RegisterID: registerID, Limit: chargeSearchLimit})
		if err != nil {
			return false, err
		}
		for _, row := range rows {
			if row.OrderID == "" && registerFeeCharge(row) && row.Status != domain.ChargeStatusAbate {
				return true, nil
			}
		}
	}
	return false, nil
}

// CheckInRegister moves NORMAL or USED to DIAGNOSING.
// Any other status is 409 ILLEGAL_STATE_TRANSITION and does not change the row.
// POST requires encounter:check-in or admin:* (REGISTRAR, DOCTOR, SYSTEM). 2026-10-12
func (s *Service) CheckInRegister(ctx context.Context, claims *auth.Claims, registerID, requestID, ip string) (*RegisterDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	registerID = strings.TrimSpace(registerID)
	if registerID == "" {
		return nil, notFound()
	}
	if s.Registers == nil {
		return nil, errors.New("register store is not configured")
	}
	reg, err := s.Registers.GetRegister(ctx, registerID)
	if errors.Is(err, port.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, err
	}
	if !hasPermScope(user, domain.PermEncounterCheckIn, "outpatient", reg.DepartmentID) {
		e := forbidden()
		s.auditRegister(ctx, start, user, domain.ActionRegisterCheckIn, registerID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	if e := s.ensureRegisterNotExpired(reg); e != nil {
		s.auditRegister(ctx, start, user, domain.ActionRegisterCheckIn, registerID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	updated, err := s.Registers.CheckInRegister(ctx, registerID, user.ID, s.now().UTC())
	if err != nil {
		switch {
		case errors.Is(err, port.ErrNotFound):
			return nil, notFound()
		case errors.Is(err, port.ErrRegisterNotCheckIn):
			e := &Error{Status: 409, Code: CodeIllegalStateTransition, Message: msgIllegalStateTransition}
			s.auditRegister(ctx, start, user, domain.ActionRegisterCheckIn, registerID, domain.OutcomeFail, e.Code, requestID, ip)
			return nil, e
		default:
			return nil, err
		}
	}
	dto := toRegisterDTO(*updated)
	s.auditRegister(ctx, start, user, domain.ActionRegisterCheckIn, dto.RegisterID, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

func (s *Service) ensureRegisterNotExpired(reg *domain.Register) *Error {
	if reg == nil || s.RegisterExpireHours <= 0 || reg.RegisteredAt.IsZero() {
		return nil
	}
	if !s.now().UTC().Before(reg.RegisteredAt.UTC().Add(time.Duration(s.RegisterExpireHours) * time.Hour)) {
		return &Error{Status: 409, Code: CodeRegisterExpired, Message: msgRegisterExpired}
	}
	return nil
}

// CompleteRegister moves DIAGNOSING to DIAGNOSED and records the current user.
// The existing encounter:check-in permission covers both check-in and completion
// in the phase-one role matrix; no clinical policy or diagnosis payload is implied.
func (s *Service) CompleteRegister(ctx context.Context, claims *auth.Claims, registerID, requestID, ip string) (*RegisterDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	registerID = strings.TrimSpace(registerID)
	if registerID == "" {
		return nil, notFound()
	}
	if s.Registers == nil {
		return nil, errors.New("register store is not configured")
	}
	reg, err := s.Registers.GetRegister(ctx, registerID)
	if errors.Is(err, port.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, err
	}
	if !hasPermScope(user, domain.PermEncounterCheckIn, "outpatient", reg.DepartmentID) {
		e := forbidden()
		s.auditRegister(ctx, start, user, domain.ActionRegisterComplete, registerID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	updated, err := s.Registers.CompleteRegister(ctx, registerID, user.ID, s.now().UTC())
	if err != nil {
		switch {
		case errors.Is(err, port.ErrNotFound):
			return nil, notFound()
		case errors.Is(err, port.ErrRegisterNotComplete):
			e := &Error{Status: 409, Code: CodeIllegalStateTransition, Message: msgIllegalStateTransition}
			s.auditRegister(ctx, start, user, domain.ActionRegisterComplete, registerID, domain.OutcomeFail, e.Code, requestID, ip)
			return nil, e
		default:
			return nil, err
		}
	}
	dto := toRegisterDTO(*updated)
	s.auditRegister(ctx, start, user, domain.ActionRegisterComplete, dto.RegisterID, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

func (s *Service) requireRegisterMasters(ctx context.Context, patientID, departmentID, doctorID string) error {
	if _, err := s.Patients.Get(ctx, patientID); err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return &Error{Status: 404, Code: CodePatientNotFound, Message: msgPatientNotFound}
		}
		return err
	}
	dept, err := s.Registers.GetDepartment(ctx, departmentID)
	if err = gateActiveDepartment(dept, err); err != nil {
		return err
	}
	doc, err := s.Registers.GetDoctor(ctx, doctorID)
	if err := gateActiveDoctor(doc, err); err != nil {
		return err
	}
	return gateDoctorDepartment(doc, departmentID)
}

func gateDoctorDepartment(d *domain.Doctor, departmentID string) error {
	if d == nil || d.DepartmentID != departmentID {
		return &Error{Status: 409, Code: CodeDoctorDepartmentMismatch, Message: msgDoctorDepartmentMismatch}
	}
	return nil
}

func hasPerm(u *domain.User, want string) bool {
	return hasPermScope(u, want, permissionModule(want), "")
}

// hasPermAcrossScopes authorizes an endpoint that returns a global snapshot and
// therefore has no department parameter to evaluate. A scoped grant is still
// a valid grant for these read-only inventory endpoints; however, any matching
// explicit DENY must remain effective because the response is not department
// filtered and cannot safely apply a narrower deny at query time.
func hasPermAcrossScopes(u *domain.User, want, moduleCode string) bool {
	if u == nil {
		return false
	}
	if u.Super || u.Developer {
		return true
	}
	if !u.AccessControlLoaded {
		return hasPerm(u, want)
	}
	hasGrant := false
	for _, grant := range u.EffectivePermissions {
		if (grant.PermissionCode == want || grant.PermissionCode == domain.PermAdminAll) &&
			(grant.ModuleCode == "" || grant.ModuleCode == moduleCode) {
			hasGrant = true
			break
		}
	}
	if !hasGrant {
		return false
	}
	for _, denied := range u.DeniedPermissions {
		if (denied.PermissionCode == want || denied.PermissionCode == domain.PermAdminAll) &&
			(denied.ModuleCode == "" || denied.ModuleCode == moduleCode) {
			return false
		}
	}
	return true
}

func hasPermScope(u *domain.User, want, moduleCode, departmentID string) bool {
	if u == nil {
		return false
	}
	if u.Super || u.Developer {
		return true
	}
	if u.AccessControlLoaded {
		for _, denied := range u.DeniedPermissions {
			if (denied.PermissionCode == want || denied.PermissionCode == domain.PermAdminAll) && permissionScopeMatches(denied, moduleCode, departmentID) {
				return false
			}
		}
		for _, permission := range u.EffectivePermissions {
			if (permission.PermissionCode == want || permission.PermissionCode == domain.PermAdminAll) && permissionScopeMatches(permission, moduleCode, departmentID) {
				return true
			}
		}
		return false
	}
	for _, role := range u.Roles {
		for _, perm := range domain.RolePermissions[role] {
			if perm == domain.PermAdminAll || perm == want {
				return true
			}
		}
	}
	return false
}

func permissionModule(permission string) string {
	if strings.HasPrefix(permission, "admin:") || strings.HasPrefix(permission, "inventory:") || strings.HasPrefix(permission, "audit:") {
		return "master"
	}
	return "outpatient"
}

func permissionScopeMatches(grant domain.PermissionGrant, moduleCode, departmentID string) bool {
	return (grant.ModuleCode == "" || grant.ModuleCode == moduleCode) && (grant.DepartmentID == "" || grant.DepartmentID == departmentID)
}

func hasAnyPermModule(u *domain.User, moduleCode string, permissions ...string) bool {
	for _, permission := range permissions {
		if hasPermScope(u, permission, moduleCode, "") {
			return true
		}
	}
	if u == nil || !u.AccessControlLoaded {
		return false
	}
	for _, grant := range u.EffectivePermissions {
		if grant.ModuleCode != "" && grant.ModuleCode != moduleCode {
			continue
		}
		for _, permission := range permissions {
			if (grant.PermissionCode == permission || grant.PermissionCode == domain.PermAdminAll) && hasPermScope(u, permission, moduleCode, grant.DepartmentID) {
				return true
			}
		}
	}
	return false
}

func hasAnyPermScope(u *domain.User, moduleCode, departmentID string, permissions ...string) bool {
	for _, permission := range permissions {
		if hasPermScope(u, permission, moduleCode, departmentID) {
			return true
		}
	}
	return false
}

func normalizeRegister(in CreateRegisterInput) (registerCanon, []string) {
	c := registerCanon{
		PatientID:    strings.TrimSpace(in.PatientID),
		DepartmentID: strings.TrimSpace(in.DepartmentID),
		DoctorID:     strings.TrimSpace(in.DoctorID),
		VisitDate:    strings.TrimSpace(in.VisitDate),
		RegisterSort: derefTrim(in.RegisterSort),
	}
	var fields []string
	if c.PatientID == "" || len(c.PatientID) > 64 {
		fields = append(fields, "patientId")
	}
	if c.DepartmentID == "" || len(c.DepartmentID) > 64 {
		fields = append(fields, "departmentId")
	}
	if c.DoctorID == "" || len(c.DoctorID) > 64 {
		fields = append(fields, "doctorId")
	}
	if _, ok := parseVisitDate(c.VisitDate); !ok {
		fields = append(fields, "visitDate")
	}
	if len(c.RegisterSort) > 32 {
		fields = append(fields, "registerSort")
	}
	cents, ok := feeCents(in.RegisterFee)
	if !ok {
		fields = append(fields, "registerFee")
	}
	c.FeeCents = cents
	return c, fields
}

func parseVisitDate(s string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil || t.Format("2006-01-02") != s {
		return time.Time{}, false
	}
	return t, true
}

func feeCents(v *float64) (int64, bool) {
	if v == nil {
		return 0, true
	}
	f := *v
	if math.IsNaN(f) || math.IsInf(f, 0) || f < 0 || f > maxRegisterFee {
		return 0, false
	}
	cents := math.Round(f * 100)
	if cents < 0 || cents > 999999999999999999 {
		return 0, false
	}
	return int64(cents), true
}

func hashRegister(c registerCanon) string {
	b, _ := json.Marshal(c)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (s *Service) lookupRegisterIdem(key, hash string) (*RegisterDTO, error) {
	s.idemMu.Lock()
	defer s.idemMu.Unlock()
	hit, ok := s.idem[idemScopeRegisterCreate+"\x00"+key]
	if !ok {
		return nil, nil
	}
	if hit.hash != hash {
		return nil, &Error{Status: 409, Code: CodeIdempotencyConflict, Message: msgIdempotencyConflict}
	}
	var dto RegisterDTO
	if err := json.Unmarshal(hit.body, &dto); err != nil {
		return nil, err
	}
	return &dto, nil
}

func (s *Service) saveRegisterIdem(key, hash string, dto RegisterDTO) (*RegisterDTO, error) {
	body, err := json.Marshal(dto)
	if err != nil {
		return nil, err
	}
	s.idemMu.Lock()
	defer s.idemMu.Unlock()
	if s.idem == nil {
		s.idem = map[string]idemHit{}
	}
	k := idemScopeRegisterCreate + "\x00" + key
	if hit, ok := s.idem[k]; ok {
		if hit.hash != hash {
			return nil, &Error{Status: 409, Code: CodeIdempotencyConflict, Message: msgIdempotencyConflict}
		}
		var prev RegisterDTO
		if err := json.Unmarshal(hit.body, &prev); err != nil {
			return nil, err
		}
		return &prev, nil
	}
	s.idem[k] = idemHit{hash: hash, body: body}
	return &dto, nil
}

func (s *Service) auditRegister(ctx context.Context, start time.Time, user *domain.User, action, resourceID, outcome, errorCode, requestID, ip string) {
	actor := ""
	roles := ""
	if user != nil {
		actor = user.ID
		roles = strings.Join(user.RoleCodes(), ",")
	}
	level := "info"
	if outcome != domain.OutcomeSuccess {
		level = "warn"
	}
	event := s.enrichBusinessAudit(ctx, domain.AuditEvent{
		ActorID:      actor,
		ActorRoles:   roles,
		Action:       action,
		ResourceType: "Register",
		ResourceID:   resourceID,
		Outcome:      outcome,
		ErrorCode:    errorCode,
		RequestID:    requestID,
		ClientIP:     ip,
	})
	s.record(ctx, start, event, level)
}

func toRegisterDTO(r domain.Register) RegisterDTO {
	return RegisterDTO{
		RegisterID:      r.RegisterID,
		RegisterNo:      r.RegisterNo,
		PatientID:       r.PatientID,
		DepartmentID:    r.DepartmentID,
		DoctorID:        r.DoctorID,
		RegisterSort:    strPtrOrNil(r.RegisterSort),
		VisitDate:       dateString(r.VisitDate),
		Status:          r.Status,
		RegisterFee:     float64(r.RegisterFeeCents) / 100,
		RegisterCharged: r.RegisterCharged,
		RegisteredAt:    r.RegisteredAt.UTC().Format(time.RFC3339),
		BackOperatorID:  strPtrOrNil(r.BackOperatorID),
		BackOperatedAt:  timePtrOrNil(r.BackOperatedAt),
	}
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func timePtrOrNil(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}
