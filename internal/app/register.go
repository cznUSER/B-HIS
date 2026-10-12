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
	CodePatientNotFound        = "PATIENT_NOT_FOUND"
	CodeDepartmentNotFound     = "DEPARTMENT_NOT_FOUND"
	CodeDoctorNotFound         = "DOCTOR_NOT_FOUND"
	CodeRegisterNotCancellable = "REGISTER_NOT_CANCELLABLE"
	CodeRegisterBackNotAllowed = "REGISTER_BACK_NOT_ALLOWED"
	CodeIllegalStateTransition = "ILLEGAL_STATE_TRANSITION"

	msgPatientNotFound        = "患者不存在"
	msgDepartmentNotFound     = "科室不存在"
	msgDoctorNotFound         = "医生不存在"
	msgRegisterNotCancellable = "当前状态不可退号"
	msgRegisterBackNotAllowed = "已退号，不能再次退号"
	msgIllegalStateTransition = "当前状态不允许该操作"

	idemScopeRegisterCreate = "register.create"
	registerSearchLimit     = 100
	maxRegisterFee          = 9999999999999999.99
)

// CreateRegisterInput is OpenAPI CreateRegisterRequest.
// RegisterFee nil means 0. IdempotencyKey comes from the header, not the body.
type CreateRegisterInput struct {
	PatientID      string   `json:"patientId"`
	DepartmentID   string   `json:"departmentId"`
	DoctorID       string   `json:"doctorId"`
	VisitDate      string   `json:"visitDate"`
	RegisterSort   *string  `json:"registerSort"`
	RegisterFee    *float64 `json:"registerFee"`
	IdempotencyKey string   `json:"-"`
}

// RegisterDTO is OpenAPI RegisterDto. Nullable fields are JSON null when empty.
type RegisterDTO struct {
	RegisterID      string  `json:"registerId"`
	RegisterNo      string  `json:"registerNo"`
	PatientID       string  `json:"patientId"`
	DepartmentID    string  `json:"departmentId"`
	DoctorID        string  `json:"doctorId"`
	RegisterSort    *string `json:"registerSort"`
	VisitDate       string  `json:"visitDate"`
	Status          string  `json:"status"`
	RegisterFee     float64 `json:"registerFee"`
	RegisterCharged bool    `json:"registerCharged"`
	RegisteredAt    string  `json:"registeredAt"`
	BackOperatorID  *string `json:"backOperatorId"`
	BackOperatedAt  *string `json:"backOperatedAt"`
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
// Idempotency-Key is process-local. 003_register has no idempotency table.
func (s *Service) CreateRegister(ctx context.Context, claims *auth.Claims, in CreateRegisterInput, requestID, ip string) (*RegisterDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !hasPerm(user, domain.PermEncounterCreate) {
		e := forbidden()
		s.auditRegister(ctx, start, user, domain.ActionRegisterCreate, "", domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	norm, fields := normalizeRegister(in)
	if in.IdempotencyKey != "" && !validIdempotencyKey(in.IdempotencyKey) {
		fields = append(fields, "idempotencyKey")
	}
	if len(fields) > 0 {
		return nil, &Error{Status: 400, Code: CodeValidation, Message: msgValidation, Details: map[string]any{"fields": fields}}
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
		createErr = s.Registers.CreateRegister(ctx, reg)
		if createErr == nil {
			created = reg
			break
		}
		if errors.Is(createErr, port.ErrDuplicateBusinessNo) {
			continue
		}
		break
	}
	if createErr != nil {
		return nil, createErr
	}
	dto := toRegisterDTO(created)
	if in.IdempotencyKey != "" {
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
	if _, err := s.actor(ctx, claims); err != nil {
		return nil, err
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
	for _, r := range rows {
		items = append(items, toRegisterDTO(r))
	}
	return &RegisterListDTO{Items: items}, nil
}

// GetRegister returns one bill or 404 NOT_FOUND.
func (s *Service) GetRegister(ctx context.Context, claims *auth.Claims, registerID string) (*RegisterDTO, error) {
	if _, err := s.actor(ctx, claims); err != nil {
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
	if !hasPerm(user, domain.PermEncounterCancel) {
		e := forbidden()
		s.auditRegister(ctx, start, user, domain.ActionRegisterCancel, strings.TrimSpace(registerID), domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	registerID = strings.TrimSpace(registerID)
	if registerID == "" {
		return nil, notFound()
	}
	if s.Registers == nil {
		return nil, errors.New("register store is not configured")
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

// CheckInRegister moves NORMAL or USED to DIAGNOSING.
// Any other status is 409 ILLEGAL_STATE_TRANSITION and does not change the row.
// POST requires encounter:check-in or admin:* (REGISTRAR, DOCTOR, SYSTEM). 2026-10-12
func (s *Service) CheckInRegister(ctx context.Context, claims *auth.Claims, registerID, requestID, ip string) (*RegisterDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !hasPerm(user, domain.PermEncounterCheckIn) {
		e := forbidden()
		s.auditRegister(ctx, start, user, domain.ActionRegisterCheckIn, strings.TrimSpace(registerID), domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	registerID = strings.TrimSpace(registerID)
	if registerID == "" {
		return nil, notFound()
	}
	if s.Registers == nil {
		return nil, errors.New("register store is not configured")
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

func (s *Service) requireRegisterMasters(ctx context.Context, patientID, departmentID, doctorID string) error {
	if _, err := s.Patients.Get(ctx, patientID); err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return &Error{Status: 404, Code: CodePatientNotFound, Message: msgPatientNotFound}
		}
		return err
	}
	ok, err := s.Registers.DepartmentExists(ctx, departmentID)
	if err != nil {
		return err
	}
	if !ok {
		return &Error{Status: 404, Code: CodeDepartmentNotFound, Message: msgDepartmentNotFound}
	}
	ok, err = s.Registers.DoctorExists(ctx, doctorID)
	if err != nil {
		return err
	}
	if !ok {
		return &Error{Status: 404, Code: CodeDoctorNotFound, Message: msgDoctorNotFound}
	}
	return nil
}

func hasPerm(u *domain.User, want string) bool {
	if u == nil {
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
	s.record(ctx, start, domain.AuditEvent{
		ActorID:      actor,
		ActorRoles:   roles,
		Action:       action,
		ResourceType: "Register",
		ResourceID:   resourceID,
		Outcome:      outcome,
		ErrorCode:    errorCode,
		RequestID:    requestID,
		ClientIP:     ip,
	}, level)
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
