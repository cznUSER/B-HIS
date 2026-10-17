export function roleScopeDepartmentState(scope, departments) {
  const departmentId = scope && scope.departmentId
  const department = (Array.isArray(departments) ? departments : []).find((item) => item.departmentId === departmentId)
  if (!department) return { valid: false, label: '当前无效 · 科室不在目录中', department: null }
  if (department.status !== 'ACTIVE') return { valid: false, label: '科室已停用 · 当前无效', department }
  if (!scope || scope.status !== 'ACTIVE') return { valid: false, label: '范围已停用', department }
  return { valid: true, label: '生效中', department }
}

export function canEnableRoleScope(scope, departments) {
  const state = roleScopeDepartmentState(scope, departments)
  return scope && scope.status !== 'ACTIVE' && state.department?.status === 'ACTIVE'
}

export function buildAssignedRoleScopePayload(scopes, assignedRoles) {
  const assigned = new Set(Array.isArray(assignedRoles) ? assignedRoles : [])
  return (Array.isArray(scopes) ? scopes : [])
    .filter((scope) => assigned.has(scope.roleCode))
    .map(({ roleCode, departmentId, status }) => ({ roleCode, departmentId, status }))
}
