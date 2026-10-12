// 2026-10-12 P1-H05-HOME-DASH 登录后侧栏。六步都在「门诊业务」；档案建档走同一路由。

export const NAV = [
  { key: 'home', label: '首页', to: { name: 'home' } },
  {
    key: 'archive',
    label: '档案管理',
    children: [
      { key: 'archive-patient', label: '患者建档', module: 'patient' },
      { key: 'archive-card', label: '就诊卡/账户', phase: '二期' },
    ],
  },
  {
    key: 'outpatient',
    label: '门诊业务',
    overview: { label: '六步总览', to: { name: 'workbench' } },
    children: [
      { key: 'patient', label: '建档', module: 'patient' },
      { key: 'register', label: '挂号', module: 'register' },
      { key: 'order', label: '开单', module: 'order' },
      { key: 'charge', label: '收费', module: 'charge' },
      { key: 'dispense', label: '发药', module: 'dispense' },
      { key: 'refund', label: '退药/退费', module: 'refund' },
    ],
  },
  {
    key: 'settings',
    label: '系统设置',
    children: [
      { key: 'settings-users', label: '用户与权限', phase: '二期' },
      { key: 'settings-dict', label: '基础字典', phase: '二期' },
    ],
  },
]

export function resolveNavChild(child, modules) {
  if (!child || child.phase) {
    return { enabled: false, reason: child && child.phase ? child.phase : '二期', to: null }
  }
  if (child.to) return { enabled: true, reason: '', to: child.to }
  const mod = (Array.isArray(modules) ? modules : []).find((item) => item.key === child.module)
  if (!mod || !mod.allowed) {
    return { enabled: false, reason: mod && mod.reason ? mod.reason : '无权限', to: null }
  }
  return {
    enabled: true,
    reason: '',
    to: { name: 'outpatient-module', params: { module: child.module } },
  }
}
