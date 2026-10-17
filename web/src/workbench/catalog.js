// 2026-10-12 P1-H05-HOME 门诊工作台：工作区顺序、徽标、按角色是否可进。

export const ROLE_LABELS = {
  REGISTRAR: '建档挂号员',
  DOCTOR: '门诊医生',
  CASHIER: '收费员',
  PHARMACIST: '药师',
  SYSTEM: '系统',
}

export const STATUS_LABELS = {
  available: '可用',
  developing: '开发中',
  disabled: '未启用',
}

// Effective permissions are returned by /auth/login and /auth/me. The role
// list remains a compatibility fallback for older responses and mock callers.
export const OUTPATIENT_PERMISSION_CODES = Object.freeze({
  patient: ['patient:create', 'encounter:create', 'encounter:check-in', 'encounter:cancel', 'order:submit', 'charge:pay', 'charge:refund', 'dispense:create', 'dispense:return'],
  register: ['encounter:create', 'encounter:check-in', 'encounter:cancel'],
  diagnosis: ['diagnose:write'],
  order: ['order:submit'],
  charge: ['charge:pay'],
  dispense: ['dispense:create', 'dispense:return'],
  refund: ['dispense:return', 'charge:refund'],
})

// 2026-10-12 P1-H05-PATIENT-UI 建档已可保存和查询。
// 2026-10-12 P1-H05-REGISTER-UI 挂号可选科室和医生。
// 2026-10-12 P1-H05-ORDER-UI 开单可按挂号暂存和提交。
// 2026-10-12 P1-H05-CHARGE-UI 收费可预览、现金支付、查询和退费。
// 2026-10-14 P2-1 退药/退费复用发药与收费工作区，保持现金退回和已发药先退药规则。
// status 为 disabled 时不提供路由，避免跳进空白业务页。
export const OUTPATIENT_MODULES = [
  {
    key: 'patient',
    name: '患者建档',
    blurb: '建档、查询主索引',
    status: 'available',
    roles: ['REGISTRAR', 'DOCTOR', 'CASHIER', 'PHARMACIST', 'SYSTEM'],
    permissions: OUTPATIENT_PERMISSION_CODES.patient,
    recommendFor: ['REGISTRAR'],
  },
  {
    key: 'register',
    name: '挂号',
    blurb: '须先有患者；选科室和医生',
    status: 'available',
    roles: ['REGISTRAR', 'SYSTEM'],
    permissions: OUTPATIENT_PERMISSION_CODES.register,
    recommendFor: ['REGISTRAR'],
  },
  {
    key: 'diagnosis',
    name: '诊断',
    blurb: '按挂号录入，可暂存、提交或作废',
    status: 'available',
    roles: ['DOCTOR', 'SYSTEM'],
    permissions: OUTPATIENT_PERMISSION_CODES.diagnosis,
    recommendFor: ['DOCTOR'],
  },
  {
    key: 'order',
    name: '开单',
    blurb: '须先挂号；可暂存、提交',
    status: 'available',
    roles: ['DOCTOR', 'SYSTEM'],
    permissions: OUTPATIENT_PERMISSION_CODES.order,
    recommendFor: ['DOCTOR'],
  },
  {
    key: 'charge',
    name: '收费',
    blurb: '现金一次付清；无挂号拒收',
    status: 'available',
    roles: ['CASHIER', 'SYSTEM'],
    permissions: OUTPATIENT_PERMISSION_CODES.charge,
    recommendFor: ['CASHIER'],
  },
  {
    key: 'dispense',
    name: '发药',
    blurb: '须已收费；可部分发、退药',
    status: 'available',
    roles: ['PHARMACIST', 'SYSTEM'],
    permissions: OUTPATIENT_PERMISSION_CODES.dispense,
    recommendFor: ['PHARMACIST'],
  },
  {
    key: 'refund',
    name: '退药/退费',
    blurb: '已发药须先退药再退费',
    status: 'available',
    roles: ['CASHIER', 'PHARMACIST', 'SYSTEM'],
    permissions: OUTPATIENT_PERMISSION_CODES.refund,
    recommendFor: ['CASHIER', 'PHARMACIST'],
  },
]

