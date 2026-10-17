package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

const (
	CodeDiagnoseNotCommittable = "DIAGNOSE_NOT_COMMITTABLE"
	CodeDiagnoseNotAbatable    = "DIAGNOSE_NOT_ABATABLE"
	CodeDiagnoseContentTooLong = "DIAGNOSE_CONTENT_TOO_LONG"
	msgDiagnoseNotCommittable  = "当前状态不可提交诊断"
	msgDiagnoseNotAbatable     = "当前状态不可作废诊断"
	msgDiagnoseContentTooLong  = "诊断内容过长"
	diagnoseSearchLimit        = 100
)

type CreateDiagnoseInput struct {
	Content string `json:"content"`
}

type DiagnoseDTO struct {
	DiagnoseID  string `json:"diagnoseId"`
	EncounterID string `json:"encounterId"`
	Status      string `json:"status"`
	Content     string `json:"content"`
}

type DiagnoseListDTO struct {
	Items []DiagnoseDTO `json:"items"`
}

func (s *Service) CreateDiagnose(ctx context.Context, claims *auth.Claims, registerID string, in CreateDiagnoseInput, requestID, ip string) (*DiagnoseDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	registerID = strings.TrimSpace(registerID)
	content := strings.TrimSpace(in.Content)
	if registerID == "" {
		return nil, notFound()
	}
	if len([]rune(content)) > 4000 {
		e := &Error{Status: 400, Code: CodeDiagnoseContentTooLong, Message: msgDiagnoseContentTooLong}
		s.auditDiagnose(ctx, start, user, domain.ActionDiagnoseCreate, "", domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	if s.Registers == nil || s.Diagnoses == nil {
		return nil, errors.New("diagnosis store is not configured")
	}
	reg, err := s.Registers.GetRegister(ctx, registerID)
	if errors.Is(err, port.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, err
	}
	if !hasPermScope(user, domain.PermDiagnoseWrite, "outpatient", reg.DepartmentID) {
		e := forbidden()
		s.auditDiagnose(ctx, start, user, domain.ActionDiagnoseCreate, "", domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	if reg.Status == domain.RegisterStatusBack {
		e := &Error{Status: 409, Code: CodeIllegalStateTransition, Message: msgIllegalStateTransition}
		s.auditDiagnose(ctx, start, user, domain.ActionDiagnoseCreate, "", domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	if e := s.ensureRegisterNotExpired(reg); e != nil {
		s.auditDiagnose(ctx, start, user, domain.ActionDiagnoseCreate, "", domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	id, err := newUUID()
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	d := domain.Diagnose{DiagnoseID: id, RegisterID: registerID, Status: domain.DiagnoseStatusSave, Content: content, CreatedAt: now, CreatedBy: user.ID, UpdatedAt: now, UpdatedBy: user.ID}
	if err := s.Diagnoses.CreateDiagnose(ctx, d); err != nil {
		return nil, err
	}
	dto := toDiagnoseDTO(d)
	s.auditDiagnose(ctx, start, user, domain.ActionDiagnoseCreate, id, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

func (s *Service) ListDiagnoses(ctx context.Context, claims *auth.Claims, registerID string) (*DiagnoseListDTO, error) {
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	registerID = strings.TrimSpace(registerID)
	if registerID == "" || s.Registers == nil || s.Diagnoses == nil {
		return nil, notFound()
	}
	reg, err := s.Registers.GetRegister(ctx, registerID)
	if errors.Is(err, port.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, err
	}
	if !hasPermScope(user, domain.PermDiagnoseWrite, "outpatient", reg.DepartmentID) {
		return nil, forbidden()
	}
	rows, err := s.Diagnoses.ListDiagnoses(ctx, port.DiagnoseQuery{RegisterID: registerID, Limit: diagnoseSearchLimit})
	if err != nil {
		return nil, err
	}
	items := make([]DiagnoseDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toDiagnoseDTO(row))
	}
	return &DiagnoseListDTO{Items: items}, nil
}

func (s *Service) CommitDiagnose(ctx context.Context, claims *auth.Claims, diagnoseID, requestID, ip string) (*DiagnoseDTO, error) {
	return s.transitionDiagnose(ctx, claims, diagnoseID, requestID, ip, domain.ActionDiagnoseCommit, func(id, operator string, at time.Time) (*domain.Diagnose, error) {
		return s.Diagnoses.CommitDiagnose(ctx, id, operator, at)
	}, CodeDiagnoseNotCommittable, msgDiagnoseNotCommittable)
}

func (s *Service) AbateDiagnose(ctx context.Context, claims *auth.Claims, diagnoseID, requestID, ip string) (*DiagnoseDTO, error) {
	return s.transitionDiagnose(ctx, claims, diagnoseID, requestID, ip, domain.ActionDiagnoseAbate, func(id, operator string, at time.Time) (*domain.Diagnose, error) {
		return s.Diagnoses.AbateDiagnose(ctx, id, operator, at)
	}, CodeDiagnoseNotAbatable, msgDiagnoseNotAbatable)
}

func (s *Service) transitionDiagnose(ctx context.Context, claims *auth.Claims, diagnoseID, requestID, ip, action string, transition func(string, string, time.Time) (*domain.Diagnose, error), code, message string) (*DiagnoseDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	diagnoseID = strings.TrimSpace(diagnoseID)
	if diagnoseID == "" || s.Diagnoses == nil || s.Registers == nil {
		return nil, notFound()
	}
	d, err := s.Diagnoses.GetDiagnose(ctx, diagnoseID)
	if errors.Is(err, port.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, err
	}
	reg, err := s.Registers.GetRegister(ctx, d.RegisterID)
	if errors.Is(err, port.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, err
	}
	if !hasPermScope(user, domain.PermDiagnoseWrite, "outpatient", reg.DepartmentID) {
		e := forbidden()
		s.auditDiagnose(ctx, start, user, action, diagnoseID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	if e := s.ensureRegisterNotExpired(reg); e != nil {
		s.auditDiagnose(ctx, start, user, action, diagnoseID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	updated, err := transition(diagnoseID, user.ID, s.now().UTC())
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, notFound()
		}
		if errors.Is(err, port.ErrDiagnoseNotCommittable) || errors.Is(err, port.ErrDiagnoseNotAbatable) {
			e := &Error{Status: 409, Code: code, Message: message}
			s.auditDiagnose(ctx, start, user, action, diagnoseID, domain.OutcomeFail, e.Code, requestID, ip)
			return nil, e
		}
		return nil, err
	}
	dto := toDiagnoseDTO(*updated)
	s.auditDiagnose(ctx, start, user, action, diagnoseID, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

func (s *Service) auditDiagnose(ctx context.Context, start time.Time, user *domain.User, action, resourceID, outcome, errorCode, requestID, ip string) {
	actor, roles := "", ""
	if user != nil {
		actor = user.ID
		roles = strings.Join(user.RoleCodes(), ",")
	}
	level := "info"
	if outcome != domain.OutcomeSuccess {
		level = "warn"
	}
	s.record(ctx, start, domain.AuditEvent{ActorID: actor, ActorRoles: roles, Action: action, ResourceType: "Diagnose", ResourceID: resourceID, Outcome: outcome, ErrorCode: errorCode, RequestID: requestID, ClientIP: ip}, level)
}

func toDiagnoseDTO(d domain.Diagnose) DiagnoseDTO {
	return DiagnoseDTO{DiagnoseID: d.DiagnoseID, EncounterID: d.RegisterID, Status: d.Status, Content: d.Content}
}
