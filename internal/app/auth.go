package app

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/logging"
	"github.com/cznUSER/B-HIS/internal/port"
)

const (
	CodeValidation         = "VALIDATION_ERROR"
	CodeInvalidCredentials = "INVALID_CREDENTIALS"
	CodeTokenExpired       = "TOKEN_EXPIRED"
	CodeTokenInvalid       = "TOKEN_INVALID"
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeAccountDisabled    = "ACCOUNT_DISABLED"
	CodeRateLimited        = "AUTH_RATE_LIMITED"

	msgValidation         = "字段校验失败"
	msgInvalidCredentials = "用户名或密码错误"
	msgTokenExpired       = "accessToken 已过期"
	msgTokenInvalid       = "accessToken 无效或已注销"
	msgUnauthorized       = "未认证"
	msgAccountDisabled    = "账号已停用"
	msgRateLimited        = "登录失败过频，请稍后再试"
)

// Error is a business failure with an H03 code. Details never include a password.
type Error struct {
	Status  int
	Code    string
	Message string
	Details map[string]any
}

func (e *Error) Error() string { return e.Code }

// LoginResult is OpenAPI LoginResponse. It is not wrapped in the error envelope.
type LoginResult struct {
	AccessToken string  `json:"accessToken"`
	TokenType   string  `json:"tokenType"`
	ExpiresIn   int64   `json:"expiresIn"`
	User        UserDTO `json:"user"`
}

