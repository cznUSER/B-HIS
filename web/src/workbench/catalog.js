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

// 门诊接口尚未接入：一期六步都可进入空态，徽标为开发中。
// status 为 disabled 时不提供路由，避免跳进空白业务页。
export const OUTPATIENT_MODULES = [
  {
    key: 'patient',
    name: '患者建档',
    blurb: '建档、查询主索引',
    status: 'developing',
    roles: ['REGISTRAR', 'DOCTOR', 'CASHIER', 'PHARMACIST', 'SYSTEM'],
    recommendFor: ['REGISTRAR'],
  },
  {
    key: 'register',
    name: '挂号',
    blurb: '须先有患者；号源简化可后续',
    status: 'developing',
    roles: ['REGISTRAR', 'SYSTEM'],
    recommendFor: ['REGISTRAR'],
  },
  {
    key: 'order',
    name: '开单',
    blurb: '须有挂号/就诊',
    status: 'developing',
    roles: ['DOCTOR', 'SYSTEM'],
    recommendFor: ['DOCTOR'],
  },
  {
    key: 'charge',
    name: '收费',
    blurb: '现金演示；无挂号拒收',
    status: 'developing',
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
