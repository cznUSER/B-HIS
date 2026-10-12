// 2026-10-12 P1-H05-HOME 门诊工作台：六步顺序、徽标、按角色是否可进。

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

// 2026-10-12 P1-H05-PATIENT-UI 建档已可保存和查询。
// 2026-10-12 P1-H05-REGISTER-UI 挂号可选科室和医生。
// 2026-10-12 P1-H05-ORDER-UI 开单可按挂号暂存和提交。
// 2026-10-12 P1-H05-CHARGE-UI 收费可预览、现金支付、查询和退费。发药、退药仍是空态。
// status 为 disabled 时不提供路由，避免跳进空白业务页。
export const OUTPATIENT_MODULES = [
  {
    key: 'patient',
    name: '患者建档',
    blurb: '建档、查询主索引',
    status: 'available',
    roles: ['REGISTRAR', 'DOCTOR', 'CASHIER', 'PHARMACIST', 'SYSTEM'],
    recommendFor: ['REGISTRAR'],
  },
  {
    key: 'register',
    name: '挂号',
    blurb: '须先有患者；选科室和医生',
    status: 'available',
    roles: ['REGISTRAR', 'SYSTEM'],
    recommendFor: ['REGISTRAR'],
  },
  {
    key: 'order',
    name: '开单',
    blurb: '须先挂号；可暂存、提交',
    status: 'available',
    roles: ['DOCTOR', 'SYSTEM'],
    recommendFor: ['DOCTOR'],
  },
  {
    key: 'charge',
    name: '收费',
    blurb: '现金一次付清；无挂号拒收',
    status: 'available',
    roles: ['CASHIER', 'SYSTEM'],
    recommendFor: ['CASHIER'],
  },
  {
    key: 'dispense',
    name: '发药',
    blurb: '须已收费；简易库存',
    status: 'developing',
    roles: ['PHARMACIST', 'SYSTEM'],
    recommendFor: ['PHARMACIST'],
  },
  {
    key: 'refund',
    name: '退药/退费',
    blurb: '已发药须先退药再退费',
    status: 'developing',
    roles: ['CASHIER', 'PHARMACIST', 'SYSTEM'],
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

export function canEnter(module, roles) {
  if (!module || module.status === 'disabled') return false
  const granted = new Set(Array.isArray(roles) ? roles : [])
  return module.roles.some((role) => granted.has(role))
}

export function blockReason(module, roles) {
  if (!module || module.status === 'disabled') return '未启用'
  if (!canEnter(module, roles)) return '无权限'
  return ''
}

export function decorateModules(roles) {
  const granted = new Set(Array.isArray(roles) ? roles : [])
  return OUTPATIENT_MODULES.map((item) => {
    const allowed = canEnter(item, roles)
    const recommended = allowed && item.recommendFor.some((role) => granted.has(role))
    return {
      ...item,
      allowed,
      recommended,
      reason: blockReason(item, roles),
      statusLabel: STATUS_LABELS[item.status] || item.status,
      path: modulePath(item.key),
    }
  })
}
