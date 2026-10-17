package memory

import (
	"context"
	"sort"
	"strings"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

var _ port.AccessStore = (*Store)(nil)
var _ port.AtomicAccessStore = (*Store)(nil)

func seedAccessControl(s *Store) {
	s.modules["outpatient"] = domain.AccessModule{ID: "module-outpatient", Code: "outpatient", Name: "门诊业务", Status: domain.StatusActive}
	s.modules["master"] = domain.AccessModule{ID: "module-master", Code: "master", Name: "系统设置", Status: domain.StatusActive}
	s.categories["patient"] = domain.PermissionCategory{ID: "patient", Name: "患者与挂号", Status: domain.StatusActive}
	s.categories["order"] = domain.PermissionCategory{ID: "order", Name: "医嘱与发药", Status: domain.StatusActive}
	s.categories["charge"] = domain.PermissionCategory{ID: "charge", Name: "收费与退费", Status: domain.StatusActive}
	s.categories["admin"] = domain.PermissionCategory{ID: "admin", Name: "系统管理", Status: domain.StatusActive}
	s.categories["inventory"] = domain.PermissionCategory{ID: "inventory", Name: "药品库存", Status: domain.StatusActive}
	s.categories["audit"] = domain.PermissionCategory{ID: "audit", Name: "操作日志", Status: domain.StatusActive}
	s.modes["password"] = domain.PermissionMode{Code: "password", Name: "用户名密码", Status: domain.StatusActive}
	s.modes["card"] = domain.PermissionMode{Code: "card", Name: "刷卡（二次认证关闭）", Status: domain.StatusInactive}
	for _, p := range []domain.AccessPermission{
		{Code: "patient:create", Name: "患者建档", ModuleCode: "outpatient", CategoryID: "patient", ModeCode: "password", Actions: []string{"N", "S"}, Status: domain.StatusActive},
		{Code: "encounter:create", Name: "挂号", ModuleCode: "outpatient", CategoryID: "patient", ModeCode: "password", Actions: []string{"N", "S"}, Status: domain.StatusActive},
		{Code: "encounter:check-in", Name: "签到", ModuleCode: "outpatient", CategoryID: "patient", ModeCode: "password", Actions: []string{"M", "S"}, Status: domain.StatusActive},
		{Code: "encounter:cancel", Name: "退号", ModuleCode: "outpatient", CategoryID: "patient", ModeCode: "password", Actions: []string{"D", "S"}, Status: domain.StatusActive},
		{Code: domain.PermDiagnoseWrite, Name: "诊断录入", ModuleCode: "outpatient", CategoryID: "patient", ModeCode: "password", Actions: []string{"N", "M", "D", "S"}, Status: domain.StatusActive},
		{Code: "order:submit", Name: "提交医嘱", ModuleCode: "outpatient", CategoryID: "order", ModeCode: "password", Actions: []string{"N", "S"}, Status: domain.StatusActive},
		{Code: "order:recipe-check", Name: "处方审核", ModuleCode: "outpatient", CategoryID: "order", ModeCode: "password", Actions: []string{"M", "S"}, Status: domain.StatusActive},
		{Code: "charge:pay", Name: "收费", ModuleCode: "outpatient", CategoryID: "charge", ModeCode: "password", Actions: []string{"N", "S"}, Status: domain.StatusActive},
		{Code: "charge:refund", Name: "退费", ModuleCode: "outpatient", CategoryID: "charge", ModeCode: "password", Actions: []string{"N", "S"}, Status: domain.StatusActive},
		{Code: "dispense:create", Name: "发药", ModuleCode: "outpatient", CategoryID: "order", ModeCode: "password", Actions: []string{"N", "S"}, Status: domain.StatusActive},
		{Code: "dispense:return", Name: "退药", ModuleCode: "outpatient", CategoryID: "order", ModeCode: "password", Actions: []string{"D", "S"}, Status: domain.StatusActive},
		{Code: "inventory:read", Name: "库存查询", ModuleCode: "master", CategoryID: "inventory", ModeCode: "password", Actions: []string{"S"}, Status: domain.StatusActive},
		{Code: domain.PermInventoryBatchRead, Name: "库存批次/效期查询", ModuleCode: "master", CategoryID: "inventory", ModeCode: "password", Actions: []string{"S"}, Status: domain.StatusActive},
		{Code: "inventory:adjust", Name: "库存调整", ModuleCode: "master", CategoryID: "inventory", ModeCode: "password", Actions: []string{"M", "S"}, Status: domain.StatusActive},
		{Code: "audit:read", Name: "操作日志查询", ModuleCode: "master", CategoryID: "audit", ModeCode: "password", Actions: []string{"S"}, Status: domain.StatusActive},
		{Code: "admin:*", Name: "全部管理权限", ModuleCode: "master", CategoryID: "admin", ModeCode: "password", Actions: []string{"N", "M", "D", "S"}, Status: domain.StatusActive},
		{Code: "admin:user", Name: "用户管理", ModuleCode: "master", CategoryID: "admin", ModeCode: "password", Actions: []string{"N", "M", "D", "S"}, Status: domain.StatusActive},
		{Code: "admin:role", Name: "角色管理", ModuleCode: "master", CategoryID: "admin", ModeCode: "password", Actions: []string{"N", "M", "D", "S"}, Status: domain.StatusActive},
		{Code: "admin:permission", Name: "权限管理", ModuleCode: "master", CategoryID: "admin", ModeCode: "password", Actions: []string{"N", "M", "D", "S"}, Status: domain.StatusActive},
	} {
		s.permissions[p.Code] = p
	}
	for role, permissions := range domain.RolePermissions {
		name := string(role)
		switch role {
		case domain.RoleRegistrar:
			name = "建档挂号员"
		case domain.RoleDoctor:
			name = "门诊医生"
		case domain.RoleCashier:
			name = "收费员"
		case domain.RolePharmacist:
			name = "药房人员"
		case domain.RoleSystem:
			name = "系统管理员"
			permissions = append(append([]string(nil), permissions...), "admin:user", "admin:role", "admin:permission", domain.PermInventoryAdjust, domain.PermAuditRead)
		}
		s.accessRoles[string(role)] = domain.AccessRole{Code: string(role), Name: name, PermissionCodes: append([]string(nil), permissions...), Status: domain.StatusActive, System: role == domain.RoleSystem}
	}
}

func (s *Store) ListAccessModules(_ context.Context) ([]domain.AccessModule, error) {
	s.accessMu.RLock()
	defer s.accessMu.RUnlock()
	out := make([]domain.AccessModule, 0, len(s.modules))
	for _, row := range s.modules {
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out, nil
}

func (s *Store) CreateAccessModule(_ context.Context, row domain.AccessModule) error {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	if _, ok := s.modules[row.Code]; ok {
		return port.ErrDuplicateAccessCatalog
	}
	s.modules[row.Code] = row
	return nil
}
func (s *Store) UpdateAccessModule(_ context.Context, row domain.AccessModule) error {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	if _, ok := s.modules[row.Code]; !ok {
		return port.ErrNotFound
	}
	s.modules[row.Code] = row
	return nil
}

func (s *Store) ListPermissionCategories(_ context.Context) ([]domain.PermissionCategory, error) {
	s.accessMu.RLock()
	defer s.accessMu.RUnlock()
	out := make([]domain.PermissionCategory, 0, len(s.categories))
	for _, row := range s.categories {
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *Store) CreatePermissionCategory(_ context.Context, row domain.PermissionCategory) error {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	if _, ok := s.categories[row.ID]; ok {
		return port.ErrDuplicateAccessCatalog
	}
	s.categories[row.ID] = row
	return nil
}
func (s *Store) UpdatePermissionCategory(_ context.Context, row domain.PermissionCategory) error {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	if _, ok := s.categories[row.ID]; !ok {
		return port.ErrNotFound
	}
	s.categories[row.ID] = row
	return nil
}

func (s *Store) ListPermissionModes(_ context.Context) ([]domain.PermissionMode, error) {
	s.accessMu.RLock()
	defer s.accessMu.RUnlock()
	out := make([]domain.PermissionMode, 0, len(s.modes))
	for _, row := range s.modes {
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out, nil
}

func (s *Store) CreatePermissionMode(_ context.Context, row domain.PermissionMode) error {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	if _, ok := s.modes[row.Code]; ok {
		return port.ErrDuplicateAccessCatalog
	}
	s.modes[row.Code] = row
	return nil
}
func (s *Store) UpdatePermissionMode(_ context.Context, row domain.PermissionMode) error {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	if _, ok := s.modes[row.Code]; !ok {
		return port.ErrNotFound
	}
	s.modes[row.Code] = row
	return nil
}

func (s *Store) ListAccessPermissions(_ context.Context, moduleCode, categoryID string) ([]domain.AccessPermission, error) {
	s.accessMu.RLock()
	defer s.accessMu.RUnlock()
	out := make([]domain.AccessPermission, 0)
	for _, row := range s.permissions {
		if moduleCode != "" && row.ModuleCode != moduleCode || categoryID != "" && row.CategoryID != categoryID {
			continue
		}
		row.Actions = append([]string(nil), row.Actions...)
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out, nil
}

func (s *Store) CreateAccessPermission(_ context.Context, permission domain.AccessPermission) error {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	if _, ok := s.permissions[permission.Code]; ok {
		return port.ErrDuplicateAccessPermission
	}
	permission.Actions = append([]string(nil), permission.Actions...)
	s.permissions[permission.Code] = permission
	return nil
}

func (s *Store) CreateAccessPermissionWithAudit(_ context.Context, permission domain.AccessPermission, event domain.AuditEvent) error {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	if _, ok := s.permissions[permission.Code]; ok {
		return port.ErrDuplicateAccessPermission
	}
	permission.Actions = append([]string(nil), permission.Actions...)
	s.permissions[permission.Code] = permission
	s.mu.Lock()
	s.appendAuditLocked(event)
	s.mu.Unlock()
	return nil
}

func (s *Store) UpdateAccessPermission(_ context.Context, permission domain.AccessPermission) error {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	if _, ok := s.permissions[permission.Code]; !ok {
		return port.ErrNotFound
	}
	permission.Actions = append([]string(nil), permission.Actions...)
	s.permissions[permission.Code] = permission
	return nil
}

func (s *Store) UpdateAccessPermissionWithAudit(_ context.Context, permission domain.AccessPermission, event domain.AuditEvent) error {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	if _, ok := s.permissions[permission.Code]; !ok {
		return port.ErrNotFound
	}
	permission.Actions = append([]string(nil), permission.Actions...)
	s.permissions[permission.Code] = permission
	s.mu.Lock()
	s.appendAuditLocked(event)
	s.mu.Unlock()
	return nil
}

func (s *Store) ListAccessRoles(_ context.Context) ([]domain.AccessRole, error) {
	s.accessMu.RLock()
	defer s.accessMu.RUnlock()
	out := make([]domain.AccessRole, 0, len(s.accessRoles))
	for _, row := range s.accessRoles {
		row.PermissionCodes = append([]string(nil), row.PermissionCodes...)
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out, nil
}

func (s *Store) CreateAccessRole(_ context.Context, role domain.AccessRole) error {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	if _, ok := s.accessRoles[role.Code]; ok {
		return port.ErrDuplicateUsername
	}
	role.PermissionCodes = uniqueStrings(role.PermissionCodes)
	s.accessRoles[role.Code] = role
	return nil
}

func (s *Store) CreateAccessRoleWithAudit(_ context.Context, role domain.AccessRole, event domain.AuditEvent) error {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	if _, ok := s.accessRoles[role.Code]; ok {
		return port.ErrDuplicateUsername
	}
	role.PermissionCodes = uniqueStrings(role.PermissionCodes)
	s.accessRoles[role.Code] = role
	s.mu.Lock()
	s.appendAuditLocked(event)
	s.mu.Unlock()
	return nil
}

func (s *Store) UpdateAccessRole(_ context.Context, role domain.AccessRole) error {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	if _, ok := s.accessRoles[role.Code]; !ok {
		return port.ErrNotFound
	}
	role.PermissionCodes = uniqueStrings(role.PermissionCodes)
	s.accessRoles[role.Code] = role
	return nil
}

func (s *Store) UpdateAccessRoleWithAudit(_ context.Context, role domain.AccessRole, event domain.AuditEvent) error {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	if _, ok := s.accessRoles[role.Code]; !ok {
		return port.ErrNotFound
	}
	role.PermissionCodes = uniqueStrings(role.PermissionCodes)
	s.accessRoles[role.Code] = role
	s.mu.Lock()
	s.appendAuditLocked(event)
	s.mu.Unlock()
	return nil
}

func (s *Store) ListUserPermissionOverrides(_ context.Context, userID string) ([]domain.PermissionOverride, error) {
	s.accessMu.RLock()
	defer s.accessMu.RUnlock()
	out := append([]domain.PermissionOverride(nil), s.overrides[userID]...)
	return out, nil
}

func (s *Store) ReplaceUserPermissionOverrides(_ context.Context, userID string, overrides []domain.PermissionOverride) error {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	for i := range overrides {
		overrides[i].UserID = userID
		overrides[i].Effect = strings.ToUpper(strings.TrimSpace(overrides[i].Effect))
	}
	s.overrides[userID] = append([]domain.PermissionOverride(nil), overrides...)
	return nil
}

func (s *Store) ReplaceUserPermissionOverridesWithAudit(_ context.Context, userID string, overrides []domain.PermissionOverride, event domain.AuditEvent) error {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	for i := range overrides {
		overrides[i].UserID = userID
		overrides[i].Effect = strings.ToUpper(strings.TrimSpace(overrides[i].Effect))
	}
	s.overrides[userID] = append([]domain.PermissionOverride(nil), overrides...)
	s.mu.Lock()
	s.appendAuditLocked(event)
	s.mu.Unlock()
	return nil
}

func (s *Store) CountSystemAdmins(_ context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, user := range s.byID {
		if user.Status == domain.StatusActive && user.HasSystemRole() {
			n++
		}
	}
	return n, nil
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, value := range in {
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}
