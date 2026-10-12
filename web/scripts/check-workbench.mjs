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
import {
  CHARGE_METRICS,
  FEE_SLICES,
  PAY_SLICES,
  PERIODS,
  PLACEHOLDER_VALUE,
  TODO_CARDS,
  VISIT_METRICS,
} from '../src/workbench/dashboard.js'
import { NAV, resolveNavChild } from '../src/workbench/nav.js'

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

const usable = new Set(['patient', 'register', 'order', 'charge', 'dispense'])
for (const item of OUTPATIENT_MODULES) {
  if (usable.has(item.key)) assert(item.status === 'available', `${item.key} is usable`)
  else assert(item.status === 'developing', `${item.key} status`)
  assert(item.status !== 'disabled', `${item.key} should stay enterable`)
}

const system = decorateModules(['SYSTEM'])
assert(system.every((item) => item.allowed), 'system can open all')
assert(system.find((item) => item.key === 'patient').statusLabel === '可用', 'patient badge')
assert(system.find((item) => item.key === 'register').statusLabel === '可用', 'register badge')
assert(system.find((item) => item.key === 'order').statusLabel === '可用', 'order badge')
assert(
  system.filter((item) => !usable.has(item.key)).every((item) => item.statusLabel === '开发中'),
  'other modules stay developing',
)
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

assert(PLACEHOLDER_VALUE === 0, 'placeholder is zero')
assert(PERIODS.join(',') === '今天,昨天,本周,本月', 'periods')
assert(CHARGE_METRICS.join(',') === '营业收入,现金,微信,支付宝,银行卡,医保,减免', 'charge metrics')
assert(VISIT_METRICS.join(',') === '营业收入,门诊客流量,门诊客单价', 'visit metrics')
assert(FEE_SLICES.map((item) => item.label).join(',') === '检验,检查,西成药,中草药,治疗及其他,卫材', 'fee slices')
assert(PAY_SLICES.map((item) => item.label).join(',') === '现金,微信,支付宝,银行卡,医保,减免', 'pay slices')
for (const slice of [...FEE_SLICES, ...PAY_SLICES]) {
  assert(!Object.prototype.hasOwnProperty.call(slice, 'value'), `${slice.key} has no magnitude`)
}

const todoLabels = ['待收费', '待发药', '进销存未上传', '未贯标提醒', '库存预警']
assert(TODO_CARDS.map((item) => item.label).join(',') === todoLabels.join(','), 'todo order')
assert(TODO_CARDS.find((item) => item.key === 'charge').module === 'charge', 'todo charge link')
assert(TODO_CARDS.find((item) => item.key === 'dispense').module === 'dispense', 'todo dispense link')
assert(TODO_CARDS.find((item) => item.key === 'charge').value === 0, 'todo charge zero')
assert(TODO_CARDS.find((item) => item.key === 'dispense').value === 0, 'todo dispense zero')
for (const key of ['upload', 'standard']) {
  const card = TODO_CARDS.find((item) => item.key === key)
  assert(card.phase === '二期' && card.value === null && !card.module, `${key} is deferred`)
}
const stock = TODO_CARDS.find((item) => item.key === 'stock')
assert(stock.value === 0 && stock.phase === '占位' && !stock.module, 'stock is a weak zero')
for (const card of TODO_CARDS) {
  if (typeof card.value === 'number') assert(card.value === 0, `${card.key} value`)
}

const home = NAV.find((item) => item.key === 'home')
assert(home.to.name === 'home', 'home nav')
const archive = NAV.find((item) => item.key === 'archive')
assert(archive.children[0].label === '患者建档' && archive.children[0].module === 'patient', 'archive patient')
assert(archive.children[1].phase === '二期' && !archive.children[1].module, 'archive card deferred')
const outpatient = NAV.find((item) => item.key === 'outpatient')
assert(
  outpatient.children.map((item) => item.label).join(',') === '建档,挂号,开单,收费,发药,退药/退费',
  'sidebar steps',
)
assert(outpatient.children.map((item) => item.module).join(',') === keys.join(','), 'sidebar modules')
assert(outpatient.overview.to.name === 'workbench', 'overview route name')
const settings = NAV.find((item) => item.key === 'settings')
assert(settings.children.every((item) => item.phase === '二期' && !item.module && !item.to), 'settings stay closed')

