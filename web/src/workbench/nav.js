// 2026-10-16 患者建档与查询作为档案管理下的独立入口。
// 2026-10-13 P1-H05-MASTER-UI 「基础字典」改为可进入的「基础数据」。

import { canMaintainMaster } from '../master/access.js'
import { visibleAccessTabs } from '../master/accessControlPolicy.js'

export const NAV = [
  { key: 'home', label: '首页', to: { name: 'home' } },
  {
    key: 'archive',
    label: '档案管理',
    children: [
      { key: 'archive-patient-create', label: '患者建档', module: 'patient', to: { name: 'patient-create' } },
      { key: 'archive-patient-search', label: '患者查询', module: 'patient', to: { name: 'patient-search' } },
      { key: 'archive-card', label: '就诊卡/账户', phase: '二期' },
    ],
  },
  {
    key: 'outpatient',
    label: '门诊业务',
    overview: { label: '门诊总览', to: { name: 'workbench' } },
    children: [
      { key: 'register', label: '挂号', module: 'register' },
      { key: 'diagnosis', label: '诊断', module: 'diagnosis' },
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
      { key: 'settings-master', label: '基础数据', to: { name: 'master' }, access: 'master' },
      { key: 'settings-users', label: '用户与权限', to: { name: 'access-users' }, access: 'users' },
    ],
  },
]

export function resolveNavChild(child, modules, user) {
  if (!child || child.phase) {
    return { enabled: false, hidden: false, reason: child && child.phase ? child.phase : '二期', to: null }
  }
  if (child.access === 'master') {
    if (!canMaintainMaster(user)) {
      return { enabled: false, hidden: true, reason: '无权限', to: null }
    }
    return { enabled: true, hidden: false, reason: '', to: child.to }
  }
  if (child.access === 'users') {
    const firstAllowedTab = visibleAccessTabs(user)[0]
    if (!firstAllowedTab) {
      return { enabled: false, hidden: true, reason: '无权限', to: null }
    }
    return { enabled: true, hidden: false, reason: '', to: { name: firstAllowedTab.name } }
  }
  if (child.module) {
    const mod = (Array.isArray(modules) ? modules : []).find((item) => item.key === child.module)
    if (!mod || !mod.allowed) {
      return { enabled: false, hidden: false, reason: mod && mod.reason ? mod.reason : '无权限', to: null }
    }
    return {
      enabled: true,
      hidden: false,
      reason: '',
      to: child.to || { name: 'outpatient-module', params: { module: child.module } },
    }
  }
  if (child.to) return { enabled: true, hidden: false, reason: '', to: child.to }
  return { enabled: false, hidden: false, reason: '无权限', to: null }
}
