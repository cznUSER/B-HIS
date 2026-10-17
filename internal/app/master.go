package app

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/cznUSER/B-HIS/internal/store/seed"
)

// Master-data maintenance (P1-H04-MASTER-API). 2026-10-13

const (
	CodeDeptCodeDuplicate           = "DEPT_CODE_DUPLICATE"
	CodeDepartmentInactive          = "DEPARTMENT_INACTIVE"
	CodeDoctorCodeDuplicate         = "DOCTOR_CODE_DUPLICATE"
	CodeDoctorInactive              = "DOCTOR_INACTIVE"
	CodeItemNoDuplicate             = "ITEM_NO_DUPLICATE"
	CodeCatalogInactive             = "CATALOG_INACTIVE"
	CodeInvalidPrice                = "INVALID_PRICE"
	CodeAdjustResultNegative        = "ADJUST_RESULT_NEGATIVE"
	CodeNotDrugSKU                  = "NOT_DRUG_SKU"
	CodeConflict                    = "CONFLICT"
	CodeUsernameDuplicate           = "USERNAME_DUPLICATE"
	CodeLastSystemAdmin             = "LAST_SYSTEM_ADMIN"
	CodeUserAccessBundleUnsupported = "USER_ACCESS_BUNDLE_UNSUPPORTED"
	CodeDictCodeDuplicate           = "DICT_CODE_DUPLICATE"
	CodeDictTypeReadonly            = "DICT_TYPE_READONLY"

	msgDepartmentInactive          = "科室已停用，不可新挂挂号或开单"
	msgDoctorInactive              = "医生已停用，不可新选"
	msgCatalogInactive             = "目录项已停用，不可新开单"
	msgDeptCodeDuplicate           = "科室代码已存在"
	msgDoctorCodeDuplicate         = "医生代码已存在"
	msgItemNoDuplicate             = "目录编码已存在"
	msgInvalidPrice                = "单价不合法"
	msgAdjustResultNegative        = "调整后库存不能为负"
	msgNotDrugSKU                  = "非药品或无库存 SKU"
	msgConflict                    = "库存已被他人修改"
	msgUsernameDuplicate           = "登录名已存在"
	msgLastSystemAdmin             = "不能停用或移除最后一个系统管理员"
	msgUserAccessBundleUnsupported = "当前存储不支持用户与权限整屏原子保存"
	msgDictCodeDuplicate           = "码表编码已存在"
	msgDictTypeReadonly            = "该字典为只读枚举"

	dictRegisterSort = "register_sort"
	maxQtyMilli      = int64(999999999999999999)
)

type DepartmentDTO struct {
	DepartmentID string `json:"departmentId"`
	DeptCode     string `json:"deptCode"`
	DeptName     string `json:"deptName"`
	Status       string `json:"status"`
	SortNo       int    `json:"sortNo"`
	CreatedAt    string `json:"createdAt,omitempty"`
	UpdatedAt    string `json:"updatedAt,omitempty"`
}

type DepartmentListDTO struct {
	Items []DepartmentDTO `json:"items"`
}

type CreateDepartmentInput struct {
	DeptCode string  `json:"deptCode"`
	DeptName string  `json:"deptName"`
	Status   *string `json:"status"`
	SortNo   *int    `json:"sortNo"`
}

type PatchDepartmentInput struct {
	DeptCode *string `json:"deptCode"`
	DeptName *string `json:"deptName"`
	Status   *string `json:"status"`
	SortNo   *int    `json:"sortNo"`
}

type DoctorDTO struct {
	DoctorID     string `json:"doctorId"`
	DoctorCode   string `json:"doctorCode"`
	DoctorName   string `json:"doctorName"`
	DepartmentID string `json:"departmentId"`
	Status       string `json:"status"`
	CreatedAt    string `json:"createdAt,omitempty"`
	UpdatedAt    string `json:"updatedAt,omitempty"`
}

type DoctorListDTO struct {
	Items []DoctorDTO `json:"items"`
}

type CreateDoctorInput struct {
	DoctorCode   string  `json:"doctorCode"`
	DoctorName   string  `json:"doctorName"`
	DepartmentID string  `json:"departmentId"`
	Status       *string `json:"status"`
}

type PatchDoctorInput struct {
	DoctorCode   *string `json:"doctorCode"`
	DoctorName   *string `json:"doctorName"`
	DepartmentID *string `json:"departmentId"`
	Status       *string `json:"status"`
}

type CatalogItemDTO struct {
	CatalogItemID  string  `json:"catalogItemId"`
	ItemNo         string  `json:"itemNo"`
	ItemName       string  `json:"itemName"`
	ItemType       string  `json:"itemType"`
	Spec           *string `json:"spec"`
	Units          string  `json:"units"`
	Price          float64 `json:"price"`
	Status         string  `json:"status"`
	ControlledFlag bool    `json:"controlledFlag"`
	IsControlled   bool    `json:"isControlled"`
	CreatedAt      string  `json:"createdAt,omitempty"`
	UpdatedAt      string  `json:"updatedAt,omitempty"`
}

type CatalogListDTO struct {
	Items []CatalogItemDTO `json:"items"`
}

type CreateCatalogInput struct {
	ItemNo         string   `json:"itemNo"`
	ItemName       string   `json:"itemName"`
	ItemType       string   `json:"itemType"`
	Spec           *string  `json:"spec"`
	Units          string   `json:"units"`
	Price          *float64 `json:"price"`
	Status         *string  `json:"status"`
	ControlledFlag *bool    `json:"controlledFlag"`
	IsControlled   *bool    `json:"isControlled"`
}

type PatchCatalogInput struct {
	ItemNo         *string  `json:"itemNo"`
	ItemName       *string  `json:"itemName"`
	Spec           *string  `json:"spec"`
	Units          *string  `json:"units"`
	Price          *float64 `json:"price"`
	Status         *string  `json:"status"`
	ControlledFlag *bool    `json:"controlledFlag"`
	IsControlled   *bool    `json:"isControlled"`
}

type AdjustInventoryInput struct {
	QtyDelta   *float64 `json:"qtyDelta"`
	Remark     string   `json:"remark"`
	RowVersion *string  `json:"rowVersion"`
}

type DictEnumDTO struct {
	DictType string   `json:"dictType"`
	Values   []string `json:"values"`
}

type DictTypeDTO struct {
	DictType string `json:"dictType"`
	Name     string `json:"name"`
}

type DictOverviewDTO struct {
	ReadonlyEnums     []DictEnumDTO `json:"readonlyEnums"`
	MaintainableTypes []DictTypeDTO `json:"maintainableTypes"`
}

