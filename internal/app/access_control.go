package app

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

const (
	adminPermissionRoles   = domain.PermAdminRole
	adminPermissionCatalog = domain.PermAdminPermission
	adminPermissionUsers   = domain.PermAdminUser
)

// Access-control DTOs intentionally expose codes and stable identifiers only. Passwords,
// database roles, and other implementation details never cross the HTTP boundary.
type AccessModuleDTO struct {
	ID                    string `json:"id"`
	Code                  string `json:"code"`
	Name                  string `json:"name"`
	Description           string `json:"description,omitempty"`
	Status                string `json:"status"`
	Hidden                bool   `json:"hidden"`
	DepartmentFunctions   string `json:"departmentFunctions"`
	NeedSpecifyDepartment bool   `json:"needSpecifyDepartment"`
	SecondPermissionID    int    `json:"secondPermissionId"`
	HomePageURL           string `json:"homePageUrl"`
}

type PermissionCategoryDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
}

type PermissionModeDTO struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type AccessPermissionDTO struct {
	Code                 string   `json:"code"`
	Name                 string   `json:"name"`
	Description          string   `json:"description,omitempty"`
	ModuleCode           string   `json:"moduleCode"`
	CategoryID           string   `json:"categoryId"`
	ModeCode             string   `json:"modeCode"`
	SecondPermissionCode string   `json:"secondPermissionCode,omitempty"`
	Actions              []string `json:"actions"`
	Status               string   `json:"status"`
	Hidden               bool     `json:"hidden"`
	DBRoleName           string   `json:"dbRoleName"`
	NeedSecondPermission bool     `json:"needSecondPermission"`
}

type CreateAccessPermissionInput struct {
	Code                 string   `json:"code"`
	Name                 string   `json:"name"`
	Description          string   `json:"description"`
	ModuleCode           string   `json:"moduleCode"`
	CategoryID           string   `json:"categoryId"`
	ModeCode             string   `json:"modeCode"`
	SecondPermissionCode string   `json:"secondPermissionCode"`
	Actions              []string `json:"actions"`
	Status               string   `json:"status"`
	Hidden               bool     `json:"hidden"`
	DBRoleName           string   `json:"dbRoleName"`
	NeedSecondPermission bool     `json:"needSecondPermission"`
}

type PatchAccessPermissionInput struct {
	Name                 *string   `json:"name"`
	Description          *string   `json:"description"`
	ModuleCode           *string   `json:"moduleCode"`
	CategoryID           *string   `json:"categoryId"`
	ModeCode             *string   `json:"modeCode"`
	SecondPermissionCode *string   `json:"secondPermissionCode"`
	Actions              *[]string `json:"actions"`
	Status               *string   `json:"status"`
	Hidden               *bool     `json:"hidden"`
	DBRoleName           *string   `json:"dbRoleName"`
	NeedSecondPermission *bool     `json:"needSecondPermission"`
}

type AccessRoleDTO struct {
	Code            string   `json:"code"`
	Name            string   `json:"name"`
	Description     string   `json:"description,omitempty"`
	ModuleCode      string   `json:"moduleCode,omitempty"`
	DepartmentID    string   `json:"departmentId,omitempty"`
	PermissionCodes []string `json:"permissionCodes"`
	Status          string   `json:"status"`
	Hidden          bool     `json:"hidden"`
	System          bool     `json:"system"`
	OrderSort       int      `json:"orderSort"`
	StandName       string   `json:"standName"`
}

type PermissionOverrideDTO struct {
	UserID         string `json:"userId"`
	PermissionCode string `json:"permissionCode"`
	ModuleCode     string `json:"moduleCode,omitempty"`
	DepartmentID   string `json:"departmentId,omitempty"`
	Effect         string `json:"effect"`
}

type AccessModuleListDTO struct {
	Items []AccessModuleDTO `json:"items"`
}
type PermissionCategoryListDTO struct {
	Items []PermissionCategoryDTO `json:"items"`
}
type PermissionModeListDTO struct {
	Items []PermissionModeDTO `json:"items"`
}
type AccessPermissionListDTO struct {
	Items []AccessPermissionDTO `json:"items"`
}
type AccessRoleListDTO struct {
	Items []AccessRoleDTO `json:"items"`
}
type PermissionOverrideListDTO struct {
	Items []PermissionOverrideDTO `json:"items"`
}

