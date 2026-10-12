package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

const (
	CodeNotFound            = "NOT_FOUND"
	CodeForbidden           = "FORBIDDEN"
	CodeDuplicateIDNumber   = "DUPLICATE_ID_NUMBER"
	CodeDuplicateCardNo     = "DUPLICATE_CARD_NO"
	CodeIdempotencyConflict = "IDEMPOTENCY_KEY_CONFLICT"

	msgNotFound            = "资源不存在"
	msgForbidden           = "无权限"
	msgDuplicateIDNumber   = "该身份证号已存在，不能重复建档"
	msgDuplicateCardNo     = "卡号已存在，不能重复建档"
	msgIdempotencyConflict = "幂等键与已有请求不一致"

	idemScopePatientCreate = "patient.create"
	patientSearchLimit     = 100
)

// shanghai is the calendar used for "today" on birthday checks and generated numbers.
// The clock value is still stored in UTC. 2026-10-12
var shanghai = time.FixedZone("Asia/Shanghai", 8*60*60)

// CreatePatientInput is OpenAPI CreatePatientRequest. Address is a pointer so JSON null is empty.
// IdempotencyKey comes from the Idempotency-Key header, not the body.
type CreatePatientInput struct {
	Name           string  `json:"name"`
	Sex            string  `json:"sex"`
	Birthday       string  `json:"birthday"`
	IDType         string  `json:"idType"`
	IDNumber       string  `json:"idNumber"`
	CardNo         string  `json:"cardNo"`
	Phone          string  `json:"phone"`
	Address        *string `json:"address"`
	FareIdentity   string  `json:"fareIdentity"`
	IdempotencyKey string  `json:"-"`
}

// PatientDTO is OpenAPI PatientDto. Success responses are not wrapped in the error envelope.
type PatientDTO struct {
	PatientID    string `json:"patientId"`
	HospPkNo     string `json:"hospPkNo"`
	PatientNo    string `json:"patientNo"`
	CardNo       string `json:"cardNo"`
	Name         string `json:"name"`
	Sex          string `json:"sex"`
	Birthday     string `json:"birthday"`
	IDNumber     string `json:"idNumber"`
	Phone        string `json:"phone"`
	Address      string `json:"address"`
	FareIdentity string `json:"fareIdentity"`
	CardStatus   string `json:"cardStatus"`
	Status       string `json:"status"`
	CreatedAt    string `json:"createdAt"`
}

// PatientListDTO is the GET /patients body.
type PatientListDTO struct {
	Items []PatientDTO `json:"items"`
}

// PatientSearch filters are AND-ed. Name and address are substrings; the others are exact.
type PatientSearch struct {
	CardNo   string
	IDNumber string
	Name     string
	HospPkNo string
	Address  string
}

type patientCanon struct {
	Name         string `json:"name"`
	Sex          string `json:"sex"`
	Birthday     string `json:"birthday"`
	IDType       string `json:"idType"`
	IDNumber     string `json:"idNumber"`
	CardNo       string `json:"cardNo"`
	Phone        string `json:"phone"`
	Address      string `json:"address"`
	FareIdentity string `json:"fareIdentity"`
}

type idemHit struct {
	hash string
	body []byte
}