// UserDTO is OpenAPI UserDto.
type UserDTO struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"displayName"`
	Roles       []string `json:"roles"`
}

// Service is the login use case. Stores are interfaces; the driver is chosen in main.
type Service struct {
	Users       port.UserStore
	Revocations port.RevocationStore
	Audit       port.AuditStore
	Tokens      *auth.Issuer
	Limiter     *auth.Limiter
	Log         *logging.Logger
	TTL         time.Duration
	Now         func() time.Time
	Patients    port.PatientStore
	Registers   port.RegisterStore
	Orders      port.OrderStore
	Charges     port.ChargeStore
	Dispenses   port.DispenseStore
	Master      port.MasterStore

	// idemMu guards process-local Idempotency-Key hits for patient create, register create, and order create.
	// 002_patient, 003_register, and 004_order have no idempotency table; keys are not persisted.
	idemMu sync.Mutex
	idem   map[string]idemHit
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Login authenticates username/password. Unknown user and wrong password share one message.
// ACCOUNT_DISABLED is returned only after the password matches, so a wrong password cannot enumerate disabled accounts.
func (s *Service) Login(ctx context.Context, username, password, requestID, ip string) (*LoginResult, error) {
	start := time.Now()
	username = strings.TrimSpace(username)
	if username == "" || password == "" || len(username) > 64 || len(password) > 72 {
		fields := make([]string, 0, 2)
		if username == "" || len(username) > 64 {
			fields = append(fields, "username")
		}
		if password == "" || len(password) > 72 {
			fields = append(fields, "password")
		}
		err := &Error{Status: 400, Code: CodeValidation, Message: msgValidation, Details: map[string]any{"fields": fields}}
		s.record(ctx, start, domain.AuditEvent{
			Action: domain.ActionLoginFailure, Outcome: domain.OutcomeFail,
			ErrorCode: err.Code, RequestID: requestID, ClientIP: ip, ResourceType: "User",
		}, "warn")
		return nil, err
	}
	now := s.now()
	if s.Limiter != nil && s.Limiter.Blocked(ip, username, now) {
		err := &Error{Status: 429, Code: CodeRateLimited, Message: msgRateLimited}
		s.record(ctx, start, domain.AuditEvent{
			Action: domain.ActionLoginFailure, Outcome: domain.OutcomeFail,
			ErrorCode: err.Code, RequestID: requestID, ClientIP: ip, ResourceType: "User",
		}, "warn")
		return nil, err
	}

	user, err := s.Users.FindByUsername(ctx, username)
	if err != nil && !errors.Is(err, port.ErrNotFound) {
		return nil, err
	}
	hash := auth.DummyHash
	if user != nil {
		hash = user.PasswordHash
	}
	if user == nil || !auth.CheckPassword(hash, password) {
		if s.Limiter != nil {
			s.Limiter.Fail(ip, username, now)
		}
		ae := domain.AuditEvent{
			Action: domain.ActionLoginFailure, Outcome: domain.OutcomeFail,
			ErrorCode: CodeInvalidCredentials, RequestID: requestID, ClientIP: ip, ResourceType: "User",
		}
		s.record(ctx, start, ae, "warn")
		return nil, &Error{Status: 401, Code: CodeInvalidCredentials, Message: msgInvalidCredentials}
	}
	if !user.Active() {
		if s.Limiter != nil {
			s.Limiter.Fail(ip, username, now)
		}
		s.record(ctx, start, domain.AuditEvent{
			ActorID: user.ID, ActorRoles: strings.Join(user.RoleCodes(), ","),
			Action: domain.ActionLoginFailure, Outcome: domain.OutcomeFail,
			ErrorCode: CodeAccountDisabled, RequestID: requestID, ClientIP: ip,
			ResourceType: "User", ResourceID: user.ID,
		}, "warn")
		return nil, &Error{Status: 403, Code: CodeAccountDisabled, Message: msgAccountDisabled}
	}

	token, _, _, err := s.Tokens.Issue(*user, now)
	if err != nil {
		return nil, err
	}
	if s.Limiter != nil {
		s.Limiter.Success(ip, username)
	}
	dto := toDTO(user)
	s.record(ctx, start, domain.AuditEvent{
		ActorID: user.ID, ActorRoles: strings.Join(dto.Roles, ","),
		Action: domain.ActionLoginSuccess, Outcome: domain.OutcomeSuccess,
		RequestID: requestID, ClientIP: ip, ResourceType: "User", ResourceID: user.ID,
	}, "info")
	return &LoginResult{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(s.TTL.Seconds()),
		User:        dto,
	}, nil
}

// Authenticate parses a raw bearer token (no "Bearer " prefix). Missing token is the caller's concern.
func (s *Service) Authenticate(ctx context.Context, raw string) (*auth.Claims, error) {
	claims, err := s.Tokens.Parse(raw, s.now())
	if err != nil {
		if errors.Is(err, auth.ErrExpired) {
			return nil, &Error{Status: 401, Code: CodeTokenExpired, Message: msgTokenExpired}
		}
		return nil, &Error{Status: 401, Code: CodeTokenInvalid, Message: msgTokenInvalid}
	}
	revoked, err := s.Revocations.IsRevoked(ctx, claims.ID)
	if err != nil {
		return nil, err
	}
	if revoked {
		return nil, &Error{Status: 401, Code: CodeTokenInvalid, Message: msgTokenInvalid}
	}
	return claims, nil
}

// Me returns the current user from the store (roles may have changed since issue).
func (s *Service) Me(ctx context.Context, claims *auth.Claims) (*UserDTO, error) {
	user, err := s.Users.FindByID(ctx, claims.Subject)
	if errors.Is(err, port.ErrNotFound) {
		return nil, &Error{Status: 401, Code: CodeTokenInvalid, Message: msgTokenInvalid}
	}
	if err != nil {
		return nil, err
	}
	if !user.Active() {
		return nil, &Error{Status: 403, Code: CodeAccountDisabled, Message: msgAccountDisabled}
	}
	dto := toDTO(user)
	return &dto, nil
}

// Logout blacklists jti until the access token would have expired.
func (s *Service) Logout(ctx context.Context, claims *auth.Claims, requestID, ip string) error {
	start := time.Now()
	exp := time.Time{}
	if claims.ExpiresAt != nil {
		exp = claims.ExpiresAt.Time
	}
	if err := s.Revocations.Revoke(ctx, claims.ID, claims.Subject, exp); err != nil {
		return err
	}
	s.record(ctx, start, domain.AuditEvent{
		ActorID: claims.Subject, ActorRoles: strings.Join(claims.Roles, ","),
		Action: domain.ActionLogout, Outcome: domain.OutcomeSuccess,
		RequestID: requestID, ClientIP: ip, ResourceType: "User", ResourceID: claims.Subject,
	}, "info")
	return nil
}

func (s *Service) record(ctx context.Context, start time.Time, e domain.AuditEvent, level string) {
	e.DurationMS = time.Since(start).Milliseconds()
	if e.OccurredAt.IsZero() {
		e.OccurredAt = s.now()
	}
	if e.ID == "" {
		if id, err := randomAuditID(); err == nil {
			e.ID = id
		}
	}
	if s.Audit != nil {
		_ = s.Audit.Append(ctx, e)
	}
	if s.Log != nil {
		s.Log.Log(logging.Event{
			Level:        level,
			RequestID:    e.RequestID,
			ActorID:      e.ActorID,
			Action:       e.Action,
			ResourceType: e.ResourceType,
			ResourceID:   e.ResourceID,
			Outcome:      e.Outcome,
			ErrorCode:    e.ErrorCode,
			DurationMS:   e.DurationMS,
		})
	}
}

func toDTO(u *domain.User) UserDTO {
	roles := u.RoleCodes()
	if roles == nil {
		roles = []string{}
	}
	return UserDTO{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Roles:       roles,
	}
}

func Unauthorized() *Error {
	return &Error{Status: 401, Code: CodeUnauthorized, Message: msgUnauthorized}
}

func TokenInvalid() *Error {
	return &Error{Status: 401, Code: CodeTokenInvalid, Message: msgTokenInvalid}
}

func Validation(reason string) *Error {
	details := map[string]any{}
	if reason != "" {
		details["reason"] = reason
	}
	return &Error{Status: 400, Code: CodeValidation, Message: msgValidation, Details: details}
}