type CreateAccessRoleInput struct {
	Code            string   `json:"code"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	ModuleCode      string   `json:"moduleCode"`
	DepartmentID    string   `json:"departmentId"`
	PermissionCodes []string `json:"permissionCodes"`
	Status          *string  `json:"status"`
	Hidden          *bool    `json:"hidden"`
	System          *bool    `json:"system"`
	OrderSort       int      `json:"orderSort"`
	StandName       string   `json:"standName"`
}

type PatchAccessRoleInput struct {
	Name            *string   `json:"name"`
	Description     *string   `json:"description"`
	ModuleCode      *string   `json:"moduleCode"`
	DepartmentID    *string   `json:"departmentId"`
	PermissionCodes *[]string `json:"permissionCodes"`
	Status          *string   `json:"status"`
	Hidden          *bool     `json:"hidden"`
	System          *bool     `json:"system"`
	OrderSort       *int      `json:"orderSort"`
	StandName       *string   `json:"standName"`
}

type CreateAccessModuleInput struct {
	Code                  string `json:"code"`
	Name                  string `json:"name"`
	Description           string `json:"description"`
	Status                string `json:"status"`
	Hidden                bool   `json:"hidden"`
	DepartmentFunctions   string `json:"departmentFunctions"`
	NeedSpecifyDepartment bool   `json:"needSpecifyDepartment"`
	SecondPermissionID    int    `json:"secondPermissionId"`
	HomePageURL           string `json:"homePageUrl"`
}

type PatchAccessModuleInput struct {
	Name                  *string `json:"name"`
	Description           *string `json:"description"`
	Status                *string `json:"status"`
	Hidden                *bool   `json:"hidden"`
	DepartmentFunctions   *string `json:"departmentFunctions"`
	NeedSpecifyDepartment *bool   `json:"needSpecifyDepartment"`
	SecondPermissionID    *int    `json:"secondPermissionId"`
	HomePageURL           *string `json:"homePageUrl"`
}

type CreatePermissionCategoryInput struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type PatchPermissionCategoryInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
}

type CreatePermissionModeInput struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Status string `json:"status"`
}
type PatchPermissionModeInput struct {
	Name   *string `json:"name"`
	Status *string `json:"status"`
}

type PermissionOverrideInput struct {
	PermissionCode string `json:"permissionCode"`
	ModuleCode     string `json:"moduleCode"`
	DepartmentID   string `json:"departmentId"`
	Effect         string `json:"effect"`
}

type ReplacePermissionOverridesInput struct {
	Overrides []PermissionOverrideInput `json:"overrides"`
}

func (s *Service) accessStore() (port.AccessStore, error) {
	if s.Master == nil {
		return nil, errors.New("master store is not configured")
	}
	store, ok := s.Master.(port.AccessStore)
	if !ok || store == nil {
		return nil, errors.New("access store is not configured")
	}
	return store, nil
}

// accessAuditEvent prepares the success event passed to an atomic access-store
// mutation. The store persists this event in the same transaction as the
// role/permission/override write, so the service must not append it again.
func (s *Service) accessAuditEvent(ctx context.Context, start time.Time, user *domain.User, action, resourceType, resourceID, requestID, ip string) domain.AuditEvent {
	event := domain.AuditEvent{
		Action: action, ResourceType: resourceType,
		ResourceID: resourceID, Outcome: domain.OutcomeSuccess, RequestID: requestID, ClientIP: ip,
	}
	if user != nil {
		event.ActorID = user.ID
		event.ActorRoles = strings.Join(user.RoleCodes(), ",")
	}
	event = s.enrichBusinessAudit(ctx, event)
	event.DurationMS = time.Since(start).Milliseconds()
	event.OccurredAt = s.now()
	if event.ID == "" {
		if id, err := randomAuditID(); err == nil {
			event.ID = id
		}
	}
	return event
}

func (s *Service) requireAccessPermission(ctx context.Context, claims *auth.Claims, permission string, readFallbacks ...string) (*domain.User, error) {
	user, err := s.requireAdminPermission(ctx, claims, permission)
	if err == nil {
		return user, err
	}
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Code != CodeForbidden {
		return nil, err
	}
	for _, fallback := range readFallbacks {
		user, err = s.requireAdminPermission(ctx, claims, fallback)
		if err == nil {
			return user, nil
		}
		if !errors.As(err, &appErr) || appErr.Code != CodeForbidden {
			return nil, err
		}
	}
	return nil, forbidden()
}

func (s *Service) ListAccessModules(ctx context.Context, claims *auth.Claims) (*AccessModuleListDTO, error) {
	if _, err := s.requireAccessPermission(ctx, claims, adminPermissionCatalog, adminPermissionUsers, adminPermissionRoles); err != nil {
		return nil, err
	}
	store, err := s.accessStore()
	if err != nil {
		return nil, err
	}
	rows, err := store.ListAccessModules(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AccessModuleDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAccessModuleDTO(row))
	}
	return &AccessModuleListDTO{Items: out}, nil
}

func (s *Service) ListPermissionCategories(ctx context.Context, claims *auth.Claims) (*PermissionCategoryListDTO, error) {
	if _, err := s.requireAccessPermission(ctx, claims, adminPermissionCatalog, adminPermissionUsers); err != nil {
		return nil, err
	}
	store, err := s.accessStore()
	if err != nil {
		return nil, err
	}
	rows, err := store.ListPermissionCategories(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PermissionCategoryDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, PermissionCategoryDTO{ID: row.ID, Name: row.Name, Description: row.Description, Status: row.Status})
	}
	return &PermissionCategoryListDTO{Items: out}, nil
}

func (s *Service) ListPermissionModes(ctx context.Context, claims *auth.Claims) (*PermissionModeListDTO, error) {
	if _, err := s.requireAccessPermission(ctx, claims, adminPermissionCatalog, adminPermissionUsers); err != nil {
		return nil, err
	}
	store, err := s.accessStore()
	if err != nil {
		return nil, err
	}
	rows, err := store.ListPermissionModes(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PermissionModeDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, toPermissionModeDTO(row))
	}
	return &PermissionModeListDTO{Items: out}, nil
}

func (s *Service) CreateAccessModule(ctx context.Context, claims *auth.Claims, in CreateAccessModuleInput, requestID, ip string) (*AccessModuleDTO, error) {
	start := time.Now()
	actor, err := s.requireAdminPermissionAudit(ctx, claims, adminPermissionCatalog, start, "module.create", "AccessModule", in.Code, requestID, ip)
	if err != nil {
		return nil, err
	}
	row, fields := normalizeCreateModule(in)
	if len(fields) > 0 {
		e := validationFields(fields)
		s.auditMasterErr(ctx, start, actor, "module.create", "AccessModule", row.Code, requestID, ip, e)
		return nil, e
	}
	store, err := s.accessStore()
	if err != nil {
		return nil, err
	}
	if err = store.CreateAccessModule(ctx, row); err != nil {
		e := mapAccessWrite(err)
		s.auditMasterErr(ctx, start, actor, "module.create", "AccessModule", row.Code, requestID, ip, e)
		return nil, e
	}
	dto := toAccessModuleDTO(row)
	s.auditMaster(ctx, start, actor, "module.create", "AccessModule", row.Code, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

func (s *Service) UpdateAccessModule(ctx context.Context, claims *auth.Claims, code string, in PatchAccessModuleInput, requestID, ip string) (*AccessModuleDTO, error) {
	start := time.Now()
	code = strings.TrimSpace(code)
	actor, err := s.requireAdminPermissionAudit(ctx, claims, adminPermissionCatalog, start, "module.update", "AccessModule", code, requestID, ip)
	if err != nil {
		return nil, err
	}
	store, err := s.accessStore()
	if err != nil {
		return nil, err
	}
	rows, err := store.ListAccessModules(ctx)
	if err != nil {
		return nil, err
	}
	var row *domain.AccessModule
	for i := range rows {
		if rows[i].Code == code {
			copy := rows[i]
			row = &copy
			break
		}
	}
	if row == nil {
		return nil, notFound()
	}
	applyPatchModule(row, in)
	if fields := validateAccessModule(*row); len(fields) > 0 {
		e := validationFields(fields)
		s.auditMasterErr(ctx, start, actor, "module.update", "AccessModule", code, requestID, ip, e)
		return nil, e
	}
	if err = store.UpdateAccessModule(ctx, *row); err != nil {
		e := mapAccessWrite(err)
		s.auditMasterErr(ctx, start, actor, "module.update", "AccessModule", code, requestID, ip, e)
		return nil, e
	}
	dto := toAccessModuleDTO(*row)
	s.auditMaster(ctx, start, actor, "module.update", "AccessModule", code, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

func (s *Service) CreatePermissionCategory(ctx context.Context, claims *auth.Claims, in CreatePermissionCategoryInput, requestID, ip string) (*PermissionCategoryDTO, error) {
	start := time.Now()
	actor, err := s.requireAdminPermissionAudit(ctx, claims, adminPermissionCatalog, start, "category.create", "PermissionCategory", in.ID, requestID, ip)
	if err != nil {
		return nil, err
	}
	row := domain.PermissionCategory{ID: strings.TrimSpace(in.ID), Name: strings.TrimSpace(in.Name), Description: strings.TrimSpace(in.Description), Status: normalizeCatalogStatus(in.Status)}
	if fields := validatePermissionCategory(row); len(fields) > 0 {
		e := validationFields(fields)
		s.auditMasterErr(ctx, start, actor, "category.create", "PermissionCategory", row.ID, requestID, ip, e)
		return nil, e
	}
	store, err := s.accessStore()
	if err != nil {
		return nil, err
	}
	if err = store.CreatePermissionCategory(ctx, row); err != nil {
		e := mapAccessWrite(err)
		s.auditMasterErr(ctx, start, actor, "category.create", "PermissionCategory", row.ID, requestID, ip, e)
		return nil, e
	}
	dto := toPermissionCategoryDTO(row)
	s.auditMaster(ctx, start, actor, "category.create", "PermissionCategory", row.ID, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

func (s *Service) UpdatePermissionCategory(ctx context.Context, claims *auth.Claims, id string, in PatchPermissionCategoryInput, requestID, ip string) (*PermissionCategoryDTO, error) {
	start := time.Now()
	id = strings.TrimSpace(id)
	actor, err := s.requireAdminPermissionAudit(ctx, claims, adminPermissionCatalog, start, "category.update", "PermissionCategory", id, requestID, ip)
	if err != nil {
		return nil, err
	}
	store, err := s.accessStore()
	if err != nil {
		return nil, err
	}
	rows, err := store.ListPermissionCategories(ctx)
	if err != nil {
		return nil, err
	}
	var row *domain.PermissionCategory
	for i := range rows {
		if rows[i].ID == id {
			copy := rows[i]
			row = &copy
			break
		}
	}
	if row == nil {
		return nil, notFound()
	}
	if in.Name != nil {
		row.Name = strings.TrimSpace(*in.Name)
	}
	if in.Description != nil {
		row.Description = strings.TrimSpace(*in.Description)
	}
	if in.Status != nil {
		row.Status = normalizeCatalogStatus(*in.Status)
	}
	if fields := validatePermissionCategory(*row); len(fields) > 0 {
		return nil, validationFields(fields)
	}
	if err = store.UpdatePermissionCategory(ctx, *row); err != nil {
		e := mapAccessWrite(err)
		s.auditMasterErr(ctx, start, actor, "category.update", "PermissionCategory", id, requestID, ip, e)
		return nil, e
	}
	dto := toPermissionCategoryDTO(*row)
	s.auditMaster(ctx, start, actor, "category.update", "PermissionCategory", id, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

func (s *Service) CreatePermissionMode(ctx context.Context, claims *auth.Claims, in CreatePermissionModeInput, requestID, ip string) (*PermissionModeDTO, error) {
	start := time.Now()
	actor, err := s.requireAdminPermissionAudit(ctx, claims, adminPermissionCatalog, start, "mode.create", "PermissionMode", in.Code, requestID, ip)
	if err != nil {
		return nil, err
	}
	row := domain.PermissionMode{Code: strings.ToLower(strings.TrimSpace(in.Code)), Name: strings.TrimSpace(in.Name), Status: normalizeCatalogStatus(in.Status)}
	if fields := validatePermissionMode(row); len(fields) > 0 {
		return nil, validationFields(fields)
	}
	store, err := s.accessStore()
	if err != nil {
		return nil, err
	}
	if err = store.CreatePermissionMode(ctx, row); err != nil {
		e := mapAccessWrite(err)
		s.auditMasterErr(ctx, start, actor, "mode.create", "PermissionMode", row.Code, requestID, ip, e)
		return nil, e
	}
	dto := toPermissionModeDTO(row)
	s.auditMaster(ctx, start, actor, "mode.create", "PermissionMode", row.Code, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

func (s *Service) UpdatePermissionMode(ctx context.Context, claims *auth.Claims, code string, in PatchPermissionModeInput, requestID, ip string) (*PermissionModeDTO, error) {
	start := time.Now()
	code = strings.ToLower(strings.TrimSpace(code))
	actor, err := s.requireAdminPermissionAudit(ctx, claims, adminPermissionCatalog, start, "mode.update", "PermissionMode", code, requestID, ip)
	if err != nil {
		return nil, err
	}
	store, err := s.accessStore()
	if err != nil {
		return nil, err
	}
	rows, err := store.ListPermissionModes(ctx)
	if err != nil {
		return nil, err
	}
	var row *domain.PermissionMode
	for i := range rows {
		if rows[i].Code == code {
			copy := rows[i]
			row = &copy
			break
		}
	}
	if row == nil {
		return nil, notFound()
	}
	if in.Name != nil {
		row.Name = strings.TrimSpace(*in.Name)
	}
	if in.Status != nil {
		row.Status = normalizeCatalogStatus(*in.Status)
	}
	if fields := validatePermissionMode(*row); len(fields) > 0 {
		return nil, validationFields(fields)
	}
	if err = store.UpdatePermissionMode(ctx, *row); err != nil {
		e := mapAccessWrite(err)
		s.auditMasterErr(ctx, start, actor, "mode.update", "PermissionMode", code, requestID, ip, e)
		return nil, e
	}
	dto := toPermissionModeDTO(*row)
	s.auditMaster(ctx, start, actor, "mode.update", "PermissionMode", code, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

func normalizeCreateModule(in CreateAccessModuleInput) (domain.AccessModule, []string) {
	functions := int64(0)
	var err error
	if strings.TrimSpace(in.DepartmentFunctions) != "" {
		functions, err = strconv.ParseInt(strings.TrimSpace(in.DepartmentFunctions), 10, 64)
	}
	row := domain.AccessModule{Code: strings.ToLower(strings.TrimSpace(in.Code)), Name: strings.TrimSpace(in.Name), Description: strings.TrimSpace(in.Description), Status: normalizeCatalogStatus(in.Status), Hidden: in.Hidden, DepartmentFunctions: functions, NeedSpecifyDepartment: in.NeedSpecifyDepartment, SecondPermissionID: in.SecondPermissionID, HomePageURL: strings.TrimSpace(in.HomePageURL)}
	row.ID = "module-" + row.Code
	fields := validateAccessModule(row)
	if err != nil {
		fields = append(fields, "departmentFunctions")
	}
	return row, fields
}

func applyPatchModule(row *domain.AccessModule, in PatchAccessModuleInput) {
	if in.Name != nil {
		row.Name = strings.TrimSpace(*in.Name)
	}
	if in.Description != nil {
		row.Description = strings.TrimSpace(*in.Description)
	}
	if in.Status != nil {
		row.Status = normalizeCatalogStatus(*in.Status)
	}
	if in.Hidden != nil {
		row.Hidden = *in.Hidden
	}
	if in.DepartmentFunctions != nil {
		value, err := strconv.ParseInt(strings.TrimSpace(*in.DepartmentFunctions), 10, 64)
		if err == nil {
			row.DepartmentFunctions = value
		} else {
			row.DepartmentFunctions = -1
		}
	}
	if in.NeedSpecifyDepartment != nil {
		row.NeedSpecifyDepartment = *in.NeedSpecifyDepartment
	}
	if in.SecondPermissionID != nil {
		row.SecondPermissionID = *in.SecondPermissionID
	}
	if in.HomePageURL != nil {
		row.HomePageURL = strings.TrimSpace(*in.HomePageURL)
	}
}

func validateAccessModule(row domain.AccessModule) []string {
	var fields []string
	if row.Code == "" || len(row.Code) > 57 {
		fields = append(fields, "code")
	}
	if row.Name == "" || utf8.RuneCountInString(row.Name) > 128 {
		fields = append(fields, "name")
	}
	if len(row.Description) > 512 {
		fields = append(fields, "description")
	}
	if row.Status != domain.StatusActive && row.Status != domain.StatusInactive && row.Status != domain.StatusDisabled {
		fields = append(fields, "status")
	}
	if len(row.HomePageURL) > 512 {
		fields = append(fields, "moduleMetadata")
	}
	return fields
}

func validatePermissionCategory(row domain.PermissionCategory) []string {
	var fields []string
	if row.ID == "" || len(row.ID) > 64 {
		fields = append(fields, "id")
	}
	if row.Name == "" || utf8.RuneCountInString(row.Name) > 128 {
		fields = append(fields, "name")
	}
	if len(row.Description) > 512 {
		fields = append(fields, "description")
	}
	if row.Status != domain.StatusActive && row.Status != domain.StatusInactive && row.Status != domain.StatusDisabled {
		fields = append(fields, "status")
	}
	return fields
}
func validatePermissionMode(row domain.PermissionMode) []string {
	var fields []string
	if row.Code == "" || len(row.Code) > 32 {
		fields = append(fields, "code")
	}
	if row.Name == "" || utf8.RuneCountInString(row.Name) > 128 {
		fields = append(fields, "name")
	}
	if row.Status != domain.StatusActive && row.Status != domain.StatusInactive && row.Status != domain.StatusDisabled {
		fields = append(fields, "status")
	}
	return fields
}
func normalizeCatalogStatus(status string) string {
	status = strings.ToUpper(strings.TrimSpace(status))
	if status == "" {
		return domain.StatusActive
	}
	return status
}
func toAccessModuleDTO(row domain.AccessModule) AccessModuleDTO {
	return AccessModuleDTO{ID: row.ID, Code: row.Code, Name: row.Name, Description: row.Description, Status: row.Status, Hidden: row.Hidden, DepartmentFunctions: fmt.Sprint(row.DepartmentFunctions), NeedSpecifyDepartment: row.NeedSpecifyDepartment, SecondPermissionID: row.SecondPermissionID, HomePageURL: row.HomePageURL}
}
func toPermissionCategoryDTO(row domain.PermissionCategory) PermissionCategoryDTO {
	return PermissionCategoryDTO{ID: row.ID, Name: row.Name, Description: row.Description, Status: row.Status}
}

func toPermissionModeDTO(row domain.PermissionMode) PermissionModeDTO {
	return PermissionModeDTO{Code: row.Code, Name: row.Name, Status: row.Status}
}

func (s *Service) ListAccessPermissions(ctx context.Context, claims *auth.Claims, moduleCode, categoryID string) (*AccessPermissionListDTO, error) {
	if _, err := s.requireAccessPermission(ctx, claims, adminPermissionCatalog, adminPermissionUsers, adminPermissionRoles); err != nil {
		return nil, err
	}
	moduleCode, categoryID, fields := normalizeAccessFilter(moduleCode, categoryID)
	if len(fields) > 0 {
		return nil, validationFields(fields)
	}
	store, err := s.accessStore()
	if err != nil {
		return nil, err
	}
	rows, err := store.ListAccessPermissions(ctx, moduleCode, categoryID)
	if err != nil {
		return nil, err
	}
	out := make([]AccessPermissionDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAccessPermissionDTO(row))
	}
	return &AccessPermissionListDTO{Items: out}, nil
}

func (s *Service) CreateAccessPermission(ctx context.Context, claims *auth.Claims, in CreateAccessPermissionInput, requestID, ip string) (*AccessPermissionDTO, error) {
	start := time.Now()
	actor, err := s.requireAdminPermissionAudit(ctx, claims, adminPermissionCatalog, start, "permission.create", "AccessPermission", in.Code, requestID, ip)
	if err != nil {
		return nil, err
	}
	permission := domain.AccessPermission{
		Code: strings.TrimSpace(in.Code), Name: strings.TrimSpace(in.Name), Description: strings.TrimSpace(in.Description),
		ModuleCode: strings.TrimSpace(in.ModuleCode), CategoryID: strings.TrimSpace(in.CategoryID), ModeCode: strings.TrimSpace(in.ModeCode),
		SecondPermissionCode: strings.TrimSpace(in.SecondPermissionCode), Actions: normalizeActions(in.Actions), Status: strings.ToUpper(strings.TrimSpace(in.Status)), Hidden: in.Hidden,
		DBRoleName: strings.TrimSpace(in.DBRoleName), NeedSecondPermission: in.NeedSecondPermission,
	}
	if permission.Status == "" {
		permission.Status = domain.StatusActive
	}
	if fields := validateAccessPermission(permission); len(fields) > 0 {
		e := validationFields(fields)
		s.auditMasterErr(ctx, start, actor, "permission.create", "AccessPermission", permission.Code, requestID, ip, e)
		return nil, e
	}
	if permission.Hidden && !actor.Developer {
		e := forbidden()
		s.auditMaster(ctx, start, actor, "permission.create", "AccessPermission", permission.Code, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	store, err := s.accessStore()
	if err != nil {
		return nil, err
	}
	if fields := validatePermissionReferences(ctx, store, permission, true); len(fields) > 0 {
		e := validationFields(fields)
		s.auditMasterErr(ctx, start, actor, "permission.create", "AccessPermission", permission.Code, requestID, ip, e)
		return nil, e
	}
	atomic, hasAtomic := store.(port.AtomicAccessStore)
	if hasAtomic {
		err = atomic.CreateAccessPermissionWithAudit(ctx, permission, s.accessAuditEvent(ctx, start, actor, "permission.create", "AccessPermission", permission.Code, requestID, ip))
	} else {
		err = store.CreateAccessPermission(ctx, permission)
	}
	if err != nil {
		e := mapAccessWrite(err)
		s.auditMasterErr(ctx, start, actor, "permission.create", "AccessPermission", permission.Code, requestID, ip, e)
		return nil, e
	}
	dto := toAccessPermissionDTO(permission)
	if !hasAtomic {
		s.auditMaster(ctx, start, actor, "permission.create", "AccessPermission", permission.Code, domain.OutcomeSuccess, "", requestID, ip)
	}
	return &dto, nil
}

func (s *Service) UpdateAccessPermission(ctx context.Context, claims *auth.Claims, code string, in PatchAccessPermissionInput, requestID, ip string) (*AccessPermissionDTO, error) {
	start := time.Now()
	actor, err := s.requireAdminPermissionAudit(ctx, claims, adminPermissionCatalog, start, "permission.update", "AccessPermission", code, requestID, ip)
	if err != nil {
		return nil, err
	}
	code = strings.TrimSpace(code)
	store, err := s.accessStore()
	if err != nil {
		return nil, err
	}
	rows, err := store.ListAccessPermissions(ctx, "", "")
	if err != nil {
		return nil, err
	}
	var permission *domain.AccessPermission
	for i := range rows {
		if rows[i].Code == code {
			copy := rows[i]
			permission = &copy
			break
		}
	}
	if permission == nil {
		e := notFound()
		s.auditMasterErr(ctx, start, actor, "permission.update", "AccessPermission", code, requestID, ip, e)
		return nil, e
	}
	applyPatchAccessPermission(permission, in)
	if in.Hidden != nil && !actor.Developer {
		e := forbidden()
		s.auditMaster(ctx, start, actor, "permission.update", "AccessPermission", code, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	if fields := validateAccessPermission(*permission); len(fields) > 0 {
		e := validationFields(fields)
		s.auditMasterErr(ctx, start, actor, "permission.update", "AccessPermission", code, requestID, ip, e)
		return nil, e
	}
	if fields := validatePermissionReferences(ctx, store, *permission, false); len(fields) > 0 {
		e := validationFields(fields)
		s.auditMasterErr(ctx, start, actor, "permission.update", "AccessPermission", code, requestID, ip, e)
		return nil, e
	}
	atomic, hasAtomic := store.(port.AtomicAccessStore)
	if hasAtomic {
		err = atomic.UpdateAccessPermissionWithAudit(ctx, *permission, s.accessAuditEvent(ctx, start, actor, "permission.update", "AccessPermission", code, requestID, ip))
	} else {
		err = store.UpdateAccessPermission(ctx, *permission)
	}
	if err != nil {
		e := mapAccessWrite(err)
		s.auditMasterErr(ctx, start, actor, "permission.update", "AccessPermission", code, requestID, ip, e)
		return nil, e
	}
	dto := toAccessPermissionDTO(*permission)
	if !hasAtomic {
		s.auditMaster(ctx, start, actor, "permission.update", "AccessPermission", code, domain.OutcomeSuccess, "", requestID, ip)
	}
	return &dto, nil
}

func applyPatchAccessPermission(permission *domain.AccessPermission, in PatchAccessPermissionInput) {
	if in.Name != nil {
		permission.Name = strings.TrimSpace(*in.Name)
	}
	if in.Description != nil {
		permission.Description = strings.TrimSpace(*in.Description)
	}
	if in.ModuleCode != nil {
		permission.ModuleCode = strings.TrimSpace(*in.ModuleCode)
	}
	if in.CategoryID != nil {
		permission.CategoryID = strings.TrimSpace(*in.CategoryID)
	}
	if in.ModeCode != nil {
		permission.ModeCode = strings.TrimSpace(*in.ModeCode)
	}
	if in.SecondPermissionCode != nil {
		permission.SecondPermissionCode = strings.TrimSpace(*in.SecondPermissionCode)
	}
	if in.Actions != nil {
		permission.Actions = normalizeActions(*in.Actions)
	}
	if in.Status != nil {
		permission.Status = strings.ToUpper(strings.TrimSpace(*in.Status))
	}
	if in.Hidden != nil {
		permission.Hidden = *in.Hidden
	}
	if in.DBRoleName != nil {
		permission.DBRoleName = strings.TrimSpace(*in.DBRoleName)
	}
	if in.NeedSecondPermission != nil {
		permission.NeedSecondPermission = *in.NeedSecondPermission
	}
}

func normalizeActions(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, raw := range in {
		code := strings.ToUpper(strings.TrimSpace(raw))
		if code != "" && !seen[code] {
			seen[code] = true
			out = append(out, code)
		}
	}
	sort.Strings(out)
	return out
}

func validateAccessPermission(permission domain.AccessPermission) []string {
	var fields []string
	if permission.Code == "" || len(permission.Code) > 128 || strings.ContainsAny(permission.Code, " \t\r\n") {
		fields = append(fields, "code")
	}
	if permission.Name == "" || utf8.RuneCountInString(permission.Name) > 128 {
		fields = append(fields, "name")
	}
	if utf8.RuneCountInString(permission.Description) > 512 {
		fields = append(fields, "description")
	}
	if permission.ModuleCode == "" || len(permission.ModuleCode) > 64 {
		fields = append(fields, "moduleCode")
	}
	if len(permission.CategoryID) > 64 {
		fields = append(fields, "categoryId")
	}
	if len(permission.ModeCode) > 32 {
		fields = append(fields, "modeCode")
	}
	if len(permission.SecondPermissionCode) > 128 {
		fields = append(fields, "secondPermissionCode")
	}
	if len(permission.DBRoleName) > 128 {
		fields = append(fields, "dbRoleName")
	}
	if permission.Status != domain.StatusActive && permission.Status != domain.StatusInactive && permission.Status != domain.StatusDisabled {
		fields = append(fields, "status")
	}
	if len(permission.Actions) > 8 {
		fields = append(fields, "actions")
	}
	for _, action := range permission.Actions {
		if !strings.Contains("NMDSPE", action) || len(action) != 1 {
			fields = append(fields, "actions")
			break
		}
	}
	return fields
}

func validatePermissionReferences(ctx context.Context, store port.AccessStore, permission domain.AccessPermission, creating bool) []string {
	modules, err := store.ListAccessModules(ctx)
	if err != nil {
		return []string{"moduleCode"}
	}
	moduleExists := false
	for _, item := range modules {
		if item.Code == permission.ModuleCode && item.Status == domain.StatusActive {
			moduleExists = true
			break
		}
	}
	var fields []string
	if !moduleExists {
		fields = append(fields, "moduleCode")
	}
	if permission.CategoryID != "" {
		categories, err := store.ListPermissionCategories(ctx)
		if err != nil {
			fields = append(fields, "categoryId")
		} else {
			found := false
			for _, item := range categories {
				if item.ID == permission.CategoryID && item.Status == domain.StatusActive {
					found = true
					break
				}
			}
			if !found {
				fields = append(fields, "categoryId")
			}
		}
	}
	if permission.ModeCode != "" {
		modes, err := store.ListPermissionModes(ctx)
		if err != nil {
			fields = append(fields, "modeCode")
		} else {
			found := false
			for _, item := range modes {
				if item.Code == permission.ModeCode {
					found = true
					break
				}
			}
			if !found {
				fields = append(fields, "modeCode")
			}
		}
	}
	if permission.SecondPermissionCode != "" {
		rows, err := store.ListAccessPermissions(ctx, "", "")
		found := false
		if err == nil {
			for _, item := range rows {
				if item.Code == permission.SecondPermissionCode {
					found = true
					break
				}
			}
		}
		if !found || (creating && permission.SecondPermissionCode == permission.Code) {
			fields = append(fields, "secondPermissionCode")
		}
	}
	return fields
}

func toAccessPermissionDTO(permission domain.AccessPermission) AccessPermissionDTO {
	return AccessPermissionDTO{Code: permission.Code, Name: permission.Name, Description: permission.Description, ModuleCode: permission.ModuleCode,
		CategoryID: permission.CategoryID, ModeCode: permission.ModeCode, SecondPermissionCode: permission.SecondPermissionCode,
		Actions: append([]string(nil), permission.Actions...), Status: permission.Status, Hidden: permission.Hidden, DBRoleName: permission.DBRoleName, NeedSecondPermission: permission.NeedSecondPermission}
}

func (s *Service) ListAccessRoles(ctx context.Context, claims *auth.Claims) (*AccessRoleListDTO, error) {
	actor, err := s.requireAccessPermission(ctx, claims, adminPermissionRoles, adminPermissionUsers)
	if err != nil {
		return nil, err
	}
	store, err := s.accessStore()
	if err != nil {
		return nil, err
	}
	rows, err := store.ListAccessRoles(ctx)
	if err != nil {
		return nil, err
	}
	if !actor.Developer {
		filtered := rows[:0]
		for _, row := range rows {
			if !row.Hidden {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}
	return &AccessRoleListDTO{Items: accessRoleDTOs(rows)}, nil
}

func (s *Service) CreateAccessRole(ctx context.Context, claims *auth.Claims, in CreateAccessRoleInput, requestID, ip string) (*AccessRoleDTO, error) {
	start := time.Now()
	actor, err := s.requireAdminPermissionAudit(ctx, claims, adminPermissionRoles, start, "role.create", "AccessRole", in.Code, requestID, ip)
	if err != nil {
		return nil, err
	}
	role, fields := normalizeCreateAccessRole(in)
	if len(fields) > 0 {
		e := validationFields(fields)
		s.auditMasterErr(ctx, start, actor, "role.create", "AccessRole", role.Code, requestID, ip, e)
		return nil, e
	}
	if roleHasAdminPermission(role) && !canManageSystemAccess(actor) {
		e := forbidden()
		s.auditMasterErr(ctx, start, actor, "role.create", "AccessRole", role.Code, requestID, ip, e)
		return nil, e
	}
	if (role.Hidden || role.System) && !actor.Developer {
		e := forbidden()
		s.auditMaster(ctx, start, actor, "role.create", "AccessRole", role.Code, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	store, err := s.accessStore()
	if err != nil {
		s.auditMasterErr(ctx, start, actor, "role.create", "AccessRole", role.Code, requestID, ip, err)
		return nil, err
	}
	if fields := s.validateRoleReferences(ctx, store, role); len(fields) > 0 {
		e := validationFields(fields)
		s.auditMasterErr(ctx, start, actor, "role.create", "AccessRole", role.Code, requestID, ip, e)
		return nil, e
	}
	atomic, hasAtomic := store.(port.AtomicAccessStore)
	if hasAtomic {
		err = atomic.CreateAccessRoleWithAudit(ctx, role, s.accessAuditEvent(ctx, start, actor, "role.create", "AccessRole", role.Code, requestID, ip))
	} else {
		err = store.CreateAccessRole(ctx, role)
	}
	if err != nil {
		e := mapAccessWrite(err)
		s.auditMasterErr(ctx, start, actor, "role.create", "AccessRole", role.Code, requestID, ip, e)
		return nil, e
	}
	dto := toAccessRoleDTO(role)
	if !hasAtomic {
		s.auditMaster(ctx, start, actor, "role.create", "AccessRole", role.Code, domain.OutcomeSuccess, "", requestID, ip)
	}
	return &dto, nil
}

func (s *Service) UpdateAccessRole(ctx context.Context, claims *auth.Claims, code string, in PatchAccessRoleInput, requestID, ip string) (*AccessRoleDTO, error) {
	start := time.Now()
	actor, err := s.requireAdminPermissionAudit(ctx, claims, adminPermissionRoles, start, "role.update", "AccessRole", code, requestID, ip)
	if err != nil {
		return nil, err
	}
	code = strings.TrimSpace(code)
	if len(code) > 32 {
		return nil, validationFields([]string{"code"})
	}
	store, err := s.accessStore()
	if err != nil {
		return nil, err
	}
	rows, err := store.ListAccessRoles(ctx)
	if err != nil {
		return nil, err
	}
	var role *domain.AccessRole
	for i := range rows {
		if rows[i].Code == code {
			copy := rows[i]
			role = &copy
			break
		}
	}
	if role == nil {
		e := notFound()
		s.auditMasterErr(ctx, start, actor, "role.update", "AccessRole", code, requestID, ip, e)
		return nil, e
	}
	wasSystem := role.System
	wasHidden := role.Hidden
	applyPatchAccessRole(role, in)
	if roleHasAdminPermission(*role) && !canManageSystemAccess(actor) {
		e := forbidden()
		s.auditMasterErr(ctx, start, actor, "role.update", "AccessRole", code, requestID, ip, e)
		return nil, e
	}
	if (in.Hidden != nil || in.System != nil) && !actor.Developer {
		e := forbidden()
		s.auditMaster(ctx, start, actor, "role.update", "AccessRole", code, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	if (wasSystem || wasHidden) && !actor.Developer {
		e := forbidden()
		s.auditMaster(ctx, start, actor, "role.update", "AccessRole", code, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	fields := validateAccessRole(*role, false)
	if len(fields) > 0 {
		e := validationFields(fields)
		s.auditMasterErr(ctx, start, actor, "role.update", "AccessRole", code, requestID, ip, e)
		return nil, e
	}
	if wasSystem && (!role.System || role.Status != domain.StatusActive) {
		e := validationFields([]string{"system"})
		s.auditMasterErr(ctx, start, actor, "role.update", "AccessRole", code, requestID, ip, e)
		return nil, e
	}
	if fields := s.validateRoleReferences(ctx, store, *role); len(fields) > 0 {
		e := validationFields(fields)
		s.auditMasterErr(ctx, start, actor, "role.update", "AccessRole", code, requestID, ip, e)
		return nil, e
	}
	atomic, hasAtomic := store.(port.AtomicAccessStore)
	if hasAtomic {
		err = atomic.UpdateAccessRoleWithAudit(ctx, *role, s.accessAuditEvent(ctx, start, actor, "role.update", "AccessRole", code, requestID, ip))
	} else {
		err = store.UpdateAccessRole(ctx, *role)
	}
	if err != nil {
		e := mapAccessWrite(err)
		s.auditMasterErr(ctx, start, actor, "role.update", "AccessRole", code, requestID, ip, e)
		return nil, e
	}
	dto := toAccessRoleDTO(*role)
	if !hasAtomic {
		s.auditMaster(ctx, start, actor, "role.update", "AccessRole", code, domain.OutcomeSuccess, "", requestID, ip)
	}
	return &dto, nil
}

func (s *Service) ListUserPermissionOverrides(ctx context.Context, claims *auth.Claims, userID string) (*PermissionOverrideListDTO, error) {
	if _, err := s.requireAccessPermission(ctx, claims, adminPermissionUsers); err != nil {
		return nil, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" || s.Users == nil {
		return nil, notFound()
	}
	if _, err := s.Users.FindByID(ctx, userID); errors.Is(err, port.ErrNotFound) {
		return nil, notFound()
	} else if err != nil {
		return nil, err
	}
	store, err := s.accessStore()
	if err != nil {
		return nil, err
	}
	rows, err := store.ListUserPermissionOverrides(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &PermissionOverrideListDTO{Items: overrideDTOs(rows)}, nil
}

func (s *Service) ReplaceUserPermissionOverrides(ctx context.Context, claims *auth.Claims, userID string, in ReplacePermissionOverridesInput, requestID, ip string) (*PermissionOverrideListDTO, error) {
	start := time.Now()
	actor, err := s.requireAdminPermissionAudit(ctx, claims, adminPermissionUsers, start, "user.permission_override", "User", userID, requestID, ip)
	if err != nil {
		return nil, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" || s.Users == nil {
		e := notFound()
		s.auditMasterErr(ctx, start, actor, "user.permission_override", "User", userID, requestID, ip, e)
		return nil, e
	}
	if _, err := s.Users.FindByID(ctx, userID); errors.Is(err, port.ErrNotFound) {
		e := notFound()
		s.auditMasterErr(ctx, start, actor, "user.permission_override", "User", userID, requestID, ip, e)
		return nil, e
	} else if err != nil {
		return nil, err
	}
	overrides, fields := normalizeOverrides(userID, in.Overrides)
	if len(fields) > 0 {
		e := validationFields(fields)
		s.auditMasterErr(ctx, start, actor, "user.permission_override", "User", userID, requestID, ip, e)
		return nil, e
	}
	if !canManageSystemAccess(actor) {
		for _, override := range overrides {
			if isAdminPermissionCode(override.PermissionCode) && override.Effect == domain.PermissionAllow {
				e := forbidden()
				s.auditMasterErr(ctx, start, actor, "user.permission_override", "User", userID, requestID, ip, e)
				return nil, e
			}
		}
	}
	store, err := s.accessStore()
	if err != nil {
		return nil, err
	}
	if fields := s.validateOverrideReferences(ctx, store, overrides); len(fields) > 0 {
		e := validationFields(fields)
		s.auditMasterErr(ctx, start, actor, "user.permission_override", "User", userID, requestID, ip, e)
		return nil, e
	}
	atomic, hasAtomic := store.(port.AtomicAccessStore)
	if hasAtomic {
		err = atomic.ReplaceUserPermissionOverridesWithAudit(ctx, userID, overrides, s.accessAuditEvent(ctx, start, actor, "user.permission_override", "User", userID, requestID, ip))
	} else {
		err = store.ReplaceUserPermissionOverrides(ctx, userID, overrides)
	}
	if err != nil {
		s.auditMasterErr(ctx, start, actor, "user.permission_override", "User", userID, requestID, ip, err)
		return nil, err
	}
	if !hasAtomic {
		s.auditMaster(ctx, start, actor, "user.permission_override", "User", userID, domain.OutcomeSuccess, "", requestID, ip)
	}
	return &PermissionOverrideListDTO{Items: overrideDTOs(overrides)}, nil
}

func normalizeAccessFilter(moduleCode, categoryID string) (string, string, []string) {
	moduleCode = strings.TrimSpace(moduleCode)
	categoryID = strings.TrimSpace(categoryID)
	var fields []string
	if len(moduleCode) > 64 {
		fields = append(fields, "moduleCode")
	}
	if len(categoryID) > 64 {
		fields = append(fields, "categoryId")
	}
	return moduleCode, categoryID, fields
}

func normalizeCreateAccessRole(in CreateAccessRoleInput) (domain.AccessRole, []string) {
	role := domain.AccessRole{Code: strings.ToUpper(strings.TrimSpace(in.Code)), Name: strings.TrimSpace(in.Name), Description: strings.TrimSpace(in.Description), ModuleCode: strings.TrimSpace(in.ModuleCode), DepartmentID: strings.TrimSpace(in.DepartmentID), PermissionCodes: uniqueAccessStrings(in.PermissionCodes), Status: domain.StatusActive, OrderSort: in.OrderSort, StandName: strings.TrimSpace(in.StandName)}
	if in.Status != nil {
		role.Status = strings.TrimSpace(*in.Status)
	}
	if in.Hidden != nil {
		role.Hidden = *in.Hidden
	}
	if in.System != nil {
		role.System = *in.System
	}
	return role, validateAccessRole(role, true)
}

func applyPatchAccessRole(role *domain.AccessRole, in PatchAccessRoleInput) {
	if in.Name != nil {
		role.Name = strings.TrimSpace(*in.Name)
	}
	if in.Description != nil {
		role.Description = strings.TrimSpace(*in.Description)
	}
	if in.ModuleCode != nil {
		role.ModuleCode = strings.TrimSpace(*in.ModuleCode)
	}
	if in.DepartmentID != nil {
		role.DepartmentID = strings.TrimSpace(*in.DepartmentID)
	}
	if in.PermissionCodes != nil {
		role.PermissionCodes = uniqueAccessStrings(*in.PermissionCodes)
	}
	if in.Status != nil {
		role.Status = strings.TrimSpace(*in.Status)
	}
	if in.Hidden != nil {
		role.Hidden = *in.Hidden
	}
	if in.System != nil {
		role.System = *in.System
	}
	if in.OrderSort != nil {
		role.OrderSort = *in.OrderSort
	}
	if in.StandName != nil {
		role.StandName = strings.TrimSpace(*in.StandName)
	}
}

func validateAccessRole(role domain.AccessRole, create bool) []string {
	var fields []string
	if role.Code == "" || len(role.Code) > 32 {
		fields = append(fields, "code")
	}
	if role.Name == "" || utf8.RuneCountInString(role.Name) > 128 {
		fields = append(fields, "name")
	}
	if len(role.Description) > 512 {
		fields = append(fields, "description")
	}
	if len(role.StandName) > 128 {
		fields = append(fields, "standName")
	}
	if len(role.ModuleCode) > 64 {
		fields = append(fields, "moduleCode")
	}
	if len(role.DepartmentID) > 64 {
		fields = append(fields, "departmentId")
	}
	if role.Status != domain.StatusActive && role.Status != domain.StatusInactive && role.Status != domain.StatusDisabled {
		fields = append(fields, "status")
	}
	if len(role.PermissionCodes) > 512 {
		fields = append(fields, "permissionCodes")
	}
	return fields
}

func normalizeOverrides(userID string, in []PermissionOverrideInput) ([]domain.PermissionOverride, []string) {
	if len(in) > 512 {
		return nil, []string{"overrides"}
	}
	out := make([]domain.PermissionOverride, 0, len(in))
	var fields []string
	seen := make(map[string]struct{}, len(in))
	for i, item := range in {
		o := domain.PermissionOverride{UserID: userID, PermissionCode: strings.TrimSpace(item.PermissionCode), ModuleCode: strings.TrimSpace(item.ModuleCode), DepartmentID: strings.TrimSpace(item.DepartmentID), Effect: strings.ToUpper(strings.TrimSpace(item.Effect))}
		if o.PermissionCode == "" || len(o.PermissionCode) > 128 {
			fields = append(fields, "overrides["+strconv.Itoa(i)+"].permissionCode")
		}
		if len(o.ModuleCode) > 64 {
			fields = append(fields, "overrides["+strconv.Itoa(i)+"].moduleCode")
		}
		if len(o.DepartmentID) > 64 {
			fields = append(fields, "overrides["+strconv.Itoa(i)+"].departmentId")
		}
		if o.Effect != domain.PermissionAllow && o.Effect != domain.PermissionDeny {
			fields = append(fields, "overrides["+strconv.Itoa(i)+"].effect")
		}
		key := strings.Join([]string{o.PermissionCode, o.ModuleCode, o.DepartmentID}, "\x00")
		if _, ok := seen[key]; ok {
			fields = append(fields, "overrides["+strconv.Itoa(i)+"].permissionCode")
		} else {
			seen[key] = struct{}{}
		}
		out = append(out, o)
	}
	return out, fields
}

func isAdminPermissionCode(code string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(code)), "admin:")
}

func roleHasAdminPermission(role domain.AccessRole) bool {
	for _, code := range role.PermissionCodes {
		if isAdminPermissionCode(code) {
			return true
		}
	}
	return false
}

func mapAccessWrite(err error) error {
	if errors.Is(err, port.ErrDuplicateAccessCatalog) {
		return &Error{Status: 409, Code: "ACCESS_CATALOG_DUPLICATE", Message: "目录编码已存在"}
	}
	if errors.Is(err, port.ErrDuplicateAccessPermission) {
		return &Error{Status: 409, Code: "PERMISSION_CODE_DUPLICATE", Message: "权限编码已存在"}
	}
	if errors.Is(err, port.ErrDuplicateUsername) {
		return &Error{Status: 409, Code: CodeConflict, Message: "角色编码已存在"}
	}
	if errors.Is(err, port.ErrNotFound) {
		return notFound()
	}
	return err
}

func (s *Service) validateRoleReferences(ctx context.Context, store port.AccessStore, role domain.AccessRole) []string {
	permissions, err := store.ListAccessPermissions(ctx, "", "")
	if err != nil {
		return []string{"permissionCodes"}
	}
	byCode := make(map[string]domain.AccessPermission, len(permissions))
	for _, permission := range permissions {
		byCode[permission.Code] = permission
	}
	var fields []string
	for _, code := range role.PermissionCodes {
		permission, ok := byCode[code]
		if !ok || (role.ModuleCode != "" && permission.ModuleCode != role.ModuleCode) {
			fields = append(fields, "permissionCodes")
			break
		}
	}
	if role.DepartmentID != "" && s.Master != nil {
		if _, err := s.Master.GetDepartment(ctx, role.DepartmentID); errors.Is(err, port.ErrNotFound) {
			fields = append(fields, "departmentId")
		} else if err != nil {
			fields = append(fields, "departmentId")
		}
	}
	return fields
}

func (s *Service) validateOverrideReferences(ctx context.Context, store port.AccessStore, overrides []domain.PermissionOverride) []string {
	permissions, err := store.ListAccessPermissions(ctx, "", "")
	if err != nil {
		return []string{"overrides"}
	}
	byCode := make(map[string]domain.AccessPermission, len(permissions))
	for _, permission := range permissions {
		byCode[permission.Code] = permission
	}
	var fields []string
	for i, override := range overrides {
		permission, ok := byCode[override.PermissionCode]
		if !ok || (override.ModuleCode != "" && override.ModuleCode != permission.ModuleCode) {
			fields = append(fields, "overrides["+strconv.Itoa(i)+"].permissionCode")
		}
		if override.DepartmentID != "" && s.Master != nil {
			if _, err := s.Master.GetDepartment(ctx, override.DepartmentID); err != nil {
				fields = append(fields, "overrides["+strconv.Itoa(i)+"].departmentId")
			}
		}
	}
	return fields
}

func uniqueAccessStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, value := range in {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func accessRoleDTOs(rows []domain.AccessRole) []AccessRoleDTO {
	out := make([]AccessRoleDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAccessRoleDTO(row))
	}
	return out
}

func toAccessRoleDTO(row domain.AccessRole) AccessRoleDTO {
	return AccessRoleDTO{Code: row.Code, Name: row.Name, Description: row.Description, ModuleCode: row.ModuleCode, DepartmentID: row.DepartmentID, PermissionCodes: append([]string(nil), row.PermissionCodes...), Status: row.Status, Hidden: row.Hidden, System: row.System, OrderSort: row.OrderSort, StandName: row.StandName}
}

func overrideDTOs(rows []domain.PermissionOverride) []PermissionOverrideDTO {
	out := make([]PermissionOverrideDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, PermissionOverrideDTO{UserID: row.UserID, PermissionCode: row.PermissionCode, ModuleCode: row.ModuleCode, DepartmentID: row.DepartmentID, Effect: row.Effect})
	}
	return out
}