// CreatePatient registers one MPI row.
// Required: name, sex (M|F|U), birthday, fareIdentity.
// Birthday must be after 1900-01-01 and on or before the Asia/Shanghai calendar date of the service clock.
// The SQL CHECK only enforces birthday > 1900-01-01; ≤ today is applied here.
// A non-empty idNumber is unique by itself (002_patient ux_patient_id_number), not by idType+idNumber.
// Empty idNumber may repeat. cardNo is unique only when non-empty. Conflicts are refused and never merged.
// POST requires patient:create or admin:* (REGISTRAR, DOCTOR, CASHIER, SYSTEM). Roles are read from the
// user store, same freshness as GET /auth/me. GET allows any authenticated active user.
// Idempotency-Key is process-local. 002_patient has no idempotency table, so keys are not written to SQL.
func (s *Service) CreatePatient(ctx context.Context, claims *auth.Claims, in CreatePatientInput, requestID, ip string) (*PatientDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !canCreatePatient(user) {
		e := forbidden()
		s.auditPatient(ctx, start, user, "", domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	now := s.now().UTC()
	norm, fields := normalizePatient(in, shanghaiDate(now))
	if in.IdempotencyKey != "" && !validIdempotencyKey(in.IdempotencyKey) {
		fields = append(fields, "idempotencyKey")
	}
	if len(fields) > 0 {
		return nil, &Error{Status: 400, Code: CodeValidation, Message: msgValidation, Details: map[string]any{"fields": fields}}
	}
	hash := hashCanon(norm)
	if in.IdempotencyKey != "" {
		if dto, err := s.lookupIdem(in.IdempotencyKey, hash); err != nil || dto != nil {
			return dto, err
		}
	}
	if s.Patients == nil {
		return nil, errors.New("patient store is not configured")
	}

	cardStatus := domain.CardStatusNotSent
	if norm.CardNo != "" {
		cardStatus = domain.CardStatusNormal
	}
	bday, _ := parseBirthday(norm.Birthday, shanghaiDate(now))

	var created domain.Patient
	var createErr error
	for attempt := 0; attempt < 5; attempt++ {
		id, err := newUUID()
		if err != nil {
			return nil, err
		}
		hosp, err := newBusinessNo("H", now)
		if err != nil {
			return nil, err
		}
		pno, err := newBusinessNo("P", now)
		if err != nil {
			return nil, err
		}
		p := domain.Patient{
			PatientID:    id,
			HospPkNo:     hosp,
			PatientNo:    pno,
			CardNo:       norm.CardNo,
			Name:         norm.Name,
			Sex:          norm.Sex,
			Birthday:     bday,
			IDType:       norm.IDType,
			IDNumber:     norm.IDNumber,
			Phone:        norm.Phone,
			Address:      norm.Address,
			FareIdentity: norm.FareIdentity,
			CardStatus:   cardStatus,
			Status:       domain.PatientStatusActive,
			CreatedAt:    now,
			CreatedBy:    user.ID,
			UpdatedAt:    now,
			UpdatedBy:    user.ID,
		}
		createErr = s.Patients.Create(ctx, p)
		if createErr == nil {
			created = p
			break
		}
		if errors.Is(createErr, port.ErrDuplicateBusinessNo) {
			continue
		}
		break
	}
	if createErr != nil {
		switch {
		case errors.Is(createErr, port.ErrDuplicateIDNumber):
			e := &Error{Status: 409, Code: CodeDuplicateIDNumber, Message: msgDuplicateIDNumber, Details: map[string]any{"fields": []string{"idNumber"}}}
			s.auditPatient(ctx, start, user, "", domain.OutcomeFail, e.Code, requestID, ip)
			return nil, e
		case errors.Is(createErr, port.ErrDuplicateCardNo):
			e := &Error{Status: 409, Code: CodeDuplicateCardNo, Message: msgDuplicateCardNo, Details: map[string]any{"fields": []string{"cardNo"}}}
			s.auditPatient(ctx, start, user, "", domain.OutcomeFail, e.Code, requestID, ip)
			return nil, e
		default:
			return nil, createErr
		}
	}

	dto := toPatientDTO(created)
	if in.IdempotencyKey != "" {
		stored, err := s.saveIdem(in.IdempotencyKey, hash, dto)
		if err != nil {
			return nil, err
		}
		if stored != nil {
			dto = *stored
		}
	}
	s.auditPatient(ctx, start, user, dto.PatientID, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

// SearchPatients returns an AND of the given filters. With no filters the list is empty.
// Name and address match by substring; cardNo, idNumber, and hospPkNo are exact.
func (s *Service) SearchPatients(ctx context.Context, claims *auth.Claims, q PatientSearch) (*PatientListDTO, error) {
	if _, err := s.actor(ctx, claims); err != nil {
		return nil, err
	}
	q.CardNo = strings.TrimSpace(q.CardNo)
	q.IDNumber = strings.TrimSpace(q.IDNumber)
	q.Name = strings.TrimSpace(q.Name)
	q.HospPkNo = strings.TrimSpace(q.HospPkNo)
	q.Address = strings.TrimSpace(q.Address)
	empty := PatientListDTO{Items: []PatientDTO{}}
	if q.CardNo == "" && q.IDNumber == "" && q.Name == "" && q.HospPkNo == "" && q.Address == "" {
		return &empty, nil
	}
	if s.Patients == nil {
		return nil, errors.New("patient store is not configured")
	}
	rows, err := s.Patients.Search(ctx, port.PatientQuery{
		CardNo: q.CardNo, IDNumber: q.IDNumber, Name: q.Name, HospPkNo: q.HospPkNo, Address: q.Address,
		Limit: patientSearchLimit,
	})
	if err != nil {
		return nil, err
	}
	items := make([]PatientDTO, 0, len(rows))
	for _, p := range rows {
		items = append(items, toPatientDTO(p))
	}
	return &PatientListDTO{Items: items}, nil
}

// GetPatient returns one record or 404 NOT_FOUND.
func (s *Service) GetPatient(ctx context.Context, claims *auth.Claims, patientID string) (*PatientDTO, error) {
	if _, err := s.actor(ctx, claims); err != nil {
		return nil, err
	}
	patientID = strings.TrimSpace(patientID)
	if patientID == "" {
		return nil, notFound()
	}
	if s.Patients == nil {
		return nil, errors.New("patient store is not configured")
	}
	p, err := s.Patients.Get(ctx, patientID)
	if errors.Is(err, port.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, err
	}
	dto := toPatientDTO(*p)
	return &dto, nil
}

func (s *Service) actor(ctx context.Context, claims *auth.Claims) (*domain.User, error) {
	if claims == nil || strings.TrimSpace(claims.Subject) == "" || s.Users == nil {
		return nil, Unauthorized()
	}
	user, err := s.Users.FindByID(ctx, claims.Subject)
	if errors.Is(err, port.ErrNotFound) {
		return nil, TokenInvalid()
	}
	if err != nil {
		return nil, err
	}
	if !user.Active() {
		return nil, &Error{Status: 403, Code: CodeAccountDisabled, Message: msgAccountDisabled}
	}
	return user, nil
}

func canCreatePatient(u *domain.User) bool {
	if u == nil {
		return false
	}
	for _, role := range u.Roles {
		for _, perm := range domain.RolePermissions[role] {
			if perm == domain.PermPatientCreate || perm == domain.PermAdminAll {
				return true
			}
		}
	}
	return false
}

func normalizePatient(in CreatePatientInput, today time.Time) (patientCanon, []string) {
	c := patientCanon{
		Name:         strings.TrimSpace(in.Name),
		Sex:          strings.TrimSpace(in.Sex),
		Birthday:     strings.TrimSpace(in.Birthday),
		IDType:       strings.TrimSpace(in.IDType),
		IDNumber:     strings.TrimSpace(in.IDNumber),
		CardNo:       strings.TrimSpace(in.CardNo),
		Phone:        strings.TrimSpace(in.Phone),
		Address:      derefTrim(in.Address),
		FareIdentity: strings.TrimSpace(in.FareIdentity),
	}
	var fields []string
	if c.Name == "" || utf8.RuneCountInString(c.Name) > 64 {
		fields = append(fields, "name")
	}
	if c.Sex != domain.SexMale && c.Sex != domain.SexFemale && c.Sex != domain.SexUnknown {
		fields = append(fields, "sex")
	}
	if _, ok := parseBirthday(c.Birthday, today); !ok {
		fields = append(fields, "birthday")
	}
	if len(c.IDType) > 16 {
		fields = append(fields, "idType")
	}
	if len(c.IDNumber) > 32 {
		fields = append(fields, "idNumber")
	}
	if len(c.CardNo) > 32 {
		fields = append(fields, "cardNo")
	}
	if len(c.Phone) > 32 {
		fields = append(fields, "phone")
	}
	if utf8.RuneCountInString(c.Address) > 256 {
		fields = append(fields, "address")
	}
	if c.FareIdentity == "" || len(c.FareIdentity) > 32 {
		fields = append(fields, "fareIdentity")
	}
	return c, fields
}

func derefTrim(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func parseBirthday(s string, today time.Time) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil || t.Format("2006-01-02") != s {
		return time.Time{}, false
	}
	floor := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
	if !t.After(floor) || t.After(today) {
		return time.Time{}, false
	}
	return t, true
}

func shanghaiDate(now time.Time) time.Time {
	t := now.In(shanghai)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func validIdempotencyKey(k string) bool {
	if k == "" || len(k) > 128 {
		return false
	}
	for _, c := range k {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_':
		default:
			return false
		}
	}
	return true
}

func hashCanon(c patientCanon) string {
	b, _ := json.Marshal(c)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (s *Service) lookupIdem(key, hash string) (*PatientDTO, error) {
	s.idemMu.Lock()
	defer s.idemMu.Unlock()
	hit, ok := s.idem[idemScopePatientCreate+"\x00"+key]
	if !ok {
		return nil, nil
	}
	if hit.hash != hash {
		return nil, &Error{Status: 409, Code: CodeIdempotencyConflict, Message: msgIdempotencyConflict}
	}
	var dto PatientDTO
	if err := json.Unmarshal(hit.body, &dto); err != nil {
		return nil, err
	}
	return &dto, nil
}

func (s *Service) saveIdem(key, hash string, dto PatientDTO) (*PatientDTO, error) {
	body, err := json.Marshal(dto)
	if err != nil {
		return nil, err
	}
	s.idemMu.Lock()
	defer s.idemMu.Unlock()
	if s.idem == nil {
		s.idem = map[string]idemHit{}
	}
	k := idemScopePatientCreate + "\x00" + key
	if hit, ok := s.idem[k]; ok {
		if hit.hash != hash {
			return nil, &Error{Status: 409, Code: CodeIdempotencyConflict, Message: msgIdempotencyConflict}
		}
		var prev PatientDTO
		if err := json.Unmarshal(hit.body, &prev); err != nil {
			return nil, err
		}
		return &prev, nil
	}
	s.idem[k] = idemHit{hash: hash, body: body}
	return &dto, nil
}

func (s *Service) auditPatient(ctx context.Context, start time.Time, user *domain.User, resourceID, outcome, errorCode, requestID, ip string) {
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
		Action:       domain.ActionPatientCreate,
		ResourceType: "Patient",
		ResourceID:   resourceID,
		Outcome:      outcome,
		ErrorCode:    errorCode,
		RequestID:    requestID,
		ClientIP:     ip,
	}, level)
}

func toPatientDTO(p domain.Patient) PatientDTO {
	return PatientDTO{
		PatientID:    p.PatientID,
		HospPkNo:     p.HospPkNo,
		PatientNo:    p.PatientNo,
		CardNo:       p.CardNo,
		Name:         p.Name,
		Sex:          p.Sex,
		Birthday:     dateString(p.Birthday),
		IDNumber:     p.IDNumber,
		Phone:        p.Phone,
		Address:      p.Address,
		FareIdentity: p.FareIdentity,
		CardStatus:   p.CardStatus,
		Status:       p.Status,
		CreatedAt:    p.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func dateString(t time.Time) string {
	y, m, d := t.Date()
	return fmt.Sprintf("%04d-%02d-%02d", y, int(m), d)
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func newBusinessNo(prefix string, now time.Time) (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return prefix + now.In(shanghai).Format("20060102") + "-" + strings.ToUpper(hex.EncodeToString(b[:])), nil
}

func notFound() *Error {
	return &Error{Status: 404, Code: CodeNotFound, Message: msgNotFound}
}

func forbidden() *Error {
	return &Error{Status: 403, Code: CodeForbidden, Message: msgForbidden}
}
