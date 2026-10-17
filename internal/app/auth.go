package app

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/logging"
	"github.com/cznUSER/B-HIS/internal/port"
)

const (
	CodeValidation               = "VALIDATION_ERROR"
	CodeInvalidCredentials       = "INVALID_CREDENTIALS"
	CodeTokenExpired             = "TOKEN_EXPIRED"
	CodeTokenInvalid             = "TOKEN_INVALID"
	CodeUnauthorized             = "UNAUTHORIZED"
	CodeAccountDisabled          = "ACCOUNT_DISABLED"
	CodeAccountLocked            = "ACCOUNT_LOCKED"
	CodePasswordChangeRequired   = "PASSWORD_CHANGE_REQUIRED"
	CodePasswordChangeNotAllowed = "PASSWORD_CHANGE_NOT_ALLOWED"
	CodeCurrentPasswordInvalid   = "CURRENT_PASSWORD_INVALID"
	CodeRateLimited              = "AUTH_RATE_LIMITED"
	CodeAuditUnavailable         = "USER_AUDIT_UNAVAILABLE"
	CodeUserAdminBusy            = "USER_ADMIN_BUSY"

	msgValidation         = "字段校验失败"
	msgInvalidCredentials = "用户名或密码错误"
	msgTokenExpired       = "accessToken 已过期"
	msgTokenInvalid       = "accessToken 无效或已注销"
	msgUnauthorized       = "未认证"
	msgAccountDisabled    = "账号已停用"
	msgAccountLocked      = "账号已锁定，请稍后再试"
	msgRateLimited        = "登录失败过频，请稍后再试"
	msgAuditUnavailable   = "用户变更未完成，审计记录暂时不可用，请重试"
	msgUserAdminBusy      = "用户权限正在被其他管理员修改，请稍后重试"
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
	ID                   string   `json:"id"`
	Username             string   `json:"username"`
	DisplayName          string   `json:"displayName"`
	Roles                []string `json:"roles"`
	Permissions          []string `json:"permissions"`
	DeniedPermissions    []string `json:"deniedPermissions"`
	Scopes               []string `json:"scopes"`
	AccessControlLoaded  bool     `json:"accessControlLoaded"`
	System               bool     `json:"system"`
	Super                bool     `json:"super"`
	Developer            bool     `json:"developer"`
	MustChangePassword   bool     `json:"mustChangePassword"`
	CannotModifyPassword bool     `json:"cannotModifyPassword"`
}