const labels = []
for (const item of NAV) {
  labels.push(item.label)
  for (const child of item.children || []) labels.push(child.label)
  if (item.overview) labels.push(item.overview.label)
}
for (const word of ['住院', '病历', '采购', '报表']) {
  assert(!labels.some((label) => label.includes(word)), `nav ${word}`)
}

const systemNav = decorateModules(['SYSTEM'])
const registrarNav = decorateModules(['REGISTRAR'])
const chargeChild = outpatient.children.find((item) => item.module === 'charge')
const patientChild = outpatient.children.find((item) => item.module === 'patient')
const archivePatient = archive.children[0]
assert(resolveNavChild(patientChild, systemNav).to.params.module === 'patient', 'patient route')
assert(resolveNavChild(archivePatient, systemNav).to.params.module === 'patient', 'archive shares patient')
assert(resolveNavChild(chargeChild, registrarNav).enabled === false, 'registrar cannot open charge from nav')
assert(resolveNavChild(chargeChild, systemNav).enabled === true, 'system can open charge from nav')
assert(resolveNavChild(archive.children[1], systemNav).to === null, 'phase item has no route')
assert(resolveNavChild(settings.children[0], systemNav).to === null, 'settings item has no route')

const uiFiles = [
  '../src/components/AppShell.vue',
  '../src/components/SideNav.vue',
  '../src/components/DonutPlaceholder.vue',
  '../src/views/DashboardView.vue',
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
  '首页',
  '工作台',
  '门诊业务',
  '建档 → 挂号 → 开单 → 收费 → 发药 → 退药/退费',
  '收费前须有挂号/就诊 · 支付演示现金 · 已发药先退药再退费',
  '个人设置',
  '退出登录',
  '接口就绪后开放',
  '机构',
  '人员',
  '演示机构',
  '经营概况',
  '演示 / 占位',
  '待办',
  '刷新',
  '收入来源分布',
  '支付方式分布',
  '二期',
]
for (const text of required) assert(ui.includes(text), `missing copy: ${text}`)
for (const text of ['住院', '病历', '报表', '采购', 'admin', 'H05', 'H02', 'H03', '搜索患者', '1959']) {
  assert(!ui.includes(text), `forbidden copy: ${text}`)
}
assert(!ui.includes('module-grid') || readFileSync(new URL('../src/views/DashboardView.vue', import.meta.url), 'utf8').includes('module-grid') === false, 'dashboard is not the card wall')
assert(readFileSync(new URL('../src/views/DashboardView.vue', import.meta.url), 'utf8').includes('module-grid') === false, 'dashboard template has no card wall')
assert(readFileSync(new URL('../src/views/WorkbenchView.vue', import.meta.url), 'utf8').includes('module-grid'), 'overview keeps six cards')

const routerSrc = readFileSync(new URL('../src/router/index.js', import.meta.url), 'utf8')
assert(/path:\s*''[\s\S]{0,180}name:\s*'home'/.test(routerSrc), 'default child is home')
assert(/path:\s*'outpatient'[\s\S]{0,180}name:\s*'workbench'/.test(routerSrc), 'overview is /app/outpatient')
assert(/name:\s*'home'/.test(readFileSync(new URL('../src/views/LoginView.vue', import.meta.url), 'utf8')), 'login lands on home')
assert(!readFileSync(new URL('../src/views/LoginView.vue', import.meta.url), 'utf8').includes("name: 'workbench'"), 'login skips card wall')

console.log('workbench checks passed')
