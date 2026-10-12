// 2026-10-12 P1-H05-HOME 工作台入口顺序与占位规则
import { readFileSync } from 'node:fs'
import {
  OUTPATIENT_MODULES,
  blockReason,
  canEnter,
  decorateModules,
  modulePath,
  roleText,
} from '../src/workbench/catalog.js'

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

const names = ['患者建档', '挂号', '开单', '收费', '发药', '退药/退费']
assert(OUTPATIENT_MODULES.map((item) => item.name).join(',') === names.join(','), 'module order')

const keys = ['patient', 'register', 'order', 'charge', 'dispense', 'refund']
assert(OUTPATIENT_MODULES.map((item) => item.key).join(',') === keys.join(','), 'module keys')
for (const key of keys) {
  assert(modulePath(key) === `/app/outpatient/${key}`, `path ${key}`)
}

for (const item of OUTPATIENT_MODULES) {
  assert(item.status === 'developing', `${item.key} status`)
  assert(item.status !== 'disabled', `${item.key} should stay enterable`)
}

const system = decorateModules(['SYSTEM'])
assert(system.every((item) => item.allowed && item.statusLabel === '开发中'), 'system can open all')
assert(system.every((item) => !item.recommended), 'system has no single recommendation')

const registrar = decorateModules(['REGISTRAR'])
assert(registrar.find((item) => item.key === 'patient').allowed, 'registrar patient')
assert(registrar.find((item) => item.key === 'register').recommended, 'registrar register')
assert(!registrar.find((item) => item.key === 'charge').allowed, 'registrar charge locked')
assert(registrar.find((item) => item.key === 'charge').reason === '无权限', 'registrar reason')

const doctor = decorateModules(['DOCTOR'])
assert(doctor.find((item) => item.key === 'order').recommended, 'doctor order')
assert(!doctor.find((item) => item.key === 'dispense').allowed, 'doctor dispense locked')

const cashier = decorateModules(['CASHIER'])
assert(cashier.find((item) => item.key === 'charge').allowed, 'cashier charge')
assert(cashier.find((item) => item.key === 'refund').recommended, 'cashier refund')
assert(!cashier.find((item) => item.key === 'order').allowed, 'cashier order locked')

const pharmacist = decorateModules(['PHARMACIST'])
assert(pharmacist.find((item) => item.key === 'dispense').recommended, 'pharmacist dispense')
assert(pharmacist.find((item) => item.key === 'refund').allowed, 'pharmacist refund')
assert(!pharmacist.find((item) => item.key === 'register').allowed, 'pharmacist register locked')

assert(decorateModules([]).every((item) => !item.allowed), 'empty roles locked')
assert(roleText(['SYSTEM']) === '系统', 'system role label')
assert(roleText(['CASHIER', 'PHARMACIST']) === '收费员、药师', 'joined roles')
assert(!roleText(['SYSTEM']).toLowerCase().includes('admin'), 'role label')

assert(canEnter({ status: 'disabled', roles: ['SYSTEM'] }, ['SYSTEM']) === false, 'disabled never navigates')
assert(blockReason({ status: 'disabled', roles: ['SYSTEM'] }, ['SYSTEM']) === '未启用', 'disabled reason')

const uiFiles = [
  '../src/components/AppShell.vue',
  '../src/views/WorkbenchView.vue',
  '../src/views/ModulePlaceholderView.vue',
]
const ui = uiFiles
  .map((file) => readFileSync(new URL(file, import.meta.url), 'utf8'))
  .join('\n')
  .replace(/<template>/g, '\n<template>')
  .match(/<template>[\s\S]*?<\/template>/g)
  .join('\n')
const required = [
  'B-HIS',
  '门诊业务',
  '建档 → 挂号 → 开单 → 收费 → 发药 → 退药/退费',
  '收费前须有挂号/就诊 · 支付演示现金 · 已发药先退药再退费',
  '个人设置',
  '退出登录',
  '接口就绪后开放',
  '搜索患者',
  '二期',
]
for (const text of required) assert(ui.includes(text), `missing copy: ${text}`)
for (const text of ['住院', '病历', '报表', '采购', 'admin', 'H05', 'H02', 'H03']) {
  assert(!ui.includes(text), `forbidden copy: ${text}`)
}

console.log('workbench checks passed')