// Service is the login use case. Stores are interfaces; the driver is chosen in main.
type Service struct {
	Users               port.UserStore
	Revocations         port.RevocationStore
	Audit               port.AuditStore
	Tokens              *auth.Issuer
	Limiter             *auth.Limiter
	Log                 *logging.Logger
	TTL                 time.Duration
	RegisterExpireHours int
	Now                 func() time.Time
	Patients            port.PatientStore
	Registers           port.RegisterStore
	Diagnoses           port.DiagnoseStore
	Orders              port.OrderStore
	Charges             port.ChargeStore
	Dispenses           port.DispenseStore
	InventoryBatches    port.InventoryBatchStore
	Master              port.MasterStore
	Dashboard           port.DashboardStore

	// idemMu guards the process-local fallback for write scopes whose store does not
	// yet provide an atomic persistent idempotency contract.
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
		if user != nil && s.Users != nil {
			if _, err := s.Users.RecordLoginFailure(ctx, user.ID, now, 3, 15*time.Minute); err != nil && !errors.Is(err, port.ErrNotFound) {
				return nil, err
			}
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
	if accountLocked(user, now) {
		return nil, &Error{Status: 423, Code: CodeAccountLocked, Message: msgAccountLocked}
	}
	if err := s.Users.RecordSuccessfulLogin(ctx, user.ID, ip, now); err != nil {
		return nil, err
	}
	user.LastLoginAt = now.UTC()
	user.LastLoginIP = ip
	user.FailedLoginCount = 0
	user.AccountLocked = false
	user.LockUntil = time.Time{}

	token, _, _, err := s.Tokens.Issue(*user, now)
	if err != nil {
		return nil, err
	}
	if s.Limiter != nil {
		s.Limiter.Success(ip, username)
	}
	dto := s.userDTO(ctx, user)
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
	if s.Users == nil {
		return nil, &Error{Status: 401, Code: CodeTokenInvalid, Message: msgTokenInvalid}
	}
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
	if accountLocked(user, s.now()) {
		return nil, &Error{Status: 423, Code: CodeAccountLocked, Message: msgAccountLocked}
	}
	if claims.AuthVersion != user.AuthVersion {
		return nil, &Error{Status: 401, Code: CodeTokenInvalid, Message: msgTokenInvalid}
	}
	claims.MustChangePassword = user.MustChangePassword
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
	if accountLocked(user, s.now()) {
		return nil, &Error{Status: 423, Code: CodeAccountLocked, Message: msgAccountLocked}
	}
	if user.MustChangePassword {
		return nil, passwordChangeRequired()
	}
	dto := s.userDTO(ctx, user)
	return &dto, nil
}

// ChangePassword completes an administrator reset. Updating auth_version revokes
// every token issued before the password change without requiring token enumeration.
func (s *Service) ChangePassword(ctx context.Context, claims *auth.Claims, currentPassword, newPassword, requestID, ip string) error {
	start := time.Now()
	if claims == nil || !claims.MustChangePassword || strings.TrimSpace(claims.Subject) == "" || s.Users == nil {
		return passwordChangeRequired()
	}
	user, err := s.Users.FindByID(ctx, claims.Subject)
	if errors.Is(err, port.ErrNotFound) {
		return &Error{Status: 401, Code: CodeTokenInvalid, Message: msgTokenInvalid}
	}
	if err != nil {
		return err
	}
	if !user.Active() {
		return &Error{Status: 403, Code: CodeAccountDisabled, Message: msgAccountDisabled}
	}
	if accountLocked(user, s.now()) {
		return &Error{Status: 423, Code: CodeAccountLocked, Message: msgAccountLocked}
	}
	if !user.MustChangePassword || user.AuthVersion != claims.AuthVersion {
		return &Error{Status: 401, Code: CodeTokenInvalid, Message: msgTokenInvalid}
	}
	if user.CannotModifyPassword {
		return &Error{Status: 403, Code: CodePasswordChangeNotAllowed, Message: "当前账号不允许自行修改密码"}
	}
	if !auth.CheckPassword(user.PasswordHash, currentPassword) {
		return &Error{Status: 400, Code: CodeCurrentPasswordInvalid, Message: "当前密码不正确"}
	}
	if len(newPassword) < 8 || len(newPassword) > 72 || newPassword == currentPassword {
		return validationFields([]string{"newPassword"})
	}
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.Users.ChangePassword(ctx, user.ID, hash, user.ID, s.now().UTC()); err != nil {
		return err
	}
	s.record(ctx, start, domain.AuditEvent{
		ActorID: user.ID, ActorRoles: strings.Join(user.RoleCodes(), ","),
		Action: "user.change_password", Outcome: domain.OutcomeSuccess,
		RequestID: requestID, ClientIP: ip, ResourceType: "User", ResourceID: user.ID,
	}, "info")
	return nil
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
	e = s.enrichBusinessAudit(ctx, e)
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
		ID:                   u.ID,
		Username:             u.Username,
		DisplayName:          u.DisplayName,
		Roles:                roles,
		Permissions:          []string{},
		DeniedPermissions:    []string{},
		Scopes:               []string{},
		AccessControlLoaded:  false,
		System:               u.System,
		Super:                u.Super,
		Developer:            u.Developer,
		MustChangePassword:   u.MustChangePassword,
		CannotModifyPassword: u.CannotModifyPassword,
	}
}

// userDTO adds the current access catalogue to the login/me response. The identity
// path remains usable before migration 002 is applied, so catalogue read failures
// intentionally leave the role-only response intact.
func (s *Service) userDTO(ctx context.Context, u *domain.User) UserDTO {
	dto := toDTO(u)
	accessStore, hasAccessStore := s.Master.(port.AccessStore)
	if !hasAccessStore || accessStore == nil {
		return dto
	}
	if err := s.loadEffectivePermissions(ctx, u); err != nil {
		return dto
	}
	dto.AccessControlLoaded = true
	permissionSet := make(map[string]bool)
	deniedSet := make(map[string]bool)
	scopeSet := make(map[string]bool)
	for _, grant := range u.EffectivePermissions {
		permissionSet[grant.PermissionCode] = true
		if grant.ModuleCode != "" || grant.DepartmentID != "" {
			scopeSet[grant.ModuleCode+":"+grant.DepartmentID] = true
		}
	}
	for _, grant := range u.DeniedPermissions {
		deniedSet[grant.PermissionCode] = true
	}
	for code := range permissionSet {
		dto.Permissions = append(dto.Permissions, code)
	}
	for code := range deniedSet {
		dto.DeniedPermissions = append(dto.DeniedPermissions, code)
	}
	for scope := range scopeSet {
		dto.Scopes = append(dto.Scopes, scope)
	}
	sort.Strings(dto.Permissions)
	sort.Strings(dto.DeniedPermissions)
	sort.Strings(dto.Scopes)
	return dto
}

func accountLocked(user *domain.User, now time.Time) bool {
	if user == nil || !user.AccountLocked {
		return false
	}
	return user.LockUntil.IsZero() || user.LockUntil.After(now)
}

func passwordChangeRequired() *Error {
	return &Error{Status: 428, Code: CodePasswordChangeRequired, Message: "请先修改管理员重置的密码"}
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