type DictItemDTO struct {
	DictItemID string `json:"dictItemId"`
	DictType   string `json:"dictType"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	SortNo     int    `json:"sortNo"`
	CreatedAt  string `json:"createdAt,omitempty"`
	UpdatedAt  string `json:"updatedAt,omitempty"`
}

type DictItemListDTO struct {
	Items []DictItemDTO `json:"items"`
}

type CreateDictItemInput struct {
	Code   string  `json:"code"`
	Name   string  `json:"name"`
	Status *string `json:"status"`
	SortNo *int    `json:"sortNo"`
}

type PatchDictItemInput struct {
	Name   *string `json:"name"`
	Status *string `json:"status"`
	SortNo *int    `json:"sortNo"`
}

type UserAdminDTO struct {
	UserID               string             `json:"userId"`
	Username             string             `json:"username"`
	LoginName            string             `json:"loginName"`
	DisplayName          string             `json:"displayName"`
	Description          string             `json:"description"`
	CardNo               string             `json:"cardNo"`
	InputCode            string             `json:"inputCode"`
	ServiceStar          *int               `json:"serviceStar,omitempty"`
	ExpandInterface1     *int32             `json:"expandInterface1,omitempty"`
	ExpandInterface2     *int32             `json:"expandInterface2,omitempty"`
	InputCodeCompareMode *int32             `json:"inputCodeCompareMode,omitempty"`
	Status               string             `json:"status"`
	Roles                []string           `json:"roles"`
	RoleScopes           []UserRoleScopeDTO `json:"roleScopes"`
	MustChangePassword   bool               `json:"mustChangePassword"`
	CannotModifyPassword bool               `json:"cannotModifyPassword"`
	AccountLocked        bool               `json:"accountLocked"`
	AccountStopped       bool               `json:"accountStopped"`
	Hidden               bool               `json:"hidden"`
	System               bool               `json:"system"`
	Super                bool               `json:"super"`
	Developer            bool               `json:"developer"`
	LastLoginAt          string             `json:"lastLoginAt,omitempty"`
	LastLoginIP          string             `json:"lastLoginIp,omitempty"`
	LockUntil            string             `json:"lockUntil,omitempty"`
	PasswordChangedAt    string             `json:"passwordChangedAt,omitempty"`
	FailedLoginCount     int                `json:"failedLoginCount"`
	CreatedAt            string             `json:"createdAt,omitempty"`
	UpdatedAt            string             `json:"updatedAt,omitempty"`
}

type UserRoleScopeDTO struct {
	RoleCode     string `json:"roleCode"`
	DepartmentID string `json:"departmentId"`
	Status       string `json:"status"`
}

type UserRoleScopeInput struct {
	RoleCode     string `json:"roleCode"`
	DepartmentID string `json:"departmentId"`
	Status       string `json:"status"`
}

type UserListDTO struct {
	Items []UserAdminDTO `json:"items"`
}

type CreateUserInput struct {
	Username             string                     `json:"username"`
	LoginName            string                     `json:"loginName"`
	DisplayName          string                     `json:"displayName"`
	Description          string                     `json:"description"`
	CardNo               string                     `json:"cardNo"`
	InputCode            string                     `json:"inputCode"`
	Password             string                     `json:"password"`
	ServiceStar          *int                       `json:"serviceStar"`
	ExpandInterface1     *int32                     `json:"expandInterface1"`
	ExpandInterface2     *int32                     `json:"expandInterface2"`
	InputCodeCompareMode *int32                     `json:"inputCodeCompareMode"`
	Roles                []string                   `json:"roles"`
	Status               *string                    `json:"status"`
	MustChangePassword   *bool                      `json:"mustChangePassword"`
	CannotModifyPassword *bool                      `json:"cannotModifyPassword"`
	AccountStopped       *bool                      `json:"accountStopped"`
	Hidden               *bool                      `json:"hidden"`
	System               *bool                      `json:"system"`
	Super                *bool                      `json:"super"`
	Developer            *bool                      `json:"developer"`
	PermissionOverrides  *[]PermissionOverrideInput `json:"permissionOverrides"`
	RoleScopes           *[]UserRoleScopeInput      `json:"roleScopes"`
}

type PatchUserInput struct {
	DisplayName          *string                    `json:"displayName"`
	Description          *string                    `json:"description"`
	CardNo               *string                    `json:"cardNo"`
	InputCode            *string                    `json:"inputCode"`
	Password             *string                    `json:"password"`
	ServiceStar          OptionalNullableInt        `json:"serviceStar"`
	ExpandInterface1     OptionalNullableInt32      `json:"expandInterface1"`
	ExpandInterface2     OptionalNullableInt32      `json:"expandInterface2"`
	InputCodeCompareMode OptionalNullableInt32      `json:"inputCodeCompareMode"`
	Status               *string                    `json:"status"`
	Roles                []string                   `json:"roles"`
	MustChangePassword   *bool                      `json:"mustChangePassword"`
	CannotModifyPassword *bool                      `json:"cannotModifyPassword"`
	AccountLocked        *bool                      `json:"accountLocked"`
	AccountStopped       *bool                      `json:"accountStopped"`
	Hidden               *bool                      `json:"hidden"`
	System               *bool                      `json:"system"`
	Super                *bool                      `json:"super"`
	Developer            *bool                      `json:"developer"`
	PermissionOverrides  *[]PermissionOverrideInput `json:"permissionOverrides"`
	RoleScopes           *[]UserRoleScopeInput      `json:"roleScopes"`
}

type OptionalNullableInt struct {
	Set   bool
	Value *int
}

func (value *OptionalNullableInt) UnmarshalJSON(raw []byte) error {
	value.Set = true
	if strings.TrimSpace(string(raw)) == "null" {
		value.Value = nil
		return nil
	}
	var parsed int
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return err
	}
	value.Value = &parsed
	return nil
}

type OptionalNullableInt32 struct {
	Set   bool
	Value *int32
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneInt32(value *int32) *int32 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func validateUserLegacyFields(serviceStar *int, _ *int32) []string {
	var fields []string
	if serviceStar != nil && (*serviceStar < 1 || *serviceStar > 5) {
		fields = append(fields, "serviceStar")
	}
	return fields
}

func (value *OptionalNullableInt32) UnmarshalJSON(raw []byte) error {
	value.Set = true
	if strings.TrimSpace(string(raw)) == "null" {
		value.Value = nil
		return nil
	}
	var parsed int32
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return err
	}
	value.Value = &parsed
	return nil
}

type masterFilter struct {
	Status       string
	Q            string
	DepartmentID string
	ItemType     string
}

func (s *Service) ListDepartments(ctx context.Context, claims *auth.Claims, status, q string) (*DepartmentListDTO, error) {
	user, err := s.requireOutpatientMasterRead(ctx, claims, domain.PermEncounterCreate, domain.PermEncounterCheckIn, domain.PermOrderSubmit, domain.PermDiagnoseWrite)
	if err != nil {
		return nil, err
	}
	f, err := normalizeMasterFilter(status, q, "", "")
	if err != nil {
		return nil, err
	}
	if !hasPerm(user, domain.PermAdminAll) {
		f.Status = domain.StatusActive
	}
	if s.Master == nil {
		return nil, errors.New("master store is not configured")
	}
	rows, err := s.Master.ListDepartments(ctx, port.DepartmentQuery{Status: f.Status, Q: f.Q})
	if err != nil {
		return nil, err
	}
	items := make([]DepartmentDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toDepartmentDTO(row))
	}
	return &DepartmentListDTO{Items: items}, nil
}

func (s *Service) GetDepartment(ctx context.Context, claims *auth.Claims, departmentID string) (*DepartmentDTO, error) {
	if _, err := s.requireAdmin(ctx, claims); err != nil {
		return nil, err
	}
	row, err := s.loadDepartment(ctx, departmentID)
	if err != nil {
		return nil, err
	}
	dto := toDepartmentDTO(*row)
	return &dto, nil
}

func (s *Service) CreateDepartment(ctx context.Context, claims *auth.Claims, in CreateDepartmentInput, requestID, ip string) (*DepartmentDTO, error) {
	start := time.Now()
	user, err := s.requireAdminAudit(ctx, claims, start, domain.ActionDepartmentCreate, "Department", "", requestID, ip)
	if err != nil {
		return nil, err
	}
	code, name, status, sortNo, fields := normalizeDeptCreate(in)
	if len(fields) > 0 {
		return nil, validationFields(fields)
	}
	if s.Master == nil {
		return nil, errors.New("master store is not configured")
	}
	now := s.now().UTC()
	id, err := newUUID()
	if err != nil {
		return nil, err
	}
	row := domain.Department{
		DepartmentID: id, DeptCode: code, DeptName: name, Status: status, SortNo: sortNo,
		CreatedAt: now, CreatedBy: user.ID, UpdatedAt: now, UpdatedBy: user.ID,
	}
	if err := s.Master.CreateDepartment(ctx, row); err != nil {
		mapped := mapDeptWrite(err)
		s.auditMasterErr(ctx, start, user, domain.ActionDepartmentCreate, "Department", "", requestID, ip, mapped)
		return nil, mapped
	}
	dto := toDepartmentDTO(row)
	s.auditMaster(ctx, start, user, domain.ActionDepartmentCreate, "Department", id, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

func (s *Service) PatchDepartment(ctx context.Context, claims *auth.Claims, departmentID string, in PatchDepartmentInput, requestID, ip string) (*DepartmentDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !hasPerm(user, domain.PermAdminAll) {
		e := forbidden()
		s.auditMaster(ctx, start, user, domain.ActionDepartmentUpdate, "Department", departmentID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	cur, err := s.loadDepartment(ctx, departmentID)
	if err != nil {
		s.auditMasterErr(ctx, start, user, domain.ActionDepartmentUpdate, "Department", departmentID, requestID, ip, err)
		return nil, err
	}
	next, fields := applyDeptPatch(*cur, in)
	if len(fields) > 0 {
		return nil, validationFields(fields)
	}
	now := s.now().UTC()
	next.UpdatedAt = now
	next.UpdatedBy = user.ID
	action := domain.ActionDepartmentUpdate
	if next.Status == domain.StatusInactive && cur.Status != domain.StatusInactive {
		action = domain.ActionDepartmentDeactivate
	}
	saved, err := s.Master.UpdateDepartment(ctx, next)
	if err != nil {
		mapped := mapDeptWrite(err)
		s.auditMasterErr(ctx, start, user, action, "Department", departmentID, requestID, ip, mapped)
		return nil, mapped
	}
	dto := toDepartmentDTO(*saved)
	s.auditMaster(ctx, start, user, action, "Department", departmentID, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

func actionForUserPatch(cur, next *domain.User) string {
	if cur != nil && next != nil && cur.Active() && !next.Active() {
		return domain.ActionUserDeactivate
	}
	return domain.ActionUserUpdate
}

func (s *Service) ListDoctors(ctx context.Context, claims *auth.Claims, departmentID, status, q string) (*DoctorListDTO, error) {
	user, err := s.requireOutpatientMasterRead(ctx, claims, domain.PermEncounterCreate, domain.PermEncounterCheckIn, domain.PermOrderSubmit, domain.PermDiagnoseWrite)
	if err != nil {
		return nil, err
	}
	f, err := normalizeMasterFilter(status, q, departmentID, "")
	if err != nil {
		return nil, err
	}
	if !hasPerm(user, domain.PermAdminAll) {
		f.Status = domain.StatusActive
	}
	if s.Master == nil {
		return nil, errors.New("master store is not configured")
	}
	rows, err := s.Master.ListDoctors(ctx, port.DoctorQuery{DepartmentID: f.DepartmentID, Status: f.Status, Q: f.Q})
	if err != nil {
		return nil, err
	}
	items := make([]DoctorDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toDoctorDTO(row))
	}
	return &DoctorListDTO{Items: items}, nil
}

func (s *Service) GetDoctor(ctx context.Context, claims *auth.Claims, doctorID string) (*DoctorDTO, error) {
	if _, err := s.requireAdmin(ctx, claims); err != nil {
		return nil, err
	}
	row, err := s.loadDoctor(ctx, doctorID)
	if err != nil {
		return nil, err
	}
	dto := toDoctorDTO(*row)
	return &dto, nil
}

func (s *Service) CreateDoctor(ctx context.Context, claims *auth.Claims, in CreateDoctorInput, requestID, ip string) (*DoctorDTO, error) {
	start := time.Now()
	user, err := s.requireAdminAudit(ctx, claims, start, domain.ActionDoctorCreate, "Doctor", "", requestID, ip)
	if err != nil {
		return nil, err
	}
	code, name, deptID, status, fields := normalizeDoctorCreate(in)
	if len(fields) > 0 {
		return nil, validationFields(fields)
	}
	if err := s.requireDepartment(ctx, deptID); err != nil {
		s.auditMasterErr(ctx, start, user, domain.ActionDoctorCreate, "Doctor", "", requestID, ip, err)
		return nil, err
	}
	if s.Master == nil {
		return nil, errors.New("master store is not configured")
	}
	now := s.now().UTC()
	id, err := newUUID()
	if err != nil {
		return nil, err
	}
	row := domain.Doctor{
		DoctorID: id, DepartmentID: deptID, DoctorCode: code, DoctorName: name, Status: status,
		CreatedAt: now, CreatedBy: user.ID, UpdatedAt: now, UpdatedBy: user.ID,
	}
	if err := s.Master.CreateDoctor(ctx, row); err != nil {
		mapped := mapDoctorWrite(err)
		s.auditMasterErr(ctx, start, user, domain.ActionDoctorCreate, "Doctor", "", requestID, ip, mapped)
		return nil, mapped
	}
	dto := toDoctorDTO(row)
	s.auditMaster(ctx, start, user, domain.ActionDoctorCreate, "Doctor", id, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

func (s *Service) PatchDoctor(ctx context.Context, claims *auth.Claims, doctorID string, in PatchDoctorInput, requestID, ip string) (*DoctorDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !hasPerm(user, domain.PermAdminAll) {
		e := forbidden()
		s.auditMaster(ctx, start, user, domain.ActionDoctorUpdate, "Doctor", doctorID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	cur, err := s.loadDoctor(ctx, doctorID)
	if err != nil {
		s.auditMasterErr(ctx, start, user, domain.ActionDoctorUpdate, "Doctor", doctorID, requestID, ip, err)
		return nil, err
	}
	next, fields := applyDoctorPatch(*cur, in)
	if len(fields) > 0 {
		return nil, validationFields(fields)
	}
	if err := s.requireDepartment(ctx, next.DepartmentID); err != nil {
		s.auditMasterErr(ctx, start, user, domain.ActionDoctorUpdate, "Doctor", doctorID, requestID, ip, err)
		return nil, err
	}
	now := s.now().UTC()
	next.UpdatedAt = now
	next.UpdatedBy = user.ID
	saved, err := s.Master.UpdateDoctor(ctx, next)
	if err != nil {
		mapped := mapDoctorWrite(err)
		s.auditMasterErr(ctx, start, user, domain.ActionDoctorUpdate, "Doctor", doctorID, requestID, ip, mapped)
		return nil, mapped
	}
	dto := toDoctorDTO(*saved)
	s.auditMaster(ctx, start, user, domain.ActionDoctorUpdate, "Doctor", doctorID, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

func (s *Service) ListCatalog(ctx context.Context, claims *auth.Claims, itemType, status, q string) (*CatalogListDTO, error) {
	user, err := s.requireOutpatientMasterRead(ctx, claims, domain.PermOrderSubmit, domain.PermOrderRecipeCheck)
	if err != nil {
		return nil, err
	}
	f, err := normalizeMasterFilter(status, q, "", itemType)
	if err != nil {
		return nil, err
	}
	if !hasPerm(user, domain.PermAdminAll) {
		f.Status = domain.StatusActive
	}
	if s.Master == nil {
		return nil, errors.New("master store is not configured")
	}
	rows, err := s.Master.ListCatalog(ctx, port.CatalogQuery{ItemType: f.ItemType, Status: f.Status, Q: f.Q})
	if err != nil {
		return nil, err
	}
	items := make([]CatalogItemDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toCatalogDTO(row))
	}
	return &CatalogListDTO{Items: items}, nil
}

func (s *Service) GetCatalogItem(ctx context.Context, claims *auth.Claims, catalogItemID string) (*CatalogItemDTO, error) {
	if _, err := s.requireAdmin(ctx, claims); err != nil {
		return nil, err
	}
	row, err := s.loadCatalog(ctx, catalogItemID)
	if err != nil {
		return nil, err
	}
	dto := toCatalogDTO(*row)
	return &dto, nil
}

func (s *Service) CreateCatalogItem(ctx context.Context, claims *auth.Claims, in CreateCatalogInput, requestID, ip string) (*CatalogItemDTO, error) {
	start := time.Now()
	user, err := s.requireAdminAudit(ctx, claims, start, domain.ActionCatalogCreate, "CatalogItem", "", requestID, ip)
	if err != nil {
		return nil, err
	}
	item, fields, priceErr := normalizeCatalogCreate(in)
	if priceErr != nil {
		s.auditMasterErr(ctx, start, user, domain.ActionCatalogCreate, "CatalogItem", "", requestID, ip, priceErr)
		return nil, priceErr
	}
	if len(fields) > 0 {
		return nil, validationFields(fields)
	}
	if s.Master == nil {
		return nil, errors.New("master store is not configured")
	}
	now := s.now().UTC()
	id, err := newUUID()
	if err != nil {
		return nil, err
	}
	skuID := ""
	if item.ItemType == domain.ItemTypeDrug {
		skuID, err = newUUID()
		if err != nil {
			return nil, err
		}
	}
	item.CatalogItemID = id
	item.CreatedAt = now
	item.CreatedBy = user.ID
	item.UpdatedAt = now
	item.UpdatedBy = user.ID
	saved, err := s.Master.CreateCatalog(ctx, item, skuID)
	if err != nil {
		mapped := mapCatalogWrite(err)
		s.auditMasterErr(ctx, start, user, domain.ActionCatalogCreate, "CatalogItem", "", requestID, ip, mapped)
		return nil, mapped
	}
	dto := toCatalogDTO(*saved)
	s.auditMaster(ctx, start, user, domain.ActionCatalogCreate, "CatalogItem", id, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

func (s *Service) PatchCatalogItem(ctx context.Context, claims *auth.Claims, catalogItemID string, in PatchCatalogInput, requestID, ip string) (*CatalogItemDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !hasPerm(user, domain.PermAdminAll) {
		e := forbidden()
		s.auditMaster(ctx, start, user, domain.ActionCatalogUpdate, "CatalogItem", catalogItemID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	cur, err := s.loadCatalog(ctx, catalogItemID)
	if err != nil {
		s.auditMasterErr(ctx, start, user, domain.ActionCatalogUpdate, "CatalogItem", catalogItemID, requestID, ip, err)
		return nil, err
	}
	next, fields, priceErr := applyCatalogPatch(*cur, in)
	if priceErr != nil {
		s.auditMasterErr(ctx, start, user, domain.ActionCatalogUpdate, "CatalogItem", catalogItemID, requestID, ip, priceErr)
		return nil, priceErr
	}
	if len(fields) > 0 {
		return nil, validationFields(fields)
	}
	now := s.now().UTC()
	next.UpdatedAt = now
	next.UpdatedBy = user.ID
	saved, err := s.Master.UpdateCatalog(ctx, next)
	if err != nil {
		mapped := mapCatalogWrite(err)
		s.auditMasterErr(ctx, start, user, domain.ActionCatalogUpdate, "CatalogItem", catalogItemID, requestID, ip, mapped)
		return nil, mapped
	}
	dto := toCatalogDTO(*saved)
	s.auditMaster(ctx, start, user, domain.ActionCatalogUpdate, "CatalogItem", catalogItemID, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

func (s *Service) AdjustInventory(ctx context.Context, claims *auth.Claims, skuID string, in AdjustInventoryInput, requestID, ip string) (*InventoryItemDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !hasPerm(user, domain.PermInventoryAdjust) {
		e := forbidden()
		s.auditMaster(ctx, start, user, domain.ActionInventoryAdjust, "InventoryItem", skuID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	skuID = strings.TrimSpace(skuID)
	remark := strings.TrimSpace(in.Remark)
	var fields []string
	if skuID == "" || len(skuID) > 64 {
		fields = addField(fields, "skuId")
	}
	if remark == "" || utf8.RuneCountInString(remark) > 256 {
		fields = addField(fields, "remark")
	}
	if in.QtyDelta == nil {
		fields = addField(fields, "qtyDelta")
	}
	var delta int64
	if in.QtyDelta != nil {
		m, ok := signedMilli(*in.QtyDelta)
		if !ok {
			fields = addField(fields, "qtyDelta")
		}
		delta = m
	}
	rowVersion := ""
	if in.RowVersion != nil {
		rowVersion = strings.TrimSpace(*in.RowVersion)
		if len(rowVersion) > 40 {
			fields = addField(fields, "rowVersion")
		}
	}
	if len(fields) > 0 {
		return nil, validationFields(fields)
	}
	if s.Master == nil {
		return nil, errors.New("master store is not configured")
	}
	ledgerID, err := newUUID()
	if err != nil {
		return nil, err
	}
	saved, err := s.Master.AdjustInventory(ctx, skuID, delta, remark, user.ID, rowVersion, ledgerID, s.now().UTC())
	if err != nil {
		mapped := mapAdjustErr(err)
		s.auditMasterErr(ctx, start, user, domain.ActionInventoryAdjust, "InventoryItem", skuID, requestID, ip, mapped)
		return nil, mapped
	}
	dto := toInventoryItemDTO(*saved)
	s.auditMaster(ctx, start, user, domain.ActionInventoryAdjust, "InventoryItem", skuID, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

func (s *Service) ListDicts(ctx context.Context, claims *auth.Claims) (*DictOverviewDTO, error) {
	if _, err := s.actor(ctx, claims); err != nil {
		return nil, err
	}
	return &DictOverviewDTO{
		ReadonlyEnums: []DictEnumDTO{
			{DictType: "sex", Values: []string{"M", "F", "U"}},
			{DictType: "cardStatus", Values: []string{"NOT_SENT", "NORMAL", "LOST", "CANCELLED"}},
			{DictType: "payMethod", Values: []string{"CASH", "OTHER_E", "WECHAT", "ALIPAY", "OTHER"}},
			{DictType: "registerStatus", Values: []string{"NORMAL", "USED", "BACK", "DIAGNOSING", "DIAGNOSED", "PAUSED"}},
			{DictType: "orderStatus", Values: []string{"SAVE", "COMMIT", "PROC", "ABATE"}},
		},
		MaintainableTypes: []DictTypeDTO{{DictType: dictRegisterSort, Name: "号别"}},
	}, nil
}

func (s *Service) ListDictItems(ctx context.Context, claims *auth.Claims, dictType, status string) (*DictItemListDTO, error) {
	if _, err := s.actor(ctx, claims); err != nil {
		return nil, err
	}
	dictType = strings.TrimSpace(dictType)
	status = strings.TrimSpace(status)
	if status != "" && !statusActiveInactive(status) {
		return nil, validationFields([]string{"status"})
	}
	if values, ok := readonlyDictValues(dictType); ok {
		items := make([]DictItemDTO, 0, len(values))
		if status == domain.StatusInactive {
			return &DictItemListDTO{Items: items}, nil
		}
		for _, code := range values {
			items = append(items, DictItemDTO{DictItemID: dictType + ":" + code, DictType: dictType, Code: code, Name: code, Status: domain.StatusActive})
		}
		return &DictItemListDTO{Items: items}, nil
	}
	if !maintainableDict(dictType) {
		return nil, notFound()
	}
	if s.Master == nil {
		return nil, errors.New("master store is not configured")
	}
	rows, err := s.Master.ListDictItems(ctx, dictType, status)
	if err != nil {
		return nil, err
	}
	items := make([]DictItemDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toDictDTO(row))
	}
	return &DictItemListDTO{Items: items}, nil
}

func (s *Service) CreateDictItem(ctx context.Context, claims *auth.Claims, dictType string, in CreateDictItemInput, requestID, ip string) (*DictItemDTO, error) {
	start := time.Now()
	user, err := s.requireAdminAudit(ctx, claims, start, domain.ActionDictCreate, "DictItem", "", requestID, ip)
	if err != nil {
		return nil, err
	}
	dictType = strings.TrimSpace(dictType)
	if _, ok := readonlyDictValues(dictType); ok {
		e := dictReadonly()
		s.auditMasterErr(ctx, start, user, domain.ActionDictCreate, "DictItem", "", requestID, ip, e)
		return nil, e
	}
	if !maintainableDict(dictType) {
		e := notFound()
		s.auditMasterErr(ctx, start, user, domain.ActionDictCreate, "DictItem", "", requestID, ip, e)
		return nil, e
	}
	code := strings.TrimSpace(in.Code)
	name := strings.TrimSpace(in.Name)
	var fields []string
	if code == "" || len(code) > 32 {
		fields = addField(fields, "code")
	}
	if name == "" || utf8.RuneCountInString(name) > 64 {
		fields = addField(fields, "name")
	}
	status := domain.StatusActive
	if in.Status != nil {
		status = strings.TrimSpace(*in.Status)
		if !statusActiveInactive(status) {
			fields = addField(fields, "status")
		}
	}
	sortNo := 0
	if in.SortNo != nil {
		sortNo = *in.SortNo
		if !sortInRange(sortNo) {
			fields = addField(fields, "sortNo")
		}
	}
	if len(fields) > 0 {
		return nil, validationFields(fields)
	}
	if s.Master == nil {
		return nil, errors.New("master store is not configured")
	}
	now := s.now().UTC()
	id, err := newUUID()
	if err != nil {
		return nil, err
	}
	row := domain.DictItem{
		DictItemID: id, DictType: dictType, Code: code, Name: name, Status: status, SortNo: sortNo,
		CreatedAt: now, CreatedBy: user.ID, UpdatedAt: now, UpdatedBy: user.ID,
	}
	if err := s.Master.CreateDictItem(ctx, row); err != nil {
		mapped := mapDictWrite(err)
		s.auditMasterErr(ctx, start, user, domain.ActionDictCreate, "DictItem", "", requestID, ip, mapped)
		return nil, mapped
	}
	dto := toDictDTO(row)
	s.auditMaster(ctx, start, user, domain.ActionDictCreate, "DictItem", id, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

func (s *Service) PatchDictItem(ctx context.Context, claims *auth.Claims, dictType, dictItemID string, in PatchDictItemInput, requestID, ip string) (*DictItemDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !hasPerm(user, domain.PermAdminAll) {
		e := forbidden()
		s.auditMaster(ctx, start, user, domain.ActionDictUpdate, "DictItem", dictItemID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	dictType = strings.TrimSpace(dictType)
	dictItemID = strings.TrimSpace(dictItemID)
	if _, ok := readonlyDictValues(dictType); ok {
		e := dictReadonly()
		s.auditMasterErr(ctx, start, user, domain.ActionDictUpdate, "DictItem", dictItemID, requestID, ip, e)
		return nil, e
	}
	if !maintainableDict(dictType) || dictItemID == "" {
		e := notFound()
		s.auditMasterErr(ctx, start, user, domain.ActionDictUpdate, "DictItem", dictItemID, requestID, ip, e)
		return nil, e
	}
	if s.Master == nil {
		return nil, errors.New("master store is not configured")
	}
	rows, err := s.Master.ListDictItems(ctx, dictType, "")
	if err != nil {
		return nil, err
	}
	var cur *domain.DictItem
	for i := range rows {
		if rows[i].DictItemID == dictItemID {
			cur = &rows[i]
			break
		}
	}
	if cur == nil {
		e := notFound()
		s.auditMasterErr(ctx, start, user, domain.ActionDictUpdate, "DictItem", dictItemID, requestID, ip, e)
		return nil, e
	}
	next := *cur
	var fields []string
	if in.Name != nil {
		next.Name = strings.TrimSpace(*in.Name)
		if next.Name == "" || utf8.RuneCountInString(next.Name) > 64 {
			fields = addField(fields, "name")
		}
	}
	if in.Status != nil {
		next.Status = strings.TrimSpace(*in.Status)
		if !statusActiveInactive(next.Status) {
			fields = addField(fields, "status")
		}
	}
	if in.SortNo != nil {
		next.SortNo = *in.SortNo
		if !sortInRange(next.SortNo) {
			fields = addField(fields, "sortNo")
		}
	}
	if len(fields) > 0 {
		return nil, validationFields(fields)
	}
	now := s.now().UTC()
	next.UpdatedAt = now
	next.UpdatedBy = user.ID
	saved, err := s.Master.UpdateDictItem(ctx, next)
	if err != nil {
		mapped := mapDictWrite(err)
		s.auditMasterErr(ctx, start, user, domain.ActionDictUpdate, "DictItem", dictItemID, requestID, ip, mapped)
		return nil, mapped
	}
	dto := toDictDTO(*saved)
	s.auditMaster(ctx, start, user, domain.ActionDictUpdate, "DictItem", dictItemID, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

func (s *Service) ListUsers(ctx context.Context, claims *auth.Claims, status, q string) (*UserListDTO, error) {
	actor, err := s.requireAdminPermission(ctx, claims, domain.PermAdminUser)
	if err != nil {
		return nil, err
	}
	status = strings.TrimSpace(status)
	q = strings.TrimSpace(q)
	if status != "" && status != domain.StatusActive && status != domain.StatusDisabled && status != domain.StatusInactive {
		return nil, validationFields([]string{"status"})
	}
	if len(q) > 64 {
		return nil, validationFields([]string{"q"})
	}
	if s.Master == nil {
		return nil, errors.New("master store is not configured")
	}
	rows, err := s.Master.ListUsers(ctx, port.UserQuery{Status: status, Q: q})
	if err != nil {
		return nil, err
	}
	if accessStore, ok := s.Master.(port.AccessStore); ok {
		for i := range rows {
			rows[i].RoleScopes, err = accessStore.ListUserRoleScopes(ctx, rows[i].ID)
			if err != nil {
				return nil, err
			}
		}
	}
	items := make([]UserAdminDTO, 0, len(rows))
	for _, row := range rows {
		if row.Hidden && !actor.Developer {
			continue
		}
		items = append(items, toUserAdminDTO(row))
	}
	return &UserListDTO{Items: items}, nil
}

func (s *Service) CreateUser(ctx context.Context, claims *auth.Claims, in CreateUserInput, requestID, ip string) (*UserAdminDTO, error) {
	start := time.Now()
	user, err := s.requireAdminPermissionAudit(ctx, claims, domain.PermAdminUser, start, domain.ActionUserCreate, "User", "", requestID, ip)
	if err != nil {
		return nil, err
	}
	username, display, roles, status, fields := normalizeUserCreate(in)
	roles, rolesOK := s.parseAssignableRoles(ctx, in.Roles)
	hasAdminRole, roleErr := s.hasAdminPermissionRole(ctx, in.Roles)
	if roleErr != nil {
		s.auditMasterErr(ctx, start, user, domain.ActionUserCreate, "User", "", requestID, ip, roleErr)
		return nil, roleErr
	}
	if hasAdminRole && !canManageSystemAccess(user) {
		e := forbidden()
		s.auditMasterErr(ctx, start, user, domain.ActionUserCreate, "User", "", requestID, ip, e)
		return nil, e
	}
	if !rolesOK {
		fields = addField(fields, "roles")
	}
	fields = append(fields, validateUserProfile(in.Description, in.CardNo, in.InputCode)...)
	fields = append(fields, validateAdminPassword(in.Password)...)
	fields = append(fields, validateUserLegacyFields(in.ServiceStar, in.InputCodeCompareMode)...)
	if boolDefault(in.CannotModifyPassword, false) && boolDefault(in.MustChangePassword, false) {
		fields = addField(fields, "mustChangePassword")
	}
	var bundleScopes []domain.UserRoleScope
	if in.RoleScopes != nil {
		accessStore, ok := s.Master.(port.AccessStore)
		if !ok {
			return nil, &Error{Status: 501, Code: CodeUserAccessBundleUnsupported, Message: msgUserAccessBundleUnsupported}
		}
		var scopeFields []string
		var scopeErr error
		bundleScopes, scopeFields, scopeErr = normalizeAndValidateRoleScopes(ctx, s.Master, accessStore, "", roleCodes(roles), *in.RoleScopes)
		if scopeErr != nil {
			return nil, scopeErr
		}
		fields = append(fields, scopeFields...)
	}
	if len(fields) > 0 {
		return nil, validationFields(fields)
	}
	if strings.TrimSpace(in.CardNo) != "" && !canManageProtectedUsers(user) {
		return nil, forbidden()
	}
	if hasDeveloperOnlyUserCreateFields(in) && !canEditDeveloperFlags(user) {
		return nil, forbidden()
	}
	if boolDefault(in.Super, false) && !canEditSuperFlag(user) {
		return nil, forbidden()
	}
	if s.Master == nil {
		return nil, errors.New("master store is not configured")
	}
	now := s.now().UTC()
	id, err := newUUID()
	if err != nil {
		return nil, err
	}
	inputCodeMode := in.InputCodeCompareMode
	if inputCodeMode == nil {
		mode := int32(2)
		inputCodeMode = &mode
	}
	passwordHash := seed.DemoPasswordHash
	if strings.TrimSpace(in.Password) != "" {
		passwordHash, err = auth.HashPassword(in.Password)
		if err != nil {
			return nil, err
		}
	}
	row := domain.User{
		ID: id, Username: username, DisplayName: display, PasswordHash: passwordHash,
		Roles: roles, Status: status, CreatedAt: now, UpdatedAt: now, CreatedBy: user.ID, UpdatedBy: user.ID,
		Description: strings.TrimSpace(in.Description), CardNo: strings.TrimSpace(in.CardNo), InputCode: strings.TrimSpace(in.InputCode),
		ServiceStar: cloneInt(in.ServiceStar), ExpandInterface1: cloneInt32(in.ExpandInterface1), ExpandInterface2: cloneInt32(in.ExpandInterface2), InputCodeCompareMode: cloneInt32(inputCodeMode),
		MustChangePassword: boolDefault(in.MustChangePassword, !boolDefault(in.CannotModifyPassword, false)), CannotModifyPassword: boolDefault(in.CannotModifyPassword, false),
		AccountStopped: boolDefault(in.AccountStopped, false), Hidden: boolDefault(in.Hidden, false),
		System: boolDefault(in.System, false), Super: boolDefault(in.Super, false), Developer: boolDefault(in.Developer, false),
	}
	if in.RoleScopes != nil {
		for i := range bundleScopes {
			bundleScopes[i].UserID = row.ID
		}
	}
	if row.AccountStopped {
		row.Status = domain.StatusDisabled
	}
	useAccessBundle := in.PermissionOverrides != nil || in.RoleScopes != nil
	var bundleOverrides []domain.PermissionOverride
	if useAccessBundle {
		accessStore, ok := s.Master.(port.AccessStore)
		if !ok {
			return nil, &Error{Status: 501, Code: CodeUserAccessBundleUnsupported, Message: msgUserAccessBundleUnsupported}
		}
		if in.PermissionOverrides != nil {
			bundleOverrides, fields = normalizeOverrides(id, *in.PermissionOverrides)
			if len(fields) > 0 {
				return nil, validationFields(fields)
			}
			if !canManageSystemAccess(user) {
				for _, override := range bundleOverrides {
					if isAdminPermissionCode(override.PermissionCode) && override.Effect == domain.PermissionAllow {
						return nil, forbidden()
					}
				}
			}
			if fields := s.validateOverrideReferences(ctx, accessStore, bundleOverrides); len(fields) > 0 {
				return nil, validationFields(fields)
			}
		}
	}
	atomicAudit := false
	var saved *domain.User
	if bundleStore, ok := s.Master.(port.AtomicUserAccessBundleStore); useAccessBundle && ok {
		auditID, idErr := randomAuditID()
		if idErr != nil {
			return nil, idErr
		}
		event := domain.AuditEvent{
			ID: auditID, OccurredAt: s.now().UTC(), ActorID: user.ID,
			ActorRoles: strings.Join(user.RoleCodes(), ","), Action: domain.ActionUserCreate,
			ResourceType: "User", ResourceID: id, Outcome: domain.OutcomeSuccess,
			RequestID: requestID, ClientIP: ip, DurationMS: time.Since(start).Milliseconds(),
		}
		saved, err = bundleStore.CreateUserAccessBundleWithAudit(ctx, row, in.RoleScopes != nil, bundleScopes, in.PermissionOverrides != nil, bundleOverrides, event)
		atomicAudit = true
	} else if useAccessBundle {
		return nil, &Error{Status: 501, Code: CodeUserAccessBundleUnsupported, Message: msgUserAccessBundleUnsupported}
	} else if atomicStore, ok := s.Master.(port.AtomicUserCreateStore); ok {
		auditID, idErr := randomAuditID()
		if idErr != nil {
			return nil, idErr
		}
		event := domain.AuditEvent{
			ID: auditID, OccurredAt: s.now().UTC(), ActorID: user.ID,
			ActorRoles: strings.Join(user.RoleCodes(), ","), Action: domain.ActionUserCreate,
			ResourceType: "User", ResourceID: id, Outcome: domain.OutcomeSuccess,
			RequestID: requestID, ClientIP: ip, DurationMS: time.Since(start).Milliseconds(),
		}
		saved, err = atomicStore.CreateUserWithAudit(ctx, row, event)
		atomicAudit = true
	} else {
		saved, err = s.Master.CreateUser(ctx, row)
	}
	if err != nil {
		mapped := mapUserWrite(err)
		s.auditMasterErr(ctx, start, user, domain.ActionUserCreate, "User", "", requestID, ip, mapped)
		return nil, mapped
	}
	if accessStore, ok := s.Master.(port.AccessStore); ok {
		if scopes, scopeErr := accessStore.ListUserRoleScopes(ctx, saved.ID); scopeErr == nil {
			saved.RoleScopes = scopes
		}
	}
	dto := toUserAdminDTO(*saved)
	if !atomicAudit {
		s.auditMaster(ctx, start, user, domain.ActionUserCreate, "User", id, domain.OutcomeSuccess, "", requestID, ip)
	}
	return &dto, nil
}

func (s *Service) PatchUser(ctx context.Context, claims *auth.Claims, userID string, in PatchUserInput, requestID, ip string) (*UserAdminDTO, error) {
	start := time.Now()
	actorUser, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !hasPerm(actorUser, domain.PermAdminUser) {
		e := forbidden()
		s.auditMaster(ctx, start, actorUser, domain.ActionUserUpdate, "User", userID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	userID = strings.TrimSpace(userID)
	if userID == "" || s.Users == nil || s.Master == nil {
		e := notFound()
		s.auditMasterErr(ctx, start, actorUser, domain.ActionUserUpdate, "User", userID, requestID, ip, e)
		return nil, e
	}
	cur, err := s.Users.FindByID(ctx, userID)
	if errors.Is(err, port.ErrNotFound) {
		e := notFound()
		s.auditMasterErr(ctx, start, actorUser, domain.ActionUserUpdate, "User", userID, requestID, ip, e)
		return nil, e
	}
	if err != nil {
		return nil, err
	}
	next := *cur
	replace := false
	var fields []string
	if in.DisplayName != nil {
		next.DisplayName = strings.TrimSpace(*in.DisplayName)
		if next.DisplayName == "" || utf8.RuneCountInString(next.DisplayName) > 128 {
			fields = addField(fields, "displayName")
		}
	}
	if in.Description != nil {
		next.Description = strings.TrimSpace(*in.Description)
	}
	if in.CardNo != nil {
		next.CardNo = strings.TrimSpace(*in.CardNo)
	}
	if in.InputCode != nil {
		next.InputCode = strings.TrimSpace(*in.InputCode)
	}
	if in.Password != nil && strings.TrimSpace(*in.Password) != "" {
		if passwordHash, hashErr := auth.HashPassword(*in.Password); hashErr != nil {
			return nil, hashErr
		} else {
			next.PasswordHash = passwordHash
			next.MustChangePassword = true
			next.PasswordChangedAt = time.Time{}
		}
	}
	if in.ServiceStar.Set {
		next.ServiceStar = cloneInt(in.ServiceStar.Value)
	}
	if in.ExpandInterface1.Set {
		next.ExpandInterface1 = cloneInt32(in.ExpandInterface1.Value)
	}
	if in.ExpandInterface2.Set {
		next.ExpandInterface2 = cloneInt32(in.ExpandInterface2.Value)
	}
	if in.InputCodeCompareMode.Set {
		next.InputCodeCompareMode = cloneInt32(in.InputCodeCompareMode.Value)
	}
	if in.Status != nil {
		next.Status = strings.TrimSpace(*in.Status)
		if next.Status != domain.StatusActive && next.Status != domain.StatusDisabled && next.Status != domain.StatusInactive {
			fields = addField(fields, "status")
		}
	}
	if in.Roles != nil {
		roles, ok := s.parseAssignableRoles(ctx, in.Roles)
		hasAdminRole, roleErr := s.hasAdminPermissionRole(ctx, in.Roles)
		if roleErr != nil {
			s.auditMasterErr(ctx, start, actorUser, domain.ActionUserUpdate, "User", userID, requestID, ip, roleErr)
			return nil, roleErr
		}
		if hasAdminRole && !canManageSystemAccess(actorUser) {
			e := forbidden()
			s.auditMasterErr(ctx, start, actorUser, domain.ActionUserUpdate, "User", userID, requestID, ip, e)
			return nil, e
		}
		if !ok {
			fields = addField(fields, "roles")
		}
		next.Roles = roles
		replace = true
	}
	if in.MustChangePassword != nil {
		next.MustChangePassword = *in.MustChangePassword
	}
	if in.CannotModifyPassword != nil {
		next.CannotModifyPassword = *in.CannotModifyPassword
	}
	if in.AccountLocked != nil {
		next.AccountLocked = *in.AccountLocked
		if !*in.AccountLocked {
			next.LockUntil = time.Time{}
			next.FailedLoginCount = 0
		} else {
			next.LockUntil = time.Time{}
		}
	}
	if in.AccountStopped != nil {
		next.AccountStopped = *in.AccountStopped
		if *in.AccountStopped {
			next.Status = domain.StatusDisabled
		} else if in.Status == nil && next.Status == domain.StatusDisabled {
			next.Status = domain.StatusActive
		}
	}
	if in.Hidden != nil {
		next.Hidden = *in.Hidden
	}
	if in.System != nil {
		next.System = *in.System
	}
	if in.Super != nil {
		next.Super = *in.Super
	}
	if in.Developer != nil {
		next.Developer = *in.Developer
	}
	if len(validateUserProfile(next.Description, next.CardNo, next.InputCode)) > 0 {
		fields = append(fields, validateUserProfile(next.Description, next.CardNo, next.InputCode)...)
	}
	if in.Password != nil {
		fields = append(fields, validateAdminPassword(*in.Password)...)
	}
	fields = append(fields, validateUserLegacyFields(next.ServiceStar, next.InputCodeCompareMode)...)
	if in.CardNo != nil && !canManageProtectedUsers(actorUser) {
		e := forbidden()
		s.auditMaster(ctx, start, actorUser, domain.ActionUserUpdate, "User", userID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	if hasDeveloperOnlyUserPatchFields(in) && !canEditDeveloperFlags(actorUser) {
		e := forbidden()
		s.auditMaster(ctx, start, actorUser, domain.ActionUserUpdate, "User", userID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	if in.Super != nil && !canEditSuperFlag(actorUser) {
		e := forbidden()
		s.auditMaster(ctx, start, actorUser, domain.ActionUserUpdate, "User", userID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	if isProtectedUser(cur) && !canManageProtectedUsers(actorUser) {
		e := forbidden()
		s.auditMaster(ctx, start, actorUser, domain.ActionUserUpdate, "User", userID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	if cur.Hidden && !actorUser.Developer {
		e := forbidden()
		s.auditMaster(ctx, start, actorUser, domain.ActionUserUpdate, "User", userID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	useAccessBundle := in.PermissionOverrides != nil || in.RoleScopes != nil || in.Roles != nil
	var bundleOverrides []domain.PermissionOverride
	var bundleScopes []domain.UserRoleScope
	if in.RoleScopes != nil {
		accessStore, ok := s.Master.(port.AccessStore)
		if !ok {
			return nil, &Error{Status: 501, Code: CodeUserAccessBundleUnsupported, Message: msgUserAccessBundleUnsupported}
		}
		var scopeFields []string
		var scopeErr error
		bundleScopes, scopeFields, scopeErr = normalizeAndValidateRoleScopes(ctx, s.Master, accessStore, userID, roleCodes(next.Roles), *in.RoleScopes)
		if scopeErr != nil {
			return nil, scopeErr
		}
		if len(scopeFields) > 0 {
			return nil, validationFields(scopeFields)
		}
	}
	if useAccessBundle {
		accessStore, ok := s.Master.(port.AccessStore)
		if !ok {
			return nil, &Error{Status: 501, Code: CodeUserAccessBundleUnsupported, Message: msgUserAccessBundleUnsupported}
		}
		if in.PermissionOverrides != nil {
			var overrideFields []string
			bundleOverrides, overrideFields = normalizeOverrides(userID, *in.PermissionOverrides)
			if len(overrideFields) > 0 {
				return nil, validationFields(overrideFields)
			}
			if !canManageSystemAccess(actorUser) {
				for _, override := range bundleOverrides {
					if isAdminPermissionCode(override.PermissionCode) && override.Effect == domain.PermissionAllow {
						return nil, forbidden()
					}
				}
			}
			if overrideFields := s.validateOverrideReferences(ctx, accessStore, bundleOverrides); len(overrideFields) > 0 {
				return nil, validationFields(overrideFields)
			}
		}
	}
	if len(fields) > 0 {
		return nil, validationFields(fields)
	}
	if next.CannotModifyPassword && next.MustChangePassword {
		e := &Error{Status: 409, Code: CodePasswordChangeNotAllowed, Message: "该账号禁止自行修改密码，不能设置强制改密"}
		return nil, e
	}
	if !sameUserAuthorization(cur, &next) {
		next.AuthVersion++
	}
	if cur.HasSystemRole() && (next.Status != domain.StatusActive || !next.HasSystemRole()) {
		if _, atomic := s.Master.(port.AtomicUserWriteStore); !atomic {
			if access, ok := s.Master.(port.AccessStore); ok {
				count, countErr := access.CountSystemAdmins(ctx)
				if countErr != nil {
					return nil, countErr
				}
				if count <= 1 {
					e := &Error{Status: 409, Code: CodeLastSystemAdmin, Message: msgLastSystemAdmin}
					s.auditMaster(ctx, start, actorUser, actionForUserPatch(cur, &next), "User", userID, domain.OutcomeFail, e.Code, requestID, ip)
					return nil, e
				}
			}
		}
	}
	now := s.now().UTC()
	next.UpdatedAt = now
	next.UpdatedBy = actorUser.ID
	action := domain.ActionUserUpdate
	if cur.Active() && !next.Active() {
		action = domain.ActionUserDeactivate
	}
	atomicAudit := false
	var saved *domain.User
	if bundleStore, ok := s.Master.(port.AtomicUserAccessBundleStore); useAccessBundle && ok {
		auditID, idErr := randomAuditID()
		if idErr != nil {
			return nil, idErr
		}
		event := domain.AuditEvent{
			ID: auditID, OccurredAt: s.now().UTC(), ActorID: actorUser.ID,
			ActorRoles: strings.Join(actorUser.RoleCodes(), ","), Action: action,
			ResourceType: "User", ResourceID: userID, Outcome: domain.OutcomeSuccess,
			RequestID: requestID, ClientIP: ip, DurationMS: time.Since(start).Milliseconds(),
		}
		saved, err = bundleStore.UpdateUserAccessBundleWithAudit(ctx, next, replace, in.RoleScopes != nil, bundleScopes, in.PermissionOverrides != nil, bundleOverrides, event)
		atomicAudit = true
	} else if useAccessBundle {
		return nil, &Error{Status: 501, Code: CodeUserAccessBundleUnsupported, Message: msgUserAccessBundleUnsupported}
	} else if atomicStore, ok := s.Master.(port.AtomicUserWriteStore); ok {
		auditID, idErr := randomAuditID()
		if idErr != nil {
			return nil, idErr
		}
		event := domain.AuditEvent{
			ID: auditID, OccurredAt: s.now().UTC(), ActorID: actorUser.ID,
			ActorRoles: strings.Join(actorUser.RoleCodes(), ","), Action: action,
			ResourceType: "User", ResourceID: userID, Outcome: domain.OutcomeSuccess,
			RequestID: requestID, ClientIP: ip, DurationMS: time.Since(start).Milliseconds(),
		}
		saved, err = atomicStore.UpdateUserWithAudit(ctx, next, replace, event)
		atomicAudit = true
	} else {
		saved, err = s.Master.UpdateUser(ctx, next, replace)
	}
	if err != nil {
		mapped := mapUserWrite(err)
		s.auditMasterErr(ctx, start, actorUser, action, "User", userID, requestID, ip, mapped)
		return nil, mapped
	}
	if accessStore, ok := s.Master.(port.AccessStore); ok {
		if scopes, scopeErr := accessStore.ListUserRoleScopes(ctx, saved.ID); scopeErr == nil {
			saved.RoleScopes = scopes
		}
	}
	dto := toUserAdminDTO(*saved)
	if !atomicAudit {
		s.auditMaster(ctx, start, actorUser, action, "User", userID, domain.OutcomeSuccess, "", requestID, ip)
	}
	return &dto, nil
}

func (s *Service) ResetUserPassword(ctx context.Context, claims *auth.Claims, userID, requestID, ip string) error {
	start := time.Now()
	actorUser, err := s.requireAdminPermissionAudit(ctx, claims, domain.PermAdminUser, start, domain.ActionUserResetPassword, "User", userID, requestID, ip)
	if err != nil {
		return err
	}
	userID = strings.TrimSpace(userID)
	if s.Master == nil || userID == "" {
		e := notFound()
		s.auditMasterErr(ctx, start, actorUser, domain.ActionUserResetPassword, "User", userID, requestID, ip, e)
		return e
	}
	target, findErr := s.Users.FindByID(ctx, userID)
	if errors.Is(findErr, port.ErrNotFound) {
		e := notFound()
		s.auditMasterErr(ctx, start, actorUser, domain.ActionUserResetPassword, "User", userID, requestID, ip, e)
		return e
	}
	if findErr != nil {
		return findErr
	}
	if target.CannotModifyPassword {
		e := &Error{Status: 409, Code: CodePasswordChangeNotAllowed, Message: "该账号禁止自行修改密码，无法强制改密"}
		s.auditMasterErr(ctx, start, actorUser, domain.ActionUserResetPassword, "User", userID, requestID, ip, e)
		return e
	}
	at := s.now().UTC()
	atomicAudit := false
	if atomicStore, ok := s.Master.(port.AtomicPasswordResetStore); ok {
		auditID, idErr := randomAuditID()
		if idErr != nil {
			return idErr
		}
		event := domain.AuditEvent{
			ID: auditID, OccurredAt: at, ActorID: actorUser.ID,
			ActorRoles: strings.Join(actorUser.RoleCodes(), ","), Action: domain.ActionUserResetPassword,
			ResourceType: "User", ResourceID: userID, Outcome: domain.OutcomeSuccess,
			RequestID: requestID, ClientIP: ip, DurationMS: time.Since(start).Milliseconds(),
		}
		err = atomicStore.ResetPasswordWithAudit(ctx, userID, seed.DemoPasswordHash, actorUser.ID, event)
		atomicAudit = true
	} else {
		err = s.Master.ResetPassword(ctx, userID, seed.DemoPasswordHash, actorUser.ID, at)
	}
	if err != nil {
		mapped := mapUserWrite(err)
		s.auditMasterErr(ctx, start, actorUser, domain.ActionUserResetPassword, "User", userID, requestID, ip, mapped)
		return mapped
	}
	if !atomicAudit {
		s.auditMaster(ctx, start, actorUser, domain.ActionUserResetPassword, "User", userID, domain.OutcomeSuccess, "", requestID, ip)
	}
	return nil
}

func gateActiveDepartment(d *domain.Department, err error) error {
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return &Error{Status: 404, Code: CodeDepartmentNotFound, Message: msgDepartmentNotFound}
		}
		return err
	}
	if d == nil || d.Status != domain.StatusActive {
		return &Error{Status: 409, Code: CodeDepartmentInactive, Message: msgDepartmentInactive}
	}
	return nil
}

func gateActiveDoctor(d *domain.Doctor, err error) error {
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return &Error{Status: 404, Code: CodeDoctorNotFound, Message: msgDoctorNotFound}
		}
		return err
	}
	if d == nil || d.Status != domain.StatusActive {
		return &Error{Status: 409, Code: CodeDoctorInactive, Message: msgDoctorInactive}
	}
	return nil
}

func (s *Service) requireDepartment(ctx context.Context, departmentID string) error {
	if s.Registers == nil {
		return errors.New("register store is not configured")
	}
	_, err := s.Registers.GetDepartment(ctx, departmentID)
	if errors.Is(err, port.ErrNotFound) {
		return &Error{Status: 404, Code: CodeDepartmentNotFound, Message: msgDepartmentNotFound}
	}
	return err
}

func (s *Service) loadDepartment(ctx context.Context, id string) (*domain.Department, error) {
	id = strings.TrimSpace(id)
	if id == "" || s.Master == nil {
		return nil, &Error{Status: 404, Code: CodeDepartmentNotFound, Message: msgDepartmentNotFound}
	}
	row, err := s.Master.GetDepartment(ctx, id)
	if errors.Is(err, port.ErrNotFound) {
		return nil, &Error{Status: 404, Code: CodeDepartmentNotFound, Message: msgDepartmentNotFound}
	}
	return row, err
}

func (s *Service) loadDoctor(ctx context.Context, id string) (*domain.Doctor, error) {
	id = strings.TrimSpace(id)
	if id == "" || s.Master == nil {
		return nil, &Error{Status: 404, Code: CodeDoctorNotFound, Message: msgDoctorNotFound}
	}
	row, err := s.Master.GetDoctor(ctx, id)
	if errors.Is(err, port.ErrNotFound) {
		return nil, &Error{Status: 404, Code: CodeDoctorNotFound, Message: msgDoctorNotFound}
	}
	return row, err
}

func (s *Service) loadCatalog(ctx context.Context, id string) (*domain.CatalogItem, error) {
	id = strings.TrimSpace(id)
	if id == "" || s.Master == nil {
		return nil, &Error{Status: 404, Code: CodeCatalogItemNotFound, Message: msgCatalogItemNotFound}
	}
	row, err := s.Master.GetCatalog(ctx, id)
	if errors.Is(err, port.ErrNotFound) {
		return nil, &Error{Status: 404, Code: CodeCatalogItemNotFound, Message: msgCatalogItemNotFound}
	}
	return row, err
}

func (s *Service) requireAdmin(ctx context.Context, claims *auth.Claims) (*domain.User, error) {
	return s.requireAdminPermission(ctx, claims, domain.PermAdminAll)
}

// requireOutpatientMasterRead gates the reference data used by outpatient
// workstations without granting any system-settings maintenance capability.
// The existing outpatient permissions are the source of truth for the read
// use cases: registration staff need encounter permissions, while clinicians
// need order/diagnosis permissions. Access administrators also need the
// department/doctor lists to edit role scopes; those admin permissions grant
// read-only reference access, while writes remain protected by admin:*.
// Non-admin callers are restricted to active rows by the list methods above.
func (s *Service) requireOutpatientMasterRead(ctx context.Context, claims *auth.Claims, permissions ...string) (*domain.User, error) {
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if hasPerm(user, domain.PermAdminAll) ||
		hasAnyPermModule(user, "master", domain.PermAdminUser, domain.PermAdminRole, domain.PermAdminPermission) ||
		hasAnyPermModule(user, "outpatient", permissions...) {
		return user, nil
	}
	return nil, forbidden()
}

func (s *Service) requireAdminPermission(ctx context.Context, claims *auth.Claims, permission string) (*domain.User, error) {
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !hasPerm(user, permission) {
		return nil, forbidden()
	}
	return user, nil
}

func (s *Service) requireAdminAudit(ctx context.Context, claims *auth.Claims, start time.Time, action, resourceType, resourceID, requestID, ip string) (*domain.User, error) {
	return s.requireAdminPermissionAudit(ctx, claims, domain.PermAdminAll, start, action, resourceType, resourceID, requestID, ip)
}

func (s *Service) requireAdminPermissionAudit(ctx context.Context, claims *auth.Claims, permission string, start time.Time, action, resourceType, resourceID, requestID, ip string) (*domain.User, error) {
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !hasPerm(user, permission) {
		e := forbidden()
		s.auditMaster(ctx, start, user, action, resourceType, resourceID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	return user, nil
}

func (s *Service) auditMaster(ctx context.Context, start time.Time, user *domain.User, action, resourceType, resourceID, outcome, errorCode, requestID, ip string) {
	actorID := ""
	roles := ""
	if user != nil {
		actorID = user.ID
		roles = strings.Join(user.RoleCodes(), ",")
	}
	level := "info"
	if outcome != domain.OutcomeSuccess {
		level = "warn"
	}
	s.record(ctx, start, domain.AuditEvent{
		ActorID: actorID, ActorRoles: roles, Action: action, ResourceType: resourceType, ResourceID: resourceID,
		Outcome: outcome, ErrorCode: errorCode, RequestID: requestID, ClientIP: ip,
	}, level)
}

func (s *Service) auditMasterErr(ctx context.Context, start time.Time, user *domain.User, action, resourceType, resourceID, requestID, ip string, err error) {
	code := ""
	var ae *Error
	if errors.As(err, &ae) {
		code = ae.Code
	}
	s.auditMaster(ctx, start, user, action, resourceType, resourceID, domain.OutcomeFail, code, requestID, ip)
}

func normalizeMasterFilter(status, q, departmentID, itemType string) (masterFilter, error) {
	f := masterFilter{
		Status: strings.TrimSpace(status), Q: strings.TrimSpace(q),
		DepartmentID: strings.TrimSpace(departmentID), ItemType: strings.TrimSpace(itemType),
	}
	var fields []string
	if f.Status != "" && !statusActiveInactive(f.Status) {
		fields = addField(fields, "status")
	}
	if len(f.Q) > 64 {
		fields = addField(fields, "q")
	}
	if len(f.DepartmentID) > 64 {
		fields = addField(fields, "departmentId")
	}
	if f.ItemType != "" && f.ItemType != domain.ItemTypeDrug && f.ItemType != domain.ItemTypeTreat {
		fields = addField(fields, "itemType")
	}
	if len(fields) > 0 {
		return f, validationFields(fields)
	}
	return f, nil
}

func normalizeDeptCreate(in CreateDepartmentInput) (code, name, status string, sortNo int, fields []string) {
	code = strings.TrimSpace(in.DeptCode)
	name = strings.TrimSpace(in.DeptName)
	status = domain.StatusActive
	if code == "" || len(code) > 32 {
		fields = addField(fields, "deptCode")
	}
	if name == "" || utf8.RuneCountInString(name) > 64 {
		fields = addField(fields, "deptName")
	}
	if in.Status != nil {
		status = strings.TrimSpace(*in.Status)
		if !statusActiveInactive(status) {
			fields = addField(fields, "status")
		}
	}
	if in.SortNo != nil {
		sortNo = *in.SortNo
		if !sortInRange(sortNo) {
			fields = addField(fields, "sortNo")
		}
	}
	return code, name, status, sortNo, fields
}

func applyDeptPatch(cur domain.Department, in PatchDepartmentInput) (domain.Department, []string) {
	var fields []string
	if in.DeptCode != nil {
		cur.DeptCode = strings.TrimSpace(*in.DeptCode)
		if cur.DeptCode == "" || len(cur.DeptCode) > 32 {
			fields = addField(fields, "deptCode")
		}
	}
	if in.DeptName != nil {
		cur.DeptName = strings.TrimSpace(*in.DeptName)
		if cur.DeptName == "" || utf8.RuneCountInString(cur.DeptName) > 64 {
			fields = addField(fields, "deptName")
		}
	}
	if in.Status != nil {
		cur.Status = strings.TrimSpace(*in.Status)
		if !statusActiveInactive(cur.Status) {
			fields = addField(fields, "status")
		}
	}
	if in.SortNo != nil {
		cur.SortNo = *in.SortNo
		if !sortInRange(cur.SortNo) {
			fields = addField(fields, "sortNo")
		}
	}
	return cur, fields
}

func normalizeDoctorCreate(in CreateDoctorInput) (code, name, deptID, status string, fields []string) {
	code = strings.TrimSpace(in.DoctorCode)
	name = strings.TrimSpace(in.DoctorName)
	deptID = strings.TrimSpace(in.DepartmentID)
	status = domain.StatusActive
	if code == "" || len(code) > 32 {
		fields = addField(fields, "doctorCode")
	}
	if name == "" || utf8.RuneCountInString(name) > 64 {
		fields = addField(fields, "doctorName")
	}
	if deptID == "" || len(deptID) > 64 {
		fields = addField(fields, "departmentId")
	}
	if in.Status != nil {
		status = strings.TrimSpace(*in.Status)
		if !statusActiveInactive(status) {
			fields = addField(fields, "status")
		}
	}
	return code, name, deptID, status, fields
}

func applyDoctorPatch(cur domain.Doctor, in PatchDoctorInput) (domain.Doctor, []string) {
	var fields []string
	if in.DoctorCode != nil {
		cur.DoctorCode = strings.TrimSpace(*in.DoctorCode)
		if cur.DoctorCode == "" || len(cur.DoctorCode) > 32 {
			fields = addField(fields, "doctorCode")
		}
	}
	if in.DoctorName != nil {
		cur.DoctorName = strings.TrimSpace(*in.DoctorName)
		if cur.DoctorName == "" || utf8.RuneCountInString(cur.DoctorName) > 64 {
			fields = addField(fields, "doctorName")
		}
	}
	if in.DepartmentID != nil {
		cur.DepartmentID = strings.TrimSpace(*in.DepartmentID)
		if cur.DepartmentID == "" || len(cur.DepartmentID) > 64 {
			fields = addField(fields, "departmentId")
		}
	}
	if in.Status != nil {
		cur.Status = strings.TrimSpace(*in.Status)
		if !statusActiveInactive(cur.Status) {
			fields = addField(fields, "status")
		}
	}
	return cur, fields
}

func normalizeCatalogCreate(in CreateCatalogInput) (domain.CatalogItem, []string, *Error) {
	var fields []string
	item := domain.CatalogItem{
		ItemNo: strings.TrimSpace(in.ItemNo), ItemName: strings.TrimSpace(in.ItemName),
		ItemType: strings.TrimSpace(in.ItemType), Units: strings.TrimSpace(in.Units), Status: domain.StatusActive,
	}
	if in.Spec != nil {
		item.Spec = strings.TrimSpace(*in.Spec)
		if utf8.RuneCountInString(item.Spec) > 64 {
			fields = addField(fields, "spec")
		}
	}
	if item.ItemNo == "" || len(item.ItemNo) > 32 {
		fields = addField(fields, "itemNo")
	}
	if item.ItemName == "" || utf8.RuneCountInString(item.ItemName) > 128 {
		fields = addField(fields, "itemName")
	}
	if item.ItemType != domain.ItemTypeDrug && item.ItemType != domain.ItemTypeTreat {
		fields = addField(fields, "itemType")
	}
	if item.Units == "" || utf8.RuneCountInString(item.Units) > 16 {
		fields = addField(fields, "units")
	}
	if in.Status != nil {
		item.Status = strings.TrimSpace(*in.Status)
		if !statusActiveInactive(item.Status) {
			fields = addField(fields, "status")
		}
	}
	controlled, ok := mergeControlled(in.ControlledFlag, in.IsControlled, false)
	if !ok {
		fields = addField(fields, "controlledFlag")
	}
	item.Controlled = controlled
	if in.Price == nil {
		return item, fields, invalidPrice()
	}
	cents, priceOK := priceCents(*in.Price)
	if !priceOK {
		return item, fields, invalidPrice()
	}
	item.PriceCents = cents
	return item, fields, nil
}

func applyCatalogPatch(cur domain.CatalogItem, in PatchCatalogInput) (domain.CatalogItem, []string, *Error) {
	var fields []string
	if in.ItemNo != nil {
		cur.ItemNo = strings.TrimSpace(*in.ItemNo)
		if cur.ItemNo == "" || len(cur.ItemNo) > 32 {
			fields = addField(fields, "itemNo")
		}
	}
	if in.ItemName != nil {
		cur.ItemName = strings.TrimSpace(*in.ItemName)
		if cur.ItemName == "" || utf8.RuneCountInString(cur.ItemName) > 128 {
			fields = addField(fields, "itemName")
		}
	}
	if in.Spec != nil {
		cur.Spec = strings.TrimSpace(*in.Spec)
		if utf8.RuneCountInString(cur.Spec) > 64 {
			fields = addField(fields, "spec")
		}
	}
	if in.Units != nil {
		cur.Units = strings.TrimSpace(*in.Units)
		if cur.Units == "" || utf8.RuneCountInString(cur.Units) > 16 {
			fields = addField(fields, "units")
		}
	}
	if in.Status != nil {
		cur.Status = strings.TrimSpace(*in.Status)
		if !statusActiveInactive(cur.Status) {
			fields = addField(fields, "status")
		}
	}
	if in.ControlledFlag != nil || in.IsControlled != nil {
		controlled, ok := mergeControlled(in.ControlledFlag, in.IsControlled, cur.Controlled)
		if !ok {
			fields = addField(fields, "controlledFlag")
		}
		cur.Controlled = controlled
	}
	if in.Price != nil {
		cents, ok := priceCents(*in.Price)
		if !ok {
			return cur, fields, invalidPrice()
		}
		cur.PriceCents = cents
	}
	return cur, fields, nil
}

func normalizeUserCreate(in CreateUserInput) (username, display string, roles []domain.Role, status string, fields []string) {
	username = strings.TrimSpace(in.Username)
	login := strings.TrimSpace(in.LoginName)
	if username == "" {
		username = login
	}
	if login != "" && username != "" && !strings.EqualFold(username, login) {
		fields = addField(fields, "username")
	}
	display = strings.TrimSpace(in.DisplayName)
	status = domain.StatusActive
	if username == "" || len(username) > 64 {
		fields = addField(fields, "username")
	}
	if display == "" || utf8.RuneCountInString(display) > 128 {
		fields = addField(fields, "displayName")
	}
	if in.Status != nil {
		status = strings.TrimSpace(*in.Status)
		if status != domain.StatusActive && status != domain.StatusDisabled && status != domain.StatusInactive {
			fields = addField(fields, "status")
		}
	}
	return username, display, []domain.Role{}, status, fields
}

func (s *Service) parseAssignableRoles(ctx context.Context, in []string) ([]domain.Role, bool) {
	if in == nil {
		return []domain.Role{}, true
	}
	if s.Master == nil {
		return parseRoles(in)
	}
	store, ok := s.Master.(port.AccessStore)
	if !ok || store == nil {
		return parseRoles(in)
	}
	rows, err := store.ListAccessRoles(ctx)
	if err != nil {
		return nil, false
	}
	valid := make(map[string]bool, len(rows))
	for _, role := range rows {
		if role.Status == domain.StatusActive {
			valid[strings.ToUpper(role.Code)] = true
		}
	}
	seen := make(map[string]bool, len(in))
	roles := make([]domain.Role, 0, len(in))
	for _, raw := range in {
		code := strings.ToUpper(strings.TrimSpace(raw))
		if code == "" || len(code) > 32 || !valid[code] {
			return nil, false
		}
		if !seen[code] {
			seen[code] = true
			roles = append(roles, domain.Role(code))
		}
	}
	return roles, true
}

func (s *Service) hasAdminPermissionRole(ctx context.Context, codes []string) (bool, error) {
	if len(codes) == 0 {
		return false, nil
	}
	wanted := make(map[string]bool, len(codes))
	for _, code := range codes {
		wanted[strings.ToUpper(strings.TrimSpace(code))] = true
	}
	if store, ok := s.Master.(port.AccessStore); ok && store != nil {
		roles, err := store.ListAccessRoles(ctx)
		if err != nil {
			return false, err
		}
		for _, role := range roles {
			if wanted[strings.ToUpper(role.Code)] && roleHasAdminPermission(role) {
				return true, nil
			}
		}
		return false, nil
	}
	for role := range domain.RolePermissions {
		if !wanted[strings.ToUpper(string(role))] {
			continue
		}
		for _, permission := range domain.RolePermissions[role] {
			if isAdminPermissionCode(permission) {
				return true, nil
			}
		}
	}
	return false, nil
}

func parseRoles(in []string) ([]domain.Role, bool) {
	if in == nil {
		return []domain.Role{}, true
	}
	seen := map[domain.Role]bool{}
	out := make([]domain.Role, 0, len(in))
	for _, raw := range in {
		switch strings.ToUpper(strings.TrimSpace(raw)) {
		case "REGISTRAR", "DOCTOR", "CASHIER", "PHARMACIST", "SYSTEM":
			role := domain.Role(strings.ToUpper(strings.TrimSpace(raw)))
			if seen[role] {
				continue
			}
			seen[role] = true
			out = append(out, role)
		default:
			return nil, false
		}
	}
	return out, true
}

func hasRoleCode(codes []string, want string) bool {
	for _, code := range codes {
		if strings.EqualFold(strings.TrimSpace(code), want) {
			return true
		}
	}
	return false
}

func boolDefault(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}

func validateUserProfile(description, cardNo, inputCode string) []string {
	var fields []string
	if utf8.RuneCountInString(strings.TrimSpace(description)) > 512 {
		fields = addField(fields, "description")
	}
	if len(strings.TrimSpace(cardNo)) > 64 {
		fields = addField(fields, "cardNo")
	}
	if len(strings.TrimSpace(inputCode)) > 64 {
		fields = addField(fields, "inputCode")
	}
	return fields
}

func validateAdminPassword(password string) []string {
	if strings.TrimSpace(password) == "" {
		return nil
	}
	if len(password) < 8 || len(password) > 72 {
		return []string{"password"}
	}
	return nil
}

func hasDeveloperOnlyUserCreateFields(in CreateUserInput) bool {
	return boolDefault(in.Hidden, false) || boolDefault(in.System, false) || boolDefault(in.Developer, false)
}

func hasDeveloperOnlyUserPatchFields(in PatchUserInput) bool {
	return in.Hidden != nil || in.System != nil || in.Developer != nil
}

func canManageProtectedUsers(actor *domain.User) bool {
	return actor != nil && (actor.Developer || actor.Super)
}

func canEditDeveloperFlags(actor *domain.User) bool {
	return actor != nil && actor.Developer
}

func canEditSuperFlag(actor *domain.User) bool {
	return actor != nil && (actor.Developer || actor.Super)
}

func canManageSystemAccess(actor *domain.User) bool {
	return actor != nil && (actor.Developer || actor.Super || actor.HasSystemRole())
}

func isProtectedUser(target *domain.User) bool {
	return target != nil && (target.System || target.Super || target.Developer || target.Hidden)
}

func sameUserAuthorization(a, b *domain.User) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Status != b.Status || a.AccountLocked != b.AccountLocked || a.AccountStopped != b.AccountStopped ||
		a.MustChangePassword != b.MustChangePassword || a.CannotModifyPassword != b.CannotModifyPassword ||
		a.Hidden != b.Hidden || a.System != b.System || a.Super != b.Super || a.Developer != b.Developer || len(a.Roles) != len(b.Roles) {
		return false
	}
	for i := range a.Roles {
		if a.Roles[i] != b.Roles[i] {
			return false
		}
	}
	return true
}

func mergeControlled(flag, alt *bool, fallback bool) (bool, bool) {
	if flag != nil && alt != nil && *flag != *alt {
		return false, false
	}
	if flag != nil {
		return *flag, true
	}
	if alt != nil {
		return *alt, true
	}
	return fallback, true
}

func readonlyDictValues(dictType string) ([]string, bool) {
	switch dictType {
	case "sex":
		return []string{"M", "F", "U"}, true
	case "cardStatus":
		return []string{"NOT_SENT", "NORMAL", "LOST", "CANCELLED"}, true
	case "payMethod":
		return []string{"CASH", "OTHER_E", "WECHAT", "ALIPAY", "OTHER"}, true
	case "registerStatus":
		return []string{"NORMAL", "USED", "BACK", "DIAGNOSING", "DIAGNOSED", "PAUSED"}, true
	case "orderStatus":
		return []string{"SAVE", "COMMIT", "PROC", "ABATE"}, true
	default:
		return nil, false
	}
}

func maintainableDict(dictType string) bool {
	return dictType == dictRegisterSort
}

func statusActiveInactive(status string) bool {
	return status == domain.StatusActive || status == domain.StatusInactive
}

func sortInRange(n int) bool {
	return n >= math.MinInt32 && n <= math.MaxInt32
}

func signedMilli(v float64) (int64, bool) {
	if math.IsNaN(v) || math.IsInf(v, 0) || v > 999999999999.999 || v < -999999999999.999 {
		return 0, false
	}
	m := math.Round(v * 1000)
	if m > float64(maxQtyMilli) || m < -float64(maxQtyMilli) {
		return 0, false
	}
	return int64(m), true
}

func invalidPrice() *Error {
	return &Error{Status: 400, Code: CodeInvalidPrice, Message: msgInvalidPrice}
}

func dictReadonly() *Error {
	return &Error{Status: 409, Code: CodeDictTypeReadonly, Message: msgDictTypeReadonly}
}

func mapDeptWrite(err error) error {
	switch {
	case errors.Is(err, port.ErrNotFound):
		return &Error{Status: 404, Code: CodeDepartmentNotFound, Message: msgDepartmentNotFound}
	case errors.Is(err, port.ErrDuplicateDeptCode):
		return &Error{Status: 409, Code: CodeDeptCodeDuplicate, Message: msgDeptCodeDuplicate}
	default:
		return err
	}
}

func mapDoctorWrite(err error) error {
	switch {
	case errors.Is(err, port.ErrNotFound):
		return &Error{Status: 404, Code: CodeDoctorNotFound, Message: msgDoctorNotFound}
	case errors.Is(err, port.ErrDuplicateDoctorCode):
		return &Error{Status: 409, Code: CodeDoctorCodeDuplicate, Message: msgDoctorCodeDuplicate}
	default:
		return err
	}
}

func mapCatalogWrite(err error) error {
	switch {
	case errors.Is(err, port.ErrNotFound):
		return &Error{Status: 404, Code: CodeCatalogItemNotFound, Message: msgCatalogItemNotFound}
	case errors.Is(err, port.ErrDuplicateItemNo):
		return &Error{Status: 409, Code: CodeItemNoDuplicate, Message: msgItemNoDuplicate}
	default:
		return err
	}
}

func mapAdjustErr(err error) error {
	switch {
	case errors.Is(err, port.ErrNotDrugSKU):
		return &Error{Status: 409, Code: CodeNotDrugSKU, Message: msgNotDrugSKU}
	case errors.Is(err, port.ErrAdjustNegative):
		return &Error{Status: 409, Code: CodeAdjustResultNegative, Message: msgAdjustResultNegative}
	case errors.Is(err, port.ErrConflict):
		return &Error{Status: 409, Code: CodeConflict, Message: msgConflict}
	default:
		return err
	}
}

func mapDictWrite(err error) error {
	switch {
	case errors.Is(err, port.ErrNotFound):
		return notFound()
	case errors.Is(err, port.ErrDuplicateDictCode):
		return &Error{Status: 409, Code: CodeDictCodeDuplicate, Message: msgDictCodeDuplicate}
	case errors.Is(err, port.ErrDictTypeReadonly):
		return dictReadonly()
	default:
		return err
	}
}

func mapUserWrite(err error) error {
	switch {
	case errors.Is(err, port.ErrNotFound):
		return notFound()
	case errors.Is(err, port.ErrDuplicateUsername):
		return &Error{Status: 409, Code: CodeUsernameDuplicate, Message: msgUsernameDuplicate}
	case errors.Is(err, port.ErrLastSystemAdmin):
		return &Error{Status: 409, Code: CodeLastSystemAdmin, Message: msgLastSystemAdmin}
	case errors.Is(err, port.ErrAuditUnavailable):
		return &Error{Status: 503, Code: CodeAuditUnavailable, Message: msgAuditUnavailable, Details: map[string]any{"retryable": true}}
	case errors.Is(err, port.ErrUserAdminBusy):
		return &Error{Status: 503, Code: CodeUserAdminBusy, Message: msgUserAdminBusy, Details: map[string]any{"retryable": true}}
	case errors.Is(err, port.ErrDuplicateUserRoleScope):
		return &Error{Status: 409, Code: "ROLE_SCOPE_CONFLICT", Message: "用户角色科室范围已被并发修改，请刷新后重试", Details: map[string]any{"retryable": true}}
	case errors.Is(err, port.ErrInvalidUserRoleScope):
		return validationFields([]string{"roleScopes"})
	default:
		return err
	}
}

func normalizeAndValidateRoleScopes(ctx context.Context, master port.MasterStore, access port.AccessStore, userID string, assignedRoles []string, input []UserRoleScopeInput) ([]domain.UserRoleScope, []string, error) {
	roles, err := access.ListAccessRoles(ctx)
	if err != nil {
		return nil, nil, err
	}
	knownRoles := make(map[string]bool, len(roles))
	for _, role := range roles {
		if role.Status == domain.StatusActive {
			knownRoles[role.Code] = true
		}
	}
	assigned := make(map[string]bool, len(assignedRoles))
	for _, role := range assignedRoles {
		assigned[strings.ToUpper(strings.TrimSpace(role))] = true
	}
	departments, err := master.ListDepartments(ctx, port.DepartmentQuery{})
	if err != nil {
		return nil, nil, err
	}
	departmentStatus := make(map[string]string, len(departments))
	for _, department := range departments {
		departmentStatus[department.DepartmentID] = department.Status
	}

	fields := make([]string, 0)
	rows := make([]domain.UserRoleScope, 0, len(input))
	seen := make(map[string]bool, len(input))
	for i, item := range input {
		roleCode := strings.ToUpper(strings.TrimSpace(item.RoleCode))
		departmentID := strings.TrimSpace(item.DepartmentID)
		status := strings.ToUpper(strings.TrimSpace(item.Status))
		prefix := "roleScopes[" + strconv.Itoa(i) + "]."
		if roleCode == "" || !knownRoles[roleCode] || !assigned[roleCode] {
			fields = addField(fields, prefix+"roleCode")
		}
		if departmentID == "" || departmentStatus[departmentID] == "" || status == domain.StatusActive && departmentStatus[departmentID] != domain.StatusActive {
			fields = addField(fields, prefix+"departmentId")
		}
		if status != domain.StatusActive && status != domain.StatusDisabled {
			fields = addField(fields, prefix+"status")
		}
		key := roleCode + "\x00" + departmentID
		if seen[key] {
			fields = addField(fields, prefix+"departmentId")
		}
		seen[key] = true
		rows = append(rows, domain.UserRoleScope{UserID: userID, RoleCode: roleCode, DepartmentID: departmentID, Status: status})
	}
	return rows, fields, nil
}

func roleCodes(roles []domain.Role) []string {
	codes := make([]string, 0, len(roles))
	for _, role := range roles {
		codes = append(codes, string(role))
	}
	return codes
}

func toDepartmentDTO(d domain.Department) DepartmentDTO {
	return DepartmentDTO{
		DepartmentID: d.DepartmentID, DeptCode: d.DeptCode, DeptName: d.DeptName, Status: d.Status, SortNo: d.SortNo,
		CreatedAt: formatMasterTime(d.CreatedAt), UpdatedAt: formatMasterTime(d.UpdatedAt),
	}
}

func toDoctorDTO(d domain.Doctor) DoctorDTO {
	return DoctorDTO{
		DoctorID: d.DoctorID, DoctorCode: d.DoctorCode, DoctorName: d.DoctorName, DepartmentID: d.DepartmentID, Status: d.Status,
		CreatedAt: formatMasterTime(d.CreatedAt), UpdatedAt: formatMasterTime(d.UpdatedAt),
	}
}

func toCatalogDTO(c domain.CatalogItem) CatalogItemDTO {
	var spec *string
	if c.Spec != "" {
		s := c.Spec
		spec = &s
	}
	return CatalogItemDTO{
		CatalogItemID: c.CatalogItemID, ItemNo: c.ItemNo, ItemName: c.ItemName, ItemType: c.ItemType,
		Spec: spec, Units: c.Units, Price: float64(c.PriceCents) / 100, Status: c.Status,
		ControlledFlag: c.Controlled, IsControlled: c.Controlled,
		CreatedAt: formatMasterTime(c.CreatedAt), UpdatedAt: formatMasterTime(c.UpdatedAt),
	}
}

func toDictDTO(it domain.DictItem) DictItemDTO {
	return DictItemDTO{
		DictItemID: it.DictItemID, DictType: it.DictType, Code: it.Code, Name: it.Name, Status: it.Status, SortNo: it.SortNo,
		CreatedAt: formatMasterTime(it.CreatedAt), UpdatedAt: formatMasterTime(it.UpdatedAt),
	}
}

func toUserAdminDTO(u domain.User) UserAdminDTO {
	roles := u.RoleCodes()
	if roles == nil {
		roles = []string{}
	}
	roleScopes := make([]UserRoleScopeDTO, 0, len(u.RoleScopes))
	for _, scope := range u.RoleScopes {
		roleScopes = append(roleScopes, UserRoleScopeDTO{RoleCode: scope.RoleCode, DepartmentID: scope.DepartmentID, Status: scope.Status})
	}
	return UserAdminDTO{
		UserID: u.ID, Username: u.Username, LoginName: u.Username, DisplayName: u.DisplayName,
		Description: u.Description, CardNo: u.CardNo, InputCode: u.InputCode, ServiceStar: cloneInt(u.ServiceStar),
		ExpandInterface1: cloneInt32(u.ExpandInterface1), ExpandInterface2: cloneInt32(u.ExpandInterface2),
		InputCodeCompareMode: cloneInt32(u.InputCodeCompareMode), Status: u.Status, Roles: roles, RoleScopes: roleScopes,
		MustChangePassword: u.MustChangePassword, CannotModifyPassword: u.CannotModifyPassword,
		AccountLocked: u.AccountLocked, AccountStopped: u.AccountStopped, Hidden: u.Hidden,
		System: u.System, Super: u.Super, Developer: u.Developer,
		LastLoginAt: formatMasterTime(u.LastLoginAt), LastLoginIP: u.LastLoginIP,
		LockUntil: formatMasterTime(u.LockUntil), PasswordChangedAt: formatMasterTime(u.PasswordChangedAt),
		FailedLoginCount: u.FailedLoginCount,
		CreatedAt:        formatMasterTime(u.CreatedAt), UpdatedAt: formatMasterTime(u.UpdatedAt),
	}
}

func formatMasterTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