export function modulePath(key) {
  return `/app/outpatient/${key}`
}

export function findModule(key) {
  return OUTPATIENT_MODULES.find((item) => item.key === key) || null
}

export function roleText(roles) {
  if (!Array.isArray(roles) || roles.length === 0) return '未分配角色'
  return roles.map((role) => ROLE_LABELS[role] || role).join('、')
}

function strings(values) {
  return Array.isArray(values) ? values.filter((value) => typeof value === 'string').map((value) => value.trim()).filter(Boolean) : []
}

function accessSubject(value) {
  if (Array.isArray(value)) return { roles: strings(value), legacy: true }
  if (!value || typeof value !== 'object') return { roles: [], legacy: false }
  const permissions = strings(value.permissions)
  const scopes = strings(value.scopes)
  const deniedPermissions = strings(value.deniedPermissions)
  const deniedScopes = strings(value.deniedScopes)
  // A real UserDto always owns permissions/scopes, including an empty array
  // after a deny-all override. Only older role-only callers may use fallback.
  const hasPermissionData = value.accessControlLoaded === true
    || Object.prototype.hasOwnProperty.call(value, 'permissions')
    || Object.prototype.hasOwnProperty.call(value, 'scopes')
    || Object.prototype.hasOwnProperty.call(value, 'deniedPermissions')
    || Object.prototype.hasOwnProperty.call(value, 'deniedScopes')
  return {
    roles: strings(value.roles),
    permissions,
    scopes,
    deniedPermissions,
    deniedScopes,
    elevated: value.super === true || value.developer === true,
    legacy: !hasPermissionData,
  }
}

function scopeModule(scope) {
  const separator = scope.indexOf(':')
  return separator < 0 ? null : scope.slice(0, separator)
}

function scopeMatchesModule(scope, moduleCode) {
  const module = scopeModule(scope)
  return module === '' || module === moduleCode
}

function deniedByScope(subject, moduleCode) {
  return subject.deniedScopes.some((scope) => scopeMatchesModule(scope, moduleCode))
}

// A denied permission is represented by its absence from the server's
// effective permission list. The optional denied fields are accepted for
// forward-compatible/mock sessions that expose the override detail directly.
export function hasEffectivePermission(value, permissionCodes, moduleCode = 'outpatient') {
  const subject = accessSubject(value)
  const required = strings(permissionCodes)
  if (!required.length) return false
  if (subject.deniedPermissions.includes('admin:*') || required.some((code) => subject.deniedPermissions.includes(code))) return false
  if (deniedByScope(subject, moduleCode)) return false
  if (subject.elevated) return true
  if (subject.legacy) return subject.roles.some((role) => OUTPATIENT_MODULES.some((item) => item.roles.includes(role) && item.permissions.some((code) => required.includes(code))))
  const permissions = new Set(subject.permissions)
  if (!required.some((code) => permissions.has(code) || permissions.has('admin:*'))) return false
  if (!subject.scopes.length) return true
  return subject.scopes.some((scope) => scopeMatchesModule(scope, moduleCode))
}

export function canEnter(module, value) {
  if (!module || module.status === 'disabled') return false
  const subject = accessSubject(value)
  if (subject.legacy) return module.roles.some((role) => subject.roles.includes(role))
  return hasEffectivePermission(value, module.permissions || [], 'outpatient')
}

export function blockReason(module, value) {
  if (!module || module.status === 'disabled') return '未启用'
  if (!canEnter(module, value)) return '无权限'
  return ''
}

export function decorateModules(value) {
  const subject = accessSubject(value)
  const granted = new Set(subject.roles)
  return OUTPATIENT_MODULES.map((item) => {
    const allowed = canEnter(item, value)
    const recommended = allowed && item.recommendFor.some((role) => granted.has(role))
    return {
      ...item,
      allowed,
      recommended,
      reason: blockReason(item, value),
      statusLabel: STATUS_LABELS[item.status] || item.status,
      path: modulePath(item.key),
    }
  })
}
